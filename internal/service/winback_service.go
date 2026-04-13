package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/rifqimalik/cashlens-backend/internal/logger"
	"github.com/rifqimalik/cashlens-backend/internal/repository"
)

var winBackMessages = map[string]string{
	"id": "Hei! Langganan CashLens Premium kamu sudah berakhir 😢\n\nJangan sampai ketinggalan fitur AI scan struk & analisis keuangan otomatis.\n\n🎉 Dapatkan diskon 50% dengan kode *WINBACK50* — berlaku 3 hari!\n\nUpgrade sekarang: https://cashlens.app/upgrade",
	"en": "Hey! Your CashLens Premium subscription has expired 😢\n\nDon't miss out on AI receipt scanning & automatic financial analysis.\n\n🎉 Get 50% off with code *WINBACK50* — valid for 3 days!\n\nUpgrade now: https://cashlens.app/upgrade",
}

// WinBackService handles win-back campaign logic
type WinBackService interface {
	RunWinBackCampaign(ctx context.Context) (int, error)
}

type winBackService struct {
	winBackRepo   repository.WinBackRepository
	chatRepo      repository.ChatLinkRepository
	telegramToken string
	httpClient    *http.Client
	log           *logger.Logger
}

func NewWinBackService(
	winBackRepo repository.WinBackRepository,
	chatRepo repository.ChatLinkRepository,
	telegramToken string,
) WinBackService {
	return &winBackService{
		winBackRepo:   winBackRepo,
		chatRepo:      chatRepo,
		telegramToken: telegramToken,
		httpClient:    &http.Client{Timeout: 10 * time.Second},
		log:           logger.GetDefault().With("component", "win_back_service"),
	}
}

// RunWinBackCampaign sends win-back messages to eligible users.
// Returns the number of users who were sent a message.
func (s *winBackService) RunWinBackCampaign(ctx context.Context) (int, error) {
	users, err := s.winBackRepo.GetUsersEligibleForWinBack(ctx, 7, 100)
	if err != nil {
		return 0, fmt.Errorf("failed to get eligible users: %w", err)
	}

	if len(users) == 0 {
		s.log.Info("No users eligible for win-back campaign")
		return 0, nil
	}

	sent := 0
	for _, u := range users {
		chatLink, err := s.chatRepo.GetByUserID(ctx, u.ID, "telegram")
		if err != nil {
			s.log.Warn("User has no Telegram link, skipping", "user_id", u.ID)
			continue
		}

		message, ok := winBackMessages[u.Language]
		if !ok {
			message = winBackMessages["en"]
		}

		if err := s.sendTelegramMessage(chatLink.ChatID, message); err != nil {
			s.log.Error("Failed to send win-back Telegram message", "user_id", u.ID, "chat_id", chatLink.ChatID, "error", err)
			continue
		}
		s.log.Info("Win-back message sent", "user_id", u.ID, "chat_id", chatLink.ChatID, "language", u.Language)

		if err := s.winBackRepo.MarkWinBackSent(ctx, u.ID); err != nil {
			s.log.Error("Failed to mark win-back sent", "user_id", u.ID, "error", err)
			continue
		}

		sent++
	}

	s.log.Info("Win-back campaign completed", "users_sent", sent, "total_eligible", len(users))
	return sent, nil
}

func (s *winBackService) sendTelegramMessage(chatID string, text string) error {
	payload := map[string]any{
		"chat_id":    chatID,
		"text":       text,
		"parse_mode": "Markdown",
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", s.telegramToken)
	resp, err := s.httpClient.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to call Telegram API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Telegram API returned status %d", resp.StatusCode)
	}
	return nil
}
