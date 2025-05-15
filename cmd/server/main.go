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
	"transaction-api-w-go/pkg/repository"
	"transaction-api-w-go/pkg/server"
	"transaction-api-w-go/pkg/server/handlers"
	"transaction-api-w-go/pkg/service"

	"github.com/rs/zerolog/log"
)

func main() {
	logger.Init()
	log.Info().Msg("Starting application...")

	cfg := config.LoadConfig()
	database.Connect(cfg)
	database.RunMigrations()

	// Repository'leri oluştur
	userRepo := repository.NewUserRepository(database.GetDB())
	transactionRepo := repository.NewTransactionRepository(database.GetDB())
	balanceRepo := repository.NewBalanceRepository(database.GetDB())

	// Servisleri oluştur
	authService := service.NewAuthService(userRepo, cfg.JWTSecret, cfg.JWTRefreshSecret)
	userService := service.NewUserService(userRepo)
	transactionService := service.NewTransactionService(transactionRepo, balanceRepo, userRepo)
	balanceService := service.NewBalanceService(balanceRepo)

	// Handler'ları oluştur
	authHandler := handlers.NewAuthHandler(authService)
	userHandler := handlers.NewUserHandler(userService)
	transactionHandler := handlers.NewTransactionHandler(transactionService)
	balanceHandler := handlers.NewBalanceHandler(balanceService)

	// HTTP sunucusunu başlat
	srv := server.NewServer(8081)
	srv.SetHandlers(authHandler, userHandler, transactionHandler, balanceHandler)

	go func() {
		if err := srv.Start(); err != nil {
			log.Fatal().Err(err).Msg("HTTP sunucusu başlatılamadı")
		}
	}()

	// Graceful shutdown için sinyal bekle
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	log.Info().Msg("Shutdown signal received")

	// Graceful shutdown için timeout ile context oluştur
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	cleanup(shutdownCtx, srv)
}

func cleanup(ctx context.Context, srv *server.Server) {
	log.Info().Msg("Temizlik işlemleri başlatılıyor...")

	done := make(chan bool)
	go func() {
		// HTTP sunucusunu kapat
		if err := srv.Shutdown(ctx); err != nil {
			log.Error().Err(err).Msg("HTTP sunucusu kapatılırken hata oluştu")
		}

		// Veritabanı bağlantısını kapat
		database.Close()
		done <- true
	}()

	select {
	case <-done:
		log.Info().Msg("Temizlik işlemleri başarıyla tamamlandı")
	case <-ctx.Done():
		log.Error().Msg("Temizlik işlemleri zaman aşımına uğradı")
	}
}
