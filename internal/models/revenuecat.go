package models

// RevenueCatWebhook represents the overall structure of a webhook from RevenueCat
type RevenueCatWebhook struct {
	APIVersion string        `json:"api_version"`
	Event      RevenueCatEvent `json:"event"`
}

// RevenueCatEvent represents the "event" object within a RevenueCat webhook
type RevenueCatEvent struct {
	ID                 string   `json:"id"`
	Type               string   `json:"type"`
	EventTimestampMS   int64    `json:"event_timestamp_ms"`
	AppUserID          string   `json:"app_user_id"`
	OriginalAppUserID  string   `json:"original_app_user_id"`
	Aliases            []string `json:"aliases"`
	ProductID          *string  `json:"product_id"`
	EntitlementIDs     []string `json:"entitlement_ids"`
	PeriodType         string   `json:"period_type"`
	PurchasedAtMS      *int64   `json:"purchased_at_ms"`
	ExpiresAtMS        *int64   `json:"expires_at_ms"`
	TransactionID      *string  `json:"transaction_id"`
	OriginalTransactionID *string `json:"original_transaction_id"`
	Store              string   `json:"store"`
	IsSandbox          bool     `json:"is_sandbox"`
	Price              *float64 `json:"price"`
	Currency           *string  `json:"currency"`
	CancelReason       *string  `json:"cancel_reason"`
}
