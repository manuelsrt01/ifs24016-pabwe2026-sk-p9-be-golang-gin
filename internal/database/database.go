package database

import (
	"fmt"
	"strings"
	"time"

	"app/internal/config"

	"github.com/glebarez/sqlite"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type User struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Username  string    `gorm:"type:varchar(255);uniqueIndex;not null" json:"username"`
	Email     string    `gorm:"type:varchar(255);uniqueIndex;not null" json:"email"`
	Password  string    `gorm:"type:text;not null" json:"-"`
	Role      string    `gorm:"type:varchar(50);default:'member';not null" json:"role"`
	Posts     []Post    `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"posts,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Post struct {
	ID             uint       `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID         uint       `gorm:"not null;index" json:"user_id"`
	User           *User      `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"author,omitempty"`
	Title          string     `gorm:"type:varchar(255);not null" json:"title"`
	Slug           string     `gorm:"type:varchar(255);uniqueIndex;not null" json:"slug"`
	Category       string     `gorm:"type:varchar(100);not null;default:'Umum'" json:"category"`
	Excerpt        string     `gorm:"type:text" json:"excerpt"`
	Content        string     `gorm:"type:text;not null" json:"content"`
	CoverImagePath string     `gorm:"type:text" json:"cover_image_path"`
	IsPublished    bool       `gorm:"default:false;not null" json:"is_published"`
	PublishedAt    *time.Time `json:"published_at"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

var openGorm = func(dialector gorm.Dialector) (*gorm.DB, error) {
	return gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
}

var hashPasswordFunc = func(password string) ([]byte, error) {
	return bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
}

func Connect(cfg *config.Config) (*gorm.DB, error) {
	var dialector gorm.Dialector
	conn := strings.ToLower(strings.TrimSpace(cfg.DBConnection))
	if conn == "sqlite" || conn == "memory" {
		dsn := cfg.DBDatabase
		if dsn == "" || dsn == "pabwe_p9_blog" {
			dsn = ":memory:"
		}
		dialector = sqlite.Open(dsn)
	} else {
		dsn := fmt.Sprintf(
			"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=Asia/Jakarta",
			cfg.DBHost, cfg.DBUsername, cfg.DBPassword, cfg.DBDatabase, cfg.DBPort, cfg.DBSSLMode,
		)
		dialector = postgres.Open(dsn)
	}

	db, err := openGorm(dialector)
	if err != nil {
		return nil, err
	}

	if err := MigrateAndSeed(db, cfg); err != nil {
		return nil, err
	}

	return db, nil
}

func ConnectTestDB() (*gorm.DB, error) {
	cfg := &config.Config{
		DBConnection:  "sqlite",
		DBDatabase:    ":memory:",
		AdminUsername: "admin",
		AdminPassword: "admin",
	}
	return Connect(cfg)
}

func MigrateAndSeed(db *gorm.DB, cfg *config.Config) error {
	if err := db.AutoMigrate(&User{}, &Post{}); err != nil {
		return err
	}
	return SeedUsers(db, cfg)
}

func SeedUsers(db *gorm.DB, cfg *config.Config) error {
	adminUser := cfg.AdminUsername
	if adminUser == "" {
		adminUser = "admin"
	}
	adminPass := cfg.AdminPassword
	if adminPass == "" {
		adminPass = "admin"
	}

	defaults := []struct {
		Username string
		Email    string
		Password string
		Role     string
	}{
		{Username: adminUser, Email: adminUser + "@delcom.org", Password: adminPass, Role: "admin"},
		{Username: "member", Email: "member@delcom.org", Password: "member", Role: "member"},
	}

	for _, u := range defaults {
		var count int64
		if err := db.Model(&User{}).Where("username = ?", u.Username).Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			hashed, err := hashPasswordFunc(u.Password)
			if err != nil {
				return err
			}
			newUser := User{
				Username: u.Username,
				Email:    u.Email,
				Password: string(hashed),
				Role:     u.Role,
			}
			if err := db.Create(&newUser).Error; err != nil {
				return err
			}
		}
	}

	return nil
}
