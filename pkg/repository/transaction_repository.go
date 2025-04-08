package repository

import (
	"context"
	"database/sql"
	"transaction-api-w-go/pkg/domain"
)

type transactionRepository struct {
	db *sql.DB
}

func NewTransactionRepository(db *sql.DB) domain.TransactionRepository {
	return &transactionRepository{db: db}
}

func (r *transactionRepository) Create(ctx context.Context, transaction *domain.Transaction) error {
	query := `
		INSERT INTO transactions (from_user_id, to_user_id, amount, state, description, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id`

	err := r.db.QueryRowContext(ctx, query,
		transaction.FromUserID,
		transaction.ToUserID,
		transaction.Amount,
		transaction.State,
		transaction.Description,
		transaction.CreatedAt,
		transaction.UpdatedAt,
	).Scan(&transaction.ID)

	if err != nil {
		return err
	}

	return nil
}

func (r *transactionRepository) GetByID(ctx context.Context, id uint) (*domain.Transaction, error) {
	query := `
		SELECT id, from_user_id, to_user_id, amount, state, description, created_at, updated_at
		FROM transactions
		WHERE id = $1`

	transaction := &domain.Transaction{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&transaction.ID,
		&transaction.FromUserID,
		&transaction.ToUserID,
		&transaction.Amount,
		&transaction.State,
		&transaction.Description,
		&transaction.CreatedAt,
		&transaction.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, domain.ErrTransactionFailed
	}
	if err != nil {
		return nil, err
	}

	return transaction, nil
}

func (r *transactionRepository) GetByUserID(ctx context.Context, userID uint) ([]*domain.Transaction, error) {
	query := `
		SELECT id, from_user_id, to_user_id, amount, state, description, created_at, updated_at
		FROM transactions
		WHERE from_user_id = $1 OR to_user_id = $1
		ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var transactions []*domain.Transaction
	for rows.Next() {
		transaction := &domain.Transaction{}
		err := rows.Scan(
			&transaction.ID,
			&transaction.FromUserID,
			&transaction.ToUserID,
			&transaction.Amount,
			&transaction.State,
			&transaction.Description,
			&transaction.CreatedAt,
			&transaction.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		transactions = append(transactions, transaction)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return transactions, nil
}

func (r *transactionRepository) Update(ctx context.Context, transaction *domain.Transaction) error {
	query := `
		UPDATE transactions
		SET state = $1, updated_at = $2
		WHERE id = $3`

	result, err := r.db.ExecContext(ctx, query,
		transaction.State,
		transaction.UpdatedAt,
		transaction.ID,
	)

	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return domain.ErrTransactionFailed
	}

	return nil
}
