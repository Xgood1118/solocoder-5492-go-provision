package handlers

import (
	"errors"
	"math/rand"
	"net/http"
	"provision-server/internal/db"
	"provision-server/internal/models"
	mqttsvc "provision-server/internal/mqtt"
	"provision-server/internal/services"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type PushOTAReq struct {
	FirmwareID uint     `json:"firmware_id" binding:"required"`
	DeviceSNs  []string `json:"device_sns"`
	BatchID    uint     `json:"batch_id"`
	ModelID    uint     `json:"model_id"`
	Region     string   `json:"region"`
	Customer   string   `json:"customer"`
	UseGray    bool     `json:"use_gray"`
}

func PushOTA(c *gin.Context) {
	var req PushOTAReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": err.Error()})
		return
	}
	result, code, msg := doPushOTA(&req)
	if code != 0 {
		c.JSON(http.StatusOK, gin.H{"code": code, "msg": msg})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "ok", "data": result})
}

func doPushOTA(req *PushOTAReq) (interface{}, int, string) {
	var fw models.Firmware
	if err := db.DB.First(&fw, req.FirmwareID).Error; err != nil {
		return nil, 1, "firmware not found"
	}

	devices, err := selectDevicesForOTA(req, &fw)
	if err != nil {
		return nil, 1, err.Error()
	}

	now := time.Now()
	successCount := 0
	failCount := 0

	for _, d := range devices {
		job := models.OTAJob{
			FirmwareID: fw.ID,
			DeviceID:   d.ID,
			Status:     "pending",
			Progress:   0,
			CreatedAt:  now,
			UpdatedAt:  now,
		}
		if err := db.DB.Create(&job).Error; err != nil {
			failCount++
			continue
		}

		if mqttsvc.Instance != nil {
			payload := map[string]interface{}{
				"job_id":       job.ID,
				"firmware_id":  fw.ID,
				"version":      fw.Version,
				"file_name":    fw.FileName,
				"file_size":    fw.FileSize,
				"md5_sum":      fw.MD5Sum,
				"min_version":  fw.MinVersion,
				"download_url": "",
			}
			topic := "device/" + d.DeviceSN + "/ota/upgrade"
			_ = mqttsvc.Instance.Publish(topic, payload)
		}
		successCount++
	}

	return gin.H{
		"total":   len(devices),
		"success": successCount,
		"failed":  failCount,
	}, 0, ""
}

func selectDevicesForOTA(req *PushOTAReq, fw *models.Firmware) ([]models.Device, error) {
	q := db.DB.Model(&models.Device{}).
		Where("status IN ?", []models.DeviceStatus{models.StatusActivated, models.StatusProvisioned, models.StatusDeactivated})

	if len(req.DeviceSNs) > 0 {
		q = q.Where("device_sn IN ?", req.DeviceSNs)
	}
	if req.BatchID > 0 {
		q = q.Where("batch_id = ?", req.BatchID)
	}
	if req.ModelID > 0 {
		q = q.Where("model_id = ?", req.ModelID)
	} else {
		q = q.Where("model_id = ?", fw.ModelID)
	}
	if req.Region != "" {
		q = q.Where("region = ?", req.Region)
	}
	if req.Customer != "" {
		q = q.Where("customer = ?", req.Customer)
	}

	var all []models.Device
	if err := q.Find(&all).Error; err != nil {
		return nil, err
	}

	if len(all) == 0 {
		return nil, errors.New("no devices matched")
	}

	if req.UseGray && fw.GrayRatio > 0 {
		ratio := fw.GrayRatio
		if ratio > 100 {
			ratio = 100
		}
		count := len(all) * ratio / 100
		if count < 1 {
			count = 1
		}
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		r.Shuffle(len(all), func(i, j int) {
			all[i], all[j] = all[j], all[i]
		})
		return all[:count], nil
	}

	return all, nil
}

func ListOTAJobs(c *gin.Context) {
	page, size := getPageSize(c)
	firmwareID := c.Query("firmware_id")
	deviceID := c.Query("device_id")
	status := c.Query("status")

	var list []models.OTAJob
	var total int64
	q := db.DB.Model(&models.OTAJob{})
	if firmwareID != "" {
		q = q.Where("firmware_id = ?", firmwareID)
	}
	if deviceID != "" {
		q = q.Where("device_id = ?", deviceID)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if err := q.Count(&total).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": err.Error()})
		return
	}
	offset := (page - 1) * size
	if err := q.Order("id DESC").Offset(offset).Limit(size).Find(&list).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "ok",
		"data": gin.H{
			"list":  list,
			"total": total,
			"page":  page,
			"size":  size,
		},
	})
}

func GetDeviceOTAStatus(c *gin.Context) {
	sn := c.Param("sn")
	svc := services.NewDeviceService()
	d, err := svc.GetBySN(sn)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "device not found"})
		return
	}
	var jobs []models.OTAJob
	if err := db.DB.Where("device_id = ?", d.ID).Order("id DESC").Limit(10).Find(&jobs).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "ok", "data": jobs})
}

func GetOTAJob(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var job models.OTAJob
	if err := db.DB.First(&job, id).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "OTA job not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "ok", "data": job})
}
