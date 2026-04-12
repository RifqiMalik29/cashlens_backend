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
	Name  string       `json:"name" validate:"required,max=100"`
	Type  CategoryType `json:"type" validate:"required,oneof=income expense"`
	Icon  string       `json:"icon,omitempty" validate:"max=50"`
	Color string       `json:"color,omitempty" validate:"max=20"`
}

type UpdateCategoryRequest struct {
	Name  *string       `json:"name,omitempty" validate:"omitempty,max=100"`
	Type  *CategoryType `json:"type,omitempty" validate:"omitempty,oneof=income expense"`
	Icon  *string       `json:"icon,omitempty" validate:"omitempty,max=50"`
	Color *string       `json:"color,omitempty" validate:"omitempty,max=20"`
}
