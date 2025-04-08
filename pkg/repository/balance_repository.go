package repository

import (
	"context"
	"database/sql"
	"transaction-api-w-go/pkg/domain"
)

type balanceRepository struct {
	db *sql.DB
}

func NewBalanceRepository(db *sql.DB) domain.BalanceRepository {
	return &balanceRepository{db: db}
}

func (r *balanceRepository) Create(ctx context.Context, balance *domain.Balance) error {
	query := `
		INSERT INTO balances (user_id, amount, currency, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING user_id`

	err := r.db.QueryRowContext(ctx, query,
		balance.UserID,
		balance.Amount,
		balance.Currency,
		balance.CreatedAt,
		balance.UpdatedAt,
	).Scan(&balance.UserID)

	if err != nil {
		return err
	}

	return nil
}

func (r *balanceRepository) GetByUserID(ctx context.Context, userID uint) (*domain.Balance, error) {
	query := `
		SELECT user_id, amount, currency, created_at, updated_at
		FROM balances
		WHERE user_id = $1`

	balance := &domain.Balance{}
	err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&balance.UserID,
		&balance.Amount,
		&balance.Currency,
		&balance.CreatedAt,
		&balance.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, domain.ErrInsufficientBalance
	}
	if err != nil {
		return nil, err
	}

	return balance, nil
}

func (r *balanceRepository) Update(ctx context.Context, balance *domain.Balance) error {
	query := `
		UPDATE balances
		SET amount = $1, updated_at = $2
		WHERE user_id = $3`

	result, err := r.db.ExecContext(ctx, query,
		balance.Amount,
		balance.UpdatedAt,
		balance.UserID,
	)

	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return domain.ErrInsufficientBalance
	}

	return nil
}
