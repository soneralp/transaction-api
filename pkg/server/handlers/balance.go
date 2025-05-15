package handlers

import (
	"net/http"
	"time"

	"transaction-api-w-go/pkg/service"

	"github.com/gin-gonic/gin"
)

type BalanceHandler struct {
	balanceService *service.BalanceService
}

func NewBalanceHandler(balanceService *service.BalanceService) *BalanceHandler {
	return &BalanceHandler{
		balanceService: balanceService,
	}
}

func (h *BalanceHandler) GetCurrentBalance(c *gin.Context) {
	userID := c.GetString("user_id")
	balance, err := h.balanceService.GetCurrentBalance(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, balance)
}

func (h *BalanceHandler) GetHistoricalBalance(c *gin.Context) {
	userID := c.GetString("user_id")
	history, err := h.balanceService.GetHistoricalBalance(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, history)
}

func (h *BalanceHandler) GetBalanceAtTime(c *gin.Context) {
	userID := c.GetString("user_id")
	timestampStr := c.Query("timestamp")

	timestamp, err := time.Parse(time.RFC3339, timestampStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "geçersiz tarih formatı"})
		return
	}

	balance, err := h.balanceService.GetBalanceAtTime(userID, timestamp)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, balance)
}
