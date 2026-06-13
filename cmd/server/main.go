package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"provision-server/internal/config"
	"provision-server/internal/db"
	"provision-server/internal/handlers"
	mqttsvc "provision-server/internal/mqtt"
	"provision-server/internal/services"

	"github.com/gin-gonic/gin"
)

func ensureDirs() {
	dirs := []string{
		config.App.FirmwareDir,
		filepath.Dir(config.App.DBPath),
	}
	for _, d := range dirs {
		if d == "" || d == "." {
			continue
		}
		if err := os.MkdirAll(d, 0755); err != nil {
			log.Printf("[WARN] mkdir %s failed: %v", d, err)
		}
	}
}

func main() {
	config.Load()
	ensureDirs()

	log.Println("[Init] config loaded, port:", config.App.Port)
	log.Println("[Init] db path:", config.App.DBPath)
	log.Println("[Init] firmware dir:", config.App.FirmwareDir)
	log.Println("[Init] mqtt broker:", config.App.MQTTBroker)

	if err := db.Init(config.App.DBPath); err != nil {
		log.Fatal("failed to init db:", err)
	}
	log.Println("[Init] db OK")

	if err := services.InitCA(); err != nil {
		log.Fatal("failed to init CA:", err)
	}
	log.Println("[Init] CA OK")

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

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"code":    0,
			"msg":     "ok",
			"status":  "running",
			"version": "1.0.0",
		})
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

		modelsGroup := api.Group("/models")
		{
			modelsGroup.GET("", handlers.ListModels)
			modelsGroup.GET("/:id", handlers.GetModel)
			modelsGroup.POST("", handlers.CreateModel)
			modelsGroup.PUT("/:id", handlers.UpdateModel)
			modelsGroup.DELETE("/:id", handlers.DeleteModel)
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
			ota.GET("/jobs/:id", handlers.GetOTAJob)
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

	log.Println("[Init] routes registered")

	mqttSvc := mqttsvc.New()
	go func() {
		if err := mqttSvc.Connect(); err != nil {
			log.Println("[WARN] mqtt connect failed (will retry):", err)
		}
	}()
	log.Println("[Init] mqtt connect started in background")

	addr := fmt.Sprintf("0.0.0.0:%d", config.App.Port)
	log.Println("[Server] starting on", addr)
	if err := r.Run(addr); err != nil {
		log.Fatal("server start failed:", err)
	}
}
