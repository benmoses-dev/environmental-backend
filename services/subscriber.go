package services

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"log"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type Payload struct {
	Time int64   `json:"time"`
	Val  float64 `json:"val"`
}

type Subscriber struct {
	cfg *Config
	db  *PostgresService
}

func NewSubscriber(cfg *Config, db *PostgresService) *Subscriber {
	return &Subscriber{
		cfg: cfg,
		db:  db,
	}
}

func (s *Subscriber) Start() {
	opts := mqtt.NewClientOptions().AddBroker(s.cfg.MQTTBroker)
	opts.SetClientID(s.cfg.MQTTClientID)
	opts.SetUsername(s.cfg.MQTTUser)
	opts.SetPassword(s.cfg.MQTTPass)
	tlsConfig := &tls.Config{
		InsecureSkipVerify: false, // true only if self-signed
	}
	opts.SetTLSConfig(tlsConfig)
	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatal(token.Error())
	}
	if token := client.Subscribe(s.cfg.MQTTTopic, s.cfg.MQTTQoS, s.handleMessage); token.Wait() && token.Error() != nil {
		log.Fatal(token.Error())
	}
	log.Println("MQTT subscriber started on topic:", s.cfg.MQTTTopic)
}

func (s *Subscriber) handleMessage(c mqtt.Client, m mqtt.Message) {
	// topic format: device/{identifier}/{readingtype}
	parts := strings.Split(m.Topic(), "/")
	if len(parts) != 3 {
		log.Println("invalid topic:", m.Topic())
		return
	}
	deviceIdentifier := parts[1]
	readingTypeName := parts[2]
	var payload Payload
	if err := json.Unmarshal(m.Payload(), &payload); err != nil {
		log.Println("invalid payload:", err)
		return
	}
	timestamp := time.Unix(payload.Time, 0)
	value := payload.Val
	log.Printf("Got message: {time: %s, value: %f}", timestamp.String(), value)
	ctx, cancel := context.WithTimeout(context.Background(), s.cfg.InsertTimeout)
	defer cancel()
	deviceID, locationID, err := s.db.GetDevice(ctx, deviceIdentifier)
	if err != nil {
		log.Println("device lookup failed:", err)
		return
	}
	sensorID, readingTypeID, err := s.db.GetSensorForReading(ctx, deviceID, readingTypeName)
	if err != nil {
		log.Println("sensor lookup failed:", err)
		return
	}
	err = s.db.InsertSensorData(
		ctx,
		timestamp,
		deviceID,
		locationID,
		sensorID,
		readingTypeID,
		value,
	)
	if err != nil {
		log.Println("insert failed:", err)
	}
}
