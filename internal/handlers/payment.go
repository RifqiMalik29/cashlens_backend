package handlers

import (
	"encoding/json"
	"net/http"

	"cashlens/internal/config"
	"cashlens/internal/logger"
	"cashlens/internal/middleware"
	"cashlens/internal/pkg/validator"
	"cashlens/internal/service"
	"github.com/google/uuid"
)

// PaymentHandler handles HTTP requests related to payments
type PaymentHandler struct {
	paymentService *service.PaymentService
	authService    service.AuthService // To get user details like email, names
	config         *config.Config
	logger         *logger.Logger
}

// NewPaymentHandler creates a new instance of PaymentHandler
func NewPaymentHandler(paymentService *service.PaymentService, authService service.AuthService, cfg *config.Config, logger *logger.Logger) *PaymentHandler {
	return &PaymentHandler{
		paymentService: paymentService,
		authService:    authService,
		config:         cfg,
		logger:         logger,
	}
}

// CreatePaymentSessionRequest represents the request body for creating a payment session
type CreatePaymentSessionRequest struct {
	Amount             float64 `json:"amount" validate:"required,gt=0"`
	Currency           string  `json:"currency" validate:"required,iso4217"` // e.g., "IDR"
	SuccessRedirectURL string  `json:"success_redirect_url" validate:"required,url"`
	FailureRedirectURL string  `json:"failure_redirect_url" validate:"required,url"`
}

// CreatePaymentSessionResponse represents the response for creating a payment session
type CreatePaymentSessionResponse struct {
	PaymentSessionID string `json:"payment_session_id"`
	PaymentLinkURL   string `json:"payment_link_url"`
}

// CreatePaymentSession handles the creation of a new Xendit payment session
func (h *PaymentHandler) CreatePaymentSession(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(*uuid.UUID)
	if !ok {
		h.logger.Error("Unauthorized attempt to create payment session: user ID not in context")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Unauthorized"})
		return
	}

	var req CreatePaymentSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Errorf("Failed to decode create payment session request: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Invalid request body"})
		return
	}

	// Validate request
	if validationErrors := validator.ValidateStruct(&req); validationErrors != nil {
		h.logger.Warnf("Validation failed for create payment session request: %v", validationErrors)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error:   "Validation failed",
			Details: validationErrors,
		})
		return
	}

	// Retrieve user details from AuthService for Xendit customer info
	user, err := h.authService.GetMe(r.Context(), *userID)
	if err != nil {
		h.logger.Errorf("Failed to get user details for userID %s: %v", userID.String(), err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized) // User not found, likely invalid token
		json.NewEncoder(w).Encode(ErrorResponse{Error: "User not found or unauthorized"})
		return
	}

	// Prepare parameters for PaymentService
	var payerName string
	if user.Name != nil {
		payerName = *user.Name
	}
	params := service.CreatePaymentSessionParams{
		UserID:             *userID,
		Amount:             req.Amount,
		Currency:           req.Currency,
		PayerEmail:         user.Email,
		PayerName:          payerName,
		SuccessRedirectURL: req.SuccessRedirectURL,
		FailureRedirectURL: req.FailureRedirectURL,
	}

	paymentSessionResult, err := h.paymentService.CreatePaymentSession(r.Context(), params)
	if err != nil {
		h.logger.Errorf("Failed to create Xendit payment session for user %s: %v", userID.String(), err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Failed to initiate payment. Please try again."})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(CreatePaymentSessionResponse{
		PaymentSessionID: paymentSessionResult.PaymentSessionID,
		PaymentLinkURL:   paymentSessionResult.PaymentLinkURL,
	})
}
