package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	apperrors "github.com/rifqimalik/cashlens-backend/internal/errors"
	"github.com/rifqimalik/cashlens-backend/internal/middleware"
	"github.com/rifqimalik/cashlens-backend/internal/models"
	"github.com/rifqimalik/cashlens-backend/internal/pkg/validator"
	"github.com/rifqimalik/cashlens-backend/internal/pkg/xendit"
	"github.com/rifqimalik/cashlens-backend/internal/repository"
	"github.com/rifqimalik/cashlens-backend/internal/service"
)

// Plan pricing and duration (in IDR)
var PlanConfig = map[string]struct {
	Price    float64
	Duration time.Duration
}{
	"monthly":        {Price: 15000, Duration: 30 * 24 * time.Hour},
	"annual":         {Price: 129000, Duration: 365 * 24 * time.Hour},
	"founder_annual": {Price: 99000, Duration: 365 * 24 * time.Hour},
}

type SubscriptionHandler struct {
	quotaService  service.QuotaService
	userRepo      repository.UserRepository
	subService    service.SubscriptionService
	xenditClient  *xendit.XenditClient
	webhookToken  string
	successURL    string
	failureURL    string
}

func NewSubscriptionHandler(
	quotaService service.QuotaService,
	userRepo repository.UserRepository,
	subService service.SubscriptionService,
	xenditClient *xendit.XenditClient,
	webhookToken string,
	successURL string,
	failureURL string,
) *SubscriptionHandler {
	return &SubscriptionHandler{
		quotaService: quotaService,
		userRepo:     userRepo,
		subService:   subService,
		xenditClient: xenditClient,
		webhookToken: webhookToken,
		successURL:   successURL,
		failureURL:   failureURL,
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
		apperrors.WriteJSONError(w, "User not found — please log in again", http.StatusUnauthorized)
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

// CreateInvoice creates a payment invoice via Xendit
func (h *SubscriptionHandler) CreateInvoice(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(*uuid.UUID)
	if !ok {
		apperrors.WriteJSONError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req models.CreateInvoiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apperrors.WriteJSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	config, ok := PlanConfig[req.Plan]
	if !ok {
		apperrors.WriteJSONError(w, "Invalid plan", http.StatusBadRequest, map[string]string{
			"plan": "must be one of: monthly, annual, founder_annual",
		})
		return
	}

	// Generate unique external invoice ID
	externalInvoiceID := fmt.Sprintf("cashlens-%s-%d", userID.String()[:8], time.Now().Unix())
	expiresAt := time.Now().Add(config.Duration)

	// Call Xendit to create invoice
	xenditReq := xendit.XenditInvoiceRequest{
		ExternalInvoiceID: externalInvoiceID,
		Amount:            config.Price,
		Description:       fmt.Sprintf("CashLens Premium %s plan", req.Plan),
		InvoiceDuration:   604800, // 7 days in seconds
		SuccessRedirectURL: h.successURL + "?invoice_id=" + externalInvoiceID,
		FailureRedirectURL: h.failureURL,
	}

	xenditResp, err := h.xenditClient.CreateInvoice(r.Context(), xenditReq)
	if err != nil {
		fmt.Printf("XENDIT API ERROR: %v", err)
		apperrors.WriteJSONError(w, "Failed to create invoice", http.StatusInternalServerError)
		return
	}

	// Store pending invoice (including Xendit's internal ID for later verification)
	err = h.subService.CreatePendingInvoice(r.Context(), *userID, req.Plan, config.Price, externalInvoiceID, xenditResp.ID, expiresAt)
	if err != nil {
		slog.Error("[CreateInvoice] failed to store pending invoice", "error", err)
		// Log error but continue — invoice was created in Xendit
	}

	// Return payment URL to client
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"data": models.CreateInvoiceResponse{
			PaymentURL: xenditResp.InvoiceURL,
			InvoiceID:  externalInvoiceID,
			ExpiresAt:  expiresAt.Format(time.RFC3339),
			Amount:     config.Price,
			Plan:       req.Plan,
		},
	})
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
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{"status": "ok", "message": "empty body"})
		return
	}

	var payload models.XenditWebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{"status": "ok", "message": "invalid payload format"})
		return
	}

	// 3. Process the webhook (idempotent)
	// Xendit sends status "SUCCEEDED" or "COMPLETED" in payment_session.completed event
	err = h.subService.ProcessPaymentWebhook(r.Context(), payload.Data.ReferenceID, payload.Data.Status, payload.Data.Amount)
	if err != nil {
		// Log error but return 200 to prevent webhook retries
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{"status": "ok", "message": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{"status": "ok", "message": "webhook processed"})
}

// VerifyPayment allows the client to manually trigger a payment check
func (h *SubscriptionHandler) VerifyPayment(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(*uuid.UUID)
	if !ok {
		apperrors.WriteJSONError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		InvoiceID string `json:"invoice_id" validate:"required"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apperrors.WriteJSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if validationErrors := validator.ValidateStruct(&req); validationErrors != nil {
		apperrors.WriteJSONError(w, "Validation failed", http.StatusBadRequest, validationErrors)
		return
	}

	slog.Info("[VerifyPayment] received", "invoice_id", req.InvoiceID, "user_id", userID.String())

	err := h.subService.VerifyPayment(r.Context(), *userID, req.InvoiceID)
	if err != nil {
		apperrors.WriteJSONError(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{"status": "ok", "message": "payment verified and upgraded"})
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
