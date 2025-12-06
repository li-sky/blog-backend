package config

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// InitDB initializes and returns a GORM DB instance
func InitDB() *gorm.DB {
	if err := godotenv.Load(); err != nil {
		log.Println("Error loading .env file")
	}

	dbConnStr := os.Getenv("DATABASE_URL")
	if dbConnStr == "" {
		log.Fatal("DATABASE_URL is not set in the environment")
	}

	maxAttempts := parseEnvInt("DB_MAX_RETRIES", 5)
	if maxAttempts < 1 {
		maxAttempts = 1
	}
	retryDelay := time.Second * time.Duration(parseEnvInt("DB_RETRY_DELAY_SECONDS", 3))
	if retryDelay <= 0 {
		retryDelay = 3 * time.Second
	}

	var (
		db  *gorm.DB
		err error
	)
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		db, err = gorm.Open(postgres.Open(dbConnStr), &gorm.Config{})
		if err == nil {
			break
		}
		log.Printf("Failed to connect to database (attempt %d/%d): %v", attempt, maxAttempts, err)
		if attempt < maxAttempts {
			time.Sleep(retryDelay)
		}
	}
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatal("Failed to get database instance:", err)
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)

	return db
}

func parseEnvInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}
