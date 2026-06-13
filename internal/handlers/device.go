package handlers

import (
	"net/http"
	"provision-server/internal/db"
	"provision-server/internal/models"
	"provision-server/internal/services"
	"strconv"

	"github.com/gin-gonic/gin"
)

func getPageSize(c *gin.Context) (int, int) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	if size <= 0 {
		size = 20
	}
	if page <= 0 {
		page = 1
	}
	return page, size
}

func ListDevices(c *gin.Context) {
	page, size := getPageSize(c)
	status := c.Query("status")
	batchID := c.Query("batch_id")
	modelID := c.Query("model_id")
	keyword := c.Query("keyword")
	tag := c.Query("tag")
	region := c.Query("region")
	customer := c.Query("customer")

	svc := services.NewDeviceService()
	list, total, err := svc.List(page, size, status, batchID, modelID, keyword, tag, region, customer)
	if err != nil {
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

func GetDevice(c *gin.Context) {
	sn := c.Param("sn")
	svc := services.NewDeviceService()
	d, err := svc.GetBySN(sn)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "device not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "ok", "data": d})
}

type CreateDeviceReq struct {
	DeviceSN   string `json:"device_sn" binding:"required"`
	DeviceMAC  string `json:"device_mac"`
	ModelID    uint   `json:"model_id" binding:"required"`
	BatchID    uint   `json:"batch_id"`
	TemplateID uint   `json:"template_id"`
	Customer   string `json:"customer"`
	Region     string `json:"region"`
	Remark     string `json:"remark"`
}

func CreateDevice(c *gin.Context) {
	var req CreateDeviceReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": err.Error()})
		return
	}
	svc := services.NewDeviceService()
	d := &models.Device{
		DeviceSN:   req.DeviceSN,
		DeviceMAC:  req.DeviceMAC,
		ModelID:    req.ModelID,
		BatchID:    req.BatchID,
		TemplateID: req.TemplateID,
		Customer:   req.Customer,
		Region:     req.Region,
		Remark:     req.Remark,
	}
	if err := svc.Create(d); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "ok", "data": d})
}

type UpdateDeviceReq struct {
	DeviceMAC  string `json:"device_mac"`
	ModelID    uint   `json:"model_id"`
	BatchID    uint   `json:"batch_id"`
	TemplateID uint   `json:"template_id"`
	Customer   string `json:"customer"`
	Region     string `json:"region"`
	Remark     string `json:"remark"`
}

func UpdateDevice(c *gin.Context) {
	sn := c.Param("sn")
	var req UpdateDeviceReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": err.Error()})
		return
	}
	svc := services.NewDeviceService()
	d, err := svc.GetBySN(sn)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "device not found"})
		return
	}
	if req.DeviceMAC != "" {
		d.DeviceMAC = req.DeviceMAC
	}
	if req.ModelID > 0 {
		d.ModelID = req.ModelID
	}
	if req.BatchID > 0 {
		d.BatchID = req.BatchID
	}
	if req.TemplateID > 0 {
		d.TemplateID = req.TemplateID
	}
	if req.Customer != "" {
		d.Customer = req.Customer
	}
	if req.Region != "" {
		d.Region = req.Region
	}
	if req.Remark != "" {
		d.Remark = req.Remark
	}
	if err := db.DB.Save(d).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "ok", "data": d})
}

func DeleteDevice(c *gin.Context) {
	sn := c.Param("sn")
	svc := services.NewDeviceService()
	d, err := svc.GetBySN(sn)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "device not found"})
		return
	}
	if err := db.DB.Delete(d).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "ok"})
}

type TransitionReq struct {
	Status string `json:"status" binding:"required"`
	Reason string `json:"reason"`
}

func TransitionDevice(c *gin.Context) {
	sn := c.Param("sn")
	var req TransitionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": err.Error()})
		return
	}
	svc := services.NewDeviceService()
	d, err := svc.Transition(sn, models.DeviceStatus(req.Status), req.Reason)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "ok", "data": d})
}

func ListDeviceEvents(c *gin.Context) {
	sn := c.Param("sn")
	page, size := getPageSize(c)
	var list []models.EventLog
	var total int64
	q := db.DB.Model(&models.EventLog{}).Where("device_sn = ?", sn)
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
