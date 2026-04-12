package models

import (
	"time"

	"github.com/google/uuid"
)

type BudgetPeriod string

const (
	BudgetPeriodWeekly  BudgetPeriod = "weekly"
	BudgetPeriodMonthly BudgetPeriod = "monthly"
	BudgetPeriodYearly  BudgetPeriod = "yearly"
)

type Budget struct {
	ID              uuid.UUID      `json:"id"`
	UserID          uuid.UUID      `json:"user_id"`
	CategoryID      uuid.UUID      `json:"category_id"`
	Amount          float64        `json:"amount"`
	PeriodType      BudgetPeriod   `json:"period_type"`
	StartDate       time.Time      `json:"start_date"`
	EndDate         time.Time      `json:"end_date"`
	AlertThreshold  *float64       `json:"alert_threshold,omitempty"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	Category        *Category      `json:"category,omitempty"`
	SpentAmount     *float64       `json:"spent_amount,omitempty"`
	PercentageUsed  *float64       `json:"percentage_used,omitempty"`
}

type CreateBudgetRequest struct {
	CategoryID     uuid.UUID    `json:"category_id" validate:"required,uuid"`
	Amount         float64      `json:"amount" validate:"required,gt=0"`
	PeriodType     BudgetPeriod `json:"period_type" validate:"required,oneof=weekly monthly yearly"`
	StartDate      time.Time    `json:"start_date" validate:"required"`
	EndDate        time.Time    `json:"end_date" validate:"required"`
	AlertThreshold *float64     `json:"alert_threshold,omitempty" validate:"omitempty,gte=0,lte=100"`
}

type UpdateBudgetRequest struct {
	CategoryID     *uuid.UUID    `json:"category_id,omitempty" validate:"omitempty,uuid"`
	Amount         *float64      `json:"amount,omitempty" validate:"omitempty,gt=0"`
	PeriodType     *BudgetPeriod `json:"period_type,omitempty" validate:"omitempty,oneof=weekly monthly yearly"`
	StartDate      *time.Time    `json:"start_date,omitempty" validate:"omitempty"`
	EndDate        *time.Time    `json:"end_date,omitempty" validate:"omitempty"`
	AlertThreshold *float64      `json:"alert_threshold,omitempty" validate:"omitempty,gte=0,lte=100"`
}
