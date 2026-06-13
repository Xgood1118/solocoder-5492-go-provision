package handlers

import (
	"net/http"
	"provision-server/internal/db"
	"provision-server/internal/models"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

func ListBatches(c *gin.Context) {
	page, size := getPageSize(c)
	modelID := c.Query("model_id")
	keyword := c.Query("keyword")

	var list []models.Batch
	var total int64
	q := db.DB.Model(&models.Batch{})
	if modelID != "" {
		q = q.Where("model_id = ?", modelID)
	}
	if keyword != "" {
		q = q.Where("batch_no LIKE ? OR customer LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
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

func GetBatch(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var b models.Batch
	if err := db.DB.First(&b, id).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "batch not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "ok", "data": b})
}

type CreateBatchReq struct {
	BatchNo    string `json:"batch_no" binding:"required"`
	ModelID    uint   `json:"model_id" binding:"required"`
	Quantity   int    `json:"quantity"`
	TemplateID uint   `json:"template_id"`
	Customer   string `json:"customer"`
	Region     string `json:"region"`
	Remark     string `json:"remark"`
	Produced   bool   `json:"produced"`
}

func CreateBatch(c *gin.Context) {
	var req CreateBatchReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": err.Error()})
		return
	}
	b := &models.Batch{
		BatchNo:    req.BatchNo,
		ModelID:    req.ModelID,
		Quantity:   req.Quantity,
		TemplateID: req.TemplateID,
		Customer:   req.Customer,
		Region:     req.Region,
		Remark:     req.Remark,
	}
	if req.Produced {
		now := time.Now()
		b.ProducedAt = &now
	}
	if err := db.DB.Create(b).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "ok", "data": b})
}

type UpdateBatchReq struct {
	BatchNo    string `json:"batch_no"`
	ModelID    uint   `json:"model_id"`
	Quantity   int    `json:"quantity"`
	TemplateID uint   `json:"template_id"`
	Customer   string `json:"customer"`
	Region     string `json:"region"`
	Remark     string `json:"remark"`
	Produced   *bool  `json:"produced"`
}

func UpdateBatch(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req UpdateBatchReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": err.Error()})
		return
	}
	var b models.Batch
	if err := db.DB.First(&b, id).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "batch not found"})
		return
	}
	if req.BatchNo != "" {
		b.BatchNo = req.BatchNo
	}
	if req.ModelID > 0 {
		b.ModelID = req.ModelID
	}
	if req.Quantity != 0 {
		b.Quantity = req.Quantity
	}
	if req.TemplateID > 0 {
		b.TemplateID = req.TemplateID
	}
	if req.Customer != "" {
		b.Customer = req.Customer
	}
	if req.Region != "" {
		b.Region = req.Region
	}
	if req.Remark != "" {
		b.Remark = req.Remark
	}
	if req.Produced != nil {
		if *req.Produced && b.ProducedAt == nil {
			now := time.Now()
			b.ProducedAt = &now
		} else if !*req.Produced {
			b.ProducedAt = nil
		}
	}
	if err := db.DB.Save(&b).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "ok", "data": b})
}

func DeleteBatch(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var b models.Batch
	if err := db.DB.First(&b, id).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "batch not found"})
		return
	}
	if err := db.DB.Delete(&b).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "ok"})
}
