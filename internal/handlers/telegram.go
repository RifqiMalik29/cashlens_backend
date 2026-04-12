package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	apperrors "github.com/rifqimalik/cashlens-backend/internal/errors"
	"github.com/rifqimalik/cashlens-backend/internal/models"
	"github.com/rifqimalik/cashlens-backend/internal/service"
)

type TelegramHandler struct {
	draftService service.DraftService
	botToken     string
}

func NewTelegramHandler(draftService service.DraftService, botToken string) *TelegramHandler {
	return &TelegramHandler{
		draftService: draftService,
		botToken:     botToken,
	}
}

// Webhook handles incoming Telegram webhook requests
func (h *TelegramHandler) Webhook(w http.ResponseWriter, r *http.Request) {
	var update TelegramUpdate
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		apperrors.WriteJSONError(w, "Invalid webhook payload", http.StatusBadRequest)
		return
	}

	// Only process messages
	if update.Message == nil || update.Message.Text == "" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Parse message into draft
	// Expected format: "amount description" or just "amount"
	_, err := h.parseMessage(update.Message.Chat.ID, update.Message.Text)
	if err != nil {
		h.sendReply(update.Message.Chat.ID, fmt.Sprintf("❌ Failed to parse: %v", err), w)
		return
	}

	// TODO: Map Telegram Chat ID to UserID (needs user_chat_links table)
	// For now, skip since we don't have user mapping
	h.sendReply(update.Message.Chat.ID, "✅ Draft created! Use the app to confirm.", w)
}

func (h *TelegramHandler) parseMessage(chatID int64, text string) (*models.DraftTransaction, error) {
	// Simple parser: "50000 lunch" -> amount=50000, description="lunch"
	// TODO: Improve with regex/NLP

	draft := &models.DraftTransaction{
		Source:    models.DraftSourceTelegram,
		Status:    models.DraftStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		RawData: map[string]any{
			"chat_id": chatID,
		},
	}

	// Placeholder - will be properly implemented when user mapping exists
	return draft, nil
}

func (h *TelegramHandler) sendReply(chatID int64, text string, w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"method":  "sendMessage",
		"chat_id": chatID,
		"text":    text,
	})
}

// Telegram webhook payload structures
type TelegramUpdate struct {
	UpdateID int64            `json:"update_id"`
	Message  *TelegramMessage `json:"message,omitempty"`
}

type TelegramMessage struct {
	MessageID int64        `json:"message_id"`
	Chat      TelegramChat `json:"chat"`
	Text      string       `json:"text"`
	Date      int64        `json:"date"`
}

type TelegramChat struct {
	ID int64 `json:"id"`
}
