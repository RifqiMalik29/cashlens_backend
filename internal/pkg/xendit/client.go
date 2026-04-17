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
	secretKey  string
	baseURL    string
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
	ExternalInvoiceID  string  `json:"external_id"`
	Amount             float64 `json:"amount"`
	PayerEmail         string  `json:"payer_email,omitempty"`
	Description        string  `json:"description"`
	InvoiceDuration    int     `json:"invoice_duration"` // seconds, default 604800 (7 days)
	CallbackURL        string  `json:"callback_url,omitempty"`
	SuccessRedirectURL string  `json:"success_redirect_url,omitempty"`
	FailureRedirectURL string  `json:"failure_redirect_url,omitempty"`
}

// XenditInvoiceResponse represents Xendit's invoice creation response
type XenditInvoiceResponse struct {
	ID         string  `json:"id"`
	ExternalID string  `json:"external_id"`
	Amount     float64 `json:"amount"`
	Status     string  `json:"status"`
	InvoiceURL string  `json:"invoice_url"`
	Created    string  `json:"created"`
	Updated    string  `json:"updated"`
}

// XenditPaymentResponse represents a payment object from GET /v1/payments
type XenditPaymentResponse struct {
	ID          string  `json:"id"`
	ReferenceID string  `json:"reference_id"`
	Status      string  `json:"status"`
	Amount      float64 `json:"amount"`
}

type xenditPaymentListResponse struct {
	Data []XenditPaymentResponse `json:"data"`
}

// PaymentSessionCustomer represents the customer details for a payment session
type PaymentSessionCustomer struct {
	ReferenceID      string            `json:"reference_id"`
	Type             string            `json:"type"` // e.g., 'INDIVIDUAL'
	Email            string            `json:"email"`
	MobileNumber     string            `json:"mobile_number,omitempty"`
	IndividualDetail *IndividualDetail `json:"individual_detail,omitempty"`
}

// IndividualDetail represents individual customer details
type IndividualDetail struct {
	GivenNames string `json:"given_names"`
	Surname    string `json:"surname"`
}

// PaymentSessionRequest represents the payload for creating a payment session
type PaymentSessionRequest struct {
	ReferenceID             string                   `json:"reference_id"`
	SessionType             string                   `json:"session_type"` // e.g., 'PAY'
	Mode                    string                   `json:"mode"`         // e.g., 'PAYMENT_LINK'
	Amount                  float64                  `json:"amount"`
	Currency                string                   `json:"currency"`
	Country                 string                   `json:"country"`
	Customer                PaymentSessionCustomer   `json:"customer"`
	SuccessReturnURL        string                   `json:"success_return_url"`
	CancelReturnURL         string                   `json:"cancel_return_url"`
	AllowSavePaymentMethod  string                   `json:"allow_save_payment_method,omitempty"` // 'OPTIONAL' or 'FORCED' for Pay and Save Flow
	ComponentsConfiguration *ComponentsConfiguration `json:"components_configuration,omitempty"`
}

// ComponentsConfiguration represents configuration for Xendit Components
type ComponentsConfiguration struct {
	Origins []string `json:"origins"`
}

// PaymentSessionResponse represents Xendit's payment session creation response
type PaymentSessionResponse struct {
	PaymentSessionID        string                   `json:"payment_session_id"`
	Created                 string                   `json:"created"`
	Updated                 string                   `json:"updated"`
	Status                  string                   `json:"status"`
	ReferenceID             string                   `json:"reference_id"`
	Currency                string                   `json:"currency"`
	Amount                  float64                  `json:"amount"`
	Country                 string                   `json:"country"`
	CustomerID              string                   `json:"customer_id"`
	ExpiresAt               string                   `json:"expires_at"`
	SessionType             string                   `json:"session_type"`
	Mode                    string                   `json:"mode"`
	Locale                  string                   `json:"locale"`
	BusinessID              string                   `json:"business_id"`
	SuccessReturnURL        string                   `json:"success_return_url"`
	CancelReturnURL         string                   `json:"cancel_return_url"`
	PaymentLinkURL          string                   `json:"payment_link_url"`
	AllowSavePaymentMethod  string                   `json:"allow_save_payment_method,omitempty"`
	ComponentsSDKKey        string                   `json:"components_sdk_key,omitempty"`
	ComponentsConfiguration *ComponentsConfiguration `json:"components_configuration,omitempty"`
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

// GetInvoiceByID retrieves invoice status using Xendit's invoice ID (from create response)
func (c *XenditClient) GetInvoiceByID(ctx context.Context, xenditInvoiceID string) (*XenditInvoiceResponse, error) {
	targetURL := c.baseURL + "/v2/invoices/" + xenditInvoiceID
	httpReq, err := http.NewRequestWithContext(ctx, "GET", targetURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.SetBasicAuth(c.secretKey, "")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to call Xendit API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Xendit API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var invoice XenditInvoiceResponse
	if err := json.NewDecoder(resp.Body).Decode(&invoice); err != nil {
		return nil, fmt.Errorf("failed to decode Xendit response: %w", err)
	}

	return &invoice, nil
}

// GetInvoiceByExternalID retrieves an invoice using external_id (the Invoice API, not Payments API)
func (c *XenditClient) GetInvoiceByExternalID(ctx context.Context, externalID string) (*XenditInvoiceResponse, error) {
	url := c.baseURL + "/v1/invoices?external_id=" + externalID
	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.SetBasicAuth(c.secretKey, "")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to call Xendit API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Xendit API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	// Returns an array
	var invoices []XenditInvoiceResponse
	if err := json.NewDecoder(resp.Body).Decode(&invoices); err != nil {
		return nil, fmt.Errorf("failed to decode Xendit response: %w", err)
	}

	if len(invoices) == 0 {
		return nil, fmt.Errorf("no invoice found for external_id: %s", externalID)
	}

	return &invoices[0], nil
}

// encodeBasicAuth creates Basic auth header value
func (c *XenditClient) encodeBasicAuth() string {
	return fmt.Sprintf("%s:", c.secretKey)
}

// CreatePaymentSession creates a new payment session via Xendit API
func (c *XenditClient) CreatePaymentSession(ctx context.Context, req PaymentSessionRequest) (*PaymentSessionResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payment session request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/sessions", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create payment session request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Basic "+c.encodeBasicAuth())
	httpReq.SetBasicAuth(c.secretKey, "")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to call Xendit payment session API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Xendit payment session API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var xenditResp PaymentSessionResponse
	if err := json.NewDecoder(resp.Body).Decode(&xenditResp); err != nil {
		return nil, fmt.Errorf("failed to decode Xendit payment session response: %w", err)
	}

	return &xenditResp, nil
}
