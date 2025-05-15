package domain

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Balance struct {
	ID        uuid.UUID    `json:"id" gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	UserID    uuid.UUID    `json:"user_id" gorm:"type:uuid;not null;uniqueIndex"`
	Amount    float64      `json:"amount" gorm:"type:decimal(19,4);not null"`
	Currency  string       `json:"currency"`
	CreatedAt time.Time    `json:"created_at" gorm:"not null"`
	UpdatedAt time.Time    `json:"updated_at" gorm:"not null"`
	mu        sync.RWMutex `json:"-"`
}

type BalanceHistory struct {
	ID        uuid.UUID `json:"id" gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	UserID    uuid.UUID `json:"user_id" gorm:"type:uuid;not null"`
	Amount    float64   `json:"amount" gorm:"type:decimal(19,4);not null"`
	Timestamp time.Time `json:"timestamp" gorm:"not null"`
	CreatedAt time.Time `json:"created_at" gorm:"not null"`
}

func NewBalance(userID uuid.UUID, initialAmount float64, currency string) (*Balance, error) {
	if initialAmount < 0 {
		return nil, ErrInvalidAmount
	}

	return &Balance{
		ID:        uuid.New(),
		UserID:    userID,
		Amount:    initialAmount,
		Currency:  currency,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}, nil
}

func (b *Balance) Add(amount float64) error {
	if amount <= 0 {
		return ErrInvalidAmount
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	b.Amount += amount
	b.UpdatedAt = time.Now()
	return nil
}

func (b *Balance) Subtract(amount float64) error {
	if amount <= 0 {
		return ErrInvalidAmount
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	if b.Amount < amount {
		return ErrInsufficientBalance
	}

	b.Amount -= amount
	b.UpdatedAt = time.Now()
	return nil
}

func (b *Balance) GetAmount() float64 {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.Amount
}

func (b *Balance) MarshalJSON() ([]byte, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	type Alias Balance
	return json.Marshal(&struct {
		*Alias
	}{
		Alias: (*Alias)(b),
	})
}
