package domain

import (
	"encoding/json"
	"errors"
	"sync"
	"time"
)

var (
	ErrTransactionNotFound = errors.New("transaction not found")
	ErrInvalidOperation    = errors.New("invalid operation")
	ErrInsufficientFunds   = errors.New("insufficient funds")
	ErrInvalidState        = errors.New("invalid transaction state")
	ErrTransactionFailed   = errors.New("transaction failed")
)

type TransactionState string

const (
	TransactionStatePending   TransactionState = "pending"
	TransactionStateCompleted TransactionState = "completed"
	TransactionStateFailed    TransactionState = "failed"
	TransactionStateCancelled TransactionState = "cancelled"
)

type Transaction struct {
	ID          uint             `json:"id"`
	FromUserID  uint             `json:"from_user_id"`
	ToUserID    uint             `json:"to_user_id"`
	Amount      float64          `json:"amount"`
	State       TransactionState `json:"state"`
	Description string           `json:"description"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
	mu          sync.Mutex       `json:"-"`
}

func NewTransaction(fromUserID, toUserID uint, amount float64, description string) (*Transaction, error) {
	if amount <= 0 {
		return nil, ErrInvalidAmount
	}

	return &Transaction{
		FromUserID:  fromUserID,
		ToUserID:    toUserID,
		Amount:      amount,
		State:       TransactionStatePending,
		Description: description,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}, nil
}

func (t *Transaction) UpdateState(newState TransactionState) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	switch t.State {
	case TransactionStatePending:
		if newState != TransactionStateCompleted && newState != TransactionStateFailed && newState != TransactionStateCancelled {
			return ErrInvalidState
		}
	case TransactionStateCompleted:
		return ErrInvalidState
	case TransactionStateFailed:
		return ErrInvalidState
	case TransactionStateCancelled:
		return ErrInvalidState
	}

	t.State = newState
	t.UpdatedAt = time.Now()
	return nil
}

func (t *Transaction) MarshalJSON() ([]byte, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	type Alias Transaction
	return json.Marshal(&struct {
		*Alias
	}{
		Alias: (*Alias)(t),
	})
}
