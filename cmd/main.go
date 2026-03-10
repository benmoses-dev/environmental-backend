package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/benmoses-dev/environmental-backend/services"
)

func logConfig(cfg *services.Config) {
	log.Printf("Database URL: %s\n", cfg.DatabaseURL)
	log.Printf("MQTT Broker: %s\n", cfg.MQTTBroker)
	log.Printf("MQTT ClientID: %s\n", cfg.MQTTClientID)
	log.Printf("MQTT QoS: %d\n", cfg.MQTTQoS)
	log.Printf("Insert Timeout: %s\n", cfg.InsertTimeout)
}

func main() {
	cfg := services.LoadConfig()
	logConfig(cfg)

	messages := make(chan *services.SensorMessage, 100)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	db := services.NewPostgresService(cfg)
	defer db.Close()

	subscriber := services.NewSubscriber(cfg)

	var wg sync.WaitGroup
	db.Start(ctx, messages, &wg)
	wg.Add(1)
	go func() {
		defer wg.Done()
		subscriber.Start(ctx, messages)
	}()

	log.Println("IoT ingestion service started")

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)
	<-sigs

	log.Println("Shutting down IoT ingestion service")
	cancel()
	close(messages)
	wg.Wait()
}
