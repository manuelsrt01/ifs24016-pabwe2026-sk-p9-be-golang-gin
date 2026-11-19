package database

import (
	"errors"
	"testing"

	"app/internal/config"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestConnectTestDB_AndSeed(t *testing.T) {
	db, err := ConnectTestDB()
	assert.NoError(t, err)
	assert.NotNil(t, db)

	var users []User
	err = db.Find(&users).Error
	assert.NoError(t, err)
	assert.Len(t, users, 2)

	// Re-run seed to verify idempotency
	err = SeedUsers(db, &config.Config{})
	assert.NoError(t, err)
	err = db.Find(&users).Error
	assert.NoError(t, err)
	assert.Len(t, users, 2)
}

func TestConnect_SqliteExplicitDSN(t *testing.T) {
	cfg := &config.Config{
		DBConnection: "sqlite",
		DBDatabase:   "file::memory:?cache=shared",
	}
	db, err := Connect(cfg)
	assert.NoError(t, err)
	assert.NotNil(t, db)
}

func TestConnect_PostgresBranchAndOpenError(t *testing.T) {
	origOpen := openGorm
	defer func() { openGorm = origOpen }()

	openGorm = func(dialector gorm.Dialector) (*gorm.DB, error) {
		return nil, errors.New("mock connection failure")
	}

	cfg := &config.Config{
		DBConnection: "pgsql",
		DBHost:       "127.0.0.1",
		DBPort:       "5432",
		DBDatabase:   "pabwe_p9_blog",
		DBUsername:   "postgres",
		DBPassword:   "postgres",
		DBSSLMode:    "disable",
	}
	db, err := Connect(cfg)
	assert.Error(t, err)
	assert.Nil(t, db)
}

func TestConnect_MigrateError(t *testing.T) {
	db, err := openGorm(sqlite.Open(":memory:"))
	assert.NoError(t, err)
	sqlDB, _ := db.DB()
	_ = sqlDB.Close()

	origOpen := openGorm
	defer func() { openGorm = origOpen }()
	openGorm = func(dialector gorm.Dialector) (*gorm.DB, error) {
		return db, nil
	}

	_, err = Connect(&config.Config{DBConnection: "sqlite"})
	assert.Error(t, err)
}

func TestSeedUsers_Errors(t *testing.T) {
	db, err := openGorm(sqlite.Open(":memory:"))
	assert.NoError(t, err)

	// Error when table does not exist during Count
	err = SeedUsers(db, &config.Config{})
	assert.Error(t, err)

	// Migrate table, then test hashPasswordFunc error
	err = db.AutoMigrate(&User{})
	assert.NoError(t, err)

	origHash := hashPasswordFunc
	hashPasswordFunc = func(password string) ([]byte, error) {
		return nil, errors.New("hash error")
	}
	err = SeedUsers(db, &config.Config{})
	assert.Error(t, err)
	hashPasswordFunc = origHash

	// Test db.Create error by adding a callback that fails on create
	db2, err := openGorm(sqlite.Open(":memory:"))
	assert.NoError(t, err)
	_ = db2.AutoMigrate(&User{})
	_ = db2.Callback().Create().Before("gorm:create").Register("fail_create", func(d *gorm.DB) {
		d.AddError(errors.New("create failed"))
	})
	err = SeedUsers(db2, &config.Config{})
	assert.Error(t, err)
}
