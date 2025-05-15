package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

func PerformanceMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {

		startTime := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method

		c.Next()

		duration := time.Since(startTime)

		if c.Writer.Status() >= 400 {
			log.Error().
				Str("method", method).
				Str("path", path).
				Int("status", c.Writer.Status()).
				Dur("duration", duration).
				Msg("Request error")
		} else {
			log.Info().
				Str("method", method).
				Str("path", path).
				Int("status", c.Writer.Status()).
				Dur("duration", duration).
				Msg("Request successful")
		}

		c.Header("X-Response-Time", duration.String())
	}
}
