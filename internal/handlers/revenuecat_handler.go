package handlers

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/rifqimalik/cashlens-backend/internal/config"
	"github.com/rifqimalik/cashlens-backend/internal/logger"
	"github.com/rifqimalik/cashlens-backend/internal/models"
)

// RevenueCatProcessor defines the interface for processing RevenueCat webhook events.
type RevenueCatProcessor interface {
	ProcessWebhook(ctx context.Context, event *models.RevenueCatEvent) error
}

// RevenueCatHandler handles incoming webhooks from RevenueCat
type RevenueCatHandler struct {
	service RevenueCatProcessor
	config  *config.Config
	logger  *logger.Logger
}

// NewRevenueCatHandler creates a new instance of RevenueCatHandler
func NewRevenueCatHandler(s RevenueCatProcessor, c *config.Config, l *logger.Logger) *RevenueCatHandler {
	return &RevenueCatHandler{
		service: s,
		config:  c,
		logger:  l,
	}
}

// Webhook is the HTTP handler for processing RevenueCat webhooks
func (h *RevenueCatHandler) Webhook(w http.ResponseWriter, r *http.Request) {
	// 1. Verify the signature
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		h.logger.Warn("RevenueCat webhook received with no Authorization header")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	tokenParts := strings.Split(authHeader, "Bearer ")
	if len(tokenParts) != 2 {
		h.logger.Warn("RevenueCat webhook received with malformed Authorization header")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	token := tokenParts[1]
	if h.config.Payment.RevenueCatWebhookSecret == "" {
		h.logger.Error("RevenueCat webhook secret is not configured on the server")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	
	if subtle.ConstantTimeCompare([]byte(token), []byte(h.config.Payment.RevenueCatWebhookSecret)) != 1 {
		h.logger.Error("RevenueCat webhook received with invalid secret token")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// 2. Decode the payload
	var webhook models.RevenueCatWebhook
	if err := json.NewDecoder(r.Body).Decode(&webhook); err != nil {
		h.logger.Error("Failed to decode RevenueCat webhook payload", "error", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// 3. Process the event
	if err := h.service.ProcessWebhook(r.Context(), &webhook.Event); err != nil {
		h.logger.Error("Failed to process RevenueCat webhook", "event_type", webhook.Event.Type, "user_id", webhook.Event.AppUserID, "error", err)
		// Return a 500 so RevenueCat will retry the webhook
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	h.logger.Info("Successfully processed RevenueCat webhook", "event_type", webhook.Event.Type, "user_id", webhook.Event.AppUserID)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
