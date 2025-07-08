package service

import (
	"context"
	"fmt"
	"time"

	"transaction-api-w-go/pkg/domain"
	"transaction-api-w-go/pkg/repository"

	"github.com/google/uuid"
)

type EventReplayService struct {
	eventStore domain.EventStore
	eventRepo  *repository.EventRepository
	logger     domain.Logger
}

func NewEventReplayService(eventStore domain.EventStore, eventRepo *repository.EventRepository, logger domain.Logger) *EventReplayService {
	return &EventReplayService{
		eventStore: eventStore,
		eventRepo:  eventRepo,
		logger:     logger,
	}
}

func (s *EventReplayService) ReplayEventsForAggregate(ctx context.Context, aggregateID uuid.UUID) error {
	s.logger.Info("Starting event replay for aggregate", "aggregate_id", aggregateID)

	events, err := s.eventStore.GetEvents(ctx, aggregateID)
	if err != nil {
		return fmt.Errorf("failed to get events for aggregate %s: %w", aggregateID, err)
	}

	if len(events) == 0 {
		s.logger.Info("No events found for aggregate", "aggregate_id", aggregateID)
		return nil
	}

	s.logger.Info("Replaying events", "aggregate_id", aggregateID, "event_count", len(events))

	firstEvent := events[0]
	aggregateType := s.determineAggregateType(firstEvent.GetType())

	switch aggregateType {
	case "transaction":
		return s.replayTransactionEvents(ctx, aggregateID, events)
	case "balance":
		return s.replayBalanceEvents(ctx, aggregateID, events)
	default:
		return fmt.Errorf("unknown aggregate type for event: %s", firstEvent.GetType())
	}
}

func (s *EventReplayService) ReplayEventsByType(ctx context.Context, eventType domain.EventType, limit, offset int) error {
	s.logger.Info("Starting event replay by type", "event_type", eventType)

	events, err := s.eventStore.GetEventsByType(ctx, eventType, limit, offset)
	if err != nil {
		return fmt.Errorf("failed to get events by type %s: %w", eventType, err)
	}

	if len(events) == 0 {
		s.logger.Info("No events found for type", "event_type", eventType)
		return nil
	}

	s.logger.Info("Replaying events by type", "event_type", eventType, "event_count", len(events))

	aggregateGroups := s.groupEventsByAggregate(events)

	for aggregateID := range aggregateGroups {
		if err := s.ReplayEventsForAggregate(ctx, aggregateID); err != nil {
			s.logger.Error("Failed to replay events for aggregate", "aggregate_id", aggregateID, "error", err)
			continue
		}
	}

	return nil
}

func (s *EventReplayService) ReplayEventsByTimeRange(ctx context.Context, startTime, endTime time.Time) error {
	s.logger.Info("Starting event replay by time range", "start_time", startTime, "end_time", endTime)

	events, err := s.eventStore.GetEventsByTimeRange(ctx, startTime, endTime)
	if err != nil {
		return fmt.Errorf("failed to get events by time range: %w", err)
	}

	if len(events) == 0 {
		s.logger.Info("No events found in time range", "start_time", startTime, "end_time", endTime)
		return nil
	}

	s.logger.Info("Replaying events by time range", "event_count", len(events))

	aggregateGroups := s.groupEventsByAggregate(events)

	for aggregateID := range aggregateGroups {
		if err := s.ReplayEventsForAggregate(ctx, aggregateID); err != nil {
			s.logger.Error("Failed to replay events for aggregate", "aggregate_id", aggregateID, "error", err)
			continue
		}
	}

	return nil
}

func (s *EventReplayService) ReplayAllEvents(ctx context.Context, batchSize int) error {
	s.logger.Info("Starting full event replay", "batch_size", batchSize)

	offset := 0
	totalProcessed := 0

	for {
		events, err := s.eventStore.GetAllEvents(ctx, batchSize, offset)
		if err != nil {
			return fmt.Errorf("failed to get events batch: %w", err)
		}

		if len(events) == 0 {
			break
		}

		s.logger.Info("Processing event batch", "batch_size", len(events), "offset", offset)

		aggregateGroups := s.groupEventsByAggregate(events)

		for aggregateID := range aggregateGroups {
			if err := s.ReplayEventsForAggregate(ctx, aggregateID); err != nil {
				s.logger.Error("Failed to replay events for aggregate", "aggregate_id", aggregateID, "error", err)
				continue
			}
		}

		totalProcessed += len(events)
		offset += batchSize

		s.logger.Info("Processed event batch", "total_processed", totalProcessed)
	}

	s.logger.Info("Completed full event replay", "total_events_processed", totalProcessed)
	return nil
}

func (s *EventReplayService) replayTransactionEvents(ctx context.Context, aggregateID uuid.UUID, events []domain.Event) error {
	transaction := &domain.EventSourcedTransaction{}

	if err := transaction.LoadFromHistory(events); err != nil {
		return fmt.Errorf("failed to load transaction from history: %w", err)
	}

	s.logger.Info("Replayed transaction events",
		"transaction_id", aggregateID,
		"user_id", transaction.UserID,
		"status", transaction.Status,
		"amount", transaction.Amount)

	return nil
}

func (s *EventReplayService) replayBalanceEvents(ctx context.Context, aggregateID uuid.UUID, events []domain.Event) error {
	balance := &domain.EventSourcedBalance{}

	if err := balance.LoadFromHistory(events); err != nil {
		return fmt.Errorf("failed to load balance from history: %w", err)
	}

	s.logger.Info("Replayed balance events",
		"balance_id", aggregateID,
		"user_id", balance.UserID,
		"amount", balance.Amount,
		"currency", balance.Currency)

	return nil
}

func (s *EventReplayService) determineAggregateType(eventType domain.EventType) string {
	switch eventType {
	case domain.EventTransactionCreated, domain.EventTransactionCompleted,
		domain.EventTransactionFailed, domain.EventTransactionCancelled:
		return "transaction"
	case domain.EventBalanceCreated, domain.EventBalanceUpdated:
		return "balance"
	case domain.EventUserCreated, domain.EventUserUpdated:
		return "user"
	default:
		return "unknown"
	}
}

func (s *EventReplayService) groupEventsByAggregate(events []domain.Event) map[uuid.UUID][]domain.Event {
	groups := make(map[uuid.UUID][]domain.Event)

	for _, event := range events {
		aggregateID := event.GetAggregateID()
		groups[aggregateID] = append(groups[aggregateID], event)
	}

	return groups
}

func (s *EventReplayService) GetReplayStatistics(ctx context.Context) (*ReplayStatistics, error) {
	stats := &ReplayStatistics{}

	allEvents, err := s.eventStore.GetAllEvents(ctx, 1000000, 0) // Büyük limit
	if err != nil {
		return nil, fmt.Errorf("failed to get all events for statistics: %w", err)
	}
	stats.TotalEvents = int64(len(allEvents))

	eventTypeCounts := make(map[domain.EventType]int64)
	for _, event := range allEvents {
		eventTypeCounts[event.GetType()]++
	}
	stats.EventTypeCounts = eventTypeCounts

	aggregateGroups := s.groupEventsByAggregate(allEvents)
	stats.TotalAggregates = int64(len(aggregateGroups))

	aggregateTypeCounts := make(map[string]int64)
	for _, events := range aggregateGroups {
		if len(events) > 0 {
			aggregateType := s.determineAggregateType(events[0].GetType())
			aggregateTypeCounts[aggregateType]++
		}
	}
	stats.AggregateTypeCounts = aggregateTypeCounts

	return stats, nil
}

type ReplayStatistics struct {
	TotalEvents         int64                      `json:"total_events"`
	TotalAggregates     int64                      `json:"total_aggregates"`
	EventTypeCounts     map[domain.EventType]int64 `json:"event_type_counts"`
	AggregateTypeCounts map[string]int64           `json:"aggregate_type_counts"`
}
