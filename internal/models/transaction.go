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
	CategoryID      uuid.UUID `json:"category_id" validate:"required,uuid"`
	Amount          float64   `json:"amount" validate:"required,gt=0"`
	Description     string    `json:"description,omitempty" validate:"max=500"`
	TransactionDate time.Time `json:"transaction_date" validate:"required"`
}

type UpdateTransactionRequest struct {
	CategoryID      *uuid.UUID `json:"category_id,omitempty" validate:"omitempty,uuid"`
	Amount          *float64   `json:"amount,omitempty" validate:"omitempty,gt=0"`
	Description     *string    `json:"description,omitempty" validate:"omitempty,max=500"`
	TransactionDate *time.Time `json:"transaction_date,omitempty" validate:"omitempty"`
}

type TransactionWithCategory struct {
	Transaction
	Category Category `json:"category"`
}

type TransactionSummary struct {
	TotalIncome   float64            `json:"total_income"`
	TotalExpense  float64            `json:"total_expense"`
	NetBalance    float64            `json:"net_balance"`
	ByCategory    []CategorySummary  `json:"by_category"`
}

type CategorySummary struct {
	CategoryID   string  `json:"category_id"`
	CategoryName string  `json:"category_name"`
	Type         string  `json:"type"`
	Total        float64 `json:"total"`
}
