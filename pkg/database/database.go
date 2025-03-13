package database

import (
	"fmt"
	"time"

	"transaction-api-w-go/config"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog/log"
)

var DB *sqlx.DB

func createConnection(cfg *config.Config) (*sqlx.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true",
		cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort, cfg.DBName)

	log.Debug().
		Str("host", cfg.DBHost).
		Str("database", cfg.DBName).
		Msg("Attempting database connection")

	db, err := sqlx.Connect("mysql", dsn)
	if err != nil {
		return nil, err
	}

	log.Info().
		Str("host", cfg.DBHost).
		Str("database", cfg.DBName).
		Msg("Database connection established")

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
		if err := DB.Close(); err != nil {
			log.Error().Err(err).Msg("Error closing database connection")
		}
	}
}
