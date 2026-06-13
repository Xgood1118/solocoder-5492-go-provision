package handlers

import (
	"net/http"
	"provision-server/internal/db"
	"provision-server/internal/models"
	"strconv"

	"github.com/gin-gonic/gin"
)

func ListModels(c *gin.Context) {
	page, size := getPageSize(c)
	var list []models.Model
	var total int64
	q := db.DB.Model(&models.Model{})
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

func GetModel(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var m models.Model
	if err := db.DB.First(&m, id).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "model not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "ok", "data": m})
}

type CreateModelReq struct {
	ModelCode   string `json:"model_code" binding:"required"`
	ModelName   string `json:"model_name" binding:"required"`
	Description string `json:"description"`
}

func CreateModel(c *gin.Context) {
	var req CreateModelReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": err.Error()})
		return
	}
	m := &models.Model{
		ModelCode:   req.ModelCode,
		ModelName:   req.ModelName,
		Description: req.Description,
	}
	if err := db.DB.Create(m).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "ok", "data": m})
}

type UpdateModelReq struct {
	ModelCode   string `json:"model_code"`
	ModelName   string `json:"model_name"`
	Description string `json:"description"`
}

func UpdateModel(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req UpdateModelReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": err.Error()})
		return
	}
	var m models.Model
	if err := db.DB.First(&m, id).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "model not found"})
		return
	}
	if req.ModelCode != "" {
		m.ModelCode = req.ModelCode
	}
	if req.ModelName != "" {
		m.ModelName = req.ModelName
	}
	if req.Description != "" {
		m.Description = req.Description
	}
	if err := db.DB.Save(&m).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "ok", "data": m})
}

func DeleteModel(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var m models.Model
	if err := db.DB.First(&m, id).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "model not found"})
		return
	}
	if err := db.DB.Delete(&m).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "ok"})
}
