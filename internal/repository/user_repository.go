package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rifqimalik/cashlens-backend/internal/models"
)

type UserRepository interface {
	Create(ctx context.Context, user *models.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	GetByEmail(ctx context.Context, email string) (*models.User, error)
	GetByDeviceID(ctx context.Context, deviceID string) ([]models.User, error) // Added
	Update(ctx context.Context, user *models.User) error
	UpdateSubscription(ctx context.Context, userID uuid.UUID, tier string, expiresAt *time.Time) error
	UpdateFounder(ctx context.Context, userID uuid.UUID, isFounder bool) error
	UpdateLanguage(ctx context.Context, userID uuid.UUID, language string) error
	UpdatePushToken(ctx context.Context, userID uuid.UUID, token string) error
	GetByConfirmationToken(ctx context.Context, token string) (*models.User, error)
	UpdateConfirmationStatus(ctx context.Context, userID uuid.UUID, isConfirmed bool) error
	UpdateConfirmationToken(ctx context.Context, userID uuid.UUID, token string, expiresAt time.Time) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetExpiredTrialUsers(ctx context.Context) ([]*models.User, error)
}

type userRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(ctx context.Context, user *models.User) error {
	query := `
		INSERT INTO users (id, email, password_hash, name, language, is_confirmed, confirmation_token, confirmation_expires_at, expo_push_token, subscription_tier, subscription_expires_at, is_founder, device_id, trial_start_at, trial_end_at, trial_status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
	`
	_, err := r.db.Exec(ctx, query,
		user.ID, user.Email, user.PasswordHash, user.Name, user.Language,
		user.IsConfirmed, user.ConfirmationToken, user.ConfirmationExpiresAt,
		user.ExpoPushToken, user.SubscriptionTier, user.SubscriptionExpiry, user.IsFounder,
		user.DeviceID, user.TrialStartAt, user.TrialEndAt, user.TrialStatus,
		user.CreatedAt, user.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}
	return nil
}

func (r *userRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	user := &models.User{}
	query := `
		SELECT id, email, password_hash, name, language, COALESCE(expo_push_token, ''), subscription_tier, subscription_expires_at, is_founder, is_confirmed, confirmation_token, confirmation_expires_at, device_id, trial_start_at, trial_end_at, trial_status, created_at, updated_at
		FROM users WHERE id = $1
	`
	err := r.db.QueryRow(ctx, query, id).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.Name, &user.Language, &user.ExpoPushToken,
		&user.SubscriptionTier, &user.SubscriptionExpiry, &user.IsFounder,
		&user.IsConfirmed, &user.ConfirmationToken, &user.ConfirmationExpiresAt,
		&user.DeviceID, &user.TrialStartAt, &user.TrialEndAt, &user.TrialStatus,
		&user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return user, nil
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	user := &models.User{}
	query := `
		SELECT id, email, password_hash, name, language, COALESCE(expo_push_token, ''), subscription_tier, subscription_expires_at, is_founder, is_confirmed, confirmation_token, confirmation_expires_at, device_id, trial_start_at, trial_end_at, trial_status, created_at, updated_at
		FROM users WHERE email = $1
	`
	err := r.db.QueryRow(ctx, query, email).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.Name, &user.Language, &user.ExpoPushToken,
		&user.SubscriptionTier, &user.SubscriptionExpiry, &user.IsFounder,
		&user.IsConfirmed, &user.ConfirmationToken, &user.ConfirmationExpiresAt,
		&user.DeviceID, &user.TrialStartAt, &user.TrialEndAt, &user.TrialStatus,
		&user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}
	return user, nil
}

// GetByDeviceID retrieves users by their device ID.
func (r *userRepository) GetByDeviceID(ctx context.Context, deviceID string) ([]models.User, error) {
	var users []models.User
	query := `
		SELECT id, email, password_hash, name, language, COALESCE(expo_push_token, ''), subscription_tier, subscription_expires_at, is_founder, is_confirmed, confirmation_token, confirmation_expires_at, device_id, trial_start_at, trial_end_at, trial_status, created_at, updated_at
		FROM users WHERE device_id = $1
	`
	rows, err := r.db.Query(ctx, query, deviceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get users by device ID: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		user := models.User{}
		err := rows.Scan(
			&user.ID, &user.Email, &user.PasswordHash, &user.Name, &user.Language, &user.ExpoPushToken,
			&user.SubscriptionTier, &user.SubscriptionExpiry, &user.IsFounder,
			&user.IsConfirmed, &user.ConfirmationToken, &user.ConfirmationExpiresAt,
			&user.DeviceID, &user.TrialStartAt, &user.TrialEndAt, &user.TrialStatus,
			&user.CreatedAt, &user.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user by device ID: %w", err)
		}
		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error after scanning users by device ID: %w", err)
	}

	return users, nil
}

func (r *userRepository) Update(ctx context.Context, user *models.User) error {
	query := `
		UPDATE users
		SET email = $2, name = $3, language = $4, expo_push_token = $5, subscription_tier = $6, subscription_expires_at = $7, is_founder = $8, is_confirmed = $9, confirmation_token = $10, confirmation_expires_at = $11, device_id = $12, trial_start_at = $13, trial_end_at = $14, trial_status = $15, updated_at = NOW()
		WHERE id = $1
	`
	_, err := r.db.Exec(ctx, query,
		user.ID, user.Email, user.Name, user.Language, user.ExpoPushToken,
		user.SubscriptionTier, user.SubscriptionExpiry, user.IsFounder,
		user.IsConfirmed, user.ConfirmationToken, user.ConfirmationExpiresAt,
		user.DeviceID, user.TrialStartAt, user.TrialEndAt, user.TrialStatus,
	)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}
	return nil
}

func (r *userRepository) UpdateFounder(ctx context.Context, userID uuid.UUID, isFounder bool) error {
	query := `UPDATE users SET is_founder = $2, updated_at = NOW() WHERE id = $1`
	_, err := r.db.Exec(ctx, query, userID, isFounder)
	if err != nil {
		return fmt.Errorf("failed to update founder flag: %w", err)
	}
	return nil
}

func (r *userRepository) UpdateLanguage(ctx context.Context, userID uuid.UUID, language string) error {
	query := `UPDATE users SET language = $2, updated_at = NOW() WHERE id = $1`
	_, err := r.db.Exec(ctx, query, userID, language)
	if err != nil {
		return fmt.Errorf("failed to update language: %w", err)
	}
	return nil
}

func (r *userRepository) UpdatePushToken(ctx context.Context, userID uuid.UUID, token string) error {
	var query string
	var err error
	if token == "" {
		query = `UPDATE users SET expo_push_token = NULL, updated_at = NOW() WHERE id = $1`
		_, err = r.db.Exec(ctx, query, userID)
	} else {
		query = `UPDATE users SET expo_push_token = $2, updated_at = NOW() WHERE id = $1`
		_, err = r.db.Exec(ctx, query, userID, token)
	}
	if err != nil {
		return fmt.Errorf("failed to update push token: %w", err)
	}
	return nil
}

func (r *userRepository) UpdateSubscription(ctx context.Context, userID uuid.UUID, tier string, expiresAt *time.Time) error {
	query := `
		UPDATE users
		SET subscription_tier = $2, subscription_expires_at = $3, updated_at = NOW()
		WHERE id = $1
	`
	_, err := r.db.Exec(ctx, query, userID, tier, expiresAt)
	if err != nil {
		return fmt.Errorf("failed to update subscription: %w", err)
	}
	return nil
}

func (r *userRepository) GetByConfirmationToken(ctx context.Context, token string) (*models.User, error) {
	user := &models.User{}
	query := `
		SELECT id, email, password_hash, name, language, COALESCE(expo_push_token, ''), subscription_tier, subscription_expires_at, is_founder, is_confirmed, confirmation_token, confirmation_expires_at, device_id, trial_start_at, trial_end_at, trial_status, created_at, updated_at
		FROM users WHERE confirmation_token = $1
	`
	err := r.db.QueryRow(ctx, query, token).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.Name, &user.Language, &user.ExpoPushToken,
		&user.SubscriptionTier, &user.SubscriptionExpiry, &user.IsFounder,
		&user.IsConfirmed, &user.ConfirmationToken, &user.ConfirmationExpiresAt,
		&user.DeviceID, &user.TrialStartAt, &user.TrialEndAt, &user.TrialStatus,
		&user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user by token: %w", err)
	}
	return user, nil
}

func (r *userRepository) UpdateConfirmationStatus(ctx context.Context, userID uuid.UUID, isConfirmed bool) error {
	query := `
		UPDATE users
		SET is_confirmed = $2, confirmation_token = NULL, confirmation_expires_at = NULL, updated_at = NOW()
		WHERE id = $1
	`
	_, err := r.db.Exec(ctx, query, userID, isConfirmed)
	if err != nil {
		return fmt.Errorf("failed to update confirmation status: %w", err)
	}
	return nil
}

func (r *userRepository) UpdateConfirmationToken(ctx context.Context, userID uuid.UUID, token string, expiresAt time.Time) error {
	query := `
		UPDATE users
		SET confirmation_token = $2, confirmation_expires_at = $3, updated_at = NOW()
		WHERE id = $1
	`
	_, err := r.db.Exec(ctx, query, userID, token, expiresAt)
	if err != nil {
		return fmt.Errorf("failed to update confirmation token: %w", err)
	}
	return nil
}

func (r *userRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM users WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	return nil
}

func (r *userRepository) GetExpiredTrialUsers(ctx context.Context) ([]*models.User, error) {
	query := `
		SELECT id, email, password_hash, name, language, COALESCE(expo_push_token, ''), subscription_tier, subscription_expires_at, is_founder, is_confirmed, confirmation_token, confirmation_expires_at, device_id, trial_start_at, trial_end_at, trial_status, created_at, updated_at
		FROM users
		WHERE trial_status = 'active' AND trial_end_at < NOW()
	`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get expired trial users: %w", err)
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		user := &models.User{}
		err := rows.Scan(
			&user.ID, &user.Email, &user.PasswordHash, &user.Name, &user.Language, &user.ExpoPushToken,
			&user.SubscriptionTier, &user.SubscriptionExpiry, &user.IsFounder,
			&user.IsConfirmed, &user.ConfirmationToken, &user.ConfirmationExpiresAt,
			&user.DeviceID, &user.TrialStartAt, &user.TrialEndAt, &user.TrialStatus,
			&user.CreatedAt, &user.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan expired trial user: %w", err)
		}
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error scanning expired trial users: %w", err)
	}
	return users, nil
}

func (r *userRepository) FindUnconfirmedUsersOlderThan(ctx context.Context, duration time.Duration) ([]models.User, error) {
	var users []models.User
	threshold := time.Now().Add(-duration)
	query := `
		SELECT id, email, password_hash, name, language, COALESCE(expo_push_token, ''), subscription_tier, subscription_expires_at, is_founder, is_confirmed, confirmation_token, confirmation_expires_at, device_id, trial_start_at, trial_end_at, trial_status, created_at, updated_at
		FROM users WHERE is_confirmed = false AND created_at < $1
	`
	rows, err := r.db.Query(ctx, query, threshold)
	if err != nil {
		return nil, fmt.Errorf("failed to find unconfirmed users: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		user := models.User{}
		err := rows.Scan(
			&user.ID, &user.Email, &user.PasswordHash, &user.Name, &user.Language, &user.ExpoPushToken,
			&user.SubscriptionTier, &user.SubscriptionExpiry, &user.IsFounder,
			&user.IsConfirmed, &user.ConfirmationToken, &user.ConfirmationExpiresAt,
			&user.DeviceID, &user.TrialStartAt, &user.TrialEndAt, &user.TrialStatus,
			&user.CreatedAt, &user.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan unconfirmed user: %w", err)
		}
		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error after scanning unconfirmed users: %w", err)
	}

	return users, nil
}
