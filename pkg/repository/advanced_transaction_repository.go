package repository

import (
	"context"
	"fmt"
	"time"

	"transaction-api-w-go/pkg/domain"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ScheduledTransactionRepositoryImpl struct {
	db *gorm.DB
}

func NewScheduledTransactionRepository(db *gorm.DB) domain.ScheduledTransactionRepository {
	return &ScheduledTransactionRepositoryImpl{db: db}
}

func (r *ScheduledTransactionRepositoryImpl) Create(ctx context.Context, scheduledTransaction *domain.ScheduledTransaction) error {
	return r.db.WithContext(ctx).Create(scheduledTransaction).Error
}

func (r *ScheduledTransactionRepositoryImpl) GetByID(ctx context.Context, id uuid.UUID) (*domain.ScheduledTransaction, error) {
	var scheduledTransaction domain.ScheduledTransaction
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&scheduledTransaction).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrScheduledTransactionNotFound
		}
		return nil, err
	}
	return &scheduledTransaction, nil
}

func (r *ScheduledTransactionRepositoryImpl) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.ScheduledTransaction, error) {
	var scheduledTransactions []*domain.ScheduledTransaction
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("scheduled_at ASC").Find(&scheduledTransactions).Error
	if err != nil {
		return nil, err
	}
	return scheduledTransactions, nil
}

func (r *ScheduledTransactionRepositoryImpl) GetPendingScheduledTransactions(ctx context.Context) ([]*domain.ScheduledTransaction, error) {
	var scheduledTransactions []*domain.ScheduledTransaction
	err := r.db.WithContext(ctx).
		Where("status = ? AND scheduled_at <= ?", "pending", time.Now()).
		Order("scheduled_at ASC").
		Find(&scheduledTransactions).Error
	if err != nil {
		return nil, err
	}
	return scheduledTransactions, nil
}

func (r *ScheduledTransactionRepositoryImpl) Update(ctx context.Context, scheduledTransaction *domain.ScheduledTransaction) error {
	return r.db.WithContext(ctx).Save(scheduledTransaction).Error
}

func (r *ScheduledTransactionRepositoryImpl) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&domain.ScheduledTransaction{}).Error
}

type BatchTransactionRepositoryImpl struct {
	db *gorm.DB
}

func NewBatchTransactionRepository(db *gorm.DB) domain.BatchTransactionRepository {
	return &BatchTransactionRepositoryImpl{db: db}
}

func (r *BatchTransactionRepositoryImpl) Create(ctx context.Context, batchTransaction *domain.BatchTransaction) error {
	return r.db.WithContext(ctx).Create(batchTransaction).Error
}

func (r *BatchTransactionRepositoryImpl) GetByID(ctx context.Context, id uuid.UUID) (*domain.BatchTransaction, error) {
	var batchTransaction domain.BatchTransaction
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&batchTransaction).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrBatchTransactionNotFound
		}
		return nil, err
	}
	return &batchTransaction, nil
}

func (r *BatchTransactionRepositoryImpl) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.BatchTransaction, error) {
	var batchTransactions []*domain.BatchTransaction
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("created_at DESC").Find(&batchTransactions).Error
	if err != nil {
		return nil, err
	}
	return batchTransactions, nil
}

func (r *BatchTransactionRepositoryImpl) Update(ctx context.Context, batchTransaction *domain.BatchTransaction) error {
	return r.db.WithContext(ctx).Save(batchTransaction).Error
}

func (r *BatchTransactionRepositoryImpl) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&domain.BatchTransaction{}).Error
}

type BatchTransactionItemRepositoryImpl struct {
	db *gorm.DB
}

func NewBatchTransactionItemRepository(db *gorm.DB) domain.BatchTransactionItemRepository {
	return &BatchTransactionItemRepositoryImpl{db: db}
}

func (r *BatchTransactionItemRepositoryImpl) Create(ctx context.Context, item *domain.BatchTransactionItem) error {
	return r.db.WithContext(ctx).Create(item).Error
}

func (r *BatchTransactionItemRepositoryImpl) GetByBatchID(ctx context.Context, batchID uuid.UUID) ([]*domain.BatchTransactionItem, error) {
	var items []*domain.BatchTransactionItem
	err := r.db.WithContext(ctx).Where("batch_id = ?", batchID).Order("created_at ASC").Find(&items).Error
	if err != nil {
		return nil, err
	}
	return items, nil
}

func (r *BatchTransactionItemRepositoryImpl) Update(ctx context.Context, item *domain.BatchTransactionItem) error {
	return r.db.WithContext(ctx).Save(item).Error
}

func (r *BatchTransactionItemRepositoryImpl) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&domain.BatchTransactionItem{}).Error
}

type TransactionLimitRepositoryImpl struct {
	db *gorm.DB
}

func NewTransactionLimitRepository(db *gorm.DB) domain.TransactionLimitRepository {
	return &TransactionLimitRepositoryImpl{db: db}
}

func (r *TransactionLimitRepositoryImpl) Create(ctx context.Context, limit *domain.TransactionLimit) error {
	return r.db.WithContext(ctx).Create(limit).Error
}

func (r *TransactionLimitRepositoryImpl) GetByUserIDAndCurrency(ctx context.Context, userID uuid.UUID, currency domain.Currency) (*domain.TransactionLimit, error) {
	var limit domain.TransactionLimit
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND currency = ?", userID, currency).
		First(&limit).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("transaction limit not found for user %s and currency %s", userID, currency)
		}
		return nil, err
	}
	return &limit, nil
}

func (r *TransactionLimitRepositoryImpl) Update(ctx context.Context, limit *domain.TransactionLimit) error {
	return r.db.WithContext(ctx).Save(limit).Error
}

func (r *TransactionLimitRepositoryImpl) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&domain.TransactionLimit{}).Error
}

type MultiCurrencyBalanceRepositoryImpl struct {
	db *gorm.DB
}

func NewMultiCurrencyBalanceRepository(db *gorm.DB) domain.MultiCurrencyBalanceRepository {
	return &MultiCurrencyBalanceRepositoryImpl{db: db}
}

func (r *MultiCurrencyBalanceRepositoryImpl) Create(ctx context.Context, balance *domain.MultiCurrencyBalance) error {
	return r.db.WithContext(ctx).Create(balance).Error
}

func (r *MultiCurrencyBalanceRepositoryImpl) GetByUserIDAndCurrency(ctx context.Context, userID uuid.UUID, currency domain.Currency) (*domain.MultiCurrencyBalance, error) {
	var balance domain.MultiCurrencyBalance
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND currency = ?", userID, currency).
		First(&balance).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("balance not found for user %s and currency %s", userID, currency)
		}
		return nil, err
	}
	return &balance, nil
}

func (r *MultiCurrencyBalanceRepositoryImpl) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.MultiCurrencyBalance, error) {
	var balances []*domain.MultiCurrencyBalance
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&balances).Error
	if err != nil {
		return nil, err
	}
	return balances, nil
}

func (r *MultiCurrencyBalanceRepositoryImpl) Update(ctx context.Context, balance *domain.MultiCurrencyBalance) error {
	return r.db.WithContext(ctx).Save(balance).Error
}

func (r *MultiCurrencyBalanceRepositoryImpl) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&domain.MultiCurrencyBalance{}).Error
}
