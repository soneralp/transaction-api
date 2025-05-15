package database

import (
	"fmt"
	"time"

	"transaction-api-w-go/config"

	"github.com/rs/zerolog/log"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func createConnection(cfg *config.Config) (*gorm.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true",
		cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort, cfg.DBName)

	log.Debug().
		Str("host", cfg.DBHost).
		Str("database", cfg.DBName).
		Msg("Attempting database connection")

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, err
	}

	log.Info().
		Str("host", cfg.DBHost).
		Str("database", cfg.DBName).
		Msg("Database connection established")

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database instance: %v", err)
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	return db, nil
}

func Connect(cfg *config.Config) {
	var err error
	maxRetries := 5
	retryDelay := time.Second * 5

	for i := 0; i < maxRetries; i++ {
		DB, err = createConnection(cfg)
		if err == nil {
			return
		}

		log.Warn().
			Int("attempt", i+1).
			Int("maxRetries", maxRetries).
			Str("host", cfg.DBHost).
			Str("database", cfg.DBName).
			Err(err).
			Msg("Failed to connect to database")

		time.Sleep(retryDelay)
	}

	log.Fatal().
		Int("maxRetries", maxRetries).
		Str("host", cfg.DBHost).
		Str("database", cfg.DBName).
		Err(err).
		Msg("Could not connect to database after maximum retries")
}

func Close() {
	if DB != nil {
		log.Info().Msg("Closing database connection")
		sqlDB, err := DB.DB()
		if err != nil {
			log.Error().Err(err).Msg("Error getting underlying sql.DB")
			return
		}
		if err := sqlDB.Close(); err != nil {
			log.Error().Err(err).Msg("Error closing database connection")
		}
	}
}

// GetDB returns the database connection
func GetDB() *gorm.DB {
	return DB
}
