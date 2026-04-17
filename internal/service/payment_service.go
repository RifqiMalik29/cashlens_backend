package service

import (
	"context"
	"fmt"
	"strings" // Added this import
	"time"

	"github.com/rifqimalik/cashlens-backend/internal/pkg/xendit"
	"github.com/google/uuid"
)

// PaymentService handles payment-related business logic
type PaymentService struct {
	xenditClient *xendit.XenditClient
	// Add other dependencies like a repository if payment sessions need to be stored
}

// NewPaymentService creates a new instance of PaymentService
func NewPaymentService(xenditClient *xendit.XenditClient) *PaymentService {
	return &PaymentService{
		xenditClient: xenditClient,
	}
}

// CreatePaymentSessionParams holds the parameters for creating a payment session
type CreatePaymentSessionParams struct {
	UserID             uuid.UUID
	Amount             float64
	Currency           string
	PayerEmail         string
	PayerName          string // Use PayerName instead of separate given names and surname
	SuccessRedirectURL string
	FailureRedirectURL string
}

// PaymentSessionResult holds the result of a created payment session
type PaymentSessionResult struct {
	PaymentSessionID string
	PaymentLinkURL   string
	ExpiresAt        time.Time
}

// CreatePaymentSession creates a new payment session with Xendit
func (s *PaymentService) CreatePaymentSession(ctx context.Context, params CreatePaymentSessionParams) (*PaymentSessionResult, error) {
	// Generate a unique reference ID for the payment session
	referenceID := uuid.New().String()

	// Split PayerName into given_names and surname
	givenNames := ""
	surname := ""
	if params.PayerName != "" {
		nameParts := strings.Fields(params.PayerName)
		givenNames = nameParts[0]
		if len(nameParts) > 1 {
			surname = nameParts[len(nameParts)-1]
		}
	} else {
		// Fallback if name is not provided
		givenNames = "Customer"
		surname = "Cashlens"
	}

	// Map internal parameters to Xendit's request struct
	xenditReq := xendit.PaymentSessionRequest{
		ReferenceID: referenceID,
		SessionType: "PAY",
		Mode:        "PAYMENT_LINK",
		Amount:      params.Amount,
		Currency:    params.Currency,
		Country:     "ID", // Assuming Indonesia for now, can be made dynamic
		Customer: xendit.PaymentSessionCustomer{
			ReferenceID: params.UserID.String(), // Use UserID as customer reference
			Type:        "INDIVIDUAL",
			Email:       params.PayerEmail,
			IndividualDetail: &xendit.IndividualDetail{
				GivenNames: givenNames,
				Surname:    surname,
			},
		},
		SuccessReturnURL: params.SuccessRedirectURL,
		CancelReturnURL:  params.FailureRedirectURL,
	}

	xenditResp, err := s.xenditClient.CreatePaymentSession(ctx, xenditReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create xendit payment session: %w", err)
	}

	expiresAt, err := time.Parse(time.RFC3339, xenditResp.ExpiresAt)
	if err != nil {
		return nil, fmt.Errorf("failed to parse expires_at timestamp: %w", err)
	}

	return &PaymentSessionResult{
		PaymentSessionID: xenditResp.PaymentSessionID,
		PaymentLinkURL:   xenditResp.PaymentLinkURL,
		ExpiresAt:        expiresAt,
	}, nil
}
