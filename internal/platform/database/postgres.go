package database

import (
	"fmt"
	"log"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// NewPostgresDB initializes the database connection with cost-saving pooling settings
func NewPostgresDB() (*gorm.DB, error) {
	// 1. Get credentials from Environment Variables (Best practice for Cloud)
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=require TimeZone=Asia/Colombo",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_PORT"),
	)

	// 2. Open Connection
	// We use a silent logger in prod to save log storage costs, but Info in dev
	config := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	}

	db, err := gorm.Open(postgres.Open(dsn), config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// 3. Configure Connection Pooling (CRITICAL for Cost/Stability)
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	// SetMaxIdleConns: specific number of connections to keep ready (free tier usually allows ~20)
	sqlDB.SetMaxIdleConns(5)

	// SetMaxOpenConns: limit total open connections to prevent crashing the DB
	sqlDB.SetMaxOpenConns(20)

	// SetConnMaxLifetime: recycle connections every hour to prevent "stale connection" errors in Azure
	sqlDB.SetConnMaxLifetime(time.Hour)

	log.Println("✅ Database connection established with pooling configured")
	return db, nil
}
