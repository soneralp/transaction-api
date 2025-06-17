package service

import (
	"time"

	"transaction-api-w-go/pkg/domain"
	"transaction-api-w-go/pkg/metrics"
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
	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		metrics.DatabaseQueryDuration.WithLabelValues("get_current_balance").Observe(duration)
	}()

	balance, err := s.balanceRepo.GetByUserID(userID)
	if err != nil {
		return nil, err
	}

	metrics.BalanceTotal.WithLabelValues(userID).Set(balance.Amount)
	return balance, nil
}

func (s *BalanceService) GetHistoricalBalance(userID string) ([]domain.BalanceHistory, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		metrics.DatabaseQueryDuration.WithLabelValues("get_historical_balance").Observe(duration)
	}()

	return s.balanceRepo.GetHistory(userID)
}

func (s *BalanceService) GetBalanceAtTime(userID string, timestamp time.Time) (*domain.BalanceHistory, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		metrics.DatabaseQueryDuration.WithLabelValues("get_balance_at_time").Observe(duration)
	}()

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
