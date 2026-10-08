package main

import (
	"context"
	"log"
	"time"

	"github.com/zxchelik/room-booking-service/internal/application"
)

func main() {
	cfg := application.LoadConfig()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := application.RunMigrations(ctx, cfg.DatabaseURL); err != nil {
		log.Fatalf("run migrations: %v", err)
	}
}
