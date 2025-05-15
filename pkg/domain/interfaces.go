package domain

import (
	"context"
	"sync"
)

type UserService interface {
	Register(ctx context.Context, user *User) error
	Authenticate(ctx context.Context, email, password string) (*User, error)
	GetByID(ctx context.Context, id uint) (*User, error)
	Update(ctx context.Context, user *User) error
	Delete(ctx context.Context, id uint) error
	HasPermission(ctx context.Context, userID uint, permission string) bool
}

type TransactionService interface {
	CreateTransaction(ctx context.Context, transaction *Transaction) error
	ProcessTransaction(ctx context.Context, transactionID uint) error
	RollbackTransaction(ctx context.Context, transactionID uint) error
	GetTransaction(ctx context.Context, transactionID uint) (*Transaction, error)
	GetUserTransactions(ctx context.Context, userID uint) ([]*Transaction, error)
	GetStats() *TransactionStats
}

type BalanceService interface {
	AddFunds(ctx context.Context, userID uint, amount float64) error
	WithdrawFunds(ctx context.Context, userID uint, amount float64) error
	GetBalance(ctx context.Context, userID uint) (*Balance, error)
	TransferFunds(ctx context.Context, fromUserID uint, toUserID uint, amount float64) error
	GetBalanceHistory(ctx context.Context, userID uint) ([]*BalanceHistory, error)
	CalculateTotalBalance(ctx context.Context, userID uint) (float64, error)
}

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
	Delete(ctx context.Context, id uint) error
}

type BalanceRepository interface {
	Create(ctx context.Context, balance *Balance) error
	GetByID(ctx context.Context, id uint) (*Balance, error)
	GetByUserID(ctx context.Context, userID uint) (*Balance, error)
	Update(ctx context.Context, balance *Balance) error
	Delete(ctx context.Context, id uint) error
	CreateHistory(ctx context.Context, history *BalanceHistory) error
	GetHistoryByUserID(ctx context.Context, userID uint) ([]*BalanceHistory, error)
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
