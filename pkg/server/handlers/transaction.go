package handlers

import (
	"net/http"

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
	transaction, err := h.transactionService.Credit(userID, req.Amount, req.Description)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, transaction)
}

func (h *TransactionHandler) Debit(c *gin.Context) {
	req := c.MustGet("validated_data").(*domain.TransactionRequest)

	userID := c.GetString("user_id")
	transaction, err := h.transactionService.Debit(userID, req.Amount, req.Description)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, transaction)
}

func (h *TransactionHandler) Transfer(c *gin.Context) {
	req := c.MustGet("validated_data").(*domain.TransferRequest)

	fromUserID := c.GetString("user_id")
	transaction, err := h.transactionService.Transfer(fromUserID, req.ToUserID.String(), req.Amount, req.Description)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, transaction)
}

func (h *TransactionHandler) GetHistory(c *gin.Context) {
	userID := c.GetString("user_id")
	transactions, err := h.transactionService.GetHistory(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, transactions)
}

func (h *TransactionHandler) GetByID(c *gin.Context) {
	userID := c.GetString("user_id")
	transactionID := c.Param("id")

	transaction, err := h.transactionService.GetByID(userID, transactionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, transaction)
}
