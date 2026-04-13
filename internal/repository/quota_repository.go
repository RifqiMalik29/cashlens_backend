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
	IncrementTransactionsIfUnderLimit(ctx context.Context, userID uuid.UUID, month, year int, limit int) (bool, error)
	IncrementScansIfUnderLimit(ctx context.Context, userID uuid.UUID, month, year int, limit int) (bool, error)
}

type quotaRepository struct {
	pool *pgxpool.Pool
}

func NewQuotaRepository(pool *pgxpool.Pool) QuotaRepository {
	return &quotaRepository{pool: pool}
}

func (r *quotaRepository) GetOrCreate(ctx context.Context, userID uuid.UUID, month, year int) (*models.UserQuota, error) {
	var quota models.UserQuota

	// Atomically insert or ignore on conflict, then select — avoids TOCTOU race
	// between concurrent first-of-month requests from the same user.
	_, err := r.pool.Exec(ctx, `
		INSERT INTO user_quotas (id, user_id, period_month, period_year, scans_used, transactions_used)
		VALUES ($1, $2, $3, $4, 0, 0)
		ON CONFLICT (user_id, period_month, period_year) DO NOTHING
	`, uuid.New(), userID, month, year)
	if err != nil {
		return nil, fmt.Errorf("failed to upsert quota: %w", err)
	}

	err = r.pool.QueryRow(ctx, `
		SELECT id, user_id, period_month, period_year, scans_used, transactions_used
		FROM user_quotas
		WHERE user_id = $1 AND period_month = $2 AND period_year = $3
	`, userID, month, year).Scan(
		&quota.ID, &quota.UserID, &quota.PeriodMonth, &quota.PeriodYear,
		&quota.ScansUsed, &quota.TransactionsUsed,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get quota: %w", err)
	}

	return &quota, nil
}

// IncrementTransactionsIfUnderLimit atomically checks the limit and increments in one SQL statement
// Returns true if increment succeeded, false if limit was reached
func (r *quotaRepository) IncrementTransactionsIfUnderLimit(ctx context.Context, userID uuid.UUID, month, year int, limit int) (bool, error) {
	now := time.Now()

	result, err := r.pool.Exec(ctx, `
		INSERT INTO user_quotas (id, user_id, period_month, period_year, scans_used, transactions_used, created_at, updated_at)
		VALUES ($1, $2, $3, $4, 0, 1, $6, $7)
		ON CONFLICT (user_id, period_month, period_year)
		DO UPDATE SET
			transactions_used = user_quotas.transactions_used + 1,
			updated_at = $7
		WHERE user_quotas.transactions_used < $5
	`, uuid.New(), userID, month, year, limit, now, now)

	if err != nil {
		return false, fmt.Errorf("failed to increment transaction quota: %w", err)
	}

	return result.RowsAffected() > 0, nil
}

// IncrementScansIfUnderLimit atomically checks the limit and increments in one SQL statement
// Returns true if increment succeeded, false if limit was reached
func (r *quotaRepository) IncrementScansIfUnderLimit(ctx context.Context, userID uuid.UUID, month, year int, limit int) (bool, error) {
	now := time.Now()

	result, err := r.pool.Exec(ctx, `
		INSERT INTO user_quotas (id, user_id, period_month, period_year, scans_used, transactions_used, created_at, updated_at)
		VALUES ($1, $2, $3, $4, 1, 0, $6, $7)
		ON CONFLICT (user_id, period_month, period_year)
		DO UPDATE SET
			scans_used = user_quotas.scans_used + 1,
			updated_at = $7
		WHERE user_quotas.scans_used < $5
	`, uuid.New(), userID, month, year, limit, now, now)

	if err != nil {
		return false, fmt.Errorf("failed to increment scan quota: %w", err)
	}

	return result.RowsAffected() > 0, nil
}
