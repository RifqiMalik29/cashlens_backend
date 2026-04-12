package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rifqimalik/cashlens-backend/internal/models"
)

type TransactionRepository interface {
	Create(ctx context.Context, tx *models.Transaction) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Transaction, error)
	ListByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*models.TransactionWithCategory, error)
	ListByDateRange(ctx context.Context, userID uuid.UUID, startDate, endDate time.Time) ([]*models.TransactionWithCategory, error)
	Update(ctx context.Context, tx *models.Transaction) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type transactionRepository struct {
	db *pgxpool.Pool
}

func NewTransactionRepository(db *pgxpool.Pool) TransactionRepository {
	return &transactionRepository{db: db}
}

func (r *transactionRepository) Create(ctx context.Context, tx *models.Transaction) error {
	query := `INSERT INTO transactions (id, user_id, category_id, amount, description, transaction_date, attachment_url, metadata, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

	_, err := r.db.Exec(ctx, query, tx.ID, tx.UserID, tx.CategoryID, tx.Amount, tx.Description, tx.TransactionDate, tx.AttachmentURL, tx.Metadata, tx.CreatedAt, tx.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create transaction: %w", err)
	}

	return nil
}

func (r *transactionRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Transaction, error) {
	tx := &models.Transaction{}
	query := `SELECT id, user_id, category_id, amount, description, transaction_date, attachment_url, metadata, created_at, updated_at FROM transactions WHERE id = $1`

	err := r.db.QueryRow(ctx, query, id).Scan(&tx.ID, &tx.UserID, &tx.CategoryID, &tx.Amount, &tx.Description, &tx.TransactionDate, &tx.AttachmentURL, &tx.Metadata, &tx.CreatedAt, &tx.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}

	return tx, nil
}

func (r *transactionRepository) ListByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*models.TransactionWithCategory, error) {
	query := `SELECT t.id, t.user_id, t.category_id, t.amount, t.description, t.transaction_date, t.attachment_url, t.metadata, t.created_at, t.updated_at, c.id, c.user_id, c.name, c.type, c.icon, c.color, c.is_system, c.created_at, c.updated_at FROM transactions t JOIN categories c ON t.category_id = c.id WHERE t.user_id = $1 ORDER BY t.transaction_date DESC LIMIT $2 OFFSET $3`

	rows, err := r.db.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list transactions: %w", err)
	}
	defer rows.Close()

	transactions := []*models.TransactionWithCategory{}

	for rows.Next() {
		twc := &models.TransactionWithCategory{}

		err := rows.Scan(&twc.Transaction.ID, &twc.Transaction.UserID, &twc.Transaction.CategoryID, &twc.Transaction.Amount, &twc.Transaction.Description, &twc.Transaction.TransactionDate, &twc.Transaction.AttachmentURL, &twc.Transaction.Metadata, &twc.Transaction.CreatedAt, &twc.Transaction.UpdatedAt, &twc.Category.ID, &twc.Category.UserID, &twc.Category.Name, &twc.Category.Type, &twc.Category.Icon, &twc.Category.Color, &twc.Category.IsSystem, &twc.Category.CreatedAt, &twc.Category.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan transaction: %w", err)
		}

		transactions = append(transactions, twc)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during iteration: %w", err)
	}

	return transactions, nil
}

func (r *transactionRepository) ListByDateRange(ctx context.Context, userID uuid.UUID, startDate, endDate time.Time) ([]*models.TransactionWithCategory, error) {
	query := `SELECT t.id, t.user_id, t.category_id, t.amount, t.description, t.transaction_date, t.attachment_url, t.metadata, t.created_at, t.updated_at, c.id, c.user_id, c.name, c.type, c.icon, c.color, c.is_system, c.created_at, c.updated_at FROM transactions t JOIN categories c ON t.category_id = c.id WHERE t.user_id = $1 AND t.transaction_date >= $2 AND t.transaction_date <= $3 ORDER BY t.transaction_date DESC`

	rows, err := r.db.Query(ctx, query, userID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to list transactions by date range: %w", err)
	}
	defer rows.Close()

	transactions := []*models.TransactionWithCategory{}

	for rows.Next() {
		twc := &models.TransactionWithCategory{}

		err := rows.Scan(&twc.Transaction.ID, &twc.Transaction.UserID, &twc.Transaction.CategoryID, &twc.Transaction.Amount, &twc.Transaction.Description, &twc.Transaction.TransactionDate, &twc.Transaction.AttachmentURL, &twc.Transaction.Metadata, &twc.Transaction.CreatedAt, &twc.Transaction.UpdatedAt, &twc.Category.ID, &twc.Category.UserID, &twc.Category.Name, &twc.Category.Type, &twc.Category.Icon, &twc.Category.Color, &twc.Category.IsSystem, &twc.Category.CreatedAt, &twc.Category.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan transaction: %w", err)
		}

		transactions = append(transactions, twc)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during iteration: %w", err)
	}

	return transactions, nil
}

func (r *transactionRepository) Update(ctx context.Context, tx *models.Transaction) error {
	query := `UPDATE transactions SET category_id = COALESCE($2, category_id), amount = COALESCE($3, amount), description = COALESCE($4, description), transaction_date = COALESCE($5, transaction_date), attachment_url = COALESCE($6, attachment_url), metadata = COALESCE($7, metadata), updated_at = $8 WHERE id = $1`

	_, err := r.db.Exec(ctx, query, tx.ID, tx.CategoryID, tx.Amount, tx.Description, tx.TransactionDate, tx.AttachmentURL, tx.Metadata, tx.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to update transaction: %w", err)
	}

	return nil
}

func (r *transactionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM transactions WHERE id = $1`

	_, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete transaction: %w", err)
	}

	return nil
}
