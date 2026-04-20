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


