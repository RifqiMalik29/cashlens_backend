package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rifqimalik/cashlens-backend/internal/models"
	"github.com/rifqimalik/cashlens-backend/internal/repository"
	"github.com/rifqimalik/cashlens-backend/internal/service"
)

type BotService struct {
	botToken   string
	draftSvc   service.DraftService
	userRepo   repository.UserRepository
	chatRepo   repository.ChatLinkRepository
	httpClient *http.Client
}

func NewBotService(botToken string, draftSvc service.DraftService, userRepo repository.UserRepository, chatRepo repository.ChatLinkRepository) *BotService {
	return &BotService{
		botToken:   botToken,
		draftSvc:   draftSvc,
		userRepo:   userRepo,
		chatRepo:   chatRepo,
		httpClient: &http.Client{Timeout: 30 * time.Second}, // Increased for long polling
	}
}

// StartPolling begins long polling for updates
func (b *BotService) StartPolling(ctx context.Context) {
	log.Println("[Telegram Bot] Starting polling...")
	offset := int64(0)

	for {
		select {
		case <-ctx.Done():
			log.Println("[Telegram Bot] Polling stopped")
			return
		default:
			updates, err := b.getUpdates(offset, 10)
			if err != nil {
				log.Printf("[Telegram Bot] Error getting updates: %v", err)
				time.Sleep(2 * time.Second)
				continue
			}

			for _, update := range updates {
				offset = update.UpdateID + 1
				b.handleUpdate(update)
			}
		}
	}
}

func (b *BotService) getUpdates(offset int64, timeout int) ([]Update, error) {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/getUpdates?offset=%d&timeout=%d", b.botToken, offset, timeout)

	resp, err := b.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get updates: %w", err)
	}
	defer resp.Body.Close()

	var apiResp APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return apiResp.Result, nil
}

func (b *BotService) handleUpdate(update Update) {
	if update.Message == nil || update.Message.Text == "" {
		return
	}

	chatID := update.Message.Chat.ID
	text := strings.TrimSpace(update.Message.Text)

	// Handle commands
	if strings.HasPrefix(text, "/") {
		b.handleCommand(chatID, text, update.Message.Chat.Username)
		return
	}

	// Handle regular messages (smart parsing)
	b.handleMessage(chatID, text)
}

func (b *BotService) handleCommand(chatID int64, text string, username *string) {
	parts := strings.Fields(text)
	command := strings.ToLower(parts[0])

	switch command {
	case "/start":
		b.sendReply(chatID, "👋 *Welcome to CashLens Bot!*\n\nI can help you track expenses on the go!\n\n📝 *How to use:*\n• Send any transaction (e.g., `35000 lunch`)\n• I'll create a draft for you to confirm in the app\n\n🔗 *Commands:*\n• `/link <email>` - Link your account\n• `/recent` - View recent drafts\n• `/help` - Show this message")
	case "/help":
		b.sendReply(chatID, "📖 *CashLens Bot Help*\n\n*Add Transaction:*\nJust send a message like:\n• `35000 lunch`\n• `50000 transport grab`\n• `120000 shopping shoes`\n\n*Commands:*\n• `/link <email>` - Link your CashLens account\n• `/recent` - View your recent drafts\n• `/help` - Show this help message")
	case "/link":
		if len(parts) < 2 {
			b.sendReply(chatID, "❌ Please provide your email:\n`/link your@email.com`")
			return
		}
		b.handleLink(chatID, parts[1], username)
	case "/recent":
		b.handleRecent(chatID)
	default:
		b.sendReply(chatID, "❓ Unknown command. Send `/help` to see available commands.")
	}
}

func (b *BotService) handleLink(chatID int64, email string, username *string) {
	user, err := b.userRepo.GetByEmail(context.Background(), email)
	if err != nil {
		b.sendReply(chatID, "❌ Email not found. Please register first in the app.")
		return
	}

	link := &models.UserChatLink{
		ID:        uuid.New(),
		UserID:    user.ID,
		Platform:  "telegram",
		ChatID:    fmt.Sprintf("%d", chatID),
		Username:  username,
		IsActive:  true,
		LinkedAt:  time.Now(),
		UpdatedAt: time.Now(),
	}

	err = b.chatRepo.Create(context.Background(), link)
	if err != nil {
		b.sendReply(chatID, "❌ Failed to link account. It may already be linked.")
		return
	}

	b.sendReply(chatID, fmt.Sprintf("✅ Account Linked!\n\nWelcome, %s!\nYou can now send transactions directly from here.", user.Name))
}

func (b *BotService) handleMessage(chatID int64, text string) {
	// Find user by chat_id
	link, err := b.chatRepo.GetByChatID(context.Background(), fmt.Sprintf("%d", chatID), "telegram")
	if err != nil {
		b.sendReply(chatID, "⚠️ Your account is not linked yet.\nSend `/link <your-email>` to get started.")
		return
	}

	// Smart parse the message
	parsed := b.smartParse(text)

	// Create draft
	draftReq := models.CreateDraftRequest{
		CategoryID:      parsed.CategoryID,
		Amount:          &parsed.Amount,
		Description:     &parsed.Description,
		TransactionDate: &parsed.Date,
		Source:          models.DraftSourceTelegram,
		RawData: map[string]any{
			"message_text": text,
			"parsed_by":    "smart_parser",
		},
	}

	draft, err := b.draftSvc.Create(context.Background(), link.UserID, draftReq)
	if err != nil {
		b.sendReply(chatID, fmt.Sprintf("❌ Failed to create draft: %v", err))
		return
	}

	desc := *draft.Description
	dateStr := draft.TransactionDate.Format("2006-01-02")
	b.sendReply(chatID, fmt.Sprintf("✅ Draft Created!\n\n💰 Amount: Rp %.0f\n📝 Description: %s\n📅 Date: %s\n\nOpen the app to confirm.", *draft.Amount, desc, dateStr))
}

type ParsedMessage struct {
	Amount      float64
	Description string
	CategoryID  *uuid.UUID
	Date        time.Time
}

func (b *BotService) smartParse(text string) ParsedMessage {
	// Pattern: "amount description" or "amount description category"
	// Examples: "35000 lunch", "50000 transport grab", "120000 shopping shoes"
	
	amountRegex := regexp.MustCompile(`^(\d+(?:[.,]\d+)?)\s+(.+)$`)
	matches := amountRegex.FindStringSubmatch(strings.TrimSpace(text))

	result := ParsedMessage{
		Date: time.Now().Truncate(24 * time.Hour),
	}

	if len(matches) >= 3 {
		amountStr := strings.ReplaceAll(matches[1], ",", "")
		amount, err := strconv.ParseFloat(amountStr, 64)
		if err == nil {
			result.Amount = amount
		}

		description := matches[2]
		result.Description = description

		// Try to detect category from keywords
		lower := strings.ToLower(description)
		if strings.Contains(lower, "makan") || strings.Contains(lower, "lunch") || strings.Contains(lower, "dinner") || strings.Contains(lower, "breakfast") || strings.Contains(lower, "kopi") || strings.Contains(lower, "coffee") {
			// cat_food - but we don't have category lookup here, so leave nil
			// User can adjust in app
		} else if strings.Contains(lower, "transport") || strings.Contains(lower, "grab") || strings.Contains(lower, "gojek") || strings.Contains(lower, "bensin") || strings.Contains(lower, "parkir") {
			// cat_transport
		} else if strings.Contains(lower, "belanja") || strings.Contains(lower, "shopping") || strings.Contains(lower, "baju") || strings.Contains(lower, "elektronik") {
			// cat_shopping
		}
	} else {
		// Fallback: try to just extract amount
		result.Description = text
	}

	return result
}

func (b *BotService) handleRecent(chatID int64) {
	link, err := b.chatRepo.GetByChatID(context.Background(), fmt.Sprintf("%d", chatID), "telegram")
	if err != nil {
		b.sendReply(chatID, "⚠️ Your account is not linked yet.\nSend `/link <your-email>` to get started.")
		return
	}

	drafts, err := b.draftSvc.List(context.Background(), link.UserID, models.DraftStatusPending)
	if err != nil {
		b.sendReply(chatID, "❌ Failed to fetch recent drafts.")
		return
	}

	if len(drafts) == 0 {
		b.sendReply(chatID, "📭 No pending drafts found.")
		return
	}

	msg := "📋 Recent Drafts:\n\n"
	for i, d := range drafts[:min(5, len(drafts))] {
		msg += fmt.Sprintf("%d. Rp %.0f - %s\n", i+1, *d.Amount, *d.Description)
	}
	msg += "\nOpen the app to confirm."

	b.sendReply(chatID, msg)
}

func (b *BotService) sendReply(chatID int64, text string) {
	reply := SendMessageRequest{
		ChatID:    chatID,
		Text:      text,
		ParseMode: "", // Plain text - no markdown parsing issues
	}

	jsonBody, err := json.Marshal(reply)
	if err != nil {
		log.Printf("[Telegram Bot] Failed to marshal reply: %v", err)
		return
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", b.botToken)
	resp, err := b.httpClient.Post(url, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		log.Printf("[Telegram Bot] Failed to send reply: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("[Telegram Bot] API error (status %d): %s", resp.StatusCode, string(body))
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Telegram API types
type APIResponse struct {
	OK     bool     `json:"ok"`
	Result []Update `json:"result"`
}

type Update struct {
	UpdateID int64   `json:"update_id"`
	Message  *Message `json:"message,omitempty"`
}

type Message struct {
	MessageID int64  `json:"message_id"`
	Chat      Chat   `json:"chat"`
	Text      string `json:"text"`
	Date      int64  `json:"date"`
}

type Chat struct {
	ID       int64   `json:"id"`
	Username *string `json:"username,omitempty"`
}

type SendMessageRequest struct {
	ChatID    int64  `json:"chat_id"`
	Text      string `json:"text"`
	ParseMode string `json:"parse_mode,omitempty"`
}
