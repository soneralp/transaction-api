package domain

import (
	"context"
	"sync"
)

type UserRepository interface {
	Create(ctx context.Context, user *User) error
	GetByID(ctx context.Context, id uint) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	Update(ctx context.Context, user *User) error
	Delete(ctx context.Context, id uint) error
}

type TransactionRepository interface {
	Create(ctx context.Context, transaction *Transaction) error
	GetByID(ctx context.Context, id uint) (*Transaction, error)
	GetByUserID(ctx context.Context, userID uint) ([]*Transaction, error)
	Update(ctx context.Context, transaction *Transaction) error
}

type BalanceRepository interface {
	Create(ctx context.Context, balance *Balance) error
	GetByUserID(ctx context.Context, userID uint) (*Balance, error)
	Update(ctx context.Context, balance *Balance) error
}

type UserService interface {
	Register(ctx context.Context, username, email, password string) (*User, error)
	GetByID(ctx context.Context, id uint) (*User, error)
	Update(ctx context.Context, user *User) error
	Delete(ctx context.Context, id uint) error
}

type TransactionStats struct {
	TotalProcessed     uint64
	TotalFailed        uint64
	TotalAmount        float64
	AverageProcessTime float64
	mu                 sync.RWMutex
}

func (s *TransactionStats) UpdateStats(amount float64, processTime float64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.TotalProcessed++
	s.TotalAmount += amount
	s.AverageProcessTime = (s.AverageProcessTime*float64(s.TotalProcessed-1) + processTime) / float64(s.TotalProcessed)
}

type TransactionService interface {
	CreateTransaction(ctx context.Context, fromUserID, toUserID uint, amount float64, description string) (*Transaction, error)
	GetTransaction(ctx context.Context, id uint) (*Transaction, error)
	GetUserTransactions(ctx context.Context, userID uint) ([]*Transaction, error)
	ProcessTransaction(ctx context.Context, transactionID uint) error
	CancelTransaction(ctx context.Context, transactionID uint) error
	GetStats() *TransactionStats
}

type BalanceService interface {
	GetBalance(ctx context.Context, userID uint) (*Balance, error)
	AddFunds(ctx context.Context, userID uint, amount float64) error
	WithdrawFunds(ctx context.Context, userID uint, amount float64) error
	TransferFunds(ctx context.Context, fromUserID, toUserID uint, amount float64) error
}
