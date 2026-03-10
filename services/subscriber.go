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
}

func NewSubscriber(cfg *Config) *Subscriber {
	return &Subscriber{
		cfg: cfg,
	}
}

func (s *Subscriber) Start(ctx context.Context, messages chan<- *SensorMessage) {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(s.cfg.MQTTBroker)
	opts.SetClientID(s.cfg.MQTTClientID)
	opts.SetUsername(s.cfg.MQTTUser)
	opts.SetPassword(s.cfg.MQTTPass)
	tlsConfig := &tls.Config{
		InsecureSkipVerify: false,
	}
	opts.SetTLSConfig(tlsConfig)
	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatal(token.Error())
	}
	if token := client.Subscribe(s.cfg.MQTTTopic, s.cfg.MQTTQoS, func(c mqtt.Client, m mqtt.Message) {
		s.handleMessage(c, m, messages)
	}); token.Wait() && token.Error() != nil {
		log.Fatal(token.Error())
	}
	log.Println("MQTT subscriber started on topic:", s.cfg.MQTTTopic)
	<-ctx.Done()
	log.Println("MQTT subscriber shutting down")
	client.Disconnect(250)
}

func (s *Subscriber) handleMessage(c mqtt.Client, m mqtt.Message, messages chan<- *SensorMessage) {
	// The format is: device/{identifier}/{readingtype}
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
	log.Printf("Got message: {type: %s, time: %s, value: %f}", readingTypeName, timestamp.String(), value)
	msg := &SensorMessage{
		Time:            timestamp,
		Value:           value,
		ReadingTypeName: readingTypeName,
		Identifier:      deviceIdentifier,
	}
	select {
	case messages <- msg:
	default:
		log.Println("message dropped, channel full")
	}
}
