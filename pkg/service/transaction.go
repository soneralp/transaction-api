package service

import (
	"errors"
	"time"

	"transaction-api-w-go/pkg/domain"
	"transaction-api-w-go/pkg/repository"

	"github.com/google/uuid"
)

type TransactionService struct {
	transactionRepo *repository.TransactionRepository
	balanceRepo     *repository.BalanceRepository
	userRepo        *repository.UserRepository
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
	}
}

func (s *TransactionService) Credit(userID string, amount float64, description string) (*domain.Transaction, error) {
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

	if err := s.transactionRepo.Create(transaction); err != nil {
		return nil, err
	}

	balance.Amount += amount
	if err := s.balanceRepo.Update(balance); err != nil {
		return nil, err
	}

	return transaction, nil
}

func (s *TransactionService) Debit(userID string, amount float64, description string) (*domain.Transaction, error) {
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

	if err := s.transactionRepo.Create(transaction); err != nil {
		return nil, err
	}

	balance.Amount -= amount
	if err := s.balanceRepo.Update(balance); err != nil {
		return nil, err
	}

	return transaction, nil
}

func (s *TransactionService) Transfer(fromUserID, toUserID string, amount float64, description string) (*domain.Transaction, error) {
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

	if err := s.transactionRepo.Create(transaction); err != nil {
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

func (s *TransactionService) GetHistory(userID string) ([]domain.Transaction, error) {
	return s.transactionRepo.GetByUserID(userID)
}

func (s *TransactionService) GetByID(userID, transactionID string) (*domain.Transaction, error) {
	return s.transactionRepo.GetByID(userID, transactionID)
}
