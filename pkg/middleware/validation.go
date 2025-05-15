package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

func ValidationMiddleware(schema interface{}) gin.HandlerFunc {
	validate := validator.New()

	return func(c *gin.Context) {
		if err := c.ShouldBindJSON(schema); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			c.Abort()
			return
		}

		if err := validate.Struct(schema); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			c.Abort()
			return
		}

		c.Set("validated_data", schema)
		c.Next()
	}
}
