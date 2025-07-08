package server

import (
	"net/http"
	"strconv"
	"time"

	"transaction-api-w-go/pkg/domain"
	"transaction-api-w-go/pkg/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type EventHandler struct {
	eventReplayService *service.EventReplayService
	eventStore         domain.EventStore
}

func NewEventHandler(eventReplayService *service.EventReplayService, eventStore domain.EventStore) *EventHandler {
	return &EventHandler{
		eventReplayService: eventReplayService,
		eventStore:         eventStore,
	}
}

func (h *EventHandler) GetEventsByAggregate(c *gin.Context) {
	aggregateIDStr := c.Param("aggregate_id")
	aggregateID, err := uuid.Parse(aggregateIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid aggregate ID"})
		return
	}

	events, err := h.eventStore.GetEvents(c.Request.Context(), aggregateID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	eventResponses := make([]gin.H, len(events))
	for i, event := range events {
		eventResponses[i] = gin.H{
			"id":           event.GetID(),
			"type":         event.GetType(),
			"aggregate_id": event.GetAggregateID(),
			"version":      event.GetVersion(),
			"timestamp":    event.GetTimestamp(),
			"data":         event.GetData(),
			"metadata":     event.GetMetadata(),
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"aggregate_id": aggregateID,
		"events":       eventResponses,
		"count":        len(events),
	})
}

func (h *EventHandler) GetEventsByType(c *gin.Context) {
	eventType := domain.EventType(c.Param("event_type"))

	limitStr := c.DefaultQuery("limit", "100")
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

	events, err := h.eventStore.GetEventsByType(c.Request.Context(), eventType, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	eventResponses := make([]gin.H, len(events))
	for i, event := range events {
		eventResponses[i] = gin.H{
			"id":           event.GetID(),
			"type":         event.GetType(),
			"aggregate_id": event.GetAggregateID(),
			"version":      event.GetVersion(),
			"timestamp":    event.GetTimestamp(),
			"data":         event.GetData(),
			"metadata":     event.GetMetadata(),
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"event_type": eventType,
		"events":     eventResponses,
		"count":      len(events),
		"limit":      limit,
		"offset":     offset,
	})
}

func (h *EventHandler) GetEventsByTimeRange(c *gin.Context) {
	startTimeStr := c.Query("start_time")
	endTimeStr := c.Query("end_time")

	if startTimeStr == "" || endTimeStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "start_time and end_time parameters are required"})
		return
	}

	startTime, err := time.Parse(time.RFC3339, startTimeStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid start_time format. Use RFC3339 format"})
		return
	}

	endTime, err := time.Parse(time.RFC3339, endTimeStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid end_time format. Use RFC3339 format"})
		return
	}

	events, err := h.eventStore.GetEventsByTimeRange(c.Request.Context(), startTime, endTime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	eventResponses := make([]gin.H, len(events))
	for i, event := range events {
		eventResponses[i] = gin.H{
			"id":           event.GetID(),
			"type":         event.GetType(),
			"aggregate_id": event.GetAggregateID(),
			"version":      event.GetVersion(),
			"timestamp":    event.GetTimestamp(),
			"data":         event.GetData(),
			"metadata":     event.GetMetadata(),
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"start_time": startTime,
		"end_time":   endTime,
		"events":     eventResponses,
		"count":      len(events),
	})
}

func (h *EventHandler) GetAllEvents(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "100")
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

	events, err := h.eventStore.GetAllEvents(c.Request.Context(), limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	eventResponses := make([]gin.H, len(events))
	for i, event := range events {
		eventResponses[i] = gin.H{
			"id":           event.GetID(),
			"type":         event.GetType(),
			"aggregate_id": event.GetAggregateID(),
			"version":      event.GetVersion(),
			"timestamp":    event.GetTimestamp(),
			"data":         event.GetData(),
			"metadata":     event.GetMetadata(),
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"events": eventResponses,
		"count":  len(events),
		"limit":  limit,
		"offset": offset,
	})
}

func (h *EventHandler) ReplayEventsForAggregate(c *gin.Context) {
	aggregateIDStr := c.Param("aggregate_id")
	aggregateID, err := uuid.Parse(aggregateIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid aggregate ID"})
		return
	}

	err = h.eventReplayService.ReplayEventsForAggregate(c.Request.Context(), aggregateID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "Events replayed successfully",
		"aggregate_id": aggregateID,
	})
}

func (h *EventHandler) ReplayEventsByType(c *gin.Context) {
	eventType := domain.EventType(c.Param("event_type"))

	limitStr := c.DefaultQuery("limit", "1000")
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

	err = h.eventReplayService.ReplayEventsByType(c.Request.Context(), eventType, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "Events replayed successfully",
		"event_type": eventType,
		"limit":      limit,
		"offset":     offset,
	})
}

func (h *EventHandler) ReplayEventsByTimeRange(c *gin.Context) {
	var request struct {
		StartTime string `json:"start_time" binding:"required"`
		EndTime   string `json:"end_time" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	startTime, err := time.Parse(time.RFC3339, request.StartTime)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid start_time format. Use RFC3339 format"})
		return
	}

	endTime, err := time.Parse(time.RFC3339, request.EndTime)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid end_time format. Use RFC3339 format"})
		return
	}

	err = h.eventReplayService.ReplayEventsByTimeRange(c.Request.Context(), startTime, endTime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "Events replayed successfully",
		"start_time": startTime,
		"end_time":   endTime,
	})
}

func (h *EventHandler) ReplayAllEvents(c *gin.Context) {
	batchSizeStr := c.DefaultQuery("batch_size", "1000")
	batchSize, err := strconv.Atoi(batchSizeStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid batch_size parameter"})
		return
	}

	err = h.eventReplayService.ReplayAllEvents(c.Request.Context(), batchSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "All events replayed successfully",
		"batch_size": batchSize,
	})
}

func (h *EventHandler) GetReplayStatistics(c *gin.Context) {
	stats, err := h.eventReplayService.GetReplayStatistics(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"total_events":          stats.TotalEvents,
		"total_aggregates":      stats.TotalAggregates,
		"event_type_counts":     stats.EventTypeCounts,
		"aggregate_type_counts": stats.AggregateTypeCounts,
	})
}

func (h *EventHandler) GetEventCount(c *gin.Context) {
	aggregateIDStr := c.Param("aggregate_id")
	aggregateID, err := uuid.Parse(aggregateIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid aggregate ID"})
		return
	}

	count, err := h.eventStore.GetEventCount(c.Request.Context(), aggregateID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"aggregate_id": aggregateID,
		"event_count":  count,
	})
}
