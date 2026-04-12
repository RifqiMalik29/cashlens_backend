package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rifqimalik/cashlens-backend/internal/models"
)

type DraftRepository interface {
	Create(ctx context.Context, draft *models.DraftTransaction) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.DraftTransaction, error)
	ListByUserID(ctx context.Context, userID uuid.UUID, status models.DraftStatus) ([]*models.DraftTransaction, error)
	Update(ctx context.Context, draft *models.DraftTransaction) error
	Delete(ctx context.Context, id uuid.UUID) error
	Confirm(ctx context.Context, draftID uuid.UUID, txID uuid.UUID) error
}

type draftRepository struct {
	db *pgxpool.Pool
}

func NewDraftRepository(db *pgxpool.Pool) DraftRepository {
	return &draftRepository{db: db}
}

func (r *draftRepository) Create(ctx context.Context, draft *models.DraftTransaction) error {
	query := `INSERT INTO draft_transactions (id, user_id, category_id, amount, description, transaction_date, source, raw_data, status, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`

	_, err := r.db.Exec(ctx, query, draft.ID, draft.UserID, draft.CategoryID, draft.Amount, draft.Description, draft.TransactionDate, draft.Source, draft.RawData, draft.Status, draft.CreatedAt, draft.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create draft: %w", err)
	}

	return nil
}

func (r *draftRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.DraftTransaction, error) {
	draft := &models.DraftTransaction{}
	query := `SELECT id, user_id, category_id, amount, description, transaction_date, source, raw_data, status, confirmed_transaction_id, created_at, updated_at FROM draft_transactions WHERE id = $1`

	err := r.db.QueryRow(ctx, query, id).Scan(&draft.ID, &draft.UserID, &draft.CategoryID, &draft.Amount, &draft.Description, &draft.TransactionDate, &draft.Source, &draft.RawData, &draft.Status, &draft.ConfirmedTransactionID, &draft.CreatedAt, &draft.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to get draft: %w", err)
	}

	return draft, nil
}

func (r *draftRepository) ListByUserID(ctx context.Context, userID uuid.UUID, status models.DraftStatus) ([]*models.DraftTransaction, error) {
	query := `SELECT id, user_id, category_id, amount, description, transaction_date, source, raw_data, status, confirmed_transaction_id, created_at, updated_at FROM draft_transactions WHERE user_id = $1 AND status = $2 ORDER BY created_at DESC`

	rows, err := r.db.Query(ctx, query, userID, status)
	if err != nil {
		return nil, fmt.Errorf("failed to list drafts: %w", err)
	}
	defer rows.Close()

	drafts := []*models.DraftTransaction{}

	for rows.Next() {
		d := &models.DraftTransaction{}

		err := rows.Scan(&d.ID, &d.UserID, &d.CategoryID, &d.Amount, &d.Description, &d.TransactionDate, &d.Source, &d.RawData, &d.Status, &d.ConfirmedTransactionID, &d.CreatedAt, &d.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan draft: %w", err)
		}

		drafts = append(drafts, d)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during iteration: %w", err)
	}

	return drafts, nil
}

func (r *draftRepository) Update(ctx context.Context, draft *models.DraftTransaction) error {
	query := `UPDATE draft_transactions SET category_id = COALESCE($2, category_id), amount = COALESCE($3, amount), description = COALESCE($4, description), transaction_date = COALESCE($5, transaction_date), status = COALESCE($6, status), updated_at = $7 WHERE id = $1`

	_, err := r.db.Exec(ctx, query, draft.ID, draft.CategoryID, draft.Amount, draft.Description, draft.TransactionDate, draft.Status, draft.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to update draft: %w", err)
	}

	return nil
}

func (r *draftRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM draft_transactions WHERE id = $1`

	_, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete draft: %w", err)
	}

	return nil
}

func (r *draftRepository) Confirm(ctx context.Context, draftID uuid.UUID, txID uuid.UUID) error {
	query := `UPDATE draft_transactions SET status = 'confirmed', confirmed_transaction_id = $1, updated_at = NOW() WHERE id = $2`

	_, err := r.db.Exec(ctx, query, txID, draftID)
	if err != nil {
		return fmt.Errorf("failed to confirm draft: %w", err)
	}

	return nil
}
