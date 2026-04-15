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

// XenditWebhookPayload matches Xendit's payment_session.completed event structure
type XenditWebhookPayload struct {
	Event string                   `json:"event"`
	Data  XenditWebhookPayloadData `json:"data"`
}

type XenditWebhookPayloadData struct {
	ReferenceID      string  `json:"reference_id"`
	PaymentSessionID string  `json:"payment_session_id"`
	PaymentID        string  `json:"payment_id"`
	Status           string  `json:"status"`
	Amount           float64 `json:"amount"`
}
