package server

import (
	"net/http"

	"transaction-api-w-go/pkg/domain"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AdvancedTransactionHandler struct {
	scheduledService     domain.ScheduledTransactionService
	batchService         domain.BatchTransactionService
	limitService         domain.TransactionLimitService
	multiCurrencyService domain.MultiCurrencyService
}

func NewAdvancedTransactionHandler(
	scheduledService domain.ScheduledTransactionService,
	batchService domain.BatchTransactionService,
	limitService domain.TransactionLimitService,
	multiCurrencyService domain.MultiCurrencyService,
) *AdvancedTransactionHandler {
	return &AdvancedTransactionHandler{
		scheduledService:     scheduledService,
		batchService:         batchService,
		limitService:         limitService,
		multiCurrencyService: multiCurrencyService,
	}
}

func (h *AdvancedTransactionHandler) CreateScheduledTransaction(c *gin.Context) {
	var req domain.ScheduledTransactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userIDStr := c.GetString("user_id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	scheduledTransaction, err := h.scheduledService.CreateScheduledTransaction(c.Request.Context(), userID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"scheduled_transaction": scheduledTransaction,
	})
}

func (h *AdvancedTransactionHandler) GetScheduledTransaction(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid scheduled transaction ID"})
		return
	}

	scheduledTransaction, err := h.scheduledService.GetScheduledTransaction(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"scheduled_transaction": scheduledTransaction,
	})
}

func (h *AdvancedTransactionHandler) GetUserScheduledTransactions(c *gin.Context) {
	userIDStr := c.GetString("user_id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	scheduledTransactions, err := h.scheduledService.GetUserScheduledTransactions(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"scheduled_transactions": scheduledTransactions,
	})
}

func (h *AdvancedTransactionHandler) UpdateScheduledTransaction(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid scheduled transaction ID"})
		return
	}

	var req domain.ScheduledTransactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err = h.scheduledService.UpdateScheduledTransaction(c.Request.Context(), id, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Scheduled transaction updated successfully",
	})
}

func (h *AdvancedTransactionHandler) CancelScheduledTransaction(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid scheduled transaction ID"})
		return
	}

	err = h.scheduledService.CancelScheduledTransaction(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Scheduled transaction cancelled successfully",
	})
}

func (h *AdvancedTransactionHandler) ExecuteScheduledTransactions(c *gin.Context) {
	err := h.scheduledService.ExecuteScheduledTransactions(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Scheduled transactions executed successfully",
	})
}

func (h *AdvancedTransactionHandler) CreateBatchTransaction(c *gin.Context) {
	var req domain.BatchTransactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userIDStr := c.GetString("user_id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	batchTransaction, err := h.batchService.CreateBatchTransaction(c.Request.Context(), userID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"batch_transaction": batchTransaction,
	})
}

func (h *AdvancedTransactionHandler) GetBatchTransaction(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid batch transaction ID"})
		return
	}

	batchTransaction, err := h.batchService.GetBatchTransaction(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"batch_transaction": batchTransaction,
	})
}

func (h *AdvancedTransactionHandler) GetBatchTransactionItems(c *gin.Context) {
	batchIDStr := c.Param("batch_id")
	batchID, err := uuid.Parse(batchIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid batch transaction ID"})
		return
	}

	items, err := h.batchService.GetBatchTransactionItems(c.Request.Context(), batchID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"batch_items": items,
	})
}

func (h *AdvancedTransactionHandler) ProcessBatchTransaction(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid batch transaction ID"})
		return
	}

	err = h.batchService.ProcessBatchTransaction(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Batch transaction processed successfully",
	})
}

func (h *AdvancedTransactionHandler) CancelBatchTransaction(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid batch transaction ID"})
		return
	}

	err = h.batchService.CancelBatchTransaction(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Batch transaction cancelled successfully",
	})
}

func (h *AdvancedTransactionHandler) CreateTransactionLimit(c *gin.Context) {
	var req domain.TransactionLimitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userIDStr := c.GetString("user_id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	limit, err := h.limitService.CreateTransactionLimit(c.Request.Context(), userID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"transaction_limit": limit,
	})
}

func (h *AdvancedTransactionHandler) GetTransactionLimit(c *gin.Context) {
	userIDStr := c.GetString("user_id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	currencyStr := c.Param("currency")
	currency := domain.Currency(currencyStr)

	limit, err := h.limitService.GetTransactionLimit(c.Request.Context(), userID, currency)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"transaction_limit": limit,
	})
}

func (h *AdvancedTransactionHandler) UpdateTransactionLimit(c *gin.Context) {
	userIDStr := c.GetString("user_id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	currencyStr := c.Param("currency")
	currency := domain.Currency(currencyStr)

	var req domain.TransactionLimitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err = h.limitService.UpdateTransactionLimit(c.Request.Context(), userID, currency, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Transaction limit updated successfully",
	})
}

func (h *AdvancedTransactionHandler) ResetTransactionLimits(c *gin.Context) {
	userIDStr := c.GetString("user_id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	currencyStr := c.Param("currency")
	currency := domain.Currency(currencyStr)

	err = h.limitService.ResetTransactionLimits(c.Request.Context(), userID, currency)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Transaction limits reset successfully",
	})
}

func (h *AdvancedTransactionHandler) CreateMultiCurrencyBalance(c *gin.Context) {
	var req struct {
		Currency      domain.Currency `json:"currency" binding:"required"`
		InitialAmount float64         `json:"initial_amount" binding:"required,gt=0"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userIDStr := c.GetString("user_id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	balance, err := h.multiCurrencyService.CreateMultiCurrencyBalance(c.Request.Context(), userID, req.Currency, req.InitialAmount)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"multi_currency_balance": balance,
	})
}

func (h *AdvancedTransactionHandler) GetMultiCurrencyBalance(c *gin.Context) {
	userIDStr := c.GetString("user_id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	currencyStr := c.Param("currency")
	currency := domain.Currency(currencyStr)

	balance, err := h.multiCurrencyService.GetMultiCurrencyBalance(c.Request.Context(), userID, currency)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"multi_currency_balance": balance,
	})
}

func (h *AdvancedTransactionHandler) GetAllBalances(c *gin.Context) {
	userIDStr := c.GetString("user_id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	balances, err := h.multiCurrencyService.GetAllBalances(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"multi_currency_balances": balances,
	})
}

func (h *AdvancedTransactionHandler) ConvertCurrency(c *gin.Context) {
	var req domain.CurrencyConversionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	response, err := h.multiCurrencyService.ConvertCurrency(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"currency_conversion": response,
	})
}

func (h *AdvancedTransactionHandler) TransferBetweenCurrencies(c *gin.Context) {
	var req struct {
		FromCurrency domain.Currency `json:"from_currency" binding:"required"`
		ToCurrency   domain.Currency `json:"to_currency" binding:"required"`
		Amount       float64         `json:"amount" binding:"required,gt=0"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userIDStr := c.GetString("user_id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	err = h.multiCurrencyService.TransferBetweenCurrencies(c.Request.Context(), userID, req.FromCurrency, req.ToCurrency, req.Amount)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Currency transfer completed successfully",
	})
}
