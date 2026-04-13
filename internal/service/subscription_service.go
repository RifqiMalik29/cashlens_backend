package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rifqimalik/cashlens-backend/internal/models"
	"github.com/rifqimalik/cashlens-backend/internal/repository"
)

type SubscriptionService interface {
	CreatePendingInvoice(ctx context.Context, userID uuid.UUID, plan string, amount float64, externalInvoiceID string, expiresAt time.Time) error
	ProcessPaymentWebhook(ctx context.Context, externalInvoiceID string, status string, paidAmount float64) error
}

type subscriptionService struct {
	userRepo      repository.UserRepository
	eventRepo     repository.SubscriptionEventRepository
	invoiceRepo   repository.PendingInvoiceRepository
	webhookToken  string
}

func NewSubscriptionService(
	userRepo repository.UserRepository,
	eventRepo repository.SubscriptionEventRepository,
	invoiceRepo repository.PendingInvoiceRepository,
	webhookToken string,
) SubscriptionService {
	return &subscriptionService{
		userRepo:     userRepo,
		eventRepo:    eventRepo,
		invoiceRepo:  invoiceRepo,
		webhookToken: webhookToken,
	}
}

func (s *subscriptionService) CreatePendingInvoice(ctx context.Context, userID uuid.UUID, plan string, amount float64, externalInvoiceID string, expiresAt time.Time) error {
	invoice := &models.PendingInvoice{
		ID:                uuid.New(),
		UserID:            userID,
		ExternalInvoiceID: externalInvoiceID,
		Plan:              plan,
		Amount:            amount,
		Status:            "pending",
		ExpiresAt:         expiresAt,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}
	return s.invoiceRepo.Create(ctx, invoice)
}

func (s *subscriptionService) ProcessPaymentWebhook(ctx context.Context, externalInvoiceID string, status string, paidAmount float64) error {
	// 1. Check idempotency — if already processed, return success
	alreadyProcessed, err := s.eventRepo.ExistsByExternalInvoiceID(ctx, externalInvoiceID)
	if err != nil {
		return fmt.Errorf("failed to check idempotency: %w", err)
	}
	if alreadyProcessed {
		return nil // Already processed, return success (idempotent)
	}

	// 2. Look up pending invoice
	invoice, err := s.invoiceRepo.GetByExternalInvoiceID(ctx, externalInvoiceID)
	if err != nil {
		return fmt.Errorf("pending invoice not found: %w", err)
	}

	// 3. Only process PAID status
	if status != "PAID" {
		// Update pending invoice status but don't activate subscription
		s.invoiceRepo.UpdateStatus(ctx, externalInvoiceID, status)
		return nil
	}

	// 4. Calculate expiry based on plan
	var expiryDuration time.Duration
	var planStr string
	switch invoice.Plan {
	case "monthly":
		expiryDuration = 30 * 24 * time.Hour
		planStr = "monthly"
	case "annual":
		expiryDuration = 365 * 24 * time.Hour
		planStr = "annual"
	case "founder_annual":
		expiryDuration = 365 * 24 * time.Hour
		planStr = "founder_annual"
	default:
		return fmt.Errorf("unknown plan: %s", invoice.Plan)
	}

	expiresAt := time.Now().Add(expiryDuration)

	// 5. Update user subscription
	user, err := s.userRepo.GetByID(ctx, invoice.UserID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	user.SubscriptionTier = "premium"
	user.SubscriptionExpiry = &expiresAt

	err = s.userRepo.Update(ctx, user)
	if err != nil {
		return fmt.Errorf("failed to update user subscription: %w", err)
	}

	// 6. Set founder flag if applicable
	if invoice.Plan == "founder_annual" {
		user.IsFounder = true
		s.userRepo.Update(ctx, user)
	}

	// 7. Update pending invoice status
	s.invoiceRepo.UpdateStatus(ctx, externalInvoiceID, "paid")

	// 8. Record subscription event
	event := &models.SubscriptionEvent{
		ID:                uuid.New(),
		UserID:            invoice.UserID,
		EventType:         "subscribed",
		Plan:              &planStr,
		PricePaid:         &paidAmount,
		ExternalInvoiceID: &externalInvoiceID,
		CreatedAt:         time.Now(),
	}
	err = s.eventRepo.Create(ctx, event)
	if err != nil {
		return fmt.Errorf("failed to record subscription event: %w", err)
	}

	return nil
}

// VerifyWebhookSignature verifies Xendit webhook callback token
// Xendit sends the token in the "x-callback-token" header
func (s *subscriptionService) VerifyWebhookSignature(token string) bool {
	return hmac.Equal([]byte(token), []byte(s.webhookToken))
}

// ComputeXenditSignature computes HMAC-SHA256 signature for verification
func ComputeXenditSignature(payload []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}
