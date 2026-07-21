package db

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Open() (*gorm.DB, error) {
	// dsn := fmt.Sprintf("file:%s?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)", path)
	godotenv.Load()
	dsn := os.Getenv("DATABASE_URL")
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.New(log.New(os.Stdout, "\r\n", log.LstdFlags), logger.Config{
			LogLevel:                  logger.Error,
			IgnoreRecordNotFoundError: true, // auth misses are expected, don't log them
		}),
	})
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	if err := db.AutoMigrate(&User{}, &Usage{}, &Provider{}, &Model{}); err != nil {
		return nil, fmt.Errorf("migrate database: %w", err)
	}
	return db, nil
}
