package repository

import (
	"errors"
	"time"

	"transaction-api-w-go/pkg/domain"

	"gorm.io/gorm"
)

type BalanceRepository struct {
	db *gorm.DB
}

func NewBalanceRepository(db *gorm.DB) *BalanceRepository {
	return &BalanceRepository{
		db: db,
	}
}

func (r *BalanceRepository) Create(balance *domain.Balance) error {
	return r.db.Create(balance).Error
}

func (r *BalanceRepository) GetByUserID(userID string) (*domain.Balance, error) {
	var balance domain.Balance
	if err := r.db.Where("user_id = ?", userID).First(&balance).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("hesap bulunamadı")
		}
		return nil, err
	}
	return &balance, nil
}

func (r *BalanceRepository) Update(balance *domain.Balance) error {
	return r.db.Save(balance).Error
}

func (r *BalanceRepository) GetHistory(userID string) ([]domain.BalanceHistory, error) {
	var history []domain.BalanceHistory
	if err := r.db.Where("user_id = ?", userID).Order("timestamp DESC").Find(&history).Error; err != nil {
		return nil, err
	}
	return history, nil
}

func (r *BalanceRepository) GetBalanceAtTime(userID string, timestamp time.Time) (*domain.BalanceHistory, error) {
	var history domain.BalanceHistory
	if err := r.db.Where("user_id = ? AND timestamp <= ?", userID, timestamp).
		Order("timestamp DESC").
		First(&history).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("belirtilen zamanda bakiye kaydı bulunamadı")
		}
		return nil, err
	}
	return &history, nil
}
