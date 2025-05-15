package repository

import (
	"errors"

	"transaction-api-w-go/pkg/domain"

	"gorm.io/gorm"
)

type TransactionRepository struct {
	db *gorm.DB
}

func NewTransactionRepository(db *gorm.DB) *TransactionRepository {
	return &TransactionRepository{
		db: db,
	}
}

func (r *TransactionRepository) Create(transaction *domain.Transaction) error {
	return r.db.Create(transaction).Error
}

func (r *TransactionRepository) GetByID(userID, transactionID string) (*domain.Transaction, error) {
	var transaction domain.Transaction
	if err := r.db.Where("id = ? AND user_id = ?", transactionID, userID).First(&transaction).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("işlem bulunamadı")
		}
		return nil, err
	}
	return &transaction, nil
}

func (r *TransactionRepository) GetByUserID(userID string) ([]domain.Transaction, error) {
	var transactions []domain.Transaction
	if err := r.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&transactions).Error; err != nil {
		return nil, err
	}
	return transactions, nil
}
