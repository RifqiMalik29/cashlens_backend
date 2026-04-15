package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/rifqimalik/cashlens-backend/internal/models"
	"github.com/rifqimalik/cashlens-backend/internal/pkg/xendit"
	"github.com/rifqimalik/cashlens-backend/internal/repository"
)

type SubscriptionService interface {
	CreatePendingInvoice(ctx context.Context, userID uuid.UUID, plan string, amount float64, externalInvoiceID string, xenditInvoiceID string, expiresAt time.Time) error
	ProcessPaymentWebhook(ctx context.Context, externalInvoiceID string, status string, paidAmount float64) error
	VerifyPayment(ctx context.Context, userID uuid.UUID, externalInvoiceID string) error
}

type subscriptionService struct {
	userRepo     repository.UserRepository
	eventRepo    repository.SubscriptionEventRepository
	invoiceRepo  repository.PendingInvoiceRepository
	xenditClient *xendit.XenditClient
	webhookToken string
}

func NewSubscriptionService(
	userRepo repository.UserRepository,
	eventRepo repository.SubscriptionEventRepository,
	invoiceRepo repository.PendingInvoiceRepository,
	xenditClient *xendit.XenditClient,
	webhookToken string,
) SubscriptionService {
	return &subscriptionService{
		userRepo:     userRepo,
		eventRepo:    eventRepo,
		invoiceRepo:  invoiceRepo,
		xenditClient: xenditClient,
		webhookToken: webhookToken,
	}
}

func (s *subscriptionService) CreatePendingInvoice(ctx context.Context, userID uuid.UUID, plan string, amount float64, externalInvoiceID string, xenditInvoiceID string, expiresAt time.Time) error {
	invoice := &models.PendingInvoice{
		ID:                uuid.New(),
		UserID:            userID,
		ExternalInvoiceID: externalInvoiceID,
		XenditInvoiceID:   xenditInvoiceID,
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

	// 3. Only process successful payment — Xendit sends "SUCCEEDED" from payments API
	// and "COMPLETED" from payment_session.completed webhook event
	if status != "SUCCEEDED" && status != "COMPLETED" {
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

	// 5. Update user subscription tier and expiry
	err = s.userRepo.UpdateSubscription(ctx, invoice.UserID, "premium", &expiresAt)
	if err != nil {
		return fmt.Errorf("failed to update user subscription: %w", err)
	}

	// 6. Set founder flag if applicable
	if invoice.Plan == "founder_annual" {
		if err := s.userRepo.UpdateFounder(ctx, invoice.UserID, true); err != nil {
			return fmt.Errorf("failed to set founder flag: %w", err)
		}
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

// VerifyPayment checks Xendit directly for invoice status and upgrades user if paid
func (s *subscriptionService) VerifyPayment(ctx context.Context, userID uuid.UUID, externalInvoiceID string) error {
	// 1. Check idempotency
	alreadyProcessed, err := s.eventRepo.ExistsByExternalInvoiceID(ctx, externalInvoiceID)
	if err != nil {
		return fmt.Errorf("failed to check idempotency: %w", err)
	}
	if alreadyProcessed {
		return nil // Already processed
	}

	// 2. Look up our pending invoice to get Xendit's invoice ID
	pendingInvoice, err := s.invoiceRepo.GetByExternalInvoiceID(ctx, externalInvoiceID)
	if err != nil {
		return fmt.Errorf("pending invoice not found: %w", err)
	}
	if pendingInvoice.XenditInvoiceID == "" {
		return fmt.Errorf("xendit invoice ID not recorded for this invoice")
	}

	slog.Info("[VerifyPayment] checking Xendit invoice", "xendit_invoice_id", pendingInvoice.XenditInvoiceID)

	// 3. Check invoice status via Xendit v2 Invoice API
	xenditInvoice, err := s.xenditClient.GetInvoiceByID(ctx, pendingInvoice.XenditInvoiceID)
	if err != nil {
		return fmt.Errorf("failed to get payment status: %w", err)
	}

	// 4. Xendit v2 Invoice API uses "SETTLED" (or "PAID") as the success status
	if xenditInvoice.Status == "SETTLED" || xenditInvoice.Status == "PAID" {
		return s.ProcessPaymentWebhook(ctx, externalInvoiceID, "SUCCEEDED", xenditInvoice.Amount)
	}

	return fmt.Errorf("payment status is %s, not SETTLED", xenditInvoice.Status)
}
