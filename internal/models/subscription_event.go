package models

import (
	"time"

	"github.com/google/uuid"
)

// SubscriptionEvent tracks subscription lifecycle events
type SubscriptionEvent struct {
	ID                uuid.UUID `json:"id"`
	UserID            uuid.UUID `json:"user_id"`
	EventType         string    `json:"event_type"`
	Plan              *string   `json:"plan,omitempty"`
	PricePaid         *float64  `json:"price_paid,omitempty"`
	ExternalInvoiceID *string   `json:"external_invoice_id,omitempty"`
	CancelReason      *string   `json:"cancel_reason,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
}


