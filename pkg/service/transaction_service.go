package service

import (
	"context"
	"sync"
	"transaction-api-w-go/pkg/domain"
)

type transactionService struct {
	transactionRepo domain.TransactionRepository
	balanceService  domain.BalanceService
	stats           *domain.TransactionStats
	mu              sync.RWMutex
}

func NewTransactionService(
	transactionRepo domain.TransactionRepository,
	balanceService domain.BalanceService,
) domain.TransactionService {
	return &transactionService{
		transactionRepo: transactionRepo,
		balanceService:  balanceService,
		stats:           &domain.TransactionStats{},
	}
}

func (s *transactionService) CreateTransaction(
	ctx context.Context,
	fromUserID, toUserID uint,
	amount float64,
	description string,
) (*domain.Transaction, error) {
	transaction, err := domain.NewTransaction(fromUserID, toUserID, amount, description)
	if err != nil {
		return nil, err
	}

	err = s.transactionRepo.Create(ctx, transaction)
	if err != nil {
		return nil, err
	}

	return transaction, nil
}

func (s *transactionService) GetTransaction(ctx context.Context, id uint) (*domain.Transaction, error) {
	return s.transactionRepo.GetByID(ctx, id)
}

func (s *transactionService) GetUserTransactions(ctx context.Context, userID uint) ([]*domain.Transaction, error) {
	return s.transactionRepo.GetByUserID(ctx, userID)
}

func (s *transactionService) ProcessTransaction(ctx context.Context, transactionID uint) error {
	transaction, err := s.transactionRepo.GetByID(ctx, transactionID)
	if err != nil {
		return err
	}

	if transaction.State != domain.TransactionStatePending {
		return domain.ErrInvalidState
	}

	err = s.balanceService.TransferFunds(ctx, transaction.FromUserID, transaction.ToUserID, transaction.Amount)
	if err != nil {
		transaction.State = domain.TransactionStateFailed
		s.transactionRepo.Update(ctx, transaction)
		return err
	}

	transaction.State = domain.TransactionStateCompleted
	return s.transactionRepo.Update(ctx, transaction)
}

func (s *transactionService) CancelTransaction(ctx context.Context, transactionID uint) error {
	transaction, err := s.transactionRepo.GetByID(ctx, transactionID)
	if err != nil {
		return err
	}

	if transaction.State != domain.TransactionStatePending {
		return domain.ErrInvalidState
	}

	transaction.State = domain.TransactionStateCancelled
	return s.transactionRepo.Update(ctx, transaction)
}

func (s *transactionService) GetStats() *domain.TransactionStats {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.stats
}
