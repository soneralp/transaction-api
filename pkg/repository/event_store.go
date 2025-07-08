package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"transaction-api-w-go/pkg/domain"
)

type EventStoreModel struct {
	ID          uuid.UUID        `json:"id" gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	Type        domain.EventType `json:"type" gorm:"type:varchar(100);not null;index"`
	AggregateID uuid.UUID        `json:"aggregate_id" gorm:"type:uuid;not null;index"`
	Version     int64            `json:"version" gorm:"not null"`
	Timestamp   time.Time        `json:"timestamp" gorm:"not null;index"`
	Data        json.RawMessage  `json:"data" gorm:"type:jsonb;not null"`
	Metadata    json.RawMessage  `json:"metadata" gorm:"type:jsonb"`
	CreatedAt   time.Time        `json:"created_at" gorm:"not null;default:CURRENT_TIMESTAMP"`
}

func (EventStoreModel) TableName() string {
	return "event_store"
}

type PostgresEventStore struct {
	db *gorm.DB
}

func NewPostgresEventStore(db *gorm.DB) domain.EventStore {
	return &PostgresEventStore{db: db}
}

func (es *PostgresEventStore) SaveEvents(ctx context.Context, aggregateID uuid.UUID, events []domain.Event, expectedVersion int64) error {
	return es.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Optimistic concurrency control
		var currentVersion int64
		err := tx.Model(&EventStoreModel{}).
			Where("aggregate_id = ?", aggregateID).
			Select("COALESCE(MAX(version), 0)").
			Scan(&currentVersion).Error

		if err != nil {
			return fmt.Errorf("failed to get current version: %w", err)
		}

		if currentVersion != expectedVersion {
			return fmt.Errorf("concurrent modification detected: expected version %d, got %d", expectedVersion, currentVersion)
		}

		for i, event := range events {
			eventModel := EventStoreModel{
				ID:          event.GetID(),
				Type:        event.GetType(),
				AggregateID: event.GetAggregateID(),
				Version:     expectedVersion + int64(i) + 1,
				Timestamp:   event.GetTimestamp(),
				Data:        event.GetData(),
				CreatedAt:   time.Now(),
			}

			if event.GetMetadata() != nil {
				metadata, err := json.Marshal(event.GetMetadata())
				if err != nil {
					return fmt.Errorf("failed to marshal metadata: %w", err)
				}
				eventModel.Metadata = metadata
			}

			if err := tx.Create(&eventModel).Error; err != nil {
				return fmt.Errorf("failed to save event: %w", err)
			}
		}

		return nil
	})
}

func (es *PostgresEventStore) GetEvents(ctx context.Context, aggregateID uuid.UUID) ([]domain.Event, error) {
	var eventModels []EventStoreModel

	err := es.db.WithContext(ctx).
		Where("aggregate_id = ?", aggregateID).
		Order("version ASC").
		Find(&eventModels).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get events: %w", err)
	}

	events := make([]domain.Event, len(eventModels))
	for i, model := range eventModels {
		event, err := es.deserializeEvent(model)
		if err != nil {
			return nil, fmt.Errorf("failed to deserialize event: %w", err)
		}
		events[i] = event
	}

	return events, nil
}

func (es *PostgresEventStore) GetEventsByType(ctx context.Context, eventType domain.EventType, limit, offset int) ([]domain.Event, error) {
	var eventModels []EventStoreModel

	err := es.db.WithContext(ctx).
		Where("type = ?", eventType).
		Order("timestamp ASC").
		Limit(limit).
		Offset(offset).
		Find(&eventModels).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get events by type: %w", err)
	}

	events := make([]domain.Event, len(eventModels))
	for i, model := range eventModels {
		event, err := es.deserializeEvent(model)
		if err != nil {
			return nil, fmt.Errorf("failed to deserialize event: %w", err)
		}
		events[i] = event
	}

	return events, nil
}

func (es *PostgresEventStore) GetEventsByTimeRange(ctx context.Context, startTime, endTime time.Time) ([]domain.Event, error) {
	var eventModels []EventStoreModel

	err := es.db.WithContext(ctx).
		Where("timestamp BETWEEN ? AND ?", startTime, endTime).
		Order("timestamp ASC").
		Find(&eventModels).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get events by time range: %w", err)
	}

	events := make([]domain.Event, len(eventModels))
	for i, model := range eventModels {
		event, err := es.deserializeEvent(model)
		if err != nil {
			return nil, fmt.Errorf("failed to deserialize event: %w", err)
		}
		events[i] = event
	}

	return events, nil
}

func (es *PostgresEventStore) GetAllEvents(ctx context.Context, limit, offset int) ([]domain.Event, error) {
	var eventModels []EventStoreModel

	err := es.db.WithContext(ctx).
		Order("timestamp ASC").
		Limit(limit).
		Offset(offset).
		Find(&eventModels).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get all events: %w", err)
	}

	events := make([]domain.Event, len(eventModels))
	for i, model := range eventModels {
		event, err := es.deserializeEvent(model)
		if err != nil {
			return nil, fmt.Errorf("failed to deserialize event: %w", err)
		}
		events[i] = event
	}

	return events, nil
}

func (es *PostgresEventStore) GetEventCount(ctx context.Context, aggregateID uuid.UUID) (int64, error) {
	var count int64

	err := es.db.WithContext(ctx).
		Model(&EventStoreModel{}).
		Where("aggregate_id = ?", aggregateID).
		Count(&count).Error

	if err != nil {
		return 0, fmt.Errorf("failed to get event count: %w", err)
	}

	return count, nil
}

func (es *PostgresEventStore) deserializeEvent(model EventStoreModel) (domain.Event, error) {
	baseEvent := domain.BaseEvent{
		ID:          model.ID,
		Type:        model.Type,
		AggregateID: model.AggregateID,
		Version:     model.Version,
		Timestamp:   model.Timestamp,
		Data:        model.Data,
	}

	if model.Metadata != nil {
		var metadata map[string]interface{}
		if err := json.Unmarshal(model.Metadata, &metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
		baseEvent.Metadata = metadata
	}

	switch model.Type {
	case domain.EventTransactionCreated:
		var event domain.TransactionCreatedEvent
		if err := json.Unmarshal(model.Data, &event); err != nil {
			return nil, fmt.Errorf("failed to unmarshal transaction created event: %w", err)
		}
		event.BaseEvent = baseEvent
		return &event, nil

	case domain.EventTransactionCompleted, domain.EventTransactionFailed, domain.EventTransactionCancelled:
		var event domain.TransactionStateChangedEvent
		if err := json.Unmarshal(model.Data, &event); err != nil {
			return nil, fmt.Errorf("failed to unmarshal transaction state changed event: %w", err)
		}
		event.BaseEvent = baseEvent
		return &event, nil

	case domain.EventBalanceCreated:
		var event domain.BalanceCreatedEvent
		if err := json.Unmarshal(model.Data, &event); err != nil {
			return nil, fmt.Errorf("failed to unmarshal balance created event: %w", err)
		}
		event.BaseEvent = baseEvent
		return &event, nil

	case domain.EventBalanceUpdated:
		var event domain.BalanceUpdatedEvent
		if err := json.Unmarshal(model.Data, &event); err != nil {
			return nil, fmt.Errorf("failed to unmarshal balance updated event: %w", err)
		}
		event.BaseEvent = baseEvent
		return &event, nil

	default:
		return &baseEvent, nil
	}
}

type EventRepository struct {
	eventStore domain.EventStore
}

func NewEventRepository(eventStore domain.EventStore) *EventRepository {
	return &EventRepository{eventStore: eventStore}
}

func (r *EventRepository) Save(ctx context.Context, aggregate domain.AggregateRoot) error {
	events := aggregate.GetUncommittedEvents()
	if len(events) == 0 {
		return nil
	}

	expectedVersion := aggregate.GetVersion()
	err := r.eventStore.SaveEvents(ctx, aggregate.GetID(), events, expectedVersion)
	if err != nil {
		return err
	}

	aggregate.MarkEventsAsCommitted()
	return nil
}

func (r *EventRepository) GetTransaction(ctx context.Context, id uuid.UUID) (*domain.EventSourcedTransaction, error) {
	events, err := r.eventStore.GetEvents(ctx, id)
	if err != nil {
		return nil, err
	}

	if len(events) == 0 {
		return nil, sql.ErrNoRows
	}

	transaction := &domain.EventSourcedTransaction{}
	if err := transaction.LoadFromHistory(events); err != nil {
		return nil, err
	}

	return transaction, nil
}

func (r *EventRepository) GetBalance(ctx context.Context, id uuid.UUID) (*domain.EventSourcedBalance, error) {
	events, err := r.eventStore.GetEvents(ctx, id)
	if err != nil {
		return nil, err
	}

	if len(events) == 0 {
		return nil, sql.ErrNoRows
	}

	balance := &domain.EventSourcedBalance{}
	if err := balance.LoadFromHistory(events); err != nil {
		return nil, err
	}

	return balance, nil
}

func (r *EventRepository) GetBalanceByUserID(ctx context.Context, userID uuid.UUID) (*domain.EventSourcedBalance, error) {
	balanceID := userID

	return r.GetBalance(ctx, balanceID)
}

func (r *EventRepository) GetEvents(ctx context.Context, aggregateID uuid.UUID) ([]domain.Event, error) {
	return r.eventStore.GetEvents(ctx, aggregateID)
}
