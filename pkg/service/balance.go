package service

import (
	"time"

	"transaction-api-w-go/pkg/domain"
	"transaction-api-w-go/pkg/repository"

	"github.com/google/uuid"
)

type BalanceService struct {
	balanceRepo *repository.BalanceRepository
}

func NewBalanceService(balanceRepo *repository.BalanceRepository) *BalanceService {
	return &BalanceService{
		balanceRepo: balanceRepo,
	}
}

func (s *BalanceService) GetCurrentBalance(userID string) (*domain.Balance, error) {
	return s.balanceRepo.GetByUserID(userID)
}

func (s *BalanceService) GetHistoricalBalance(userID string) ([]domain.BalanceHistory, error) {
	return s.balanceRepo.GetHistory(userID)
}

func (s *BalanceService) GetBalanceAtTime(userID string, timestamp time.Time) (*domain.BalanceHistory, error) {
	return s.balanceRepo.GetBalanceAtTime(userID, timestamp)
}

func (s *BalanceService) CreateInitialBalance(userID string) error {
	balance := &domain.Balance{
		ID:        uuid.New(),
		UserID:    uuid.MustParse(userID),
		Amount:    0,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	return s.balanceRepo.Create(balance)
}
