package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"transaction-api-w-go/pkg/domain"

	"github.com/google/uuid"
)

type ScheduledTransactionServiceImpl struct {
	scheduledRepo   domain.ScheduledTransactionRepository
	transactionRepo domain.TransactionRepository
	balanceRepo     domain.BalanceRepository
	logger          domain.Logger
	mu              sync.RWMutex
}

func NewScheduledTransactionService(
	scheduledRepo domain.ScheduledTransactionRepository,
	transactionRepo domain.TransactionRepository,
	balanceRepo domain.BalanceRepository,
	logger domain.Logger,
) domain.ScheduledTransactionService {
	return &ScheduledTransactionServiceImpl{
		scheduledRepo:   scheduledRepo,
		transactionRepo: transactionRepo,
		balanceRepo:     balanceRepo,
		logger:          logger,
	}
}

func (s *ScheduledTransactionServiceImpl) CreateScheduledTransaction(ctx context.Context, userID uuid.UUID, req domain.ScheduledTransactionRequest) (*domain.ScheduledTransaction, error) {
	scheduledTransaction, err := domain.NewScheduledTransaction(userID, req)
	if err != nil {
		return nil, err
	}

	err = s.scheduledRepo.Create(ctx, scheduledTransaction)
	if err != nil {
		return nil, err
	}

	s.logger.Info("Scheduled transaction created",
		"id", scheduledTransaction.ID,
		"user_id", userID,
		"scheduled_at", req.ScheduledAt)

	return scheduledTransaction, nil
}

func (s *ScheduledTransactionServiceImpl) GetScheduledTransaction(ctx context.Context, id uuid.UUID) (*domain.ScheduledTransaction, error) {
	return s.scheduledRepo.GetByID(ctx, id)
}

func (s *ScheduledTransactionServiceImpl) GetUserScheduledTransactions(ctx context.Context, userID uuid.UUID) ([]*domain.ScheduledTransaction, error) {
	return s.scheduledRepo.GetByUserID(ctx, userID)
}

func (s *ScheduledTransactionServiceImpl) UpdateScheduledTransaction(ctx context.Context, id uuid.UUID, req domain.ScheduledTransactionRequest) error {
	scheduledTransaction, err := s.scheduledRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	scheduledTransaction.Type = req.Type
	scheduledTransaction.Amount = req.Amount
	scheduledTransaction.Currency = req.Currency
	scheduledTransaction.Description = req.Description
	scheduledTransaction.ReferenceID = req.ReferenceID
	scheduledTransaction.ToUserID = req.ToUserID
	scheduledTransaction.ScheduledAt = req.ScheduledAt
	scheduledTransaction.RecurringType = req.RecurringType
	scheduledTransaction.RecurringConfig = req.RecurringConfig
	scheduledTransaction.UpdatedAt = time.Now()

	if req.MaxRetries != nil {
		scheduledTransaction.MaxRetries = *req.MaxRetries
	}

	return s.scheduledRepo.Update(ctx, scheduledTransaction)
}

func (s *ScheduledTransactionServiceImpl) CancelScheduledTransaction(ctx context.Context, id uuid.UUID) error {
	scheduledTransaction, err := s.scheduledRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	scheduledTransaction.UpdateStatus("cancelled")
	return s.scheduledRepo.Update(ctx, scheduledTransaction)
}

func (s *ScheduledTransactionServiceImpl) ExecuteScheduledTransactions(ctx context.Context) error {
	pendingTransactions, err := s.scheduledRepo.GetPendingScheduledTransactions(ctx)
	if err != nil {
		return err
	}

	for _, scheduledTransaction := range pendingTransactions {
		if err := s.executeScheduledTransaction(ctx, scheduledTransaction); err != nil {
			s.logger.Error("Failed to execute scheduled transaction",
				"id", scheduledTransaction.ID,
				"error", err)
			continue
		}
	}

	return nil
}

func (s *ScheduledTransactionServiceImpl) executeScheduledTransaction(ctx context.Context, scheduledTransaction *domain.ScheduledTransaction) error {
	transaction, err := domain.NewTransaction(scheduledTransaction.UserID, scheduledTransaction.Amount, scheduledTransaction.Description)
	if err != nil {
		scheduledTransaction.UpdateStatus("failed")
		s.scheduledRepo.Update(ctx, scheduledTransaction)
		return err
	}

	transaction.Type = scheduledTransaction.Type
	transaction.ReferenceID = scheduledTransaction.ReferenceID

	switch scheduledTransaction.Type {
	case domain.TransactionTypeCredit:
		err = s.processCreditTransaction(ctx, transaction)
	case domain.TransactionTypeDebit:
		err = s.processDebitTransaction(ctx, transaction)
	case domain.TransactionTypeTransfer:
		if scheduledTransaction.ToUserID != nil {
			err = s.processTransferTransaction(ctx, transaction, *scheduledTransaction.ToUserID)
		} else {
			err = fmt.Errorf("transfer transaction requires to_user_id")
		}
	default:
		err = domain.ErrInvalidTransactionStatus
	}

	if err != nil {
		scheduledTransaction.IncrementRetry()
		if scheduledTransaction.CanRetry() {
			scheduledTransaction.UpdateStatus("failed")
		} else {
			scheduledTransaction.UpdateStatus("cancelled")
		}
		s.scheduledRepo.Update(ctx, scheduledTransaction)
		return err
	}

	scheduledTransaction.UpdateStatus("completed")
	return s.scheduledRepo.Update(ctx, scheduledTransaction)
}

func (s *ScheduledTransactionServiceImpl) processCreditTransaction(ctx context.Context, transaction *domain.Transaction) error {
	balance, err := s.balanceRepo.GetByUserID(ctx, uint(transaction.UserID.ID()))
	if err != nil {
		return err
	}

	if err := balance.Add(transaction.Amount); err != nil {
		return err
	}

	if err := s.balanceRepo.Update(ctx, balance); err != nil {
		return err
	}

	transaction.BalanceAfter = balance.GetAmount()
	transaction.UpdateState(domain.TransactionStateCompleted)
	return s.transactionRepo.Create(ctx, transaction)
}

func (s *ScheduledTransactionServiceImpl) processDebitTransaction(ctx context.Context, transaction *domain.Transaction) error {
	balance, err := s.balanceRepo.GetByUserID(ctx, uint(transaction.UserID.ID()))
	if err != nil {
		return err
	}

	if err := balance.Subtract(transaction.Amount); err != nil {
		return err
	}

	if err := s.balanceRepo.Update(ctx, balance); err != nil {
		return err
	}

	transaction.BalanceAfter = balance.GetAmount()
	transaction.UpdateState(domain.TransactionStateCompleted)
	return s.transactionRepo.Create(ctx, transaction)
}

func (s *ScheduledTransactionServiceImpl) processTransferTransaction(ctx context.Context, transaction *domain.Transaction, toUserID uuid.UUID) error {
	sourceBalance, err := s.balanceRepo.GetByUserID(ctx, uint(transaction.UserID.ID()))
	if err != nil {
		return err
	}

	destBalance, err := s.balanceRepo.GetByUserID(ctx, uint(toUserID.ID()))
	if err != nil {
		return err
	}

	if err := sourceBalance.Subtract(transaction.Amount); err != nil {
		return err
	}

	if err := destBalance.Add(transaction.Amount); err != nil {
		return err
	}

	if err := s.balanceRepo.Update(ctx, sourceBalance); err != nil {
		return err
	}
	if err := s.balanceRepo.Update(ctx, destBalance); err != nil {
		return err
	}

	transaction.BalanceAfter = sourceBalance.GetAmount()
	transaction.UpdateState(domain.TransactionStateCompleted)
	return s.transactionRepo.Create(ctx, transaction)
}

type BatchTransactionServiceImpl struct {
	batchRepo       domain.BatchTransactionRepository
	batchItemRepo   domain.BatchTransactionItemRepository
	transactionRepo domain.TransactionRepository
	balanceRepo     domain.BalanceRepository
	logger          domain.Logger
	mu              sync.RWMutex
}

func NewBatchTransactionService(
	batchRepo domain.BatchTransactionRepository,
	batchItemRepo domain.BatchTransactionItemRepository,
	transactionRepo domain.TransactionRepository,
	balanceRepo domain.BalanceRepository,
	logger domain.Logger,
) domain.BatchTransactionService {
	return &BatchTransactionServiceImpl{
		batchRepo:       batchRepo,
		batchItemRepo:   batchItemRepo,
		transactionRepo: transactionRepo,
		balanceRepo:     balanceRepo,
		logger:          logger,
	}
}

func (s *BatchTransactionServiceImpl) CreateBatchTransaction(ctx context.Context, userID uuid.UUID, req domain.BatchTransactionRequest) (*domain.BatchTransaction, error) {
	batchTransaction, err := domain.NewBatchTransaction(userID, req)
	if err != nil {
		return nil, err
	}

	err = s.batchRepo.Create(ctx, batchTransaction)
	if err != nil {
		return nil, err
	}

	for _, item := range req.Items {
		batchItem := &domain.BatchTransactionItem{
			ID:            uuid.New(),
			BatchID:       batchTransaction.ID,
			TransactionID: uuid.New(), // Will be updated when transaction is created
			Amount:        item.Amount,
			Description:   item.Description,
			ReferenceID:   item.ReferenceID,
			Status:        "pending",
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}

		err = s.batchItemRepo.Create(ctx, batchItem)
		if err != nil {
			s.logger.Error("Failed to create batch item", "error", err)
			continue
		}
	}

	s.logger.Info("Batch transaction created",
		"id", batchTransaction.ID,
		"user_id", userID,
		"item_count", len(req.Items))

	return batchTransaction, nil
}

func (s *BatchTransactionServiceImpl) GetBatchTransaction(ctx context.Context, id uuid.UUID) (*domain.BatchTransaction, error) {
	return s.batchRepo.GetByID(ctx, id)
}

func (s *BatchTransactionServiceImpl) GetBatchTransactionItems(ctx context.Context, batchID uuid.UUID) ([]*domain.BatchTransactionItem, error) {
	return s.batchItemRepo.GetByBatchID(ctx, batchID)
}

func (s *BatchTransactionServiceImpl) ProcessBatchTransaction(ctx context.Context, id uuid.UUID) error {
	batchTransaction, err := s.batchRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if batchTransaction.Status != "pending" {
		return fmt.Errorf("batch transaction is not in pending status")
	}

	batchTransaction.UpdateStatus("processing")
	s.batchRepo.Update(ctx, batchTransaction)

	items, err := s.batchItemRepo.GetByBatchID(ctx, id)
	if err != nil {
		return err
	}

	successCount := 0
	failedCount := 0

	for _, item := range items {
		if err := s.processBatchItem(ctx, batchTransaction, item); err != nil {
			failedCount++
			s.logger.Error("Failed to process batch item",
				"item_id", item.ID,
				"error", err)
		} else {
			successCount++
		}
	}

	if failedCount == 0 {
		batchTransaction.UpdateStatus("completed")
	} else if successCount == 0 {
		batchTransaction.UpdateStatus("failed")
	} else {
		batchTransaction.UpdateStatus("partial")
	}

	return s.batchRepo.Update(ctx, batchTransaction)
}

func (s *BatchTransactionServiceImpl) CancelBatchTransaction(ctx context.Context, id uuid.UUID) error {
	batchTransaction, err := s.batchRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if batchTransaction.Status != "pending" {
		return fmt.Errorf("can only cancel pending batch transactions")
	}

	batchTransaction.UpdateStatus("cancelled")
	return s.batchRepo.Update(ctx, batchTransaction)
}

func (s *BatchTransactionServiceImpl) processBatchItem(ctx context.Context, batchTransaction *domain.BatchTransaction, item *domain.BatchTransactionItem) error {
	transaction, err := domain.NewTransaction(batchTransaction.UserID, item.Amount, item.Description)
	if err != nil {
		item.Status = "failed"
		errorMsg := err.Error()
		item.ErrorMessage = &errorMsg
		item.UpdatedAt = time.Now()
		s.batchItemRepo.Update(ctx, item)
		return err
	}

	transaction.Type = batchTransaction.Type
	transaction.ReferenceID = item.ReferenceID

	var processErr error
	switch batchTransaction.Type {
	case domain.TransactionTypeCredit:
		processErr = s.processCreditTransaction(ctx, transaction)
	case domain.TransactionTypeDebit:
		processErr = s.processDebitTransaction(ctx, transaction)
	case domain.TransactionTypeTransfer:
		processErr = fmt.Errorf("batch transfers not implemented")
	default:
		processErr = domain.ErrInvalidTransactionStatus
	}

	if processErr != nil {
		item.Status = "failed"
		errorMsg := processErr.Error()
		item.ErrorMessage = &errorMsg
		item.UpdatedAt = time.Now()
		s.batchItemRepo.Update(ctx, item)
		return processErr
	}

	item.TransactionID = transaction.ID
	item.Status = "completed"
	now := time.Now()
	item.ProcessedAt = &now
	item.UpdatedAt = time.Now()

	return s.batchItemRepo.Update(ctx, item)
}

func (s *BatchTransactionServiceImpl) processCreditTransaction(ctx context.Context, transaction *domain.Transaction) error {
	balance, err := s.balanceRepo.GetByUserID(ctx, uint(transaction.UserID.ID()))
	if err != nil {
		return err
	}

	if err := balance.Add(transaction.Amount); err != nil {
		return err
	}

	if err := s.balanceRepo.Update(ctx, balance); err != nil {
		return err
	}

	transaction.BalanceAfter = balance.GetAmount()
	transaction.UpdateState(domain.TransactionStateCompleted)
	return s.transactionRepo.Create(ctx, transaction)
}

func (s *BatchTransactionServiceImpl) processDebitTransaction(ctx context.Context, transaction *domain.Transaction) error {
	balance, err := s.balanceRepo.GetByUserID(ctx, uint(transaction.UserID.ID()))
	if err != nil {
		return err
	}

	if err := balance.Subtract(transaction.Amount); err != nil {
		return err
	}

	if err := s.balanceRepo.Update(ctx, balance); err != nil {
		return err
	}

	transaction.BalanceAfter = balance.GetAmount()
	transaction.UpdateState(domain.TransactionStateCompleted)
	return s.transactionRepo.Create(ctx, transaction)
}
