package database

import (
	"fmt"
	"log"

	"github.com/CodeEnthusiast09/mini-brimble/server/internal/config"
	"github.com/CodeEnthusiast09/mini-brimble/server/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Connect(cfg config.DBConfig) (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable TimeZone=UTC",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Name,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	if err := db.Exec("CREATE EXTENSION IF NOT EXISTS pgcrypto").Error; err != nil {
		log.Printf("[db] pgcrypto extension: %v (non-fatal)", err)
	}

	if err := db.AutoMigrate(
		&models.Deployment{},
		&models.LogEntry{},
	); err != nil {
		return nil, fmt.Errorf("auto-migrate: %w", err)
	}

	log.Println("[db] connected and migrated")
	return db, nil
}
