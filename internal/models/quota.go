package models

import "github.com/google/uuid"

type UserQuota struct {
	ID                 uuid.UUID `json:"id"`
	UserID             uuid.UUID `json:"user_id"`
	PeriodMonth        int       `json:"period_month"`
	PeriodYear         int       `json:"period_year"`
	ScansUsed          int       `json:"scans_used"`
	TransactionsUsed   int       `json:"transactions_used"`
}

type QuotaLimits struct {
	MaxTransactionsPerMonth int
	MaxScansPerMonth        int
}

var FreeTierLimits = QuotaLimits{
	MaxTransactionsPerMonth: 50,
	MaxScansPerMonth:        5,
}

var PremiumTierLimits = QuotaLimits{
	MaxTransactionsPerMonth: -1, // Unlimited
	MaxScansPerMonth:        -1, // Unlimited
}
