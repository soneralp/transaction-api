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
	eventHandler       *EventHandler
	cacheHandler       *CacheHandler
	advancedHandler    *AdvancedTransactionHandler
	haHandler          *HAHandler
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

		advanced := api.Group("/advanced")
		{
			scheduled := advanced.Group("/scheduled")
			{
				scheduled.POST("", s.advancedHandler.CreateScheduledTransaction)
				scheduled.GET("", s.advancedHandler.GetUserScheduledTransactions)
				scheduled.GET("/:id", s.advancedHandler.GetScheduledTransaction)
				scheduled.PUT("/:id", s.advancedHandler.UpdateScheduledTransaction)
				scheduled.DELETE("/:id", s.advancedHandler.CancelScheduledTransaction)
				scheduled.POST("/execute", s.advancedHandler.ExecuteScheduledTransactions)
			}

			batch := advanced.Group("/batch")
			{
				batch.POST("", s.advancedHandler.CreateBatchTransaction)
				batch.GET("/:id", s.advancedHandler.GetBatchTransaction)
				batch.GET("/:batch_id/items", s.advancedHandler.GetBatchTransactionItems)
				batch.POST("/:id/process", s.advancedHandler.ProcessBatchTransaction)
				batch.DELETE("/:id", s.advancedHandler.CancelBatchTransaction)
			}

			limits := advanced.Group("/limits")
			{
				limits.POST("", s.advancedHandler.CreateTransactionLimit)
				limits.GET("/:currency", s.advancedHandler.GetTransactionLimit)
				limits.PUT("/:currency", s.advancedHandler.UpdateTransactionLimit)
				limits.POST("/:currency/reset", s.advancedHandler.ResetTransactionLimits)
			}

			multiCurrency := advanced.Group("/multi-currency")
			{
				multiCurrency.POST("/balance", s.advancedHandler.CreateMultiCurrencyBalance)
				multiCurrency.GET("/balance/:currency", s.advancedHandler.GetMultiCurrencyBalance)
				multiCurrency.GET("/balances", s.advancedHandler.GetAllBalances)
				multiCurrency.POST("/convert", s.advancedHandler.ConvertCurrency)
				multiCurrency.POST("/transfer", s.advancedHandler.TransferBetweenCurrencies)
			}
		}

		events := api.Group("/events")
		events.Use(middleware.RoleMiddleware("admin")) // Sadece admin'ler event'leri görebilir
		{
			events.GET("/aggregate/:aggregate_id", s.eventHandler.GetEventsByAggregate)
			events.GET("/type/:event_type", s.eventHandler.GetEventsByType)
			events.GET("/time-range", s.eventHandler.GetEventsByTimeRange)
			events.GET("", s.eventHandler.GetAllEvents)
			events.GET("/count/:aggregate_id", s.eventHandler.GetEventCount)

			events.POST("/replay/aggregate/:aggregate_id", s.eventHandler.ReplayEventsForAggregate)
			events.POST("/replay/type/:event_type", s.eventHandler.ReplayEventsByType)
			events.POST("/replay/time-range", s.eventHandler.ReplayEventsByTimeRange)
			events.POST("/replay/all", s.eventHandler.ReplayAllEvents)
			events.GET("/replay/statistics", s.eventHandler.GetReplayStatistics)
		}

		cache := api.Group("/cache")
		cache.Use(middleware.RoleMiddleware("admin")) // Sadece admin'ler cache'i yönetebilir
		{
			cache.GET("/stats", s.cacheHandler.GetCacheStats)
			cache.DELETE("/flush", s.cacheHandler.FlushAllCache)
			cache.GET("/ttl/:key", s.cacheHandler.GetCacheTTL)
			cache.GET("/exists/:key", s.cacheHandler.CheckCacheExists)
			cache.POST("/increment/:key", s.cacheHandler.IncrementCacheKey)

			cache.POST("/warmup/users", s.cacheHandler.WarmupUsers)
			cache.POST("/warmup/transactions", s.cacheHandler.WarmupTransactions)
			cache.POST("/warmup/balances", s.cacheHandler.WarmupBalances)
			cache.POST("/warmup/aggregate-events", s.cacheHandler.WarmupAggregateEvents)

			cache.DELETE("/invalidate/user/:user_id", s.cacheHandler.InvalidateUser)
			cache.DELETE("/invalidate/transaction/:transaction_id", s.cacheHandler.InvalidateTransaction)
			cache.DELETE("/invalidate/balance/:user_id", s.cacheHandler.InvalidateBalance)
			cache.DELETE("/invalidate/aggregate-events/:aggregate_id", s.cacheHandler.InvalidateAggregateEvents)

			cache.GET("/user/:user_id", s.cacheHandler.GetCachedUser)
			cache.GET("/transaction/:transaction_id", s.cacheHandler.GetCachedTransaction)
			cache.GET("/balance/:user_id", s.cacheHandler.GetCachedBalance)
			cache.GET("/user/:user_id/transactions", s.cacheHandler.GetCachedUserTransactions)
			cache.GET("/aggregate-events/:aggregate_id", s.cacheHandler.GetCachedAggregateEvents)
		}

		ha := api.Group("/ha")
		ha.Use(middleware.RoleMiddleware("admin")) // Sadece admin'ler HA'yı yönetebilir
		{
			ha.GET("/health", s.haHandler.GetSystemHealth)
			ha.GET("/metrics", s.haHandler.GetHAMetrics)

			ha.GET("/database/health", s.haHandler.GetDatabaseHealth)
			ha.GET("/database/health/:node", s.haHandler.GetDatabaseNodeHealth)
			ha.POST("/database/failover", s.haHandler.ForceDatabaseFailover)

			ha.GET("/loadbalancer/stats", s.haHandler.GetLoadBalancerStats)
			ha.POST("/loadbalancer/backends", s.haHandler.AddLoadBalancerBackend)
			ha.DELETE("/loadbalancer/backends/:id", s.haHandler.RemoveLoadBalancerBackend)

			ha.GET("/circuitbreakers", s.haHandler.GetAllCircuitBreakers)
			ha.GET("/circuitbreakers/:name", s.haHandler.GetCircuitBreakerStats)
			ha.POST("/circuitbreakers", s.haHandler.CreateCircuitBreaker)
			ha.POST("/circuitbreakers/:name/open", s.haHandler.ForceCircuitBreakerOpen)
			ha.POST("/circuitbreakers/:name/close", s.haHandler.ForceCircuitBreakerClose)
			ha.POST("/circuitbreakers/:name/reset", s.haHandler.ResetCircuitBreaker)

			ha.GET("/fallback/stats", s.haHandler.GetFallbackStats)
			ha.POST("/fallback/test", s.haHandler.TestFallback)

			ha.GET("/config", s.haHandler.GetHAConfig)
			ha.PUT("/config", s.haHandler.UpdateHAConfig)
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
	eventHandler *EventHandler,
	cacheHandler *CacheHandler,
	advancedHandler *AdvancedTransactionHandler,
	haHandler *HAHandler,
) {
	s.authHandler = authHandler
	s.userHandler = userHandler
	s.transactionHandler = transactionHandler
	s.balanceHandler = balanceHandler
	s.eventHandler = eventHandler
	s.cacheHandler = cacheHandler
	s.advancedHandler = advancedHandler
	s.haHandler = haHandler
	s.setupRoutes()
}
