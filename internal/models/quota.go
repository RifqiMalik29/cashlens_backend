package models

import (
	"time"

	"github.com/google/uuid"
)

const (
	FreeTierTransactionLimit = 50
	FreeTierScanLimit        = 5
)

type UserQuota struct {
	ID               uuid.UUID `json:"id"`
	UserID           uuid.UUID `json:"user_id"`
	PeriodMonth      int       `json:"period_month"`
	PeriodYear       int       `json:"period_year"`
	TransactionsUsed int       `json:"transactions_used"`
	ScansUsed        int       `json:"scans_used"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type QuotaStatus struct {
	TransactionsUsed  int  `json:"transactions_used"`
	TransactionsLimit int  `json:"transactions_limit"`
	ScansUsed         int  `json:"scans_used"`
	ScansLimit        int  `json:"scans_limit"`
	IsTransactionsFull bool `json:"is_transactions_full"`
	IsScansFull        bool `json:"is_scans_full"`
}
