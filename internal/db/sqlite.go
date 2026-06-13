package db

import (
	"log"
	"provision-server/internal/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB

func Init(dbPath string) error {
	var err error
	DB, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		return err
	}

	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)

	err = DB.AutoMigrate(
		&models.Device{},
		&models.Model{},
		&models.Batch{},
		&models.Template{},
		&models.Firmware{},
		&models.OTAJob{},
		&models.Tag{},
		&models.EventLog{},
		&models.DeviceCert{},
	)
	if err != nil {
		return err
	}

	log.Println("[DB] SQLite initialized and migrated at", dbPath)
	return nil
}
