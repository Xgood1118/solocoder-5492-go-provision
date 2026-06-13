package handlers

import (
	"net/http"
	"provision-server/internal/db"
	"provision-server/internal/models"
	"provision-server/internal/services"
	"strconv"

	"github.com/gin-gonic/gin"
)

func ListTags(c *gin.Context) {
	var list []models.Tag
	if err := db.DB.Order("id DESC").Find(&list).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "ok", "data": list})
}

type CreateTagReq struct {
	Name  string `json:"name" binding:"required"`
	Color string `json:"color"`
}

func CreateTag(c *gin.Context) {
	var req CreateTagReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": err.Error()})
		return
	}
	t := &models.Tag{
		Name:  req.Name,
		Color: req.Color,
	}
	if err := db.DB.Create(t).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "ok", "data": t})
}

func DeleteTag(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var t models.Tag
	if err := db.DB.First(&t, id).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "tag not found"})
		return
	}
	if err := db.DB.Delete(&t).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "ok"})
}

type AddTagToDeviceReq struct {
	TagID uint `json:"tag_id" binding:"required"`
}

func AddTagToDevice(c *gin.Context) {
	sn := c.Param("sn")
	var req AddTagToDeviceReq
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
	var tag models.Tag
	if err := db.DB.First(&tag, req.TagID).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "tag not found"})
		return
	}
	if err := db.DB.Model(d).Association("Tags").Append(&tag); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "ok"})
}

func RemoveTagFromDevice(c *gin.Context) {
	sn := c.Param("sn")
	tagID, _ := strconv.Atoi(c.Param("tagId"))
	svc := services.NewDeviceService()
	d, err := svc.GetBySN(sn)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "device not found"})
		return
	}
	var tag models.Tag
	if err := db.DB.First(&tag, tagID).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "tag not found"})
		return
	}
	if err := db.DB.Model(d).Association("Tags").Delete(&tag); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "ok"})
}
