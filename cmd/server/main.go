package main

import (
	"log"

	"app/internal/config"
	"app/internal/database"
	"app/internal/server"
)

func main() {
	cfg := config.LoadConfig()

	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	r := server.NewRouter(db, cfg)
	if err := r.Run(":" + cfg.AppPort); err != nil {
		log.Fatalf("Failed to start Gin server: %v", err)
	}
}
