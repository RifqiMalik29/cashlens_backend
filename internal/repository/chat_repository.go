package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rifqimalik/cashlens-backend/internal/models"
)

type ChatLinkRepository interface {
	Create(ctx context.Context, link *models.UserChatLink) error
	GetByChatID(ctx context.Context, chatID string, platform string) (*models.UserChatLink, error)
	GetByUserID(ctx context.Context, userID uuid.UUID, platform string) (*models.UserChatLink, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type chatLinkRepository struct {
	db *pgxpool.Pool
}

func NewChatLinkRepository(db *pgxpool.Pool) ChatLinkRepository {
	return &chatLinkRepository{db: db}
}

func (r *chatLinkRepository) Create(ctx context.Context, link *models.UserChatLink) error {
	query := `INSERT INTO user_chat_links (id, user_id, platform, chat_id, username, is_active, linked_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	_, err := r.db.Exec(ctx, query, link.ID, link.UserID, link.Platform, link.ChatID, link.Username, link.IsActive, link.LinkedAt, link.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create chat link: %w", err)
	}

	return nil
}

func (r *chatLinkRepository) GetByChatID(ctx context.Context, chatID string, platform string) (*models.UserChatLink, error) {
	link := &models.UserChatLink{}
	query := `SELECT id, user_id, platform, chat_id, username, is_active, linked_at, updated_at FROM user_chat_links WHERE chat_id = $1 AND platform = $2 AND is_active = TRUE`

	err := r.db.QueryRow(ctx, query, chatID, platform).Scan(&link.ID, &link.UserID, &link.Platform, &link.ChatID, &link.Username, &link.IsActive, &link.LinkedAt, &link.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to get chat link: %w", err)
	}

	return link, nil
}

func (r *chatLinkRepository) GetByUserID(ctx context.Context, userID uuid.UUID, platform string) (*models.UserChatLink, error) {
	link := &models.UserChatLink{}
	query := `SELECT id, user_id, platform, chat_id, username, is_active, linked_at, updated_at FROM user_chat_links WHERE user_id = $1 AND platform = $2 AND is_active = TRUE`

	err := r.db.QueryRow(ctx, query, userID, platform).Scan(&link.ID, &link.UserID, &link.Platform, &link.ChatID, &link.Username, &link.IsActive, &link.LinkedAt, &link.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to get chat link by user: %w", err)
	}

	return link, nil
}

func (r *chatLinkRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM user_chat_links WHERE id = $1`

	_, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete chat link: %w", err)
	}

	return nil
}
