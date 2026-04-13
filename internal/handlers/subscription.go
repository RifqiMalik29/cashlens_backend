package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	apperrors "github.com/rifqimalik/cashlens-backend/internal/errors"
	"github.com/rifqimalik/cashlens-backend/internal/middleware"
	"github.com/rifqimalik/cashlens-backend/internal/models"
	"github.com/rifqimalik/cashlens-backend/internal/repository"
	"github.com/rifqimalik/cashlens-backend/internal/service"
)

// Plan pricing (in IDR)
var PlanPrices = map[string]float64{
	"monthly":        15000,
	"annual":         129000,
	"founder_annual": 99000,
}

type SubscriptionHandler struct {
	quotaService  service.QuotaService
	userRepo      repository.UserRepository
	subService    service.SubscriptionService
	webhookToken  string
}

func NewSubscriptionHandler(
	quotaService service.QuotaService,
	userRepo repository.UserRepository,
	subService service.SubscriptionService,
	webhookToken string,
) *SubscriptionHandler {
	return &SubscriptionHandler{
		quotaService: quotaService,
		userRepo:     userRepo,
		subService:   subService,
		webhookToken: webhookToken,
	}
}

// GetSubscriptionStatus returns the current subscription status and quota usage
func (h *SubscriptionHandler) GetSubscriptionStatus(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(*uuid.UUID)
	if !ok {
		apperrors.WriteJSONError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	user, err := h.userRepo.GetByID(r.Context(), *userID)
	if err != nil {
		apperrors.WriteJSONError(w, "Failed to get user", http.StatusInternalServerError)
		return
	}

	tier := user.SubscriptionTier
	var expiresAt *string
	if user.SubscriptionExpiry != nil {
		expStr := user.SubscriptionExpiry.Format(time.RFC3339)
		expiresAt = &expStr

		if tier == "premium" && user.SubscriptionExpiry.Before(time.Now()) {
			tier = "free"
		}
	}

	quota, err := h.quotaService.GetCurrentUsage(r.Context(), *userID)
	if err != nil {
		apperrors.WriteJSONError(w, "Failed to get quota", http.StatusInternalServerError)
		return
	}

	limits := models.FreeTierLimits
	if tier == "premium" {
		limits = models.PremiumTierLimits
	}

	txLimit := limits.MaxTransactionsPerMonth
	scanLimit := limits.MaxScansPerMonth

	if txLimit == -1 {
		txLimit = 0
	}
	if scanLimit == -1 {
		scanLimit = 0
	}

	response := models.SubscriptionStatus{
		Tier:      tier,
		ExpiresAt: expiresAt,
		Quota: &models.QuotaStatus{
			TransactionsUsed:  quota.TransactionsUsed,
			TransactionsLimit: txLimit,
			ScansUsed:         quota.ScansUsed,
			ScansLimit:        scanLimit,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"data": response,
	})
}

// CreateInvoice creates a payment invoice for subscription upgrade
// Placeholder — returns 501 until Xendit integration is complete
func (h *SubscriptionHandler) CreateInvoice(w http.ResponseWriter, r *http.Request) {
	var req models.CreateInvoiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apperrors.WriteJSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	price, ok := PlanPrices[req.Plan]
	if !ok {
		apperrors.WriteJSONError(w, "Invalid plan", http.StatusBadRequest)
		return
	}

	// TODO: Integrate Xendit Invoice API
	// 1. Call Xendit to create invoice
	// 2. Store external_invoice_id in pending_invoices table
	// 3. Return payment_url

	_ = price // Will be used when Xendit integration is added

	apperrors.WriteJSONError(w, "Payment integration not yet implemented — Xendit integration required", http.StatusNotImplemented)
}

// PaymentWebhook handles Xendit webhook callbacks
func (h *SubscriptionHandler) PaymentWebhook(w http.ResponseWriter, r *http.Request) {
	// 1. Verify Xendit webhook signature
	callbackToken := r.Header.Get("x-callback-token")
	if callbackToken == "" || !h.verifyWebhookToken(callbackToken) {
		apperrors.WriteJSONError(w, "Invalid webhook signature", http.StatusUnauthorized)
		return
	}

	// 2. Read and parse payload
	body, err := io.ReadAll(r.Body)
	if err != nil {
		// Return 200 to prevent Xendit retries even on parse errors
		w.WriteHeader(http.StatusOK)
		return
	}

	var payload models.XenditWebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		w.WriteHeader(http.StatusOK)
		return
	}

	// 3. Process the webhook (idempotent)
	err = h.subService.ProcessPaymentWebhook(r.Context(), payload.ExternalInvoiceID, payload.Status, payload.PaidAmount)
	if err != nil {
		// Log error but return 200 to prevent webhook retries
		// In production: add proper slog.Error logging here
		w.WriteHeader(http.StatusOK)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// verifyWebhookToken verifies the Xendit callback token using HMAC
func (h *SubscriptionHandler) verifyWebhookToken(token string) bool {
	if h.webhookToken == "" {
		return false
	}
	return hmac.Equal([]byte(token), []byte(h.webhookToken))
}

// ComputeXenditSignature computes HMAC-SHA256 for testing/debugging
func ComputeXenditSignature(payload []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}
