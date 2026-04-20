package revenuecat

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Client handles RevenueCat API calls
type Client struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new RevenueCat client
func NewClient(apiKey string) *Client {
	return &Client{
		apiKey:     apiKey,
		baseURL:    "https://api.revenuecat.com/v1",
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// SubscriberResponse represents the full response from the GET /subscribers/{app_user_id} endpoint
type SubscriberResponse struct {
	RequestDate   string     `json:"request_date"`
	RequestDateMS int64      `json:"request_date_ms"`
	Subscriber    Subscriber `json:"subscriber"`
}

// Subscriber represents the subscriber object in the RevenueCat API
type Subscriber struct {
	Entitlements       map[string]Entitlement `json:"entitlements"`
	FirstSeen          string                 `json:"first_seen"`
	LastSeen           string                 `json:"last_seen"`
	ManagementURL      *string                `json:"management_url"`
	OriginalAppUserID  string                 `json:"original_app_user_id"`
	Subscriptions      map[string]Subscription `json:"subscriptions"`
}

// Entitlement represents an entitlement object
type Entitlement struct {
	ExpiresDate      *string `json:"expires_date"`
	ProductIdentifier string  `json:"product_identifier"`
	PurchaseDate     string  `json:"purchase_date"`
}

// Subscription represents a subscription object
type Subscription struct {
	BillingIssuesDetectedAt *string `json:"billing_issues_detected_at"`
	ExpiresDate             *string `json:"expires_date"`
	IsSandbox               bool    `json:"is_sandbox"`
	OriginalPurchaseDate    string  `json:"original_purchase_date"`
	PeriodType              string  `json:"period_type"`
	PurchaseDate            string  `json:"purchase_date"`
	Store                   string  `json:"store"`
	UnsubscribeDetectedAt   *string `json:"unsubscribe_detected_at"`
}

// GetSubscriber retrieves a subscriber's details from RevenueCat
func (c *Client) GetSubscriber(ctx context.Context, appUserID string) (*SubscriberResponse, error) {
	url := fmt.Sprintf("%s/subscribers/%s", c.baseURL, appUserID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call RevenueCat API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// You might want to decode an error response body here
		return nil, fmt.Errorf("RevenueCat API error (status %d)", resp.StatusCode)
	}

	var subResp SubscriberResponse
	if err := json.NewDecoder(resp.Body).Decode(&subResp); err != nil {
		return nil, fmt.Errorf("failed to decode RevenueCat response: %w", err)
	}

	return &subResp, nil
}
