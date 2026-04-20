package models

type SubscriptionStatus struct {
	Tier      string       `json:"tier"`
	ExpiresAt *string      `json:"expires_at,omitempty"`
	Quota     *QuotaStatus `json:"quota,omitempty"`
}

type QuotaStatus struct {
	TransactionsUsed  int `json:"transactions_used"`
	TransactionsLimit int `json:"transactions_limit"`
	ScansUsed         int `json:"scans_used"`
	ScansLimit        int `json:"scans_limit"`
}


