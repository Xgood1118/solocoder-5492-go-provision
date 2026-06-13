package handlers

import (
	"net/http"
	"provision-server/internal/db"
	"provision-server/internal/models"
	"strconv"

	"github.com/gin-gonic/gin"
)

func ListTemplates(c *gin.Context) {
	page, size := getPageSize(c)
	tplType := c.Query("type")

	var list []models.Template
	var total int64
	q := db.DB.Model(&models.Template{})
	if tplType != "" {
		q = q.Where("type = ?", tplType)
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

func GetTemplate(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var t models.Template
	if err := db.DB.First(&t, id).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "template not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "ok", "data": t})
}

type CreateTemplateReq struct {
	Name       string `json:"name" binding:"required"`
	Type       string `json:"type" binding:"required"`
	WifiSSID   string `json:"wifi_ssid"`
	WifiPSK    string `json:"wifi_psk"`
	MqttHost   string `json:"mqtt_host"`
	MqttPort   int    `json:"mqtt_port"`
	MqttUser   string `json:"mqtt_user"`
	MqttPass   string `json:"mqtt_pass"`
	MqttUseTLS bool   `json:"mqtt_use_tls"`
	ExtraVars  string `json:"extra_vars"`
}

func CreateTemplate(c *gin.Context) {
	var req CreateTemplateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": err.Error()})
		return
	}
	t := &models.Template{
		Name:       req.Name,
		Type:       req.Type,
		WifiSSID:   req.WifiSSID,
		WifiPSK:    req.WifiPSK,
		MqttHost:   req.MqttHost,
		MqttPort:   req.MqttPort,
		MqttUser:   req.MqttUser,
		MqttPass:   req.MqttPass,
		MqttUseTLS: req.MqttUseTLS,
		ExtraVars:  req.ExtraVars,
	}
	if t.MqttPort == 0 {
		t.MqttPort = 1883
	}
	if err := db.DB.Create(t).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "ok", "data": t})
}

type UpdateTemplateReq struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	WifiSSID   string `json:"wifi_ssid"`
	WifiPSK    string `json:"wifi_psk"`
	MqttHost   string `json:"mqtt_host"`
	MqttPort   int    `json:"mqtt_port"`
	MqttUser   string `json:"mqtt_user"`
	MqttPass   string `json:"mqtt_pass"`
	MqttUseTLS *bool  `json:"mqtt_use_tls"`
	ExtraVars  string `json:"extra_vars"`
}

func UpdateTemplate(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req UpdateTemplateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": err.Error()})
		return
	}
	var t models.Template
	if err := db.DB.First(&t, id).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "template not found"})
		return
	}
	if req.Name != "" {
		t.Name = req.Name
	}
	if req.Type != "" {
		t.Type = req.Type
	}
	if req.WifiSSID != "" {
		t.WifiSSID = req.WifiSSID
	}
	if req.WifiPSK != "" {
		t.WifiPSK = req.WifiPSK
	}
	if req.MqttHost != "" {
		t.MqttHost = req.MqttHost
	}
	if req.MqttPort != 0 {
		t.MqttPort = req.MqttPort
	}
	if req.MqttUser != "" {
		t.MqttUser = req.MqttUser
	}
	if req.MqttPass != "" {
		t.MqttPass = req.MqttPass
	}
	if req.MqttUseTLS != nil {
		t.MqttUseTLS = *req.MqttUseTLS
	}
	if req.ExtraVars != "" {
		t.ExtraVars = req.ExtraVars
	}
	if err := db.DB.Save(&t).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "ok", "data": t})
}

func DeleteTemplate(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var t models.Template
	if err := db.DB.First(&t, id).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "template not found"})
		return
	}
	if err := db.DB.Delete(&t).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "ok"})
}
