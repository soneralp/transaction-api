package domain

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Currency para birimi
type Currency string

const (
	CurrencyUSD Currency = "USD"
	CurrencyEUR Currency = "EUR"
	CurrencyTRY Currency = "TRY"
	CurrencyGBP Currency = "GBP"
)

type ExchangeRate struct {
	FromCurrency Currency  `json:"from_currency"`
	ToCurrency   Currency  `json:"to_currency"`
	Rate         float64   `json:"rate"`
	LastUpdated  time.Time `json:"last_updated"`
	Source       string    `json:"source"`
}

type ScheduledTransaction struct {
	ID              uuid.UUID       `json:"id" gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	UserID          uuid.UUID       `json:"user_id" gorm:"type:uuid;not null"`
	Type            TransactionType `json:"type" gorm:"type:varchar(20);not null"`
	Amount          float64         `json:"amount" gorm:"type:decimal(19,4);not null"`
	Currency        Currency        `json:"currency" gorm:"type:varchar(3);not null;default:'USD'"`
	Description     string          `json:"description" gorm:"type:text"`
	ReferenceID     string          `json:"reference_id" gorm:"type:varchar(100)"`
	ToUserID        *uuid.UUID      `json:"to_user_id,omitempty" gorm:"type:uuid"`
	ScheduledAt     time.Time       `json:"scheduled_at" gorm:"not null;index"`
	Status          string          `json:"status" gorm:"type:varchar(20);not null;default:'pending'"`
	RecurringType   *string         `json:"recurring_type,omitempty" gorm:"type:varchar(20)"`
	RecurringConfig *string         `json:"recurring_config,omitempty" gorm:"type:jsonb"`
	MaxRetries      int             `json:"max_retries" gorm:"not null;default:3"`
	RetryCount      int             `json:"retry_count" gorm:"not null;default:0"`
	LastRetryAt     *time.Time      `json:"last_retry_at,omitempty"`
	NextRetryAt     *time.Time      `json:"next_retry_at,omitempty"`
	CreatedAt       time.Time       `json:"created_at" gorm:"not null"`
	UpdatedAt       time.Time       `json:"updated_at" gorm:"not null"`
	mu              sync.RWMutex    `json:"-"`
}

type ScheduledTransactionRequest struct {
	Type            TransactionType `json:"type" binding:"required"`
	Amount          float64         `json:"amount" binding:"required,gt=0"`
	Currency        Currency        `json:"currency" binding:"required"`
	Description     string          `json:"description"`
	ReferenceID     string          `json:"reference_id"`
	ToUserID        *uuid.UUID      `json:"to_user_id,omitempty"`
	ScheduledAt     time.Time       `json:"scheduled_at" binding:"required"`
	RecurringType   *string         `json:"recurring_type,omitempty"`
	RecurringConfig *string         `json:"recurring_config,omitempty"`
	MaxRetries      *int            `json:"max_retries,omitempty"`
}

type BatchTransaction struct {
	ID          uuid.UUID       `json:"id" gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	UserID      uuid.UUID       `json:"user_id" gorm:"type:uuid;not null"`
	Type        TransactionType `json:"type" gorm:"type:varchar(20);not null"`
	Currency    Currency        `json:"currency" gorm:"type:varchar(3);not null;default:'USD'"`
	Description string          `json:"description" gorm:"type:text"`
	Status      string          `json:"status" gorm:"type:varchar(20);not null;default:'pending'"`
	TotalAmount float64         `json:"total_amount" gorm:"type:decimal(19,4);not null"`
	ItemCount   int             `json:"item_count" gorm:"not null"`
	ProcessedAt *time.Time      `json:"processed_at,omitempty"`
	CreatedAt   time.Time       `json:"created_at" gorm:"not null"`
	UpdatedAt   time.Time       `json:"updated_at" gorm:"not null"`
	mu          sync.RWMutex    `json:"-"`
}

type BatchTransactionItem struct {
	ID            uuid.UUID  `json:"id" gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	BatchID       uuid.UUID  `json:"batch_id" gorm:"type:uuid;not null"`
	TransactionID uuid.UUID  `json:"transaction_id" gorm:"type:uuid;not null"`
	Amount        float64    `json:"amount" gorm:"type:decimal(19,4);not null"`
	Description   string     `json:"description" gorm:"type:text"`
	ReferenceID   string     `json:"reference_id" gorm:"type:varchar(100)"`
	Status        string     `json:"status" gorm:"type:varchar(20);not null;default:'pending'"`
	ErrorMessage  *string    `json:"error_message,omitempty" gorm:"type:text"`
	ProcessedAt   *time.Time `json:"processed_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at" gorm:"not null"`
	UpdatedAt     time.Time  `json:"updated_at" gorm:"not null"`
}

type BatchTransactionRequest struct {
	Type        TransactionType `json:"type" binding:"required"`
	Currency    Currency        `json:"currency" binding:"required"`
	Description string          `json:"description"`
	Items       []BatchItem     `json:"items" binding:"required,min=1,max=1000"`
}

type BatchItem struct {
	Amount      float64    `json:"amount" binding:"required,gt=0"`
	Description string     `json:"description"`
	ReferenceID string     `json:"reference_id"`
	ToUserID    *uuid.UUID `json:"to_user_id,omitempty"`
}

type TransactionLimit struct {
	ID            uuid.UUID    `json:"id" gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	UserID        uuid.UUID    `json:"user_id" gorm:"type:uuid;not null;uniqueIndex"`
	Currency      Currency     `json:"currency" gorm:"type:varchar(3);not null"`
	DailyLimit    float64      `json:"daily_limit" gorm:"type:decimal(19,4);not null"`
	WeeklyLimit   float64      `json:"weekly_limit" gorm:"type:decimal(19,4);not null"`
	MonthlyLimit  float64      `json:"monthly_limit" gorm:"type:decimal(19,4);not null"`
	SingleLimit   float64      `json:"single_limit" gorm:"type:decimal(19,4);not null"`
	DailyCount    int          `json:"daily_count" gorm:"not null;default:0"`
	WeeklyCount   int          `json:"weekly_count" gorm:"not null;default:0"`
	MonthlyCount  int          `json:"monthly_count" gorm:"not null;default:0"`
	DailyAmount   float64      `json:"daily_amount" gorm:"type:decimal(19,4);not null;default:0"`
	WeeklyAmount  float64      `json:"weekly_amount" gorm:"type:decimal(19,4);not null;default:0"`
	MonthlyAmount float64      `json:"monthly_amount" gorm:"type:decimal(19,4);not null;default:0"`
	LastResetDate time.Time    `json:"last_reset_date" gorm:"not null"`
	IsActive      bool         `json:"is_active" gorm:"not null;default:true"`
	CreatedAt     time.Time    `json:"created_at" gorm:"not null"`
	UpdatedAt     time.Time    `json:"updated_at" gorm:"not null"`
	mu            sync.RWMutex `json:"-"`
}

type TransactionLimitRequest struct {
	Currency     Currency `json:"currency" binding:"required"`
	DailyLimit   float64  `json:"daily_limit" binding:"required,gt=0"`
	WeeklyLimit  float64  `json:"weekly_limit" binding:"required,gt=0"`
	MonthlyLimit float64  `json:"monthly_limit" binding:"required,gt=0"`
	SingleLimit  float64  `json:"single_limit" binding:"required,gt=0"`
}

type MultiCurrencyBalance struct {
	ID        uuid.UUID    `json:"id" gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	UserID    uuid.UUID    `json:"user_id" gorm:"type:uuid;not null;uniqueIndex"`
	Currency  Currency     `json:"currency" gorm:"type:varchar(3);not null"`
	Amount    float64      `json:"amount" gorm:"type:decimal(19,4);not null"`
	CreatedAt time.Time    `json:"created_at" gorm:"not null"`
	UpdatedAt time.Time    `json:"updated_at" gorm:"not null"`
	mu        sync.RWMutex `json:"-"`
}

type CurrencyConversionRequest struct {
	FromCurrency Currency `json:"from_currency" binding:"required"`
	ToCurrency   Currency `json:"to_currency" binding:"required"`
	Amount       float64  `json:"amount" binding:"required,gt=0"`
}

type CurrencyConversionResponse struct {
	FromCurrency Currency  `json:"from_currency"`
	ToCurrency   Currency  `json:"to_currency"`
	FromAmount   float64   `json:"from_amount"`
	ToAmount     float64   `json:"to_amount"`
	Rate         float64   `json:"rate"`
	LastUpdated  time.Time `json:"last_updated"`
}

type RecurringConfig struct {
	Type           string     `json:"type"`
	Interval       int        `json:"interval"`
	EndDate        *time.Time `json:"end_date,omitempty"`
	MaxOccurrences *int       `json:"max_occurrences,omitempty"`
	DayOfWeek      *int       `json:"day_of_week,omitempty"`
	DayOfMonth     *int       `json:"day_of_month,omitempty"`
	MonthOfYear    *int       `json:"month_of_year,omitempty"`
}

func NewScheduledTransaction(userID uuid.UUID, req ScheduledTransactionRequest) (*ScheduledTransaction, error) {
	if req.Amount <= 0 {
		return nil, ErrInvalidAmount
	}

	if req.ScheduledAt.Before(time.Now()) {
		return nil, ErrInvalidScheduledTime
	}

	maxRetries := 3
	if req.MaxRetries != nil {
		maxRetries = *req.MaxRetries
	}

	return &ScheduledTransaction{
		ID:              uuid.New(),
		UserID:          userID,
		Type:            req.Type,
		Amount:          req.Amount,
		Currency:        req.Currency,
		Description:     req.Description,
		ReferenceID:     req.ReferenceID,
		ToUserID:        req.ToUserID,
		ScheduledAt:     req.ScheduledAt,
		Status:          "pending",
		RecurringType:   req.RecurringType,
		RecurringConfig: req.RecurringConfig,
		MaxRetries:      maxRetries,
		RetryCount:      0,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}, nil
}

func NewBatchTransaction(userID uuid.UUID, req BatchTransactionRequest) (*BatchTransaction, error) {
	if len(req.Items) == 0 {
		return nil, ErrInvalidBatchItems
	}

	if len(req.Items) > 1000 {
		return nil, ErrBatchSizeExceeded
	}

	totalAmount := 0.0
	for _, item := range req.Items {
		if item.Amount <= 0 {
			return nil, ErrInvalidAmount
		}
		totalAmount += item.Amount
	}

	return &BatchTransaction{
		ID:          uuid.New(),
		UserID:      userID,
		Type:        req.Type,
		Currency:    req.Currency,
		Description: req.Description,
		Status:      "pending",
		TotalAmount: totalAmount,
		ItemCount:   len(req.Items),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}, nil
}

func NewTransactionLimit(userID uuid.UUID, req TransactionLimitRequest) (*TransactionLimit, error) {
	if req.DailyLimit <= 0 || req.WeeklyLimit <= 0 || req.MonthlyLimit <= 0 || req.SingleLimit <= 0 {
		return nil, ErrInvalidLimit
	}

	return &TransactionLimit{
		ID:            uuid.New(),
		UserID:        userID,
		Currency:      req.Currency,
		DailyLimit:    req.DailyLimit,
		WeeklyLimit:   req.WeeklyLimit,
		MonthlyLimit:  req.MonthlyLimit,
		SingleLimit:   req.SingleLimit,
		DailyCount:    0,
		WeeklyCount:   0,
		MonthlyCount:  0,
		DailyAmount:   0,
		WeeklyAmount:  0,
		MonthlyAmount: 0,
		LastResetDate: time.Now(),
		IsActive:      true,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}, nil
}

func NewMultiCurrencyBalance(userID uuid.UUID, currency Currency, initialAmount float64) (*MultiCurrencyBalance, error) {
	if initialAmount < 0 {
		return nil, ErrInvalidAmount
	}

	return &MultiCurrencyBalance{
		ID:        uuid.New(),
		UserID:    userID,
		Currency:  currency,
		Amount:    initialAmount,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}, nil
}

func (st *ScheduledTransaction) ShouldExecute() bool {
	st.mu.RLock()
	defer st.mu.RUnlock()

	return st.Status == "pending" && time.Now().After(st.ScheduledAt)
}

func (st *ScheduledTransaction) CanRetry() bool {
	st.mu.RLock()
	defer st.mu.RUnlock()

	return st.Status == "failed" && st.RetryCount < st.MaxRetries
}

func (st *ScheduledTransaction) IncrementRetry() {
	st.mu.Lock()
	defer st.mu.Unlock()

	st.RetryCount++
	now := time.Now()
	st.LastRetryAt = &now
	st.UpdatedAt = time.Now()
}

func (st *ScheduledTransaction) UpdateStatus(status string) {
	st.mu.Lock()
	defer st.mu.Unlock()

	st.Status = status
	st.UpdatedAt = time.Now()
}

func (bt *BatchTransaction) UpdateStatus(status string) {
	bt.mu.Lock()
	defer bt.mu.Unlock()

	bt.Status = status
	bt.UpdatedAt = time.Now()

	if status == "completed" || status == "failed" {
		now := time.Now()
		bt.ProcessedAt = &now
	}
}

func (tl *TransactionLimit) CheckSingleLimit(amount float64) error {
	tl.mu.RLock()
	defer tl.mu.RUnlock()

	if !tl.IsActive {
		return nil
	}

	if amount > tl.SingleLimit {
		return ErrTransactionLimitExceeded
	}

	return nil
}

func (tl *TransactionLimit) CheckDailyLimit(amount float64) error {
	tl.mu.Lock()
	defer tl.mu.Unlock()

	if !tl.IsActive {
		return nil
	}

	if time.Now().Sub(tl.LastResetDate) >= 24*time.Hour {
		tl.resetDailyLimits()
	}

	if tl.DailyAmount+amount > tl.DailyLimit {
		return ErrDailyLimitExceeded
	}

	if tl.DailyCount >= 100 {
		return ErrDailyCountExceeded
	}

	return nil
}

func (tl *TransactionLimit) UpdateDailyUsage(amount float64) {
	tl.mu.Lock()
	defer tl.mu.Unlock()

	tl.DailyAmount += amount
	tl.DailyCount++
	tl.UpdatedAt = time.Now()
}

func (tl *TransactionLimit) resetDailyLimits() {
	tl.DailyAmount = 0
	tl.DailyCount = 0
	tl.LastResetDate = time.Now()
}

func (mcb *MultiCurrencyBalance) Add(amount float64) error {
	if amount <= 0 {
		return ErrInvalidAmount
	}

	mcb.mu.Lock()
	defer mcb.mu.Unlock()

	mcb.Amount += amount
	mcb.UpdatedAt = time.Now()
	return nil
}

func (mcb *MultiCurrencyBalance) Subtract(amount float64) error {
	if amount <= 0 {
		return ErrInvalidAmount
	}

	mcb.mu.Lock()
	defer mcb.mu.Unlock()

	if mcb.Amount < amount {
		return ErrInsufficientBalance
	}

	mcb.Amount -= amount
	mcb.UpdatedAt = time.Now()
	return nil
}

func (mcb *MultiCurrencyBalance) GetAmount() float64 {
	mcb.mu.RLock()
	defer mcb.mu.RUnlock()
	return mcb.Amount
}

func (st *ScheduledTransaction) MarshalJSON() ([]byte, error) {
	st.mu.RLock()
	defer st.mu.RUnlock()

	type Alias ScheduledTransaction
	return json.Marshal(&struct {
		*Alias
	}{
		Alias: (*Alias)(st),
	})
}

func (bt *BatchTransaction) MarshalJSON() ([]byte, error) {
	bt.mu.RLock()
	defer bt.mu.RUnlock()

	type Alias BatchTransaction
	return json.Marshal(&struct {
		*Alias
	}{
		Alias: (*Alias)(bt),
	})
}

func (tl *TransactionLimit) MarshalJSON() ([]byte, error) {
	tl.mu.RLock()
	defer tl.mu.RUnlock()

	type Alias TransactionLimit
	return json.Marshal(&struct {
		*Alias
	}{
		Alias: (*Alias)(tl),
	})
}

func (mcb *MultiCurrencyBalance) MarshalJSON() ([]byte, error) {
	mcb.mu.RLock()
	defer mcb.mu.RUnlock()

	type Alias MultiCurrencyBalance
	return json.Marshal(&struct {
		*Alias
	}{
		Alias: (*Alias)(mcb),
	})
}
