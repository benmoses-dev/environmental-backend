package services

import (
	"log"
	"os"
	"strconv"
	"time"
)

type Config struct {
	DatabaseURL   string
	MQTTBroker    string
	MQTTUser      string
	MQTTPass      string
	MQTTClientID  string
	MQTTTopic     string
	MQTTQoS       byte
	InsertTimeout time.Duration
	DBWorkers     int
}

func LoadConfig() *Config {
	cfg := &Config{
		DatabaseURL:   getEnv("DATABASE_URL", ""),
		MQTTBroker:    getEnv("MQTT_BROKER", "tcp://localhost:8883"),
		MQTTUser:      getEnv("MQTT_USER", "user"),
		MQTTPass:      getEnv("MQTT_PASS", "password"),
		MQTTClientID:  getEnv("MQTT_CLIENT_ID", "backend"),
		MQTTTopic:     getEnv("MQTT_TOPIC", "device/+/+"),
		MQTTQoS:       byte(getEnvAsInt("MQTT_QOS", 2)),
		InsertTimeout: getEnvAsDuration("INSERT_TIMEOUT", 5*time.Second),
		DBWorkers:     getEnvAsInt("DB_WORKERS", 5),
	}
	return cfg
}

func getEnv(key string, defaultVal string) string {
	val, exists := os.LookupEnv(key)
	if exists {
		return val
	}
	return defaultVal
}

func getEnvAsInt(key string, defaultVal int) int {
	valStr, exists := os.LookupEnv(key)
	if exists {
		if val, err := strconv.Atoi(valStr); err == nil {
			return val
		}
		log.Printf("Warning: invalid integer for %s, using default %d\n", key, defaultVal)
	}
	return defaultVal
}

func getEnvAsDuration(key string, defaultVal time.Duration) time.Duration {
	valStr, exists := os.LookupEnv(key)
	if exists {
		if val, err := time.ParseDuration(valStr); err == nil {
			return val
		}
		log.Printf("Warning: invalid duration for %s, using default %s\n", key, defaultVal)
	}
	return defaultVal
}
