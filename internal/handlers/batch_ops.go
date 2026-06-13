package handlers

import (
	"net/http"
	"provision-server/internal/db"
	"provision-server/internal/models"
	"provision-server/internal/services"

	"github.com/gin-gonic/gin"
)

type BatchProvisionReq struct {
	Devices    []BatchProvisionItem `json:"devices" binding:"required"`
	TemplateID uint                 `json:"template_id"`
	BatchID    uint                 `json:"batch_id"`
	ModelID    uint                 `json:"model_id"`
	Customer   string               `json:"customer"`
	Region     string               `json:"region"`
}

type BatchProvisionItem struct {
	DeviceSN  string `json:"device_sn" binding:"required"`
	DeviceMAC string `json:"device_mac"`
}

func BatchProvision(c *gin.Context) {
	var req BatchProvisionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": err.Error()})
		return
	}

	if len(req.Devices) == 0 {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "devices required"})
		return
	}

	svc := services.NewProvisionService()
	results := make([]gin.H, 0, len(req.Devices))
	success := 0
	failed := 0

	for _, item := range req.Devices {
		provReq := &services.ScanProvisionRequest{
			DeviceSN:   item.DeviceSN,
			DeviceMAC:  item.DeviceMAC,
			ModelID:    req.ModelID,
			BatchID:    req.BatchID,
			TemplateID: req.TemplateID,
			Customer:   req.Customer,
			Region:     req.Region,
		}
		_, err := svc.ScanProvision(provReq)
		if err != nil {
			failed++
			results = append(results, gin.H{
				"device_sn": item.DeviceSN,
				"success":   false,
				"error":     err.Error(),
			})
		} else {
			success++
			results = append(results, gin.H{
				"device_sn": item.DeviceSN,
				"success":   true,
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "ok",
		"data": gin.H{
			"total":   len(req.Devices),
			"success": success,
			"failed":  failed,
			"results": results,
		},
	})
}

type BatchOTAReq struct {
	FirmwareID uint     `json:"firmware_id" binding:"required"`
	DeviceSNs  []string `json:"device_sns"`
	BatchID    uint     `json:"batch_id"`
	ModelID    uint     `json:"model_id"`
	Region     string   `json:"region"`
	Customer   string   `json:"customer"`
	UseGray    bool     `json:"use_gray"`
}

func BatchOTA(c *gin.Context) {
	var req BatchOTAReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": err.Error()})
		return
	}
	otaReq := PushOTAReq{
		FirmwareID: req.FirmwareID,
		DeviceSNs:  req.DeviceSNs,
		BatchID:    req.BatchID,
		ModelID:    req.ModelID,
		Region:     req.Region,
		Customer:   req.Customer,
		UseGray:    req.UseGray,
	}
	c.Set("ota_req", &otaReq)
	PushOTA(c)
}

type BatchStatusChangeReq struct {
	DeviceSNs []string `json:"device_sns"`
	BatchID   uint     `json:"batch_id"`
	ModelID   uint     `json:"model_id"`
	Region    string   `json:"region"`
	Customer  string   `json:"customer"`
	Status    string   `json:"status" binding:"required"`
	Reason    string   `json:"reason"`
}

func BatchStatusChange(c *gin.Context) {
	var req BatchStatusChangeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": err.Error()})
		return
	}

	q := db.DB.Model(&models.Device{})
	if len(req.DeviceSNs) > 0 {
		q = q.Where("device_sn IN ?", req.DeviceSNs)
	}
	if req.BatchID > 0 {
		q = q.Where("batch_id = ?", req.BatchID)
	}
	if req.ModelID > 0 {
		q = q.Where("model_id = ?", req.ModelID)
	}
	if req.Region != "" {
		q = q.Where("region = ?", req.Region)
	}
	if req.Customer != "" {
		q = q.Where("customer = ?", req.Customer)
	}

	var devices []models.Device
	if err := q.Find(&devices).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": err.Error()})
		return
	}

	if len(devices) == 0 {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "no devices matched"})
		return
	}

	svc := services.NewDeviceService()
	success := 0
	failed := 0
	results := make([]gin.H, 0, len(devices))

	for _, d := range devices {
		_, err := svc.Transition(d.DeviceSN, models.DeviceStatus(req.Status), req.Reason)
		if err != nil {
			failed++
			results = append(results, gin.H{
				"device_sn": d.DeviceSN,
				"success":   false,
				"error":     err.Error(),
			})
		} else {
			success++
			results = append(results, gin.H{
				"device_sn": d.DeviceSN,
				"success":   true,
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "ok",
		"data": gin.H{
			"total":   len(devices),
			"success": success,
			"failed":  failed,
			"results": results,
		},
	})
}
