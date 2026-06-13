package main

import (
	"fmt"
	"log"
	"provision-server/internal/config"
	"provision-server/internal/db"
	"provision-server/internal/handlers"
	mqttsvc "provision-server/internal/mqtt"
	"provision-server/internal/services"

	"github.com/gin-gonic/gin"
)

func main() {
	config.Load()

	if err := db.Init(config.App.DBPath); err != nil {
		log.Fatal("failed to init db:", err)
	}

	if err := services.InitCA(); err != nil {
		log.Fatal("failed to init CA:", err)
	}

	mqttSvc := mqttsvc.New()
	if err := mqttSvc.Connect(); err != nil {
		log.Println("[WARN] mqtt connect failed:", err)
	}

	r := gin.Default()

	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type,Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	api := r.Group("/api/v1")
	{
		device := api.Group("/devices")
		{
			device.GET("", handlers.ListDevices)
			device.GET("/:sn", handlers.GetDevice)
			device.POST("", handlers.CreateDevice)
			device.PUT("/:sn", handlers.UpdateDevice)
			device.DELETE("/:sn", handlers.DeleteDevice)
			device.POST("/:sn/transition", handlers.TransitionDevice)
			device.GET("/:sn/events", handlers.ListDeviceEvents)
		}

		provision := api.Group("/provision")
		{
			provision.POST("/scan", handlers.ScanProvision)
			provision.GET("/:sn/download", handlers.DownloadProvision)
		}

		models := api.Group("/models")
		{
			models.GET("", handlers.ListModels)
			models.GET("/:id", handlers.GetModel)
			models.POST("", handlers.CreateModel)
			models.PUT("/:id", handlers.UpdateModel)
			models.DELETE("/:id", handlers.DeleteModel)
		}

		batches := api.Group("/batches")
		{
			batches.GET("", handlers.ListBatches)
			batches.GET("/:id", handlers.GetBatch)
			batches.POST("", handlers.CreateBatch)
			batches.PUT("/:id", handlers.UpdateBatch)
			batches.DELETE("/:id", handlers.DeleteBatch)
		}

		templates := api.Group("/templates")
		{
			templates.GET("", handlers.ListTemplates)
			templates.GET("/:id", handlers.GetTemplate)
			templates.POST("", handlers.CreateTemplate)
			templates.PUT("/:id", handlers.UpdateTemplate)
			templates.DELETE("/:id", handlers.DeleteTemplate)
		}

		tags := api.Group("/tags")
		{
			tags.GET("", handlers.ListTags)
			tags.POST("", handlers.CreateTag)
			tags.DELETE("/:id", handlers.DeleteTag)
			tags.POST("/devices/:sn", handlers.AddTagToDevice)
			tags.DELETE("/devices/:sn/:tagId", handlers.RemoveTagFromDevice)
		}

		firmware := api.Group("/firmware")
		{
			firmware.GET("", handlers.ListFirmware)
			firmware.GET("/:id", handlers.GetFirmware)
			firmware.POST("", handlers.UploadFirmware)
			firmware.PUT("/:id", handlers.UpdateFirmware)
			firmware.DELETE("/:id", handlers.DeleteFirmware)
			firmware.GET("/:id/download", handlers.DownloadFirmware)
		}

		ota := api.Group("/ota")
		{
			ota.POST("/push", handlers.PushOTA)
			ota.GET("/jobs", handlers.ListOTAJobs)
			ota.GET("/devices/:sn", handlers.GetDeviceOTAStatus)
		}

		batchOps := api.Group("/batch-ops")
		{
			batchOps.POST("/provision", handlers.BatchProvision)
			batchOps.POST("/ota", handlers.BatchOTA)
			batchOps.POST("/status", handlers.BatchStatusChange)
		}

		stats := api.Group("/stats")
		{
			stats.GET("/overview", handlers.StatsOverview)
			stats.GET("/by-model", handlers.StatsByModel)
			stats.GET("/by-batch", handlers.StatsByBatch)
			stats.GET("/activation-rate", handlers.StatsActivationRate)
			stats.GET("/failure-rate", handlers.StatsFailureRate)
		}
	}

	addr := fmt.Sprintf("0.0.0.0:%d", config.App.Port)
	log.Println("[Server] starting on", addr)
	if err := r.Run(addr); err != nil {
		log.Fatal("server start failed:", err)
	}
}
