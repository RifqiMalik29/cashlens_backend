package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ReminderUser holds the minimal user data needed for expiry reminders.
type ReminderUser struct {
	ID            uuid.UUID
	Language      string
	ExpoPushToken string
}

type ReminderRepository interface {
	GetUsersEligibleForReminder(ctx context.Context, daysBeforeExpiry int) ([]ReminderUser, error)
	MarkReminderSent(ctx context.Context, userID uuid.UUID, daysBeforeExpiry int) error
}

type reminderRepository struct {
	pool *pgxpool.Pool
}

func NewReminderRepository(pool *pgxpool.Pool) ReminderRepository {
	return &reminderRepository{pool: pool}
}

// GetUsersEligibleForReminder returns premium users whose subscription expires
// in exactly N days (within a 1-day window) and haven't been sent this reminder yet.
func (r *reminderRepository) GetUsersEligibleForReminder(ctx context.Context, daysBeforeExpiry int) ([]ReminderUser, error) {
	var sentAtColumn string
	switch daysBeforeExpiry {
	case 3:
		sentAtColumn = "reminder_3d_sent_at"
	case 1:
		sentAtColumn = "reminder_1d_sent_at"
	default:
		return nil, fmt.Errorf("unsupported daysBeforeExpiry: %d (must be 1 or 3)", daysBeforeExpiry)
	}

	query := fmt.Sprintf(`
		SELECT id, language, COALESCE(expo_push_token, '')
		FROM users
		WHERE subscription_tier = 'premium'
		  AND subscription_expires_at >= NOW() + ($1 || ' days')::INTERVAL
		  AND subscription_expires_at <  NOW() + ($2 || ' days')::INTERVAL
		  AND %s IS NULL
	`, sentAtColumn)

	rows, err := r.pool.Query(ctx, query, daysBeforeExpiry, daysBeforeExpiry+1)
	if err != nil {
		return nil, fmt.Errorf("failed to query eligible users: %w", err)
	}
	defer rows.Close()

	var users []ReminderUser
	for rows.Next() {
		var u ReminderUser
		if err := rows.Scan(&u.ID, &u.Language, &u.ExpoPushToken); err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}
	return users, nil
}

// MarkReminderSent sets reminder_3d_sent_at or reminder_1d_sent_at to NOW().
func (r *reminderRepository) MarkReminderSent(ctx context.Context, userID uuid.UUID, daysBeforeExpiry int) error {
	var query string
	switch daysBeforeExpiry {
	case 3:
		query = `UPDATE users SET reminder_3d_sent_at = NOW(), updated_at = NOW() WHERE id = $1`
	case 1:
		query = `UPDATE users SET reminder_1d_sent_at = NOW(), updated_at = NOW() WHERE id = $1`
	default:
		return fmt.Errorf("unsupported daysBeforeExpiry: %d (must be 1 or 3)", daysBeforeExpiry)
	}
	_, err := r.pool.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to mark reminder sent: %w", err)
	}
	return nil
}
