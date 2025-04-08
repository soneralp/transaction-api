package domain

import (
	"encoding/json"
	"errors"
	"sync"
	"time"
)

var (
	ErrInsufficientBalance = errors.New("insufficient balance")
	ErrInvalidAmount       = errors.New("invalid amount")
)

type Balance struct {
	UserID    uint         `json:"user_id"`
	Amount    float64      `json:"amount"`
	Currency  string       `json:"currency"`
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at"`
	mu        sync.RWMutex `json:"-"`
}

func NewBalance(userID uint, initialAmount float64, currency string) (*Balance, error) {
	if initialAmount < 0 {
		return nil, ErrInvalidAmount
	}

	return &Balance{
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
