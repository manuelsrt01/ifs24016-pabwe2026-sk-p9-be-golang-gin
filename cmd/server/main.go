package main

import (
	"log"

	"app/internal/config"
	"app/internal/database"
	"app/internal/server"

	"github.com/gin-gonic/gin"
)

var (
	connectDB = database.Connect
	runRouter = func(r *gin.Engine, addr string) error {
		return r.Run(addr)
	}
	fatalf = log.Fatalf
)

func run() error {
	cfg := config.LoadConfig()

	db, err := connectDB(cfg)
	if err != nil {
		fatalf("Failed to connect to database: %v", err)
		return err
	}

	r := server.NewRouter(db, cfg)
	if err := runRouter(r, ":"+cfg.AppPort); err != nil {
		fatalf("Failed to start Gin server: %v", err)
		return err
	}
	return nil
}

func main() {
	_ = run()
}
