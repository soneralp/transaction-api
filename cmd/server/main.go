package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"transaction-api-w-go/config"
	"transaction-api-w-go/pkg/database"
	"transaction-api-w-go/pkg/logger"

	"github.com/rs/zerolog/log"
)

func main() {
	logger.Init()
	log.Info().Msg("Starting application...")

	cfg := config.LoadConfig()
	database.Connect(cfg)
	database.RunMigrations()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	log.Info().Msg("Shutdown signal received")

	shutdownCtx, shutdownCancel := context.WithTimeout(ctx, 30*time.Second)
	defer shutdownCancel()

	cleanup(shutdownCtx)
}

func cleanup(ctx context.Context) {
	log.Info().Msg("Starting cleanup...")

	done := make(chan bool)
	go func() {
		database.Close()
		done <- true
	}()

	select {
	case <-done:
		log.Info().Msg("Cleanup completed successfully")
	case <-ctx.Done():
		log.Error().Msg("Cleanup timed out")
	}
}
