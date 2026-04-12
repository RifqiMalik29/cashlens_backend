package models

import (
	"time"

	"github.com/google/uuid"
)

type CategoryType string

const (
	CategoryTypeIncome  CategoryType = "income"
	CategoryTypeExpense CategoryType = "expense"
)

type Category struct {
	ID        uuid.UUID     `json:"id"`
	UserID    *uuid.UUID    `json:"user_id,omitempty"` // nil for system categories
	Name      string        `json:"name"`
	Type      CategoryType  `json:"type"`
	Icon      *string       `json:"icon,omitempty"`
	Color     *string       `json:"color,omitempty"`
	IsSystem  bool          `json:"is_system"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
}

type CreateCategoryRequest struct {
	Name  string       `json:"name"`
	Type  CategoryType `json:"type"`
	Icon  string       `json:"icon,omitempty"`
	Color string       `json:"color,omitempty"`
}

type UpdateCategoryRequest struct {
	Name  *string       `json:"name,omitempty"`
	Type  *CategoryType `json:"type,omitempty"`
	Icon  *string       `json:"icon,omitempty"`
	Color *string       `json:"color,omitempty"`
}
