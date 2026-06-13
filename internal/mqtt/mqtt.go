package mqttsvc

import (
	"encoding/json"
	"fmt"
	"log"
	"provision-server/internal/config"
	"provision-server/internal/db"
	"provision-server/internal/models"
	"provision-server/internal/services"
	"strconv"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type Service struct {
	client   mqtt.Client
	devSvc   *services.DeviceService
	evtLog   *services.EventLogService
	stopCh   chan struct{}
}

type HeartbeatMsg struct {
	DeviceSN    string `json:"device_sn"`
	FirmwareVer string `json:"fw_ver"`
	Uptime      int64  `json:"uptime"`
}

type OTAProgressMsg struct {
	DeviceSN   string `json:"device_sn"`
	FirmwareID uint   `json:"firmware_id"`
	Progress   int    `json:"progress"`
	Status     string `json:"status"`
	ErrMsg     string `json:"err_msg"`
}

var Instance *Service

func New() *Service {
	return &Service{
		devSvc: services.NewDeviceService(),
		evtLog: services.NewEventLog(),
		stopCh: make(chan struct{}),
	}
}

func (s *Service) Connect() error {
	opts := mqtt.NewClientOptions().
		AddBroker(config.App.MQTTBroker).
		SetClientID(config.App.MQTTClientID + "-" + strconv.Itoa(int(time.Now().Unix()))).
		SetAutoReconnect(true).
		SetConnectRetry(false).
		SetMaxReconnectInterval(60 * time.Second).
		SetConnectTimeout(10 * time.Second)

	if config.App.MQTTUser != "" {
		opts = opts.SetUsername(config.App.MQTTUser)
	}
	if config.App.MQTTPass != "" {
		opts = opts.SetPassword(config.App.MQTTPass)
	}

	opts.OnConnect = func(c mqtt.Client) {
		log.Println("[MQTT] connected")
		s.subscribe()
	}
	opts.OnConnectionLost = func(c mqtt.Client, err error) {
		log.Println("[MQTT] connection lost:", err)
	}
	opts.OnReconnecting = func(c mqtt.Client, o *mqtt.ClientOptions) {
		log.Println("[MQTT] reconnecting...")
	}

	s.client = mqtt.NewClient(opts)
	tok := s.client.Connect()
	if !tok.WaitTimeout(10 * time.Second) {
		return fmt.Errorf("mqtt connect timeout after 10s")
	}
	if tok.Error() != nil {
		return tok.Error()
	}
	Instance = s
	go s.watchdogLoop()
	return nil
}

func (s *Service) subscribe() {
	topics := map[string]byte{
		"device/+/heartbeat": 1,
		"device/+/ota/progress": 1,
		"device/+/event": 1,
	}
	for t, q := range topics {
		tok := s.client.Subscribe(t, q, s.onMessage)
		tok.Wait()
		if tok.Error() != nil {
			log.Println("[MQTT] subscribe", t, "error:", tok.Error())
		} else {
			log.Println("[MQTT] subscribed:", t)
		}
	}
}

func (s *Service) onMessage(c mqtt.Client, m mqtt.Message) {
	parts := strings.Split(m.Topic(), "/")
	if len(parts) < 3 {
		return
	}
	deviceSN := parts[1]
	action := parts[2]

	switch action {
	case "heartbeat":
		s.handleHeartbeat(deviceSN, m.Payload())
	case "ota":
		if len(parts) >= 4 && parts[3] == "progress" {
			s.handleOTAProgress(deviceSN, m.Payload())
		}
	case "event":
		_ = s.evtLog.Record(deviceSN, "device_event", "", "", string(m.Payload()))
	}
}

func (s *Service) handleHeartbeat(deviceSN string, payload []byte) {
	var msg HeartbeatMsg
	if err := json.Unmarshal(payload, &msg); err == nil {
		if msg.DeviceSN != "" {
			deviceSN = msg.DeviceSN
		}
	}
	if err := s.devSvc.Heartbeat(deviceSN); err != nil {
		log.Println("[MQTT] heartbeat error for", deviceSN, ":", err)
		return
	}
	if msg.FirmwareVer != "" {
		_ = db.DB.Model(&models.Device{}).Where("device_sn = ?", deviceSN).Update("firmware_ver", msg.FirmwareVer).Error
	}
}

func (s *Service) handleOTAProgress(deviceSN string, payload []byte) {
	var msg OTAProgressMsg
	if err := json.Unmarshal(payload, &msg); err != nil {
		return
	}
	if msg.DeviceSN != "" {
		deviceSN = msg.DeviceSN
	}
	d, err := s.devSvc.GetBySN(deviceSN)
	if err != nil {
		return
	}
	var job models.OTAJob
	if err := db.DB.Where("device_id = ? AND firmware_id = ?", d.ID, msg.FirmwareID).
		Order("id DESC").First(&job).Error; err != nil {
		return
	}
	updates := map[string]interface{}{
		"progress": msg.Progress,
		"status":   msg.Status,
	}
	if msg.ErrMsg != "" {
		updates["err_msg"] = msg.ErrMsg
	}
	if msg.Status == "downloading" && job.StartedAt == nil {
		now := time.Now()
		updates["started_at"] = &now
	}
	if msg.Status == "success" || msg.Status == "failed" {
		now := time.Now()
		updates["finished_at"] = &now
		if msg.Status == "success" {
			var fw models.Firmware
			if db.DB.First(&fw, msg.FirmwareID).Error == nil {
				_ = db.DB.Model(d).Update("firmware_ver", fw.Version).Error
			}
		}
	}
	_ = db.DB.Model(&job).Updates(updates).Error
}

func (s *Service) watchdogLoop() {
	tick := time.NewTicker(30 * time.Second)
	defer tick.Stop()
	for {
		select {
		case <-s.stopCh:
			return
		case <-tick.C:
			s.checkTimeouts()
		}
	}
}

func (s *Service) checkTimeouts() {
	ttl := time.Duration(config.App.HeartbeatTTL) * time.Second
	cutoff := time.Now().Add(-ttl)

	var ids []uint
	db.DB.Model(&models.Device{}).
		Where("status = ? AND last_heartbeat IS NOT NULL AND last_heartbeat < ?", models.StatusActivated, cutoff).
		Pluck("id", &ids)

	for _, id := range ids {
		var d models.Device
		if err := db.DB.First(&d, id).Error; err != nil {
			continue
		}
		_, err := s.devSvc.Transition(d.DeviceSN, models.StatusDeactivated,
			fmt.Sprintf("heartbeat timeout (> %ds)", config.App.HeartbeatTTL))
		if err != nil {
			log.Println("[MQTT] watchdog transition error:", d.DeviceSN, err)
		} else {
			log.Println("[MQTT] watchdog deactivated:", d.DeviceSN)
		}
	}
}

func (s *Service) Publish(topic string, payload interface{}) error {
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	tok := s.client.Publish(topic, 1, false, b)
	tok.Wait()
	return tok.Error()
}

func (s *Service) Close() {
	close(s.stopCh)
	if s.client != nil && s.client.IsConnected() {
		s.client.Disconnect(500)
	}
}
