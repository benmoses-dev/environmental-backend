package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/benmoses-dev/environmental-backend/services"
)

func main() {
    // Todo: goroutines, buffering
	cfg := services.LoadConfig()
	db := services.NewPostgresService(cfg)
	defer db.Close()
	subscriber := services.NewSubscriber(cfg, db)
	subscriber.Start()
	log.Println("IoT ingestion service started")
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)
	<-sigs
	log.Println("Shutting down IoT ingestion service")
}
