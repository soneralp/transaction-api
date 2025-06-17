package handlers

import (
	"net/http"
	"strconv"

	"transaction-api-w-go/pkg/domain"
	"transaction-api-w-go/pkg/service"

	"github.com/gin-gonic/gin"
)

type TransactionHandler struct {
	transactionService *service.TransactionService
}

func NewTransactionHandler(transactionService *service.TransactionService) *TransactionHandler {
	return &TransactionHandler{
		transactionService: transactionService,
	}
}

func (h *TransactionHandler) Credit(c *gin.Context) {
	req := c.MustGet("validated_data").(*domain.TransactionRequest)

	userID := c.GetString("user_id")
	transaction, err := h.transactionService.Credit(c.Request.Context(), userID, req.Amount, req.Description)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, transaction)
}

func (h *TransactionHandler) Debit(c *gin.Context) {
	req := c.MustGet("validated_data").(*domain.TransactionRequest)

	userID := c.GetString("user_id")
	transaction, err := h.transactionService.Debit(c.Request.Context(), userID, req.Amount, req.Description)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, transaction)
}

func (h *TransactionHandler) Transfer(c *gin.Context) {
	req := c.MustGet("validated_data").(*domain.TransferRequest)

	fromUserID := c.GetString("user_id")
	transaction, err := h.transactionService.Transfer(c.Request.Context(), fromUserID, req.ToUserID.String(), req.Amount, req.Description)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, transaction)
}

func (h *TransactionHandler) GetHistory(c *gin.Context) {
	userIDStr := c.GetString("user_id")
	userID, err := strconv.ParseUint(userIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Geçersiz user ID"})
		return
	}
	transactions, err := h.transactionService.GetHistory(c.Request.Context(), uint(userID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, transactions)
}

func (h *TransactionHandler) GetByID(c *gin.Context) {
	transactionIDStr := c.Param("id")
	transactionID, err := strconv.ParseUint(transactionIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Geçersiz transaction ID"})
		return
	}
	transaction, err := h.transactionService.GetByID(c.Request.Context(), uint(transactionID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, transaction)
}
