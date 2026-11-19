package main

import (
	"errors"
	"testing"

	"app/internal/config"
	"app/internal/database"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestCmdMainAndRun(t *testing.T) {
	origConnect := connectDB
	origRunRouter := runRouter
	origFatalf := fatalf
	defer func() {
		connectDB = origConnect
		runRouter = origRunRouter
		fatalf = origFatalf
	}()

	var fatalMsg string
	fatalf = func(format string, v ...any) {
		fatalMsg = format
	}

	connectDB = func(cfg *config.Config) (*gorm.DB, error) {
		return database.ConnectTestDB()
	}
	runRouter = func(r *gin.Engine, addr string) error {
		return nil
	}
	main()
	assert.Empty(t, fatalMsg)

	runRouter = func(r *gin.Engine, addr string) error {
		return errors.New("bind error")
	}
	err := run()
	assert.Error(t, err)
	assert.Contains(t, fatalMsg, "Failed to start Gin server")

	connectDB = func(cfg *config.Config) (*gorm.DB, error) {
		return nil, errors.New("db error")
	}
	err = run()
	assert.Error(t, err)
	assert.Contains(t, fatalMsg, "Failed to connect to database")
}
