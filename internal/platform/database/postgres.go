package database

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// NewPostgresDB initializes the database connection with cost-saving pooling settings
func NewPostgresDB() (*gorm.DB, error) {
	// 1. If DB_HOST is not configured, fall back to local SQLite for development & offline testing
	if os.Getenv("DB_HOST") == "" {
		log.Println("⚠️ DB_HOST environment variable not set. Falling back to local SQLite database (learnsteer_dev.db)")
		db, err := gorm.Open(sqlite.Open("learnsteer_dev.db"), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Info),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to connect to local sqlite database: %w", err)
		}
		log.Println("✅ Local SQLite database connection established for dev mode")
		return db, nil
	}

	// 2. Get credentials from Environment Variables
	sslMode := os.Getenv("DB_SSLMODE")
	if sslMode == "" {
		sslMode = "disable"
	}

	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=Asia/Colombo",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_PORT"),
		sslMode,
	)

	// 3. Open Connection
	config := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	}

	db, err := gorm.Open(postgres.Open(dsn), config)
	if err != nil {
		log.Printf("⚠️ PostgreSQL connection failed (%v). Falling back to local SQLite (learnsteer_dev.db)", err)
		sqliteDB, sqliteErr := gorm.Open(sqlite.Open("learnsteer_dev.db"), config)
		if sqliteErr != nil {
			return nil, fmt.Errorf("failed to connect to postgres: %v, and sqlite fallback failed: %w", err, sqliteErr)
		}
		log.Println("✅ Local SQLite database connection established for dev mode")
		return sqliteDB, nil
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
