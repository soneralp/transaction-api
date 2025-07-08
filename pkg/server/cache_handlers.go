package server

import (
	"net/http"
	"strconv"

	"transaction-api-w-go/pkg/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type CacheHandler struct {
	cacheService *service.CacheService
}

func NewCacheHandler(cacheService *service.CacheService) *CacheHandler {
	return &CacheHandler{
		cacheService: cacheService,
	}
}

func (h *CacheHandler) GetCacheStats(c *gin.Context) {
	stats, err := h.cacheService.GetCacheStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"stats": stats,
	})
}

func (h *CacheHandler) FlushAllCache(c *gin.Context) {
	err := h.cacheService.FlushAll(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "All cache flushed successfully",
	})
}

func (h *CacheHandler) GetCacheTTL(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Key parameter is required"})
		return
	}

	ttl, err := h.cacheService.GetTTL(c.Request.Context(), key)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"key":         key,
		"ttl":         ttl.String(),
		"ttl_seconds": int64(ttl.Seconds()),
	})
}

func (h *CacheHandler) CheckCacheExists(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Key parameter is required"})
		return
	}

	exists, err := h.cacheService.Exists(c.Request.Context(), key)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"key":    key,
		"exists": exists,
	})
}

func (h *CacheHandler) IncrementCacheKey(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Key parameter is required"})
		return
	}

	valueStr := c.DefaultQuery("value", "1")
	value, err := strconv.ParseInt(valueStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid value parameter"})
		return
	}

	result, err := h.cacheService.Increment(c.Request.Context(), key, value)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"key":    key,
		"value":  value,
		"result": result,
	})
}

func (h *CacheHandler) WarmupUsers(c *gin.Context) {
	var request struct {
		UserIDs []uuid.UUID `json:"user_ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.cacheService.WarmupUsers(c.Request.Context(), request.UserIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "Users warmed up successfully",
		"user_ids": request.UserIDs,
	})
}

func (h *CacheHandler) WarmupTransactions(c *gin.Context) {
	var request struct {
		TransactionIDs []uuid.UUID `json:"transaction_ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.cacheService.WarmupTransactions(c.Request.Context(), request.TransactionIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":         "Transactions warmed up successfully",
		"transaction_ids": request.TransactionIDs,
	})
}

func (h *CacheHandler) WarmupBalances(c *gin.Context) {
	var request struct {
		UserIDs []uuid.UUID `json:"user_ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.cacheService.WarmupBalances(c.Request.Context(), request.UserIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "Balances warmed up successfully",
		"user_ids": request.UserIDs,
	})
}

func (h *CacheHandler) WarmupAggregateEvents(c *gin.Context) {
	var request struct {
		AggregateIDs []uuid.UUID `json:"aggregate_ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.cacheService.WarmupAggregateEvents(c.Request.Context(), request.AggregateIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "Aggregate events warmed up successfully",
		"aggregate_ids": request.AggregateIDs,
	})
}

func (h *CacheHandler) InvalidateUser(c *gin.Context) {
	userIDStr := c.Param("user_id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	err = h.cacheService.InvalidateUser(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "User cache invalidated successfully",
		"user_id": userID,
	})
}

func (h *CacheHandler) InvalidateTransaction(c *gin.Context) {
	transactionIDStr := c.Param("transaction_id")
	transactionID, err := uuid.Parse(transactionIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid transaction ID"})
		return
	}

	err = h.cacheService.InvalidateTransaction(c.Request.Context(), transactionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":        "Transaction cache invalidated successfully",
		"transaction_id": transactionID,
	})
}

func (h *CacheHandler) InvalidateBalance(c *gin.Context) {
	userIDStr := c.Param("user_id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	err = h.cacheService.InvalidateBalance(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Balance cache invalidated successfully",
		"user_id": userID,
	})
}

func (h *CacheHandler) InvalidateAggregateEvents(c *gin.Context) {
	aggregateIDStr := c.Param("aggregate_id")
	aggregateID, err := uuid.Parse(aggregateIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid aggregate ID"})
		return
	}

	err = h.cacheService.InvalidateAggregateEvents(c.Request.Context(), aggregateID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "Aggregate events cache invalidated successfully",
		"aggregate_id": aggregateID,
	})
}

func (h *CacheHandler) GetCachedUser(c *gin.Context) {
	userIDStr := c.Param("user_id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	user, err := h.cacheService.GetUser(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user": user,
	})
}

func (h *CacheHandler) GetCachedTransaction(c *gin.Context) {
	transactionIDStr := c.Param("transaction_id")
	transactionID, err := uuid.Parse(transactionIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid transaction ID"})
		return
	}

	transaction, err := h.cacheService.GetTransaction(c.Request.Context(), transactionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"transaction": transaction,
	})
}

func (h *CacheHandler) GetCachedBalance(c *gin.Context) {
	userIDStr := c.Param("user_id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	balance, err := h.cacheService.GetBalance(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"balance": balance,
	})
}

func (h *CacheHandler) GetCachedUserTransactions(c *gin.Context) {
	userIDStr := c.Param("user_id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	limitStr := c.DefaultQuery("limit", "10")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid limit parameter"})
		return
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid offset parameter"})
		return
	}

	transactions, err := h.cacheService.GetUserTransactions(c.Request.Context(), userID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"transactions": transactions,
		"user_id":      userID,
		"limit":        limit,
		"offset":       offset,
	})
}

func (h *CacheHandler) GetCachedAggregateEvents(c *gin.Context) {
	aggregateIDStr := c.Param("aggregate_id")
	aggregateID, err := uuid.Parse(aggregateIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid aggregate ID"})
		return
	}

	events, err := h.cacheService.GetAggregateEvents(c.Request.Context(), aggregateID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"events":       events,
		"aggregate_id": aggregateID,
		"count":        len(events),
	})
}
