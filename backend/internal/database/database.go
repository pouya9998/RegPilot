package database

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/regpilot/regpilot/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Connect(dbPath string) (*gorm.DB, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, fmt.Errorf("create database directory: %w", err)
	}

	db, err := gorm.Open(sqlite.Open(dbPath+"?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(1)

	return db, nil
}

func Migrate(db *gorm.DB) error {
	slog.Info("running database migrations")
	if err := db.AutoMigrate(
		&models.User{},
		&models.Registry{},
		&models.AuditLog{},
		&models.RetentionPolicy{},
	); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	slog.Info("database migrations complete")
	return nil
}
