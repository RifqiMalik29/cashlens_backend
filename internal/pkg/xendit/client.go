package xendit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// XenditClient handles Xendit Invoice API calls
type XenditClient struct {
	secretKey string
	baseURL   string
	httpClient *http.Client
}

func NewXenditClient(secretKey string) *XenditClient {
	return &XenditClient{
		secretKey:  secretKey,
		baseURL:    "https://api.xendit.co",
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// XenditInvoiceRequest represents the payload for creating an invoice
type XenditInvoiceRequest struct {
	ExternalInvoiceID string  `json:"external_invoice_id"`
	Amount            float64 `json:"amount"`
	PayerEmail        string  `json:"payer_email,omitempty"`
	Description       string  `json:"description"`
	InvoiceDuration   int     `json:"invoice_duration"` // seconds, default 604800 (7 days)
	CallbackURL       string  `json:"callback_url,omitempty"`
	SuccessRedirectURL string `json:"success_redirect_url,omitempty"`
	FailureRedirectURL string `json:"failure_redirect_url,omitempty"`
}

// XenditInvoiceResponse represents Xendit's response
type XenditInvoiceResponse struct {
	ID                string  `json:"id"`
	ExternalInvoiceID string  `json:"external_invoice_id"`
	UserID            string  `json:"user_id"`
	Amount            float64 `json:"amount"`
	Status            string  `json:"status"`
	InvoiceURL        string  `json:"invoice_url"`
	Created           string  `json:"created"`
	Updated           string  `json:"updated"`
}

// CreateInvoice creates a new invoice via Xendit API
func (c *XenditClient) CreateInvoice(ctx context.Context, req XenditInvoiceRequest) (*XenditInvoiceResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/invoices", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Basic "+c.encodeBasicAuth())
	httpReq.SetBasicAuth(c.secretKey, "")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to call Xendit API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Xendit API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var xenditResp XenditInvoiceResponse
	if err := json.NewDecoder(resp.Body).Decode(&xenditResp); err != nil {
		return nil, fmt.Errorf("failed to decode Xendit response: %w", err)
	}

	return &xenditResp, nil
}

// encodeBasicAuth creates Basic auth header value
func (c *XenditClient) encodeBasicAuth() string {
	return fmt.Sprintf("%s:", c.secretKey)
}
