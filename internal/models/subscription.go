package models

type SubscriptionStatus struct {
	Tier      string        `json:"tier"`
	ExpiresAt *string       `json:"expires_at,omitempty"`
	Quota     *QuotaStatus  `json:"quota,omitempty"`
}

type QuotaStatus struct {
	TransactionsUsed int `json:"transactions_used"`
	TransactionsLimit int `json:"transactions_limit"`
	ScansUsed        int `json:"scans_used"`
	ScansLimit       int `json:"scans_limit"`
}

type CreateInvoiceRequest struct {
	Plan string `json:"plan" validate:"required,oneof=monthly annual founder_annual"`
}

type CreateInvoiceResponse struct {
	PaymentURL string  `json:"payment_url"`
	InvoiceID  string  `json:"invoice_id"`
	ExpiresAt  string  `json:"expires_at"`
	Amount     float64 `json:"amount"`
	Plan       string  `json:"plan"`
}

type WebhookPayload struct {
	ExternalInvoiceID string  `json:"external_invoice_id"`
	Status            string  `json:"status"`
	PaidAmount        float64 `json:"paid_amount"`
	PaidAt            string  `json:"paid_at,omitempty"`
}
