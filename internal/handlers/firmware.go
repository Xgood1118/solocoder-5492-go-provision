package handlers

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"provision-server/internal/config"
	"provision-server/internal/db"
	"provision-server/internal/models"
	"provision-server/pkg/utils"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

func ListFirmware(c *gin.Context) {
	page, size := getPageSize(c)
	modelID := c.Query("model_id")
	version := c.Query("version")

	var list []models.Firmware
	var total int64
	q := db.DB.Model(&models.Firmware{})
	if modelID != "" {
		q = q.Where("model_id = ?", modelID)
	}
	if version != "" {
		q = q.Where("version LIKE ?", "%"+version+"%")
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

func GetFirmware(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var f models.Firmware
	if err := db.DB.First(&f, id).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "firmware not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "ok", "data": f})
}

func UploadFirmware(c *gin.Context) {
	modelID, _ := strconv.Atoi(c.PostForm("model_id"))
	version := c.PostForm("version")
	description := c.PostForm("description")
	minVersion := c.PostForm("min_version")
	isActive := c.PostForm("is_active") == "true"
	grayRatio, _ := strconv.Atoi(c.DefaultPostForm("gray_ratio", "0"))

	if modelID == 0 || version == "" {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "model_id and version required"})
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "file required"})
		return
	}

	_ = os.MkdirAll(config.App.FirmwareDir, 0755)

	fileExt := filepath.Ext(file.Filename)
	savedName := fmt.Sprintf("model%d-%s-%d%s", modelID, version, time.Now().Unix(), fileExt)
	savedPath := filepath.Join(config.App.FirmwareDir, savedName)

	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": err.Error()})
		return
	}
	defer src.Close()

	data := make([]byte, file.Size)
	_, err = src.Read(data)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": err.Error()})
		return
	}

	if err := os.WriteFile(savedPath, data, 0644); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": err.Error()})
		return
	}

	md5Sum := utils.MD5File(data)

	f := &models.Firmware{
		ModelID:     uint(modelID),
		Version:     version,
		FileName:    savedName,
		FileSize:    file.Size,
		MD5Sum:      md5Sum,
		MinVersion:  minVersion,
		Description: description,
		IsActive:    isActive,
		GrayRatio:   grayRatio,
	}

	if f.IsActive {
		db.DB.Model(&models.Firmware{}).Where("model_id = ?", f.ModelID).Update("is_active", false)
	}

	if err := db.DB.Create(f).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "ok", "data": f})
}

type UpdateFirmwareReq struct {
	Description string `json:"description"`
	MinVersion  string `json:"min_version"`
	IsActive     *bool  `json:"is_active"`
	GrayRatio    int    `json:"gray_ratio"`
}

func UpdateFirmware(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req UpdateFirmwareReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": err.Error()})
		return
	}
	var f models.Firmware
	if err := db.DB.First(&f, id).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "firmware not found"})
		return
	}
	if req.Description != "" {
		f.Description = req.Description
	}
	if req.MinVersion != "" {
		f.MinVersion = req.MinVersion
	}
	if req.GrayRatio >= 0 && req.GrayRatio <= 100 {
		f.GrayRatio = req.GrayRatio
	}
	if req.IsActive != nil {
		if *req.IsActive {
			db.DB.Model(&models.Firmware{}).Where("model_id = ? AND id != ?", f.ModelID, f.ID).Update("is_active", false)
		}
		f.IsActive = *req.IsActive
	}
	if err := db.DB.Save(&f).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "ok", "data": f})
}

func DeleteFirmware(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var f models.Firmware
	if err := db.DB.First(&f, id).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "firmware not found"})
		return
	}
	filePath := filepath.Join(config.App.FirmwareDir, f.FileName)
	_ = os.Remove(filePath)
	if err := db.DB.Delete(&f).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "ok"})
}

func DownloadFirmware(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var f models.Firmware
	if err := db.DB.First(&f, id).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "firmware not found"})
		return
	}
	filePath := filepath.Join(config.App.FirmwareDir, f.FileName)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "file not found"})
		return
	}
	c.Header("Content-MD5", f.MD5Sum)
	c.Header("X-Firmware-Version", f.Version)
	c.FileAttachment(filePath, f.FileName)
}
