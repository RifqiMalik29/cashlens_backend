package models

import (
	"time"

	"github.com/google/uuid"
)

type Transaction struct {
	ID              uuid.UUID              `json:"id"`
	UserID          uuid.UUID              `json:"user_id"`
	CategoryID      uuid.UUID              `json:"category_id"`
	Amount          float64                `json:"amount"`
	Description     *string                `json:"description,omitempty"`
	TransactionDate time.Time              `json:"transaction_date"`
	AttachmentURL   *string                `json:"attachment_url,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
}

type CreateTransactionRequest struct {
	CategoryID      uuid.UUID `json:"category_id"`
	Amount          float64   `json:"amount"`
	Description     string    `json:"description,omitempty"`
	TransactionDate time.Time `json:"transaction_date"`
}

type UpdateTransactionRequest struct {
	CategoryID      *uuid.UUID `json:"category_id,omitempty"`
	Amount          *float64   `json:"amount,omitempty"`
	Description     *string    `json:"description,omitempty"`
	TransactionDate *time.Time `json:"transaction_date,omitempty"`
}

type TransactionWithCategory struct {
	Transaction
	Category Category `json:"category"`
}
