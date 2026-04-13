package models

import (
	"time"

	"github.com/google/uuid"
)

// SubscriptionEvent tracks subscription lifecycle events
type SubscriptionEvent struct {
	ID                uuid.UUID  `json:"id"`
	UserID            uuid.UUID  `json:"user_id"`
	EventType         string     `json:"event_type"`
	Plan              *string    `json:"plan,omitempty"`
	PricePaid         *float64   `json:"price_paid,omitempty"`
	ExternalInvoiceID *string    `json:"external_invoice_id,omitempty"`
	CancelReason      *string    `json:"cancel_reason,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
}

// PendingInvoice tracks invoices waiting for payment confirmation
type PendingInvoice struct {
	ID                uuid.UUID `json:"id"`
	UserID            uuid.UUID `json:"user_id"`
	ExternalInvoiceID string    `json:"external_invoice_id"`
	Plan              string    `json:"plan"`
	Amount            float64   `json:"amount"`
	Status            string    `json:"status"` // pending, paid, expired, failed
	ExpiresAt         time.Time `json:"expires_at"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// Xendit webhook payload (simplified)
type XenditWebhookPayload struct {
	ExternalInvoiceID string  `json:"external_invoice_id"`
	Status            string  `json:"status"` // PAID, EXPIRED, FAILED
	PaidAmount        float64 `json:"paid_amount"`
	PaidAt            string  `json:"paid_at,omitempty"`
}
