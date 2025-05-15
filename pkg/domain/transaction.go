package domain

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/google/uuid"
)

type TransactionState string

const (
	TransactionStatePending   TransactionState = "pending"
	TransactionStateCompleted TransactionState = "completed"
	TransactionStateFailed    TransactionState = "failed"
	TransactionStateCancelled TransactionState = "cancelled"
)

type TransactionType string

const (
	TransactionTypeCredit   TransactionType = "CREDIT"
	TransactionTypeDebit    TransactionType = "DEBIT"
	TransactionTypeTransfer TransactionType = "TRANSFER"
)

type Transaction struct {
	ID           uuid.UUID       `json:"id" gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	UserID       uuid.UUID       `json:"user_id" gorm:"type:uuid;not null"`
	Type         TransactionType `json:"type" gorm:"type:varchar(20);not null"`
	Amount       float64         `json:"amount" gorm:"type:decimal(19,4);not null"`
	Description  string          `json:"description" gorm:"type:text"`
	ReferenceID  string          `json:"reference_id" gorm:"type:varchar(100)"`
	BalanceAfter float64         `json:"balance_after" gorm:"type:decimal(19,4);not null"`
	Status       string          `json:"status" gorm:"type:varchar(20);not null;default:'pending'"`
	CreatedAt    time.Time       `json:"created_at" gorm:"not null"`
	UpdatedAt    time.Time       `json:"updated_at" gorm:"not null"`
	mu           sync.Mutex      `json:"-"`
}

type TransactionRequest struct {
	Amount      float64 `json:"amount" binding:"required,gt=0"`
	Description string  `json:"description"`
}

type TransferRequest struct {
	Amount      float64   `json:"amount" binding:"required,gt=0"`
	ToUserID    uuid.UUID `json:"to_user_id" binding:"required"`
	Description string    `json:"description"`
}

func NewTransaction(userID uuid.UUID, amount float64, description string) (*Transaction, error) {
	if amount <= 0 {
		return nil, ErrInvalidAmount
	}

	return &Transaction{
		ID:          uuid.New(),
		UserID:      userID,
		Amount:      amount,
		Type:        TransactionTypeTransfer,
		Status:      string(TransactionStatePending),
		Description: description,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}, nil
}

func (t *Transaction) UpdateState(newState TransactionState) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	switch t.Status {
	case "pending":
		if newState != TransactionStateCompleted && newState != TransactionStateFailed && newState != TransactionStateCancelled {
			return ErrInvalidState
		}
	case "completed":
		return ErrInvalidState
	case "failed":
		return ErrInvalidState
	case "rolled_back":
		return ErrInvalidState
	}

	t.Status = string(newState)
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
