package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rifqimalik/cashlens-backend/internal/models"
)

type RefreshTokenRepository interface {
	Create(ctx context.Context, token *models.RefreshToken) error
	GetByToken(ctx context.Context, token string) (*models.RefreshToken, error)
	GetByUserID(ctx context.Context, userID uuid.UUID, activeOnly bool) ([]*models.RefreshToken, error)
	Revoke(ctx context.Context, id uuid.UUID) error
	RevokeByUserID(ctx context.Context, userID uuid.UUID) error
	MarkReplaced(ctx context.Context, oldTokenID, newTokenID uuid.UUID) error
	DeleteExpired(ctx context.Context, before time.Time) (int64, error)
}

type refreshTokenRepository struct {
	db *pgxpool.Pool
}

func NewRefreshTokenRepository(db *pgxpool.Pool) RefreshTokenRepository {
	return &refreshTokenRepository{db: db}
}

func (r *refreshTokenRepository) Create(ctx context.Context, token *models.RefreshToken) error {
	query := `
		INSERT INTO refresh_tokens (id, user_id, token, expires_at, ip_address, user_agent, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := r.db.Exec(ctx, query,
		token.ID, token.UserID, token.Token, token.ExpiresAt,
		token.IPAddress, token.UserAgent, token.CreatedAt, token.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create refresh token: %w", err)
	}
	return nil
}

func (r *refreshTokenRepository) GetByToken(ctx context.Context, token string) (*models.RefreshToken, error) {
	rt := &models.RefreshToken{}
	query := `
		SELECT id, user_id, token, expires_at, revoked_at, replaced_by_token_id, 
		       ip_address, user_agent, created_at, updated_at
		FROM refresh_tokens
		WHERE token = $1
	`
	err := r.db.QueryRow(ctx, query, token).Scan(
		&rt.ID, &rt.UserID, &rt.Token, &rt.ExpiresAt,
		&rt.RevokedAt, &rt.ReplacedByTokenID,
		&rt.IPAddress, &rt.UserAgent, &rt.CreatedAt, &rt.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get refresh token: %w", err)
	}
	return rt, nil
}

func (r *refreshTokenRepository) GetByUserID(ctx context.Context, userID uuid.UUID, activeOnly bool) ([]*models.RefreshToken, error) {
	var tokens []*models.RefreshToken
	
	query := `
		SELECT id, user_id, token, expires_at, revoked_at, replaced_by_token_id,
		       ip_address, user_agent, created_at, updated_at
		FROM refresh_tokens
		WHERE user_id = $1
	`
	
	if activeOnly {
		query += " AND revoked_at IS NULL AND expires_at > NOW()"
	}
	
	query += " ORDER BY created_at DESC"
	
	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query refresh tokens: %w", err)
	}
	defer rows.Close()
	
	for rows.Next() {
		rt := &models.RefreshToken{}
		err := rows.Scan(
			&rt.ID, &rt.UserID, &rt.Token, &rt.ExpiresAt,
			&rt.RevokedAt, &rt.ReplacedByTokenID,
			&rt.IPAddress, &rt.UserAgent, &rt.CreatedAt, &rt.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan refresh token: %w", err)
		}
		tokens = append(tokens, rt)
	}
	
	return tokens, nil
}

func (r *refreshTokenRepository) Revoke(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE refresh_tokens
		SET revoked_at = NOW(), updated_at = NOW()
		WHERE id = $1
	`
	_, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to revoke token: %w", err)
	}
	return nil
}

func (r *refreshTokenRepository) RevokeByUserID(ctx context.Context, userID uuid.UUID) error {
	query := `
		UPDATE refresh_tokens
		SET revoked_at = NOW(), updated_at = NOW()
		WHERE user_id = $1 AND revoked_at IS NULL
	`
	_, err := r.db.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to revoke user tokens: %w", err)
	}
	return nil
}

func (r *refreshTokenRepository) MarkReplaced(ctx context.Context, oldTokenID, newTokenID uuid.UUID) error {
	query := `
		UPDATE refresh_tokens
		SET replaced_by_token_id = $1, updated_at = NOW()
		WHERE id = $2
	`
	_, err := r.db.Exec(ctx, query, newTokenID, oldTokenID)
	if err != nil {
		return fmt.Errorf("failed to mark token as replaced: %w", err)
	}
	return nil
}

func (r *refreshTokenRepository) DeleteExpired(ctx context.Context, before time.Time) (int64, error) {
	query := `
		DELETE FROM refresh_tokens
		WHERE expires_at < $1 OR (revoked_at IS NOT NULL AND revoked_at < $1)
	`
	tag, err := r.db.Exec(ctx, query, before)
	if err != nil {
		return 0, fmt.Errorf("failed to delete expired tokens: %w", err)
	}
	return tag.RowsAffected(), nil
}
