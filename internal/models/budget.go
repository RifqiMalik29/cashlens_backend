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
	CategoryID     uuid.UUID    `json:"category_id"`
	Amount         float64      `json:"amount"`
	PeriodType     BudgetPeriod `json:"period_type"`
	StartDate      time.Time    `json:"start_date"`
	EndDate        time.Time    `json:"end_date"`
	AlertThreshold *float64     `json:"alert_threshold,omitempty"`
}

type UpdateBudgetRequest struct {
	CategoryID     *uuid.UUID    `json:"category_id,omitempty"`
	Amount         *float64      `json:"amount,omitempty"`
	PeriodType     *BudgetPeriod `json:"period_type,omitempty"`
	StartDate      *time.Time    `json:"start_date,omitempty"`
	EndDate        *time.Time    `json:"end_date,omitempty"`
	AlertThreshold *float64      `json:"alert_threshold,omitempty"`
}
