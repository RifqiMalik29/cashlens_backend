package repository

import (
	"context"
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

// TODO: Implement all methods (placeholder)
func (r *transactionRepository) Create(ctx context.Context, tx *models.Transaction) error {
	return nil
}

func (r *transactionRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Transaction, error) {
	return nil, nil
}

func (r *transactionRepository) ListByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*models.TransactionWithCategory, error) {
	return nil, nil
}

func (r *transactionRepository) ListByDateRange(ctx context.Context, userID uuid.UUID, startDate, endDate time.Time) ([]*models.TransactionWithCategory, error) {
	return nil, nil
}

func (r *transactionRepository) Update(ctx context.Context, tx *models.Transaction) error {
	return nil
}

func (r *transactionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return nil
}
