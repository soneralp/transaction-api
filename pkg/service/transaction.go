package service

import (
	"context"
	"errors"
	"time"

	"transaction-api-w-go/pkg/domain"
	"transaction-api-w-go/pkg/metrics"
	"transaction-api-w-go/pkg/repository"

	"github.com/google/uuid"
)

type TransactionService struct {
	transactionRepo *repository.TransactionRepository
	balanceRepo     *repository.BalanceRepository
	userRepo        *repository.UserRepository
	stats           *domain.TransactionStats
}

func NewTransactionService(
	transactionRepo *repository.TransactionRepository,
	balanceRepo *repository.BalanceRepository,
	userRepo *repository.UserRepository,
) *TransactionService {
	return &TransactionService{
		transactionRepo: transactionRepo,
		balanceRepo:     balanceRepo,
		userRepo:        userRepo,
		stats:           &domain.TransactionStats{},
	}
}

func (s *TransactionService) Credit(ctx context.Context, userID string, amount float64, description string) (*domain.Transaction, error) {
	balance, err := s.balanceRepo.GetByUserID(userID)
	if err != nil {
		balance = &domain.Balance{
			ID:        uuid.New(),
			UserID:    uuid.MustParse(userID),
			Amount:    0,
			Currency:  "TRY",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		if err := s.balanceRepo.Create(balance); err != nil {
			return nil, err
		}
	}

	transaction := &domain.Transaction{
		ID:           uuid.New(),
		UserID:       uuid.MustParse(userID),
		Type:         domain.TransactionTypeCredit,
		Amount:       amount,
		Description:  description,
		BalanceAfter: balance.Amount + amount,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := s.transactionRepo.Create(ctx, transaction); err != nil {
		return nil, err
	}

	balance.Amount += amount
	if err := s.balanceRepo.Update(balance); err != nil {
		return nil, err
	}

	return transaction, nil
}

func (s *TransactionService) Debit(ctx context.Context, userID string, amount float64, description string) (*domain.Transaction, error) {
	balance, err := s.balanceRepo.GetByUserID(userID)
	if err != nil {
		return nil, err
	}

	if balance.Amount < amount {
		return nil, errors.New("insufficient balance")
	}

	transaction := &domain.Transaction{
		ID:           uuid.New(),
		UserID:       uuid.MustParse(userID),
		Type:         domain.TransactionTypeDebit,
		Amount:       amount,
		Description:  description,
		BalanceAfter: balance.Amount - amount,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := s.transactionRepo.Create(ctx, transaction); err != nil {
		return nil, err
	}

	balance.Amount -= amount
	if err := s.balanceRepo.Update(balance); err != nil {
		return nil, err
	}

	return transaction, nil
}

func (s *TransactionService) Transfer(ctx context.Context, fromUserID, toUserID string, amount float64, description string) (*domain.Transaction, error) {
	fromBalance, err := s.balanceRepo.GetByUserID(fromUserID)
	if err != nil {
		return nil, err
	}

	if fromBalance.Amount < amount {
		return nil, errors.New("insufficient balance")
	}

	toBalance, err := s.balanceRepo.GetByUserID(toUserID)
	if err != nil {
		return nil, err
	}

	transaction := &domain.Transaction{
		ID:           uuid.New(),
		UserID:       uuid.MustParse(fromUserID),
		Type:         domain.TransactionTypeTransfer,
		Amount:       amount,
		Description:  description,
		BalanceAfter: fromBalance.Amount - amount,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := s.transactionRepo.Create(ctx, transaction); err != nil {
		return nil, err
	}

	fromBalance.Amount -= amount
	if err := s.balanceRepo.Update(fromBalance); err != nil {
		return nil, err
	}

	toBalance.Amount += amount
	if err := s.balanceRepo.Update(toBalance); err != nil {
		return nil, err
	}

	return transaction, nil
}

func (s *TransactionService) GetHistory(ctx context.Context, userID uint) ([]*domain.Transaction, error) {
	return s.transactionRepo.GetByUserID(ctx, userID)
}

func (s *TransactionService) GetByID(ctx context.Context, transactionID uint) (*domain.Transaction, error) {
	return s.transactionRepo.GetByID(ctx, transactionID)
}

func (s *TransactionService) ProcessTransaction(ctx context.Context, transactionID uint) error {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		metrics.DatabaseQueryDuration.WithLabelValues("process_transaction").Observe(duration)
	}()

	transaction, err := s.transactionRepo.GetByID(ctx, transactionID)
	if err != nil {
		metrics.TransactionTotal.WithLabelValues("process", "failed").Inc()
		return err
	}

	metrics.TransactionTotal.WithLabelValues("process", "success").Inc()
	metrics.TransactionAmount.WithLabelValues("process").Observe(transaction.Amount)
	return nil
}

func (s *TransactionService) CreateTransaction(ctx context.Context, transaction *domain.Transaction) error {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		metrics.DatabaseQueryDuration.WithLabelValues("create_transaction").Observe(duration)
	}()

	if err := s.transactionRepo.Create(ctx, transaction); err != nil {
		return err
	}

	metrics.TransactionTotal.WithLabelValues("create", "success").Inc()
	metrics.TransactionAmount.WithLabelValues("create").Observe(transaction.Amount)
	return nil
}
