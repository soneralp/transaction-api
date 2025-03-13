package database

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/log"
)

func RunMigrations() {
	log.Info().Msg("Running database migrations...")

	migrationPath := filepath.Join("migrations", "init.sql")
	migrationSQL, err := os.ReadFile(migrationPath)
	if err != nil {
		log.Fatal().Err(err).Str("path", migrationPath).Msg("Failed to read migration file")
	}

	queries := strings.Split(string(migrationSQL), ";")

	for _, query := range queries {
		query = strings.TrimSpace(query)
		if query == "" {
			continue
		}

		_, err = DB.Exec(query)
		if err != nil {
			log.Fatal().Err(err).Str("query", query).Msg("Failed to execute migration query")
		}
	}

	log.Info().Msg("Database migrations completed successfully")
}
