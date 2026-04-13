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
	_ "github.com/rifqimalik/cashlens-backend/internal/logger" // For future structured logging migration
	"github.com/rifqimalik/cashlens-backend/internal/models"
	"github.com/rifqimalik/cashlens-backend/internal/repository"
	"github.com/rifqimalik/cashlens-backend/internal/service"
)

type BotService struct {
	botToken          string
	geminiAPIKey      string
	geminiModel       string
	draftSvc          service.DraftService
	transactionSvc    service.TransactionService
	budgetSvc         service.BudgetService
	draftRepo         repository.DraftRepository
	transactionRepo   repository.TransactionRepository
	budgetRepo        repository.BudgetRepository
	userRepo          repository.UserRepository
	chatRepo          repository.ChatLinkRepository
	categoryRepo      repository.CategoryRepository
	httpClient        *http.Client
}

func NewBotService(botToken string, geminiAPIKey string, geminiModel string, draftSvc service.DraftService, transactionSvc service.TransactionService, budgetSvc service.BudgetService, draftRepo repository.DraftRepository, transactionRepo repository.TransactionRepository, budgetRepo repository.BudgetRepository, userRepo repository.UserRepository, chatRepo repository.ChatLinkRepository, categoryRepo repository.CategoryRepository) *BotService {
	return &BotService{
		botToken:          botToken,
		geminiAPIKey:      geminiAPIKey,
		geminiModel:       geminiModel,
		draftSvc:          draftSvc,
		transactionSvc:    transactionSvc,
		budgetSvc:         budgetSvc,
		draftRepo:         draftRepo,
		transactionRepo:   transactionRepo,
		budgetRepo:        budgetRepo,
		userRepo:          userRepo,
		chatRepo:          chatRepo,
		categoryRepo:      categoryRepo,
		httpClient:        &http.Client{Timeout: 30 * time.Second},
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
			// Recover from panics to keep bot running
			func() {
				defer func() {
					if r := recover(); r != nil {
						log.Printf("[Telegram Bot] Panic recovered: %v", r)
					}
				}()

				updates, err := b.getUpdates(offset, 10)
				if err != nil {
					log.Printf("[Telegram Bot] Error getting updates: %v", err)
					time.Sleep(2 * time.Second)
					return
				}

				for _, update := range updates {
					offset = update.UpdateID + 1
					b.handleUpdate(update)
				}
			}()
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
	// Recover from panics to keep bot running
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[Telegram Bot] Panic in handleUpdate: %v", r)
		}
	}()

	if update.Message != nil && update.Message.Text != "" {
		chatID := update.Message.Chat.ID
		text := strings.TrimSpace(update.Message.Text)

		// Handle commands
		if strings.HasPrefix(text, "/") {
			b.handleCommand(chatID, text, update.Message.Chat.Username)
			return
		}

		// Handle regular messages (smart parsing)
		b.handleMessage(chatID, text)
		return
	}

	// Handle callback queries (inline button presses)
	if update.CallbackQuery != nil {
		b.handleCallbackQuery(*update.CallbackQuery)
	}
}

func (b *BotService) handleCallbackQuery(query CallbackQuery) {
	if query.Message.Chat.ID == 0 {
		log.Printf("[Telegram Bot] Callback query has no chat ID")
		b.answerCallbackQuery(query.ID, "Error: Invalid message")
		return
	}

	chatID := query.Message.Chat.ID
	messageID := query.Message.MessageID
	data := query.Data

	log.Printf("[Telegram Bot] Received callback: %s from chat %d", data, chatID)

	// Parse callback data: "action:draft_id" or "setcat:draft_id:category_id"
	parts := strings.Split(data, ":")
	if len(parts) < 2 || len(parts) > 3 {
		b.answerCallbackQuery(query.ID, "Invalid action")
		return
	}

	action := parts[0]
	draftID := parts[1]

	switch action {
	case "setcat":
		// Category ID is the third part: "setcat:draft_id:category_id"
		if len(parts) != 3 {
			b.answerCallbackQuery(query.ID, "Invalid category selection")
			return
		}
		categoryID := parts[2]
		b.handleSetCategory(chatID, messageID, draftID, categoryID, query.ID)
	case "quick":
		// Quick template: "quick:amount:description"
		if len(parts) < 3 {
			b.answerCallbackQuery(query.ID, "Invalid quick template")
			return
		}
		amountStr := parts[1]
		description := strings.Join(parts[2:], ":")
		b.handleQuickTemplate(chatID, messageID, amountStr, description, query.ID)
	case "confirm":
		b.handleConfirmDraft(chatID, messageID, draftID, query.ID)
	case "reject":
		b.handleRejectDraft(chatID, messageID, draftID, query.ID)
	default:
		b.answerCallbackQuery(query.ID, "Unknown action")
	}
}

func (b *BotService) handleSetCategory(chatID int64, messageID int64, shortDraftID string, shortCatID string, callbackID string) {
	link, err := b.chatRepo.GetByChatID(context.Background(), fmt.Sprintf("%d", chatID), "telegram")
	if err != nil {
		b.answerCallbackQuery(callbackID, "Account not linked")
		return
	}

	// Find draft by short ID prefix
	drafts, err := b.draftRepo.ListByUserID(context.Background(), link.UserID, models.DraftStatusPending)
	if err != nil {
		b.answerCallbackQuery(callbackID, "Failed to find draft")
		return
	}

	var draft *models.DraftTransaction
	for _, d := range drafts {
		if strings.HasPrefix(d.ID.String(), shortDraftID) {
			draft = d
			break
		}
	}
	if draft == nil {
		b.answerCallbackQuery(callbackID, "Draft not found")
		return
	}

	// Find category by short ID prefix
	categories, err := b.categoryRepo.ListByUserID(context.Background(), link.UserID)
	if err != nil {
		b.answerCallbackQuery(callbackID, "Failed to find category")
		return
	}

	var categoryID uuid.UUID
	var categoryName string
	for _, cat := range categories {
		if strings.HasPrefix(cat.ID.String(), shortCatID) {
			categoryID = cat.ID
			categoryName = cat.Name
			break
		}
	}
	if categoryID == uuid.Nil {
		b.answerCallbackQuery(callbackID, "Category not found")
		return
	}

	// Update draft with category
	draft.CategoryID = &categoryID
	err = b.draftRepo.Update(context.Background(), draft)
	if err != nil {
		log.Printf("[Telegram Bot] Failed to update draft category: %v", err)
		b.answerCallbackQuery(callbackID, "Failed to set category")
		return
	}

	b.answerCallbackQuery(callbackID, fmt.Sprintf("Category set to %s", categoryName))
	
	// Edit message to show Confirm/Reject buttons
	b.editMessageText(chatID, messageID, fmt.Sprintf("✅ Category Set: %s\n\n💰 Amount: Rp %.0f\n📝 Description: %s\n📅 Date: %s\n\nTap below to confirm or reject.", categoryName, *draft.Amount, *draft.Description, draft.TransactionDate.Format("2006-01-02")))
	
	// Show Confirm/Reject buttons
	b.showConfirmRejectButtons(chatID, draft)
}

func (b *BotService) showConfirmRejectButtons(chatID int64, draft *models.DraftTransaction) {
	b.sendReplyWithKeyboard(chatID, "Ready to confirm?", &InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				{Text: "✅ Confirm", CallbackData: fmt.Sprintf("confirm:%s", draft.ID.String())},
				{Text: "❌ Reject", CallbackData: fmt.Sprintf("reject:%s", draft.ID.String())},
			},
		},
	})
}

func (b *BotService) handleConfirmDraft(chatID int64, messageID int64, draftIDStr string, callbackID string) {
	// Recover from panic to keep bot running
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[Telegram Bot] Panic in handleConfirmDraft: %v", r)
			b.answerCallbackQuery(callbackID, "Error processing request")
		}
	}()

	draftID, err := uuid.Parse(draftIDStr)
	if err != nil {
		b.answerCallbackQuery(callbackID, "Invalid draft ID")
		return
	}

	link, err := b.chatRepo.GetByChatID(context.Background(), fmt.Sprintf("%d", chatID), "telegram")
	if err != nil {
		log.Printf("[Telegram Bot] Chat not linked: %d", chatID)
		b.answerCallbackQuery(callbackID, "Account not linked. Send /link <email>")
		return
	}

	// Get the draft
	draft, err := b.draftSvc.Get(context.Background(), draftID, link.UserID)
	if err != nil {
		log.Printf("[Telegram Bot] Draft not found: %s", draftIDStr)
		b.answerCallbackQuery(callbackID, "Draft not found")
		return
	}

	// Validate required fields
	if draft.Amount == nil || draft.Description == nil || draft.TransactionDate == nil {
		log.Printf("[Telegram Bot] Draft has nil fields: %s", draftIDStr)
		b.answerCallbackQuery(callbackID, "Invalid draft data")
		return
	}

	// Category is required for transactions
	if draft.CategoryID == nil {
		b.editMessageText(chatID, messageID, "⚠️ Category Required\n\nThis draft doesn't have a category yet.\nPlease open the app to select a category and confirm.")
		b.answerCallbackQuery(callbackID, "Category required")
		return
	}

	// Prepare confirmation request
	confirmReq := models.ConfirmDraftRequest{
		CategoryID:      *draft.CategoryID,
		Amount:          *draft.Amount,
		Description:     *draft.Description,
		TransactionDate: *draft.TransactionDate,
	}

	tx, err := b.draftSvc.Confirm(context.Background(), draftID, link.UserID, confirmReq)
	if err != nil {
		log.Printf("[Telegram Bot] Failed to confirm draft: %v", err)
		b.answerCallbackQuery(callbackID, "Failed to confirm")
		return
	}

	log.Printf("[Telegram Bot] Draft confirmed: %s -> Transaction %s", draftIDStr, tx.ID.String())

	// Edit original message to remove buttons
	b.editMessageText(chatID, messageID, fmt.Sprintf("✅ Transaction Confirmed!\n\n💰 Amount: Rp %.0f\n📝 Description: %s\n📅 Date: %s\n\nTransaction ID: %s", tx.Amount, *tx.Description, tx.TransactionDate.Format("2006-01-02"), tx.ID.String()))
	b.answerCallbackQuery(callbackID, "Transaction confirmed!")
}

func (b *BotService) handleRejectDraft(chatID int64, messageID int64, draftIDStr string, callbackID string) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[Telegram Bot] Panic in handleRejectDraft: %v", r)
			b.answerCallbackQuery(callbackID, "Error processing request")
		}
	}()

	draftID, err := uuid.Parse(draftIDStr)
	if err != nil {
		b.answerCallbackQuery(callbackID, "Invalid draft ID")
		return
	}

	link, err := b.chatRepo.GetByChatID(context.Background(), fmt.Sprintf("%d", chatID), "telegram")
	if err != nil {
		b.answerCallbackQuery(callbackID, "Account not linked")
		return
	}

	err = b.draftSvc.Delete(context.Background(), draftID, link.UserID)
	if err != nil {
		log.Printf("[Telegram Bot] Failed to reject draft: %v", err)
		b.answerCallbackQuery(callbackID, "Failed to reject")
		return
	}

	log.Printf("[Telegram Bot] Draft rejected: %s", draftIDStr)

	// Edit original message to remove buttons
	b.editMessageText(chatID, messageID, "❌ Draft Rejected\n\nThe draft has been deleted.")
	b.answerCallbackQuery(callbackID, "Draft rejected")
}

func (b *BotService) editMessageText(chatID int64, messageID int64, text string) {
	edit := EditMessageRequest{
		ChatID:      chatID,
		MessageID:   messageID,
		Text:        text,
		ReplyMarkup: nil, // Explicitly remove buttons
	}

	jsonBody, err := json.Marshal(edit)
	if err != nil {
		log.Printf("[Telegram Bot] Failed to marshal edit: %v", err)
		return
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/editMessageText", b.botToken)
	resp, err := b.httpClient.Post(url, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		log.Printf("[Telegram Bot] Failed to edit message: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("[Telegram Bot] Edit API error (status %d): %s", resp.StatusCode, string(body))
	}
}

func (b *BotService) answerCallbackQuery(callbackID string, text string) {
	answer := CallbackAnswerRequest{
		CallbackQueryID: callbackID,
		Text:            text,
		ShowAlert:       false,
	}

	jsonBody, err := json.Marshal(answer)
	if err != nil {
		log.Printf("[Telegram Bot] Failed to marshal callback answer: %v", err)
		return
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/answerCallbackQuery", b.botToken)
	resp, err := b.httpClient.Post(url, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		log.Printf("[Telegram Bot] Failed to answer callback: %v", err)
		return
	}
	defer resp.Body.Close()
}

func (b *BotService) handleCommand(chatID int64, text string, username *string) {
	parts := strings.Fields(text)
	command := strings.ToLower(parts[0])

	switch command {
	case "/start":
		b.sendReply(chatID, "👋 Welcome to CashLens Bot!\n\nI can help you track expenses on the go!\n\n📝 How to use:\n• Send any transaction (e.g., 35000 lunch)\n• I'll create a draft for you to confirm\n\n🔗 Commands:\n• /link <email> - Link your account\n• /balance - View budget status\n• /quick - Quick add transactions\n• /history - View confirmed transactions\n• /recent - View recent drafts\n• /unlink - Unlink your account\n• /help - Show this message")
	case "/help":
		b.sendReply(chatID, "📖 CashLens Bot Help\n\nAdd Transaction:\nJust send a message like:\n• 35000 lunch\n• 50000 transport grab\n\nCommands:\n• /link <email> - Link your account\n• /balance - View budget vs spending\n• /quick - Quick add common transactions\n• /history - View confirmed transactions\n• /recent - View recent drafts\n• /unlink - Unlink your account\n• /help - Show this help message")
	case "/link":
		if len(parts) < 2 {
			b.sendReply(chatID, "❌ Please provide your email:\n/link your@email.com")
			return
		}
		b.handleLink(chatID, parts[1], username)
	case "/unlink":
		b.handleUnlink(chatID)
	case "/balance":
		b.handleBalance(chatID)
	case "/quick":
		b.handleQuick(chatID)
	case "/history":
		b.handleHistory(chatID)
	case "/recent":
		b.handleRecent(chatID)
	default:
		b.sendReply(chatID, "❓ Unknown command. Send /help to see available commands.")
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

	name := "User"
	if user.Name != nil {
		name = *user.Name
	}
	b.sendReply(chatID, fmt.Sprintf("✅ Account Linked!\n\nWelcome, %s!\nYou can now send transactions directly from here.", name))
}

func (b *BotService) handleMessage(chatID int64, text string) {
	// Find user by chat_id
	link, err := b.chatRepo.GetByChatID(context.Background(), fmt.Sprintf("%d", chatID), "telegram")
	if err != nil {
		b.sendReply(chatID, "⚠️ Your account is not linked yet.\nSend /link <your-email> to get started.")
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

	// Show AI-powered category selector
	b.showAICategorySelector(chatID, draft, text)
}

func (b *BotService) showAICategorySelector(chatID int64, draft *models.DraftTransaction, originalMessage string) {
	// Get all categories for this user
	categories, err := b.categoryRepo.ListByUserID(context.Background(), draft.UserID)
	if err != nil {
		log.Printf("[Telegram Bot] Failed to get categories: %v", err)
		b.sendReply(chatID, "❌ Failed to load categories.")
		return
	}

	// Ask AI to suggest a category
	aiSuggestion := b.detectCategoryWithAI(*draft.Description, categories)

	// Send initial message with AI suggestion
	b.sendReply(chatID, fmt.Sprintf("✅ Draft Created!\n\n💰 Amount: Rp %.0f\n📝 Description: %s\n📅 Date: %s\n\n🤖 AI suggests: %s\n\nTap a category to confirm or choose another:", *draft.Amount, *draft.Description, draft.TransactionDate.Format("2006-01-02"), aiSuggestion))

	// Show category buttons
	b.showCategoryButtons(chatID, draft, categories, aiSuggestion)
}

func (b *BotService) showCategoryButtons(chatID int64, draft *models.DraftTransaction, categories []*models.Category, aiSuggestion string) {
	// Build inline keyboard with categories
	// Use short IDs (first 8 chars) to stay under 64-byte callback limit
	var keyboard []InlineKeyboardButton
	for _, cat := range categories {
		emoji := "📌"
		if cat.Name == aiSuggestion {
			emoji = "✨"
		}
		shortDraft := draft.ID.String()[:8]
		shortCat := cat.ID.String()[:8]
		keyboard = append(keyboard, InlineKeyboardButton{
			Text:         fmt.Sprintf("%s %s", emoji, cat.Name),
			CallbackData: fmt.Sprintf("setcat:%s:%s", shortDraft, shortCat),
		})
	}

	// Split into rows of 2
	var rows [][]InlineKeyboardButton
	for i := 0; i < len(keyboard); i += 2 {
		end := i + 2
		if end > len(keyboard) {
			end = len(keyboard)
		}
		rows = append(rows, keyboard[i:end])
	}

	b.sendReplyWithKeyboard(chatID, "Choose a category:", &InlineKeyboardMarkup{
		InlineKeyboard: rows,
	})
}

func (b *BotService) detectCategoryWithAI(description string, categories []*models.Category) string {
	// Build category list for AI prompt
	var catNames []string
	for _, cat := range categories {
		catNames = append(catNames, cat.Name)
	}

	prompt := fmt.Sprintf(`You are a category detection expert. Given a transaction description, choose the best matching category from this list: %s

Transaction: "%s"

Return ONLY the category name, nothing else. If no category matches well, return "Other" or the closest match.`, strings.Join(catNames, ", "), description)

	// Call Gemini API
	result, err := b.callGeminiForCategory(prompt)
	if err != nil {
		log.Printf("[Telegram Bot] AI category detection failed: %v", err)
		// Fallback to first category or "Other"
		if len(categories) > 0 {
			return categories[0].Name
		}
		return "Unknown"
	}

	// Match AI response to actual category
	result = strings.TrimSpace(result)
	for _, cat := range categories {
		if strings.EqualFold(cat.Name, result) {
			return cat.Name
		}
	}

	// If no exact match, return the AI suggestion anyway (user can correct)
	return result
}

func (b *BotService) callGeminiForCategory(prompt string) (string, error) {
	if b.geminiAPIKey == "" {
		return "", fmt.Errorf("Gemini API key not configured")
	}

	requestBody := GeminiTextRequest{
		Contents: []GeminiContent{
			{
				Parts: []GeminiPart{
					{Text: prompt},
				},
			},
		},
		GenerationConfig: &GeminiGenerationConfig{
			Temperature: 0.1,
		},
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", b.geminiModel, b.geminiAPIKey)
	
	resp, err := b.httpClient.Post(url, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to call Gemini API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Gemini API error (status %d): %s", resp.StatusCode, string(body))
	}

	var geminiResp GeminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&geminiResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("empty response from Gemini API")
	}

	return geminiResp.Candidates[0].Content.Parts[0].Text, nil
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

// Feature #2: Budget Status
func (b *BotService) handleBalance(chatID int64) {
	link, err := b.chatRepo.GetByChatID(context.Background(), fmt.Sprintf("%d", chatID), "telegram")
	if err != nil {
		b.sendReply(chatID, "⚠️ Your account is not linked yet.\nSend /link <your-email> to get started.")
		return
	}

	// Get current month budgets
	now := time.Now()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	endOfMonth := startOfMonth.AddDate(0, 1, -1)

	budgets, err := b.budgetSvc.List(context.Background(), link.UserID)
	if err != nil {
		b.sendReply(chatID, "❌ Failed to fetch budgets.")
		return
	}

	if len(budgets) == 0 {
		b.sendReply(chatID, "📭 No budgets set for this month.\nUse the app to create budgets.")
		return
	}

	msg := "💰 Budget Status (This Month):\n\n"
	for _, budget := range budgets {
		if budget.StartDate.After(endOfMonth) || budget.EndDate.Before(startOfMonth) {
			continue
		}

		// Calculate spent amount for this category this month
		transactions, err := b.transactionSvc.ListByDateRange(context.Background(), link.UserID, startOfMonth, endOfMonth)
		if err != nil {
			continue
		}

		spent := 0.0
		for _, tx := range transactions {
			if tx.CategoryID == budget.CategoryID {
				spent += tx.Amount
			}
		}

		percentage := 0.0
		if budget.Amount > 0 {
			percentage = (spent / budget.Amount) * 100
		}

		status := "✅"
		if percentage >= 90 {
			status = "🔴"
		} else if percentage >= 70 {
			status = "🟡"
		}

		msg += fmt.Sprintf("%s %s\n   Spent: Rp %.0f / Rp %.0f (%.0f%%)\n\n", status, budget.Category.Name, spent, budget.Amount, percentage)
	}

	b.sendReply(chatID, msg)
}

// Feature #3: Quick Templates
func (b *BotService) handleQuick(chatID int64) {
	// Check if user is linked
	_, err := b.chatRepo.GetByChatID(context.Background(), fmt.Sprintf("%d", chatID), "telegram")
	if err != nil {
		b.sendReply(chatID, "⚠️ Your account is not linked yet.\nSend /link <your-email> to get started.")
		return
	}

	// Common quick templates
	templates := []struct {
		Amount      float64
		Description string
	}{
		{25000, "Grab food"},
		{50000, "Bensin"},
		{35000, "Kopi"},
		{150000, "Makan siang"},
		{10000, "Parkir"},
		{200000, "Belanja bulanan"},
	}

	// Build inline keyboard
	var keyboard []InlineKeyboardButton
	for _, tmpl := range templates {
		keyboard = append(keyboard, InlineKeyboardButton{
			Text:         fmt.Sprintf("💰 Rp %.0f - %s", tmpl.Amount, tmpl.Description),
			CallbackData: fmt.Sprintf("quick:%.0f:%s", tmpl.Amount, tmpl.Description),
		})
	}

	// Split into rows of 1 (full width buttons)
	var rows [][]InlineKeyboardButton
	for i := 0; i < len(keyboard); i++ {
		rows = append(rows, []InlineKeyboardButton{keyboard[i]})
	}

	b.sendReplyWithKeyboard(chatID, "⚡ Quick Add Transaction:\n\nTap to create instantly:", &InlineKeyboardMarkup{
		InlineKeyboard: rows,
	})
}

// Handle quick template button press
func (b *BotService) handleQuickTemplate(chatID int64, messageID int64, amountStr string, description string, callbackID string) {
	link, err := b.chatRepo.GetByChatID(context.Background(), fmt.Sprintf("%d", chatID), "telegram")
	if err != nil {
		b.answerCallbackQuery(callbackID, "Account not linked")
		return
	}

	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil {
		b.answerCallbackQuery(callbackID, "Invalid amount")
		return
	}

	now := time.Now()
	desc := description
	draftReq := models.CreateDraftRequest{
		Amount:          &amount,
		Description:     &desc,
		TransactionDate: &now,
		Source:          models.DraftSourceTelegram,
	}

	draft, err := b.draftSvc.Create(context.Background(), link.UserID, draftReq)
	if err != nil {
		b.answerCallbackQuery(callbackID, "Failed to create draft")
		return
	}

	b.answerCallbackQuery(callbackID, "Draft created!")
	b.showAICategorySelector(chatID, draft, description)
}

// Feature #4: Transaction History
func (b *BotService) handleHistory(chatID int64) {
	link, err := b.chatRepo.GetByChatID(context.Background(), fmt.Sprintf("%d", chatID), "telegram")
	if err != nil {
		b.sendReply(chatID, "⚠️ Your account is not linked yet.\nSend /link <your-email> to get started.")
		return
	}

	// Get transactions from last 7 days
	now := time.Now()
	weekAgo := now.AddDate(0, 0, -7)

	transactions, err := b.transactionSvc.ListByDateRange(context.Background(), link.UserID, weekAgo, now)
	if err != nil {
		b.sendReply(chatID, "❌ Failed to fetch transactions.")
		return
	}

	if len(transactions) == 0 {
		b.sendReply(chatID, "📭 No transactions in the last 7 days.")
		return
	}

	totalSpent := 0.0
	msg := "📊 Recent Transactions (Last 7 Days):\n\n"
	for i, tx := range transactions[:min(10, len(transactions))] {
		msg += fmt.Sprintf("%d. %s - Rp %.0f\n   📅 %s | %s\n\n", i+1, tx.Category.Name, tx.Amount, tx.TransactionDate.Format("Jan 2"), *tx.Description)
		totalSpent += tx.Amount
	}

	msg += fmt.Sprintf("💰 Total Spent: Rp %.0f", totalSpent)
	b.sendReply(chatID, msg)
}

// Feature #5: Unlink Account
func (b *BotService) handleUnlink(chatID int64) {
	link, err := b.chatRepo.GetByChatID(context.Background(), fmt.Sprintf("%d", chatID), "telegram")
	if err != nil {
		b.sendReply(chatID, "⚠️ Your account is not linked yet.")
		return
	}

	err = b.chatRepo.Delete(context.Background(), link.ID)
	if err != nil {
		b.sendReply(chatID, "❌ Failed to unlink account.")
		return
	}

	b.sendReply(chatID, "✅ Account Unlinked!\n\nYour Telegram account has been disconnected from CashLens.\nSend /link <email> to reconnect.")
}

func (b *BotService) sendReply(chatID int64, text string) {
	b.sendReplyWithKeyboard(chatID, text, nil)
}

func (b *BotService) sendReplyWithKeyboard(chatID int64, text string, keyboard *InlineKeyboardMarkup) {
	reply := SendMessageRequest{
		ChatID:        chatID,
		Text:          text,
		ParseMode:     "", // Plain text
		ReplyMarkup:   keyboard,
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

// Gemini API types
type GeminiTextRequest struct {
	Contents         []GeminiContent         `json:"contents"`
	GenerationConfig *GeminiGenerationConfig `json:"generationConfig,omitempty"`
}

type GeminiContent struct {
	Parts []GeminiPart `json:"parts"`
}

type GeminiPart struct {
	Text       string           `json:"text,omitempty"`
	InlineData *GeminiImageData `json:"inlineData,omitempty"`
}

type GeminiImageData struct {
	MimeType string `json:"mimeType"`
	Data     string `json:"data"`
}

type GeminiGenerationConfig struct {
	Temperature float64 `json:"temperature"`
}

type GeminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

// Telegram API types
type APIResponse struct {
	OK     bool     `json:"ok"`
	Result []Update `json:"result"`
}

type Update struct {
	UpdateID      int64          `json:"update_id"`
	Message       *Message       `json:"message,omitempty"`
	CallbackQuery *CallbackQuery `json:"callback_query,omitempty"`
}

type CallbackQuery struct {
	ID      string   `json:"id"`
	Message Message  `json:"message"`
	Data    string   `json:"data"`
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
	ChatID        int64                `json:"chat_id"`
	Text          string               `json:"text"`
	ParseMode     string               `json:"parse_mode,omitempty"`
	ReplyMarkup   *InlineKeyboardMarkup `json:"reply_markup,omitempty"`
}

type InlineKeyboardMarkup struct {
	InlineKeyboard [][]InlineKeyboardButton `json:"inline_keyboard"`
}

type InlineKeyboardButton struct {
	Text         string `json:"text"`
	CallbackData string `json:"callback_data,omitempty"`
}

type CallbackAnswerRequest struct {
	CallbackQueryID string `json:"callback_query_id"`
	Text            string `json:"text,omitempty"`
	ShowAlert       bool   `json:"show_alert,omitempty"`
}

type EditMessageRequest struct {
	ChatID          int64                `json:"chat_id"`
	MessageID       int64                `json:"message_id"`
	Text            string               `json:"text"`
	ParseMode       string               `json:"parse_mode,omitempty"`
	ReplyMarkup     *InlineKeyboardMarkup `json:"reply_markup,omitempty"`
}
