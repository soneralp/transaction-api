package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"transaction-api-w-go/pkg/domain"
	"transaction-api-w-go/pkg/middleware"
	"transaction-api-w-go/pkg/server/handlers"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
	"golang.org/x/time/rate"
)

type Server struct {
	engine             *gin.Engine
	server             *http.Server
	limiter            *rate.Limiter
	authHandler        *handlers.AuthHandler
	userHandler        *handlers.UserHandler
	transactionHandler *handlers.TransactionHandler
	balanceHandler     *handlers.BalanceHandler
	jwtSecret          string
}

func NewServer(port int) *Server {
	engine := gin.Default()

	limiter := rate.NewLimiter(rate.Limit(100), 100)

	server := &Server{
		engine: engine,
		server: &http.Server{
			Addr:         fmt.Sprintf(":%d", port),
			Handler:      engine,
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 15 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
		limiter:   limiter,
		jwtSecret: "your-secret-key",
	}

	server.setupMiddleware()

	return server
}

func (s *Server) setupMiddleware() {
	s.engine.Use(middleware.ErrorHandlerMiddleware())
	s.engine.Use(middleware.PerformanceMiddleware())
	s.engine.Use(middleware.MetricsMiddleware())

	s.engine.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	s.engine.Use(func(c *gin.Context) {
		c.Writer.Header().Set("X-Content-Type-Options", "nosniff")
		c.Writer.Header().Set("X-Frame-Options", "DENY")
		c.Writer.Header().Set("X-XSS-Protection", "1; mode=block")
		c.Writer.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		c.Writer.Header().Set("Content-Security-Policy", "default-src 'self'")

		c.Next()
	})

	s.engine.Use(func(c *gin.Context) {
		if !s.limiter.Allow() {
			c.AbortWithStatus(http.StatusTooManyRequests)
			return
		}
		c.Next()
	})

	s.engine.Use(func(c *gin.Context) {
		start := time.Now()

		c.Next()

		log.Info().
			Str("method", c.Request.Method).
			Str("path", c.Request.URL.Path).
			Int("status", c.Writer.Status()).
			Dur("duration", time.Since(start)).
			Str("ip", c.ClientIP()).
			Str("user_agent", c.Request.UserAgent()).
			Msg("HTTP request")
	})
}

func (s *Server) setupRoutes() {
	s.engine.GET("/metrics", gin.WrapH(promhttp.Handler()))

	auth := s.engine.Group("/api/v1/auth")
	{
		auth.POST("/register", middleware.ValidationMiddleware(&domain.RegisterRequest{}), s.authHandler.Register)
		auth.POST("/login", middleware.ValidationMiddleware(&domain.LoginRequest{}), s.authHandler.Login)
		auth.POST("/refresh", middleware.ValidationMiddleware(&domain.RefreshTokenRequest{}), s.authHandler.RefreshToken)
	}

	api := s.engine.Group("/api/v1")
	api.Use(middleware.AuthMiddleware(s.jwtSecret))
	{
		users := api.Group("/users")
		users.Use(middleware.RoleMiddleware("admin"))
		{
			users.GET("", s.userHandler.GetUsers)
			users.GET("/:id", s.userHandler.GetUser)
			users.PUT("/:id", middleware.ValidationMiddleware(&domain.User{}), s.userHandler.UpdateUser)
			users.DELETE("/:id", s.userHandler.DeleteUser)
		}

		transactions := api.Group("/transactions")
		{
			transactions.POST("/credit", middleware.ValidationMiddleware(&domain.TransactionRequest{}), s.transactionHandler.Credit)
			transactions.POST("/debit", middleware.ValidationMiddleware(&domain.TransactionRequest{}), s.transactionHandler.Debit)
			transactions.POST("/transfer", middleware.ValidationMiddleware(&domain.TransferRequest{}), s.transactionHandler.Transfer)
			transactions.GET("/history", s.transactionHandler.GetHistory)
			transactions.GET("/:id", s.transactionHandler.GetByID)
		}

		balances := api.Group("/balances")
		{
			balances.GET("/current", s.balanceHandler.GetCurrentBalance)
			balances.GET("/historical", s.balanceHandler.GetHistoricalBalance)
			balances.GET("/at-time", s.balanceHandler.GetBalanceAtTime)
		}
	}
}

func (s *Server) Start() error {
	log.Info().Str("addr", s.server.Addr).Msg("Starting HTTP server")
	return s.server.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	log.Info().Msg("Shutting down HTTP server")
	return s.server.Shutdown(ctx)
}

func (s *Server) GetEngine() *gin.Engine {
	return s.engine
}

func (s *Server) healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (s *Server) SetHandlers(
	authHandler *handlers.AuthHandler,
	userHandler *handlers.UserHandler,
	transactionHandler *handlers.TransactionHandler,
	balanceHandler *handlers.BalanceHandler,
) {
	s.authHandler = authHandler
	s.userHandler = userHandler
	s.transactionHandler = transactionHandler
	s.balanceHandler = balanceHandler
	s.setupRoutes()
}
