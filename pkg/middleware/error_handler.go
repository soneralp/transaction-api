package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
	Code    int    `json:"code"`
}

func ErrorHandlerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) > 0 {
			err := c.Errors.Last()

			statusCode := http.StatusInternalServerError
			errorMessage := "Internal Server Error"

			switch err.Type {
			case gin.ErrorTypeBind:
				statusCode = http.StatusBadRequest
				errorMessage = "Invalid request"
			case gin.ErrorTypePrivate:
				statusCode = http.StatusInternalServerError
				errorMessage = "Internal Server Error"
			case gin.ErrorTypePublic:
				statusCode = http.StatusBadRequest
				errorMessage = err.Error()
			}

			c.JSON(statusCode, ErrorResponse{
				Error:   errorMessage,
				Message: err.Error(),
				Code:    statusCode,
			})
		}
	}
}
