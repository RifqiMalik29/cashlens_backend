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

type reminderMessages struct {
	telegram  string
	pushTitle string
	pushBody  string
}

var expiryMessages = map[int]map[string]reminderMessages{
	3: {
		"id": {
			telegram:  "Hei! Langganan CashLens Premium kamu akan berakhir dalam 3 hari 😢\n\nJangan sampai ketinggalan fitur AI scan struk & analisis keuangan otomatis.\n\n🎉 Perpanjang sekarang dan nikmati terus semua fitur premium!\n\nPerbarui langganan: https://cashlens.app/upgrade",
			pushTitle: "Langganan Hampir Berakhir 😢",
			pushBody:  "Premium kamu berakhir dalam 3 hari! Perpanjang sekarang.",
		},
		"en": {
			telegram:  "Hey! Your CashLens Premium subscription expires in 3 days 😢\n\nDon't miss out on AI receipt scanning & automatic financial analysis.\n\n🎉 Renew now and keep enjoying all premium features!\n\nRenew here: https://cashlens.app/upgrade",
			pushTitle: "Subscription Expiring Soon 😢",
			pushBody:  "Your Premium expires in 3 days! Renew now.",
		},
	},
	1: {
		"id": {
			telegram:  "⚠️ Langganan CashLens Premium kamu berakhir BESOK!\n\nSegera perpanjang agar tidak kehilangan akses ke AI scan struk & analisis keuangan otomatis.\n\n🎉 Perpanjang sekarang: https://cashlens.app/upgrade",
			pushTitle: "Langganan Berakhir Besok! ⚠️",
			pushBody:  "Premium kamu berakhir besok! Jangan sampai kehabisan.",
		},
		"en": {
			telegram:  "⚠️ Your CashLens Premium subscription expires TOMORROW!\n\nRenew now to keep access to AI receipt scanning & automatic financial analysis.\n\n🎉 Renew now: https://cashlens.app/upgrade",
			pushTitle: "Subscription Expires Tomorrow! ⚠️",
			pushBody:  "Your Premium expires tomorrow! Don't lose access.",
		},
	},
}

type ExpiryReminderService interface {
	RunReminders(ctx context.Context, daysBeforeExpiry int) (int, error)
}

type expiryReminderService struct {
	reminderRepo  repository.ReminderRepository
	chatRepo      repository.ChatLinkRepository
	telegramToken string
	httpClient    *http.Client
	log           *logger.Logger
}

func NewExpiryReminderService(
	reminderRepo repository.ReminderRepository,
	chatRepo repository.ChatLinkRepository,
	telegramToken string,
) ExpiryReminderService {
	return &expiryReminderService{
		reminderRepo:  reminderRepo,
		chatRepo:      chatRepo,
		telegramToken: telegramToken,
		httpClient:    &http.Client{Timeout: 10 * time.Second},
		log:           logger.GetDefault().With("component", "expiry_reminder_service"),
	}
}

// RunReminders sends expiry reminders to eligible users daysBeforeExpiry days before their subscription ends.
// Returns the number of users successfully notified on at least one channel.
func (s *expiryReminderService) RunReminders(ctx context.Context, daysBeforeExpiry int) (int, error) {
	users, err := s.reminderRepo.GetUsersEligibleForReminder(ctx, daysBeforeExpiry)
	if err != nil {
		return 0, fmt.Errorf("failed to get eligible users: %w", err)
	}

	if len(users) == 0 {
		s.log.Info("No users eligible for expiry reminder", "days_before", daysBeforeExpiry)
		return 0, nil
	}

	sent := 0
	for _, u := range users {
		msgs, ok := expiryMessages[daysBeforeExpiry][u.Language]
		if !ok {
			msgs = expiryMessages[daysBeforeExpiry]["en"]
		}

		atLeastOne := false

		// Try Telegram
		chatLink, err := s.chatRepo.GetByUserID(ctx, u.ID, "telegram")
		if err == nil {
			if err := s.sendTelegramMessage(chatLink.ChatID, msgs.telegram); err != nil {
				s.log.Error("Failed to send reminder via Telegram", "user_id", u.ID, "error", err)
			} else {
				s.log.Info("Reminder sent via Telegram", "user_id", u.ID, "days_before", daysBeforeExpiry)
				atLeastOne = true
			}
		}

		// Try Expo push
		if u.ExpoPushToken != "" {
			if err := s.sendExpoPush(u.ExpoPushToken, msgs.pushTitle, msgs.pushBody); err != nil {
				s.log.Error("Failed to send reminder via Expo push", "user_id", u.ID, "error", err)
			} else {
				s.log.Info("Reminder sent via Expo push", "user_id", u.ID, "days_before", daysBeforeExpiry)
				atLeastOne = true
			}
		}

		if !atLeastOne {
			s.log.Warn("No channel succeeded for user, will retry next run", "user_id", u.ID)
			continue
		}

		if err := s.reminderRepo.MarkReminderSent(ctx, u.ID, daysBeforeExpiry); err != nil {
			s.log.Error("Failed to mark reminder sent", "user_id", u.ID, "error", err)
			continue
		}

		sent++
	}

	s.log.Info("Expiry reminder run completed", "days_before", daysBeforeExpiry, "users_sent", sent, "total_eligible", len(users))
	return sent, nil
}

func (s *expiryReminderService) sendTelegramMessage(chatID string, text string) error {
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

func (s *expiryReminderService) sendExpoPush(token, title, body string) error {
	payload := map[string]any{
		"to":    token,
		"title": title,
		"body":  body,
	}
	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal push payload: %w", err)
	}
	resp, err := s.httpClient.Post("https://exp.host/push/send", "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to call Expo push API: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Expo push API returned status %d", resp.StatusCode)
	}
	return nil
}
