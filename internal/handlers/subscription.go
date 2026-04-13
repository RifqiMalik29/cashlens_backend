package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	apperrors "github.com/rifqimalik/cashlens-backend/internal/errors"
	"github.com/rifqimalik/cashlens-backend/internal/middleware"
	"github.com/rifqimalik/cashlens-backend/internal/models"
	"github.com/rifqimalik/cashlens-backend/internal/pkg/validator"
	"github.com/rifqimalik/cashlens-backend/internal/repository"
	"github.com/rifqimalik/cashlens-backend/internal/service"
)

type SubscriptionHandler struct {
	quotaService service.QuotaService
	userRepo     repository.UserRepository
}

func NewSubscriptionHandler(quotaService service.QuotaService, userRepo repository.UserRepository) *SubscriptionHandler {
	return &SubscriptionHandler{
		quotaService: quotaService,
		userRepo:     userRepo,
	}
}

// GetSubscriptionStatus returns the current subscription status and quota usage
func (h *SubscriptionHandler) GetSubscriptionStatus(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(*uuid.UUID)
	if !ok {
		apperrors.WriteJSONError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get real user data
	user, err := h.userRepo.GetByID(r.Context(), *userID)
	if err != nil {
		apperrors.WriteJSONError(w, "Failed to get user", http.StatusInternalServerError)
		return
	}

	// Determine effective tier (check expiry)
	tier := user.SubscriptionTier
	var expiresAt *string
	if user.SubscriptionExpiry != nil {
		expStr := user.SubscriptionExpiry.Format(time.RFC3339)
		expiresAt = &expStr
		
		// Auto-downgrade expired premium users
		if tier == "premium" && user.SubscriptionExpiry.Before(time.Now()) {
			tier = "free"
		}
	}

	// Get current quota usage
	quota, err := h.quotaService.GetCurrentUsage(r.Context(), *userID)
	if err != nil {
		apperrors.WriteJSONError(w, "Failed to get quota", http.StatusInternalServerError)
		return
	}

	// Determine limits based on tier
	limits := models.FreeTierLimits
	if tier == "premium" {
		limits = models.PremiumTierLimits
	}

	txLimit := limits.MaxTransactionsPerMonth
	scanLimit := limits.MaxScansPerMonth

	// -1 means unlimited for premium
	if txLimit == -1 {
		txLimit = 0 // 0 in response means unlimited
	}
	if scanLimit == -1 {
		scanLimit = 0
	}

	response := models.SubscriptionStatus{
		Tier:      tier,
		ExpiresAt: expiresAt,
		Quota: &models.QuotaStatus{
			TransactionsUsed: quota.TransactionsUsed,
			TransactionsLimit: txLimit,
			ScansUsed:        quota.ScansUsed,
			ScansLimit:       scanLimit,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"data": response,
	})
}

// CreateInvoice creates a payment invoice for subscription upgrade
func (h *SubscriptionHandler) CreateInvoice(w http.ResponseWriter, r *http.Request) {
	var req models.CreateInvoiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apperrors.WriteJSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if validationErrors := validator.ValidateStruct(&req); validationErrors != nil {
		apperrors.WriteJSONError(w, "Validation failed", http.StatusBadRequest, validationErrors)
		return
	}

	// TODO: Implement Xendit/Midtrans integration
	apperrors.WriteJSONError(w, "Payment integration not yet implemented", http.StatusNotImplemented)
}

// PaymentWebhook handles payment provider webhook callbacks
// TODO: Implement webhook signature verification before enabling
func (h *SubscriptionHandler) PaymentWebhook(w http.ResponseWriter, r *http.Request) {
	// Disabled until signature verification is implemented
	apperrors.WriteJSONError(w, "Payment webhook not yet implemented — signature verification required", http.StatusNotImplemented)
}
