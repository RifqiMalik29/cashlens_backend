package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rifqimalik/cashlens-backend/internal/models"
)

type QuotaRepository interface {
	GetOrCreate(ctx context.Context, userID uuid.UUID, month, year int) (*models.UserQuota, error)
	IncrementTransactions(ctx context.Context, userID uuid.UUID, month, year int) error
	IncrementScans(ctx context.Context, userID uuid.UUID, month, year int) error
}

type quotaRepository struct {
	db *pgxpool.Pool
}

func NewQuotaRepository(db *pgxpool.Pool) QuotaRepository {
	return &quotaRepository{db: db}
}

func (r *quotaRepository) GetOrCreate(ctx context.Context, userID uuid.UUID, month, year int) (*models.UserQuota, error) {
	query := `
		INSERT INTO user_quotas (id, user_id, period_month, period_year, transactions_used, scans_used, created_at, updated_at)
		VALUES (uuid_generate_v4(), $1, $2, $3, 0, 0, NOW(), NOW())
		ON CONFLICT (user_id, period_month, period_year) DO UPDATE SET updated_at = user_quotas.updated_at
		RETURNING id, user_id, period_month, period_year, transactions_used, scans_used, created_at, updated_at
	`
	q := &models.UserQuota{}
	err := r.db.QueryRow(ctx, query, userID, month, year).Scan(
		&q.ID, &q.UserID, &q.PeriodMonth, &q.PeriodYear,
		&q.TransactionsUsed, &q.ScansUsed, &q.CreatedAt, &q.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get or create quota: %w", err)
	}
	return q, nil
}

func (r *quotaRepository) IncrementTransactions(ctx context.Context, userID uuid.UUID, month, year int) error {
	query := `
		INSERT INTO user_quotas (id, user_id, period_month, period_year, transactions_used, scans_used, created_at, updated_at)
		VALUES (uuid_generate_v4(), $1, $2, $3, 1, 0, NOW(), NOW())
		ON CONFLICT (user_id, period_month, period_year) DO UPDATE
		SET transactions_used = user_quotas.transactions_used + 1, updated_at = NOW()
	`
	_, err := r.db.Exec(ctx, query, userID, month, year)
	if err != nil {
		return fmt.Errorf("failed to increment transaction quota: %w", err)
	}
	return nil
}

func (r *quotaRepository) IncrementScans(ctx context.Context, userID uuid.UUID, month, year int) error {
	query := `
		INSERT INTO user_quotas (id, user_id, period_month, period_year, transactions_used, scans_used, created_at, updated_at)
		VALUES (uuid_generate_v4(), $1, $2, $3, 0, 1, NOW(), NOW())
		ON CONFLICT (user_id, period_month, period_year) DO UPDATE
		SET scans_used = user_quotas.scans_used + 1, updated_at = NOW()
	`
	now := time.Now()
	_ = now
	_, err := r.db.Exec(ctx, query, userID, month, year)
	if err != nil {
		return fmt.Errorf("failed to increment scan quota: %w", err)
	}
	return nil
}
