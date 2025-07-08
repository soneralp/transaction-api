package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type EventType string

const (
	EventTransactionCreated    EventType = "transaction.created"
	EventTransactionCompleted  EventType = "transaction.completed"
	EventTransactionFailed     EventType = "transaction.failed"
	EventTransactionCancelled  EventType = "transaction.cancelled"
	EventTransactionRolledBack EventType = "transaction.rolled_back"

	EventBalanceCreated  EventType = "balance.created"
	EventBalanceUpdated  EventType = "balance.updated"
	EventBalanceDebited  EventType = "balance.debited"
	EventBalanceCredited EventType = "balance.credited"

	EventUserCreated EventType = "user.created"
	EventUserUpdated EventType = "user.updated"
)

type BaseEvent struct {
	ID          uuid.UUID              `json:"id"`
	Type        EventType              `json:"type"`
	AggregateID uuid.UUID              `json:"aggregate_id"`
	Version     int64                  `json:"version"`
	Timestamp   time.Time              `json:"timestamp"`
	Data        json.RawMessage        `json:"data"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

type Event interface {
	GetID() uuid.UUID
	GetType() EventType
	GetAggregateID() uuid.UUID
	GetVersion() int64
	GetTimestamp() time.Time
	GetData() json.RawMessage
	GetMetadata() map[string]interface{}
}

func (e *BaseEvent) GetID() uuid.UUID                    { return e.ID }
func (e *BaseEvent) GetType() EventType                  { return e.Type }
func (e *BaseEvent) GetAggregateID() uuid.UUID           { return e.AggregateID }
func (e *BaseEvent) GetVersion() int64                   { return e.Version }
func (e *BaseEvent) GetTimestamp() time.Time             { return e.Timestamp }
func (e *BaseEvent) GetData() json.RawMessage            { return e.Data }
func (e *BaseEvent) GetMetadata() map[string]interface{} { return e.Metadata }

type TransactionCreatedEvent struct {
	BaseEvent
	TransactionID uuid.UUID       `json:"transaction_id"`
	UserID        uuid.UUID       `json:"user_id"`
	Type          TransactionType `json:"type"`
	Amount        float64         `json:"amount"`
	Description   string          `json:"description"`
	ReferenceID   string          `json:"reference_id"`
}

type TransactionStateChangedEvent struct {
	BaseEvent
	TransactionID uuid.UUID        `json:"transaction_id"`
	UserID        uuid.UUID        `json:"user_id"`
	OldState      TransactionState `json:"old_state"`
	NewState      TransactionState `json:"new_state"`
	Reason        string           `json:"reason,omitempty"`
}

type BalanceCreatedEvent struct {
	BaseEvent
	UserID   uuid.UUID `json:"user_id"`
	Amount   float64   `json:"amount"`
	Currency string    `json:"currency"`
}

type BalanceUpdatedEvent struct {
	BaseEvent
	UserID        uuid.UUID `json:"user_id"`
	OldAmount     float64   `json:"old_amount"`
	NewAmount     float64   `json:"new_amount"`
	Change        float64   `json:"change"`
	Operation     string    `json:"operation"` // "credit", "debit"
	TransactionID uuid.UUID `json:"transaction_id,omitempty"`
}

type UserCreatedEvent struct {
	BaseEvent
	UserID    uuid.UUID `json:"user_id"`
	Email     string    `json:"email"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
}

type UserUpdatedEvent struct {
	BaseEvent
	UserID    uuid.UUID `json:"user_id"`
	Email     string    `json:"email,omitempty"`
	FirstName string    `json:"first_name,omitempty"`
	LastName  string    `json:"last_name,omitempty"`
}

func NewTransactionCreatedEvent(transaction *Transaction) *TransactionCreatedEvent {
	data, _ := json.Marshal(transaction)

	return &TransactionCreatedEvent{
		BaseEvent: BaseEvent{
			ID:          uuid.New(),
			Type:        EventTransactionCreated,
			AggregateID: transaction.ID,
			Version:     1,
			Timestamp:   time.Now(),
			Data:        data,
		},
		TransactionID: transaction.ID,
		UserID:        transaction.UserID,
		Type:          transaction.Type,
		Amount:        transaction.Amount,
		Description:   transaction.Description,
		ReferenceID:   transaction.ReferenceID,
	}
}

func NewTransactionStateChangedEvent(transaction *Transaction, oldState, newState TransactionState, reason string) *TransactionStateChangedEvent {
	return &TransactionStateChangedEvent{
		BaseEvent: BaseEvent{
			ID:          uuid.New(),
			Type:        EventTransactionStateChangedEventType(newState),
			AggregateID: transaction.ID,
			Version:     1,
			Timestamp:   time.Now(),
		},
		TransactionID: transaction.ID,
		UserID:        transaction.UserID,
		OldState:      oldState,
		NewState:      newState,
		Reason:        reason,
	}
}

func NewBalanceCreatedEvent(balance *Balance) *BalanceCreatedEvent {
	data, _ := json.Marshal(balance)

	return &BalanceCreatedEvent{
		BaseEvent: BaseEvent{
			ID:          uuid.New(),
			Type:        EventBalanceCreated,
			AggregateID: balance.ID,
			Version:     1,
			Timestamp:   time.Now(),
			Data:        data,
		},
		UserID:   balance.UserID,
		Amount:   balance.Amount,
		Currency: balance.Currency,
	}
}

func NewBalanceUpdatedEvent(balance *Balance, oldAmount, change float64, operation string, transactionID uuid.UUID) *BalanceUpdatedEvent {
	return &BalanceUpdatedEvent{
		BaseEvent: BaseEvent{
			ID:          uuid.New(),
			Type:        EventBalanceUpdated,
			AggregateID: balance.ID,
			Version:     1,
			Timestamp:   time.Now(),
		},
		UserID:        balance.UserID,
		OldAmount:     oldAmount,
		NewAmount:     balance.Amount,
		Change:        change,
		Operation:     operation,
		TransactionID: transactionID,
	}
}

func EventTransactionStateChangedEventType(state TransactionState) EventType {
	switch state {
	case TransactionStateCompleted:
		return EventTransactionCompleted
	case TransactionStateFailed:
		return EventTransactionFailed
	case TransactionStateCancelled:
		return EventTransactionCancelled
	default:
		return EventTransactionStateChangedEventType(TransactionStatePending)
	}
}
