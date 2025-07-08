package domain

import (
	"context"
	"sync"

	"github.com/google/uuid"
)

type Logger interface {
	Info(msg string, keysAndValues ...interface{})
	Error(msg string, keysAndValues ...interface{})
	Warn(msg string, keysAndValues ...interface{})
	Debug(msg string, keysAndValues ...interface{})
}

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

type ScheduledTransactionService interface {
	CreateScheduledTransaction(ctx context.Context, userID uuid.UUID, req ScheduledTransactionRequest) (*ScheduledTransaction, error)
	GetScheduledTransaction(ctx context.Context, id uuid.UUID) (*ScheduledTransaction, error)
	GetUserScheduledTransactions(ctx context.Context, userID uuid.UUID) ([]*ScheduledTransaction, error)
	UpdateScheduledTransaction(ctx context.Context, id uuid.UUID, req ScheduledTransactionRequest) error
	CancelScheduledTransaction(ctx context.Context, id uuid.UUID) error
	ExecuteScheduledTransactions(ctx context.Context) error
}

type BatchTransactionService interface {
	CreateBatchTransaction(ctx context.Context, userID uuid.UUID, req BatchTransactionRequest) (*BatchTransaction, error)
	GetBatchTransaction(ctx context.Context, id uuid.UUID) (*BatchTransaction, error)
	GetBatchTransactionItems(ctx context.Context, batchID uuid.UUID) ([]*BatchTransactionItem, error)
	ProcessBatchTransaction(ctx context.Context, id uuid.UUID) error
	CancelBatchTransaction(ctx context.Context, id uuid.UUID) error
}

type TransactionLimitService interface {
	CreateTransactionLimit(ctx context.Context, userID uuid.UUID, req TransactionLimitRequest) (*TransactionLimit, error)
	GetTransactionLimit(ctx context.Context, userID uuid.UUID, currency Currency) (*TransactionLimit, error)
	UpdateTransactionLimit(ctx context.Context, userID uuid.UUID, currency Currency, req TransactionLimitRequest) error
	CheckTransactionLimit(ctx context.Context, userID uuid.UUID, currency Currency, amount float64) error
	UpdateTransactionUsage(ctx context.Context, userID uuid.UUID, currency Currency, amount float64) error
	ResetTransactionLimits(ctx context.Context, userID uuid.UUID, currency Currency) error
}

type MultiCurrencyService interface {
	CreateMultiCurrencyBalance(ctx context.Context, userID uuid.UUID, currency Currency, initialAmount float64) (*MultiCurrencyBalance, error)
	GetMultiCurrencyBalance(ctx context.Context, userID uuid.UUID, currency Currency) (*MultiCurrencyBalance, error)
	GetAllBalances(ctx context.Context, userID uuid.UUID) ([]*MultiCurrencyBalance, error)
	ConvertCurrency(ctx context.Context, req CurrencyConversionRequest) (*CurrencyConversionResponse, error)
	TransferBetweenCurrencies(ctx context.Context, userID uuid.UUID, fromCurrency, toCurrency Currency, amount float64) error
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

type ScheduledTransactionRepository interface {
	Create(ctx context.Context, scheduledTransaction *ScheduledTransaction) error
	GetByID(ctx context.Context, id uuid.UUID) (*ScheduledTransaction, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]*ScheduledTransaction, error)
	GetPendingScheduledTransactions(ctx context.Context) ([]*ScheduledTransaction, error)
	Update(ctx context.Context, scheduledTransaction *ScheduledTransaction) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type BatchTransactionRepository interface {
	Create(ctx context.Context, batchTransaction *BatchTransaction) error
	GetByID(ctx context.Context, id uuid.UUID) (*BatchTransaction, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]*BatchTransaction, error)
	Update(ctx context.Context, batchTransaction *BatchTransaction) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type BatchTransactionItemRepository interface {
	Create(ctx context.Context, item *BatchTransactionItem) error
	GetByBatchID(ctx context.Context, batchID uuid.UUID) ([]*BatchTransactionItem, error)
	Update(ctx context.Context, item *BatchTransactionItem) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type TransactionLimitRepository interface {
	Create(ctx context.Context, limit *TransactionLimit) error
	GetByUserIDAndCurrency(ctx context.Context, userID uuid.UUID, currency Currency) (*TransactionLimit, error)
	Update(ctx context.Context, limit *TransactionLimit) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type MultiCurrencyBalanceRepository interface {
	Create(ctx context.Context, balance *MultiCurrencyBalance) error
	GetByUserIDAndCurrency(ctx context.Context, userID uuid.UUID, currency Currency) (*MultiCurrencyBalance, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]*MultiCurrencyBalance, error)
	Update(ctx context.Context, balance *MultiCurrencyBalance) error
	Delete(ctx context.Context, id uuid.UUID) error
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

type ExchangeRateService interface {
	GetExchangeRate(ctx context.Context, fromCurrency, toCurrency Currency) (*ExchangeRate, error)
	UpdateExchangeRate(ctx context.Context, fromCurrency, toCurrency Currency, rate float64) error
	GetSupportedCurrencies(ctx context.Context) ([]Currency, error)
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
