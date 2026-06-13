package services

import (
	"provision-server/internal/db"
	"provision-server/internal/models"
	"provision-server/internal/statemachine"
	"time"
)

type EventLogService struct{}

func NewEventLog() *EventLogService {
	return &EventLogService{}
}

func (s *EventLogService) Record(deviceSN, eventType, from, to, msg string) error {
	log := &models.EventLog{
		DeviceSN:   deviceSN,
		EventType:  eventType,
		FromStatus: from,
		ToStatus:   to,
		Message:    msg,
		CreatedAt:  time.Now(),
	}
	return db.DB.Create(log).Error
}

type DeviceService struct {
	eventLog *EventLogService
}

func NewDeviceService() *DeviceService {
	return &DeviceService{eventLog: NewEventLog()}
}

func (s *DeviceService) Create(d *models.Device) error {
	d.Status = models.StatusPending
	return db.DB.Create(d).Error
}

func (s *DeviceService) GetBySN(sn string) (*models.Device, error) {
	var d models.Device
	err := db.DB.Where("device_sn = ?", sn).Preload("Tags").First(&d).Error
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (s *DeviceService) Transition(deviceSN string, to models.DeviceStatus, reason string) (*models.Device, error) {
	d, err := s.GetBySN(deviceSN)
	if err != nil {
		return nil, err
	}
	from := d.Status
	if err := statemachine.ValidateTransition(from, to); err != nil {
		return nil, err
	}
	now := time.Now()
	d.Status = to
	switch to {
	case models.StatusProvisioned:
		d.ProvisionedAt = &now
	case models.StatusActivated:
		if d.FirstActiveAt == nil {
			d.FirstActiveAt = &now
		}
		d.LastHeartbeat = &now
	case models.StatusRetired:
		d.RetiredAt = &now
	}
	if err := db.DB.Save(d).Error; err != nil {
		return nil, err
	}
	_ = s.eventLog.Record(deviceSN, "status_change", string(from), string(to), reason)
	return d, nil
}

func (s *DeviceService) Heartbeat(deviceSN string) error {
	d, err := s.GetBySN(deviceSN)
	if err != nil {
		return err
	}
	now := time.Now()
	if d.Status == models.StatusProvisioned {
		_, _ = s.Transition(deviceSN, models.StatusActivated, "first heartbeat auto activate")
	} else if d.Status == models.StatusDeactivated {
		_, _ = s.Transition(deviceSN, models.StatusActivated, "heartbeat resume from deactivated")
	}
	d.LastHeartbeat = &now
	return db.DB.Model(&models.Device{}).Where("id = ?", d.ID).Update("last_heartbeat", now).Error
}

func (s *DeviceService) List(page, size int, status, batchID, modelID, keyword, tag, region, customer string) ([]models.Device, int64, error) {
	var list []models.Device
	var total int64
	q := db.DB.Model(&models.Device{})
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if batchID != "" {
		q = q.Where("batch_id = ?", batchID)
	}
	if modelID != "" {
		q = q.Where("model_id = ?", modelID)
	}
	if keyword != "" {
		q = q.Where("device_sn LIKE ? OR device_mac LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}
	if region != "" {
		q = q.Where("region = ?", region)
	}
	if customer != "" {
		q = q.Where("customer = ?", customer)
	}
	if tag != "" {
		q = q.Joins("JOIN device_tags ON device_tags.device_id = devices.id JOIN tags ON tags.id = device_tags.tag_id").
			Where("tags.name = ?", tag)
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * size
	if offset < 0 {
		offset = 0
	}
	err := q.Preload("Tags").Order("id DESC").Offset(offset).Limit(size).Find(&list).Error
	return list, total, err
}
