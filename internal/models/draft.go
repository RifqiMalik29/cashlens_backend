package models

import (
	"time"

	"github.com/google/uuid"
)

type DraftSource string
type DraftStatus string

const (
	DraftSourceTelegram DraftSource = "telegram"
	DraftSourceWhatsApp DraftSource = "whatsapp"
	DraftSourceReceipt  DraftSource = "receipt_scan"
	DraftSourceManual   DraftSource = "manual"

	DraftStatusPending   DraftStatus = "pending"
	DraftStatusConfirmed DraftStatus = "confirmed"
	DraftStatusRejected  DraftStatus = "rejected"
)

type DraftTransaction struct {
	ID                     uuid.UUID      `json:"id"`
	UserID                 uuid.UUID      `json:"user_id"`
	CategoryID             *uuid.UUID     `json:"category_id,omitempty"`
	Amount                 *float64       `json:"amount,omitempty"`
	Currency               string         `json:"currency"`
	Description            *string        `json:"description,omitempty"`
	TransactionDate        *time.Time     `json:"transaction_date,omitempty"`
	Source                 DraftSource    `json:"source"`
	RawData                map[string]any `json:"raw_data,omitempty"`
	Status                 DraftStatus    `json:"status"`
	ConfirmedTransactionID *uuid.UUID     `json:"confirmed_transaction_id,omitempty"`
	CreatedAt              time.Time      `json:"created_at"`
	UpdatedAt              time.Time      `json:"updated_at"`
	Category               *Category      `json:"category,omitempty"`
}

type CreateDraftRequest struct {
	CategoryID      *uuid.UUID     `json:"category_id,omitempty"`
	Amount          *float64       `json:"amount,omitempty"`
	Currency        string         `json:"currency,omitempty"`
	Description     *string        `json:"description,omitempty"`
	TransactionDate *time.Time     `json:"transaction_date,omitempty"`
	Source          DraftSource    `json:"source"`
	RawData         map[string]any `json:"raw_data,omitempty"`
}

type ConfirmDraftRequest struct {
	CategoryID      uuid.UUID `json:"category_id"`
	Amount          float64   `json:"amount"`
	Currency        string    `json:"currency,omitempty"`
	Description     string    `json:"description,omitempty"`
	TransactionDate time.Time `json:"transaction_date"`
}

type UserChatLink struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	Platform  string    `json:"platform"`
	ChatID    string    `json:"chat_id"`
	Username  *string   `json:"username,omitempty"`
	IsActive  bool      `json:"is_active"`
	LinkedAt  time.Time `json:"linked_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
