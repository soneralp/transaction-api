package domain

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

type AggregateRoot interface {
	GetID() uuid.UUID
	GetVersion() int64
	GetUncommittedEvents() []Event
	MarkEventsAsCommitted()
	ApplyEvent(event Event) error
	LoadFromHistory(events []Event) error
}

type BaseAggregate struct {
	ID                uuid.UUID    `json:"id"`
	Version           int64        `json:"version"`
	uncommittedEvents []Event      `json:"-"`
	mu                sync.RWMutex `json:"-"`
}

func (a *BaseAggregate) GetID() uuid.UUID {
	return a.ID
}

func (a *BaseAggregate) GetVersion() int64 {
	return a.Version
}

func (a *BaseAggregate) GetUncommittedEvents() []Event {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.uncommittedEvents
}

func (a *BaseAggregate) MarkEventsAsCommitted() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.uncommittedEvents = nil
	a.Version++
}

func (a *BaseAggregate) AddEvent(event Event) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.uncommittedEvents = append(a.uncommittedEvents, event)
}

type EventStore interface {
	SaveEvents(ctx context.Context, aggregateID uuid.UUID, events []Event, expectedVersion int64) error
	GetEvents(ctx context.Context, aggregateID uuid.UUID) ([]Event, error)
	GetEventsByType(ctx context.Context, eventType EventType, limit, offset int) ([]Event, error)
	GetEventsByTimeRange(ctx context.Context, startTime, endTime time.Time) ([]Event, error)
	GetAllEvents(ctx context.Context, limit, offset int) ([]Event, error)
	GetEventCount(ctx context.Context, aggregateID uuid.UUID) (int64, error)
}

type EventPublisher interface {
	PublishEvent(ctx context.Context, event Event) error
	PublishEvents(ctx context.Context, events []Event) error
}

type EventSourcedTransaction struct {
	BaseAggregate
	UserID       uuid.UUID        `json:"user_id"`
	Type         TransactionType  `json:"type"`
	Amount       float64          `json:"amount"`
	Description  string           `json:"description"`
	ReferenceID  string           `json:"reference_id"`
	BalanceAfter float64          `json:"balance_after"`
	Status       TransactionState `json:"status"`
	CreatedAt    time.Time        `json:"created_at"`
	UpdatedAt    time.Time        `json:"updated_at"`
}

func NewEventSourcedTransaction(userID uuid.UUID, amount float64, description string) (*EventSourcedTransaction, error) {
	if amount <= 0 {
		return nil, ErrInvalidAmount
	}

	transaction := &EventSourcedTransaction{
		BaseAggregate: BaseAggregate{
			ID:      uuid.New(),
			Version: 0,
		},
		UserID:      userID,
		Amount:      amount,
		Type:        TransactionTypeTransfer,
		Status:      TransactionStatePending,
		Description: description,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	event := NewTransactionCreatedEvent(&Transaction{
		ID:          transaction.ID,
		UserID:      transaction.UserID,
		Type:        transaction.Type,
		Amount:      transaction.Amount,
		Description: transaction.Description,
		ReferenceID: transaction.ReferenceID,
		Status:      string(transaction.Status),
		CreatedAt:   transaction.CreatedAt,
		UpdatedAt:   transaction.UpdatedAt,
	})

	transaction.AddEvent(event)
	return transaction, nil
}

func (t *EventSourcedTransaction) UpdateState(newState TransactionState, reason string) error {
	oldState := t.Status

	switch t.Status {
	case TransactionStatePending:
		if newState != TransactionStateCompleted && newState != TransactionStateFailed && newState != TransactionStateCancelled {
			return ErrInvalidState
		}
	case TransactionStateCompleted, TransactionStateFailed, TransactionStateCancelled:
		return ErrInvalidState
	}

	t.Status = newState
	t.UpdatedAt = time.Now()

	event := NewTransactionStateChangedEvent(&Transaction{
		ID:     t.ID,
		UserID: t.UserID,
		Status: string(t.Status),
	}, oldState, newState, reason)

	t.AddEvent(event)
	return nil
}

func (t *EventSourcedTransaction) ApplyEvent(event Event) error {
	switch event.GetType() {
	case EventTransactionCreated:
		var data Transaction
		if err := json.Unmarshal(event.GetData(), &data); err != nil {
			return err
		}
		t.UserID = data.UserID
		t.Type = data.Type
		t.Amount = data.Amount
		t.Description = data.Description
		t.ReferenceID = data.ReferenceID
		t.Status = TransactionState(data.Status)
		t.CreatedAt = data.CreatedAt
		t.UpdatedAt = data.UpdatedAt

	case EventTransactionCompleted, EventTransactionFailed, EventTransactionCancelled:
		var stateEvent TransactionStateChangedEvent
		if err := json.Unmarshal(event.GetData(), &stateEvent); err != nil {
			return err
		}
		t.Status = stateEvent.NewState
		t.UpdatedAt = event.GetTimestamp()

	default:
		return fmt.Errorf("unknown event type: %s", event.GetType())
	}

	return nil
}

func (t *EventSourcedTransaction) LoadFromHistory(events []Event) error {
	for _, event := range events {
		if err := t.ApplyEvent(event); err != nil {
			return err
		}
	}
	return nil
}

type EventSourcedBalance struct {
	BaseAggregate
	UserID    uuid.UUID `json:"user_id"`
	Amount    float64   `json:"amount"`
	Currency  string    `json:"currency"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func NewEventSourcedBalance(userID uuid.UUID, initialAmount float64, currency string) (*EventSourcedBalance, error) {
	if initialAmount < 0 {
		return nil, ErrInvalidAmount
	}

	balance := &EventSourcedBalance{
		BaseAggregate: BaseAggregate{
			ID:      uuid.New(),
			Version: 0,
		},
		UserID:    userID,
		Amount:    initialAmount,
		Currency:  currency,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	event := NewBalanceCreatedEvent(&Balance{
		ID:        balance.ID,
		UserID:    balance.UserID,
		Amount:    balance.Amount,
		Currency:  balance.Currency,
		CreatedAt: balance.CreatedAt,
		UpdatedAt: balance.UpdatedAt,
	})

	balance.AddEvent(event)
	return balance, nil
}

func (b *EventSourcedBalance) Add(amount float64, transactionID uuid.UUID) error {
	if amount <= 0 {
		return ErrInvalidAmount
	}

	oldAmount := b.Amount
	b.Amount += amount
	b.UpdatedAt = time.Now()

	event := NewBalanceUpdatedEvent(&Balance{
		ID:        b.ID,
		UserID:    b.UserID,
		Amount:    b.Amount,
		Currency:  b.Currency,
		UpdatedAt: b.UpdatedAt,
	}, oldAmount, amount, "credit", transactionID)

	b.AddEvent(event)
	return nil
}

func (b *EventSourcedBalance) Subtract(amount float64, transactionID uuid.UUID) error {
	if amount <= 0 {
		return ErrInvalidAmount
	}

	if b.Amount < amount {
		return ErrInsufficientBalance
	}

	oldAmount := b.Amount
	b.Amount -= amount
	b.UpdatedAt = time.Now()

	event := NewBalanceUpdatedEvent(&Balance{
		ID:        b.ID,
		UserID:    b.UserID,
		Amount:    b.Amount,
		Currency:  b.Currency,
		UpdatedAt: b.UpdatedAt,
	}, oldAmount, amount, "debit", transactionID)

	b.AddEvent(event)
	return nil
}

func (b *EventSourcedBalance) ApplyEvent(event Event) error {
	switch event.GetType() {
	case EventBalanceCreated:
		var data Balance
		if err := json.Unmarshal(event.GetData(), &data); err != nil {
			return err
		}
		b.UserID = data.UserID
		b.Amount = data.Amount
		b.Currency = data.Currency
		b.CreatedAt = data.CreatedAt
		b.UpdatedAt = data.UpdatedAt

	case EventBalanceUpdated:
		var updateEvent BalanceUpdatedEvent
		if err := json.Unmarshal(event.GetData(), &updateEvent); err != nil {
			return err
		}
		b.Amount = updateEvent.NewAmount
		b.UpdatedAt = event.GetTimestamp()

	default:
		return fmt.Errorf("unknown event type: %s", event.GetType())
	}

	return nil
}

func (b *EventSourcedBalance) LoadFromHistory(events []Event) error {
	for _, event := range events {
		if err := b.ApplyEvent(event); err != nil {
			return err
		}
	}
	return nil
}
