package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// WinBackRepository handles win-back campaign tracking
type WinBackRepository interface {
	GetUsersEligibleForWinBack(ctx context.Context, daysSinceExpiry int, limit int) ([]uuid.UUID, error)
	MarkWinBackSent(ctx context.Context, userID uuid.UUID) error
}

type winBackRepository struct {
	pool *pgxpool.Pool
}

func NewWinBackRepository(pool *pgxpool.Pool) WinBackRepository {
	return &winBackRepository{pool: pool}
}

// GetUsersEligibleForWinBack returns user IDs who:
// - Had premium subscription expire X days ago
// - Haven't resubscribed
// - Haven't been sent a win-back message yet
func (r *winBackRepository) GetUsersEligibleForWinBack(ctx context.Context, daysSinceExpiry int, limit int) ([]uuid.UUID, error) {
	cutoffDate := time.Now().AddDate(0, 0, -daysSinceExpiry)

	query := `
		SELECT id FROM users
		WHERE subscription_tier = 'premium'
		  AND subscription_expires_at < $1
		  AND subscription_expires_at >= $1 - INTERVAL '1 day'
		  AND win_back_sent_at IS NULL
		LIMIT $2
	`

	rows, err := r.pool.Query(ctx, query, cutoffDate, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query eligible users: %w", err)
	}
	defer rows.Close()

	var userIDs []uuid.UUID
	for rows.Next() {
		var userID uuid.UUID
		if err := rows.Scan(&userID); err != nil {
			return nil, fmt.Errorf("failed to scan user ID: %w", err)
		}
		userIDs = append(userIDs, userID)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return userIDs, nil
}

// MarkWinBackSent sets win_back_sent_at to now()
func (r *winBackRepository) MarkWinBackSent(ctx context.Context, userID uuid.UUID) error {
	query := `UPDATE users SET win_back_sent_at = NOW(), updated_at = NOW() WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to mark win-back sent: %w", err)
	}
	return nil
}
