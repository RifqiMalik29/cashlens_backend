package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rifqimalik/cashlens-backend/internal/models"
)

type SubscriptionEventRepository interface {
	Create(ctx context.Context, event *models.SubscriptionEvent) error
	ExistsByExternalInvoiceID(ctx context.Context, externalInvoiceID string) (bool, error)
}

type subscriptionEventRepository struct {
	pool *pgxpool.Pool
}

func NewSubscriptionEventRepository(pool *pgxpool.Pool) SubscriptionEventRepository {
	return &subscriptionEventRepository{pool: pool}
}

func (r *subscriptionEventRepository) Create(ctx context.Context, event *models.SubscriptionEvent) error {
	query := `
		INSERT INTO subscription_events (id, user_id, event_type, plan, price_paid, external_invoice_id, cancel_reason, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := r.pool.Exec(ctx, query,
		event.ID, event.UserID, event.EventType, event.Plan,
		event.PricePaid, event.ExternalInvoiceID, event.CancelReason, time.Now(),
	)
	if err != nil {
		return fmt.Errorf("failed to create subscription event: %w", err)
	}
	return nil
}

func (r *subscriptionEventRepository) ExistsByExternalInvoiceID(ctx context.Context, externalInvoiceID string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM subscription_events WHERE external_invoice_id = $1 AND event_type = 'subscribed')
	`, externalInvoiceID).Scan(&exists)

	if err != nil {
		return false, fmt.Errorf("failed to check subscription event existence: %w", err)
	}
	return exists, nil
}

type PendingInvoiceRepository interface {
	Create(ctx context.Context, invoice *models.PendingInvoice) error
	GetByExternalInvoiceID(ctx context.Context, externalInvoiceID string) (*models.PendingInvoice, error)
	UpdateStatus(ctx context.Context, externalInvoiceID string, status string) error
	ExpireStale(ctx context.Context) (int64, error)
}

type pendingInvoiceRepository struct {
	pool *pgxpool.Pool
}

func NewPendingInvoiceRepository(pool *pgxpool.Pool) PendingInvoiceRepository {
	return &pendingInvoiceRepository{pool: pool}
}

func (r *pendingInvoiceRepository) Create(ctx context.Context, invoice *models.PendingInvoice) error {
	query := `
		INSERT INTO pending_invoices (id, user_id, external_invoice_id, xendit_invoice_id, plan, amount, status, expires_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	_, err := r.pool.Exec(ctx, query,
		invoice.ID, invoice.UserID, invoice.ExternalInvoiceID, invoice.XenditInvoiceID, invoice.Plan,
		invoice.Amount, invoice.Status, invoice.ExpiresAt, time.Now(), time.Now(),
	)
	if err != nil {
		return fmt.Errorf("failed to create pending invoice: %w", err)
	}
	return nil
}

func (r *pendingInvoiceRepository) GetByExternalInvoiceID(ctx context.Context, externalInvoiceID string) (*models.PendingInvoice, error) {
	invoice := &models.PendingInvoice{}
	query := `
		SELECT id, user_id, external_invoice_id, COALESCE(xendit_invoice_id, ''), plan, amount, status, expires_at, created_at, updated_at
		FROM pending_invoices WHERE external_invoice_id = $1
	`
	err := r.pool.QueryRow(ctx, query, externalInvoiceID).Scan(
		&invoice.ID, &invoice.UserID, &invoice.ExternalInvoiceID, &invoice.XenditInvoiceID, &invoice.Plan,
		&invoice.Amount, &invoice.Status, &invoice.ExpiresAt, &invoice.CreatedAt, &invoice.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("pending invoice not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get pending invoice: %w", err)
	}
	return invoice, nil
}

func (r *pendingInvoiceRepository) UpdateStatus(ctx context.Context, externalInvoiceID string, status string) error {
	query := `UPDATE pending_invoices SET status = $1, updated_at = $2 WHERE external_invoice_id = $3`
	_, err := r.pool.Exec(ctx, query, status, time.Now(), externalInvoiceID)
	if err != nil {
		return fmt.Errorf("failed to update pending invoice status: %w", err)
	}
	return nil
}

func (r *pendingInvoiceRepository) ExpireStale(ctx context.Context) (int64, error) {
	result, err := r.pool.Exec(ctx, `
		UPDATE pending_invoices SET status = 'expired', updated_at = NOW()
		WHERE status = 'pending' AND expires_at < NOW()
	`)
	if err != nil {
		return 0, fmt.Errorf("failed to expire stale invoices: %w", err)
	}
	return result.RowsAffected(), nil
}
