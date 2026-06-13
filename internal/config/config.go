package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port         int
	DBPath       string
	MQTTBroker   string
	MQTTClientID string
	MQTTUser     string
	MQTTPass     string
	HeartbeatTTL int
	CertValidDays int
	FirmwareDir  string
}

var App Config

func Load() {
	App.Port = getEnvInt("PORT", 8080)
	App.DBPath = getEnv("DB_PATH", "provision.db")
	App.MQTTBroker = getEnv("MQTT_BROKER", "tcp://127.0.0.1:1883")
	App.MQTTClientID = getEnv("MQTT_CLIENT_ID", "provision-server")
	App.MQTTUser = getEnv("MQTT_USER", "")
	App.MQTTPass = getEnv("MQTT_PASS", "")
	App.HeartbeatTTL = getEnvInt("HEARTBEAT_TTL_SEC", 120)
	App.CertValidDays = getEnvInt("CERT_VALID_DAYS", 3650)
	App.FirmwareDir = getEnv("FIRMWARE_DIR", "uploads/firmware")
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getEnvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}
