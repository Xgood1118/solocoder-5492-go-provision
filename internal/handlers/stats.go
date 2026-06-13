package handlers

import (
	"net/http"
	"provision-server/internal/db"
	"provision-server/internal/models"
	"strconv"

	"github.com/gin-gonic/gin"
)

func StatsOverview(c *gin.Context) {
	var total int64
	var pending, provisioned, activated, deactivated, retired int64

	db.DB.Model(&models.Device{}).Count(&total)
	db.DB.Model(&models.Device{}).Where("status = ?", models.StatusPending).Count(&pending)
	db.DB.Model(&models.Device{}).Where("status = ?", models.StatusProvisioned).Count(&provisioned)
	db.DB.Model(&models.Device{}).Where("status = ?", models.StatusActivated).Count(&activated)
	db.DB.Model(&models.Device{}).Where("status = ?", models.StatusDeactivated).Count(&deactivated)
	db.DB.Model(&models.Device{}).Where("status = ?", models.StatusRetired).Count(&retired)

	activationRate := 0.0
	shipment := provisioned + activated + deactivated
	if shipment > 0 {
		activationRate = float64(activated) / float64(shipment) * 100
	}

	var totalModels, totalBatches, totalFirmware int64
	db.DB.Model(&models.Model{}).Count(&totalModels)
	db.DB.Model(&models.Batch{}).Count(&totalBatches)
	db.DB.Model(&models.Firmware{}).Count(&totalFirmware)

	var otaFailed int64
	db.DB.Model(&models.OTAJob{}).Where("status = ?", "failed").Count(&otaFailed)
	failureRate := 0.0
	var otaTotal int64
	db.DB.Model(&models.OTAJob{}).Where("status IN ?", []string{"success", "failed"}).Count(&otaTotal)
	if otaTotal > 0 {
		failureRate = float64(otaFailed) / float64(otaTotal) * 100
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "ok",
		"data": gin.H{
			"total_devices":     total,
			"pending":           pending,
			"provisioned":       provisioned,
			"activated":         activated,
			"deactivated":       deactivated,
			"retired":           retired,
			"shipment_count":    shipment,
			"activation_rate":   activationRate,
			"failure_rate":      failureRate,
			"total_models":      totalModels,
			"total_batches":     totalBatches,
			"total_firmware":    totalFirmware,
			"ota_jobs_total":    otaTotal,
			"ota_jobs_failed":   otaFailed,
		},
	})
}

type StatsGroup struct {
	ID            uint   `json:"id"`
	Name          string `json:"name"`
	Total         int64  `json:"total"`
	Pending       int64  `json:"pending"`
	Provisioned   int64  `json:"provisioned"`
	Activated     int64  `json:"activated"`
	Deactivated   int64  `json:"deactivated"`
	Retired       int64  `json:"retired"`
	ActivationRate float64 `json:"activation_rate"`
	FailureRate   float64 `json:"failure_rate"`
}

func StatsByModel(c *gin.Context) {
	var modelsList []models.Model
	db.DB.Find(&modelsList)

	result := make([]StatsGroup, 0, len(modelsList))
	for _, m := range modelsList {
		sg := StatsGroup{
			ID:   m.ID,
			Name: m.ModelName,
		}
		db.DB.Model(&models.Device{}).Where("model_id = ?", m.ID).Count(&sg.Total)
		db.DB.Model(&models.Device{}).Where("model_id = ? AND status = ?", m.ID, models.StatusPending).Count(&sg.Pending)
		db.DB.Model(&models.Device{}).Where("model_id = ? AND status = ?", m.ID, models.StatusProvisioned).Count(&sg.Provisioned)
		db.DB.Model(&models.Device{}).Where("model_id = ? AND status = ?", m.ID, models.StatusActivated).Count(&sg.Activated)
		db.DB.Model(&models.Device{}).Where("model_id = ? AND status = ?", m.ID, models.StatusDeactivated).Count(&sg.Deactivated)
		db.DB.Model(&models.Device{}).Where("model_id = ? AND status = ?", m.ID, models.StatusRetired).Count(&sg.Retired)

		shipment := sg.Provisioned + sg.Activated + sg.Deactivated
		if shipment > 0 {
			sg.ActivationRate = float64(sg.Activated) / float64(shipment) * 100
		}

		var devIDs []uint
		db.DB.Model(&models.Device{}).Where("model_id = ?", m.ID).Pluck("id", &devIDs)
		if len(devIDs) > 0 {
			var otaTotal, otaFailed int64
			db.DB.Model(&models.OTAJob{}).Where("device_id IN ? AND status IN ?", devIDs, []string{"success", "failed"}).Count(&otaTotal)
			db.DB.Model(&models.OTAJob{}).Where("device_id IN ? AND status = ?", devIDs, "failed").Count(&otaFailed)
			if otaTotal > 0 {
				sg.FailureRate = float64(otaFailed) / float64(otaTotal) * 100
			}
		}
		result = append(result, sg)
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "ok", "data": result})
}

func StatsByBatch(c *gin.Context) {
	var batches []models.Batch
	db.DB.Find(&batches)

	result := make([]StatsGroup, 0, len(batches))
	for _, b := range batches {
		sg := StatsGroup{
			ID:   b.ID,
			Name: b.BatchNo,
		}
		db.DB.Model(&models.Device{}).Where("batch_id = ?", b.ID).Count(&sg.Total)
		db.DB.Model(&models.Device{}).Where("batch_id = ? AND status = ?", b.ID, models.StatusPending).Count(&sg.Pending)
		db.DB.Model(&models.Device{}).Where("batch_id = ? AND status = ?", b.ID, models.StatusProvisioned).Count(&sg.Provisioned)
		db.DB.Model(&models.Device{}).Where("batch_id = ? AND status = ?", b.ID, models.StatusActivated).Count(&sg.Activated)
		db.DB.Model(&models.Device{}).Where("batch_id = ? AND status = ?", b.ID, models.StatusDeactivated).Count(&sg.Deactivated)
		db.DB.Model(&models.Device{}).Where("batch_id = ? AND status = ?", b.ID, models.StatusRetired).Count(&sg.Retired)

		shipment := sg.Provisioned + sg.Activated + sg.Deactivated
		if shipment > 0 {
			sg.ActivationRate = float64(sg.Activated) / float64(shipment) * 100
		}

		var devIDs []uint
		db.DB.Model(&models.Device{}).Where("batch_id = ?", b.ID).Pluck("id", &devIDs)
		if len(devIDs) > 0 {
			var otaTotal, otaFailed int64
			db.DB.Model(&models.OTAJob{}).Where("device_id IN ? AND status IN ?", devIDs, []string{"success", "failed"}).Count(&otaTotal)
			db.DB.Model(&models.OTAJob{}).Where("device_id IN ? AND status = ?", devIDs, "failed").Count(&otaFailed)
			if otaTotal > 0 {
				sg.FailureRate = float64(otaFailed) / float64(otaTotal) * 100
			}
		}
		result = append(result, sg)
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "ok", "data": result})
}

type TrendPoint struct {
	Date           string  `json:"date"`
	ShipmentCount  int64   `json:"shipment_count"`
	ActivationCount int64  `json:"activation_count"`
	ActivationRate  float64 `json:"activation_rate"`
}

func StatsActivationRate(c *gin.Context) {
	days, _ := strconv.Atoi(c.DefaultQuery("days", "7"))
	if days <= 0 {
		days = 7
	}
	if days > 90 {
		days = 90
	}

	points := make([]TrendPoint, 0, days)

	for i := days - 1; i >= 0; i-- {
		dateExpr := "DATE('now', '-" + strconv.Itoa(i) + " days')"

		var shipment, activation int64
		db.DB.Model(&models.Device{}).Where("DATE(provisioned_at) = " + dateExpr).Count(&shipment)
		if shipment == 0 {
			db.DB.Model(&models.Device{}).Where("DATE(created_at) = " + dateExpr + " AND status != ?", models.StatusPending).Count(&shipment)
		}
		db.DB.Model(&models.Device{}).Where("DATE(first_active_at) = " + dateExpr).Count(&activation)

		rate := 0.0
		if shipment > 0 {
			rate = float64(activation) / float64(shipment) * 100
		}

		var dateStr string
		db.DB.Raw("SELECT " + dateExpr).Scan(&dateStr)
		points = append(points, TrendPoint{
			Date:            dateStr,
			ShipmentCount:   shipment,
			ActivationCount: activation,
			ActivationRate:  rate,
		})
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "ok", "data": points})
}

type FailurePoint struct {
	Date        string  `json:"date"`
	OTATotal    int64   `json:"ota_total"`
	OTAFailed   int64   `json:"ota_failed"`
	FailureRate float64 `json:"failure_rate"`
	Deactivated int64   `json:"deactivated"`
}

func StatsFailureRate(c *gin.Context) {
	days, _ := strconv.Atoi(c.DefaultQuery("days", "7"))
	if days <= 0 {
		days = 7
	}
	if days > 90 {
		days = 90
	}

	points := make([]FailurePoint, 0, days)

	for i := days - 1; i >= 0; i-- {
		dateExpr := "DATE('now', '-" + strconv.Itoa(i) + " days')"

		var otaTotal, otaFailed, deactivated int64
		db.DB.Model(&models.OTAJob{}).Where("DATE(created_at) = " + dateExpr + " AND status IN ?", []string{"success", "failed"}).Count(&otaTotal)
		db.DB.Model(&models.OTAJob{}).Where("DATE(created_at) = " + dateExpr + " AND status = ?", "failed").Count(&otaFailed)

		var dateStr string
		db.DB.Raw("SELECT " + dateExpr).Scan(&dateStr)

		rate := 0.0
		if otaTotal > 0 {
			rate = float64(otaFailed) / float64(otaTotal) * 100
		}

		var ids []uint
		db.DB.Model(&models.Device{}).Where("status = ?", models.StatusDeactivated).Pluck("id", &ids)
		if len(ids) > 0 {
			db.DB.Model(&models.EventLog{}).
				Where("DATE(created_at) = "+dateExpr+" AND to_status = ?", models.StatusDeactivated).
				Count(&deactivated)
		}

		points = append(points, FailurePoint{
			Date:        dateStr,
			OTATotal:    otaTotal,
			OTAFailed:   otaFailed,
			FailureRate: rate,
			Deactivated: deactivated,
		})
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "ok", "data": points})
}
