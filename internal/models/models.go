package models

import (
	"time"
)

type DeviceStatus string

const (
	StatusPending     DeviceStatus = "pending"
	StatusProvisioned DeviceStatus = "provisioned"
	StatusActivated   DeviceStatus = "activated"
	StatusDeactivated DeviceStatus = "deactivated"
	StatusRetired     DeviceStatus = "retired"
)

type Device struct {
	ID            uint         `gorm:"primaryKey" json:"id"`
	DeviceSN      string       `gorm:"uniqueIndex;size:100;not null" json:"device_sn"`
	DeviceMAC     string       `gorm:"size:32" json:"device_mac"`
	ModelID       uint         `gorm:"not null" json:"model_id"`
	BatchID       uint         `json:"batch_id"`
	Status        DeviceStatus `gorm:"size:32;not null;default:'pending'" json:"status"`
	Customer      string       `gorm:"size:100" json:"customer"`
	Region        string       `gorm:"size:100" json:"region"`
	Tags          []*Tag       `gorm:"many2many:device_tags;" json:"tags,omitempty"`
	FirmwareVer   string       `gorm:"size:50" json:"firmware_ver"`
	TemplateID    uint         `json:"template_id"`
	LastHeartbeat *time.Time   `json:"last_heartbeat"`
	FirstActiveAt *time.Time   `json:"first_active_at"`
	ProvisionedAt *time.Time   `json:"provisioned_at"`
	RetiredAt     *time.Time   `json:"retired_at"`
	Remark        string       `gorm:"size:500" json:"remark"`
	CreatedAt     time.Time    `json:"created_at"`
	UpdatedAt     time.Time    `json:"updated_at"`
}

type Model struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	ModelCode   string    `gorm:"uniqueIndex;size:50;not null" json:"model_code"`
	ModelName   string    `gorm:"size:100;not null" json:"model_name"`
	Description string    `gorm:"size:500" json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Batch struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	BatchNo      string    `gorm:"uniqueIndex;size:50;not null" json:"batch_no"`
	ModelID      uint      `gorm:"not null" json:"model_id"`
	Quantity     int       `gorm:"not null;default:0" json:"quantity"`
	TemplateID   uint      `json:"template_id"`
	Customer     string    `gorm:"size:100" json:"customer"`
	Region       string    `gorm:"size:100" json:"region"`
	Remark       string    `gorm:"size:500" json:"remark"`
	ProducedAt   *time.Time `json:"produced_at"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Template struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	Name       string    `gorm:"size:100;not null" json:"name"`
	Type       string    `gorm:"size:32;not null" json:"type"`
	WifiSSID   string    `gorm:"size:100" json:"wifi_ssid"`
	WifiPSK    string    `gorm:"size:100" json:"wifi_psk"`
	MqttHost   string    `gorm:"size:200" json:"mqtt_host"`
	MqttPort   int       `gorm:"default:1883" json:"mqtt_port"`
	MqttUser   string    `gorm:"size:100" json:"mqtt_user"`
	MqttPass   string    `gorm:"size:100" json:"mqtt_pass"`
	MqttUseTLS bool      `gorm:"default:false" json:"mqtt_use_tls"`
	ExtraVars  string    `gorm:"type:text" json:"extra_vars"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type Firmware struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	ModelID     uint      `gorm:"not null" json:"model_id"`
	Version     string    `gorm:"size:50;not null" json:"version"`
	FileName    string    `gorm:"size:255;not null" json:"file_name"`
	FileSize    int64     `gorm:"not null" json:"file_size"`
	MD5Sum      string    `gorm:"size:64" json:"md5_sum"`
	MinVersion  string    `gorm:"size:50" json:"min_version"`
	Description string    `gorm:"size:500" json:"description"`
	IsActive    bool      `gorm:"default:false" json:"is_active"`
	GrayRatio   int       `gorm:"default:0" json:"gray_ratio"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type OTAJob struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	FirmwareID    uint      `gorm:"not null" json:"firmware_id"`
	DeviceID      uint      `gorm:"not null" json:"device_id"`
	Status        string    `gorm:"size:32;default:'pending'" json:"status"`
	Progress      int       `gorm:"default:0" json:"progress"`
	ErrMsg        string    `gorm:"size:500" json:"err_msg"`
	StartedAt     *time.Time `json:"started_at"`
	FinishedAt    *time.Time `json:"finished_at"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type Tag struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"uniqueIndex;size:50;not null" json:"name"`
	Color     string    `gorm:"size:16" json:"color"`
	CreatedAt time.Time `json:"created_at"`
}

type EventLog struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	DeviceSN   string    `gorm:"index;size:100" json:"device_sn"`
	EventType  string    `gorm:"size:50;not null" json:"event_type"`
	FromStatus string    `gorm:"size:32" json:"from_status"`
	ToStatus   string    `gorm:"size:32" json:"to_status"`
	Message    string    `gorm:"size:1000" json:"message"`
	CreatedAt  time.Time `json:"created_at"`
}

type DeviceCert struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	DeviceID   uint      `gorm:"uniqueIndex;not null" json:"device_id"`
	CertSN     string    `gorm:"size:100" json:"cert_sn"`
	CertPEM    string    `gorm:"type:text" json:"-"`
	KeyPEM     string    `gorm:"type:text" json:"-"`
	ExpiresAt  time.Time `json:"expires_at"`
	CreatedAt  time.Time `json:"created_at"`
}
