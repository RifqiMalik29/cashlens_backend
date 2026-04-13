package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// WinBackUser holds the minimal user data needed for a win-back campaign.
type WinBackUser struct {
	ID       uuid.UUID
	Language string
}

// WinBackRepository handles win-back campaign tracking
type WinBackRepository interface {
	GetUsersEligibleForWinBack(ctx context.Context, daysSinceExpiry int, limit int) ([]WinBackUser, error)
	MarkWinBackSent(ctx context.Context, userID uuid.UUID) error
}

type winBackRepository struct {
	pool *pgxpool.Pool
}

func NewWinBackRepository(pool *pgxpool.Pool) WinBackRepository {
	return &winBackRepository{pool: pool}
}

// GetUsersEligibleForWinBack returns users who:
// - Had premium subscription expire X days ago
// - Haven't resubscribed
// - Haven't been sent a win-back message yet
func (r *winBackRepository) GetUsersEligibleForWinBack(ctx context.Context, daysSinceExpiry int, limit int) ([]WinBackUser, error) {
	cutoffDate := time.Now().AddDate(0, 0, -daysSinceExpiry)

	query := `
		SELECT id, language FROM users
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

	var users []WinBackUser
	for rows.Next() {
		var u WinBackUser
		if err := rows.Scan(&u.ID, &u.Language); err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, u)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return users, nil
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
