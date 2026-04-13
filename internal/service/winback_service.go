package service

import (
	"context"
	"fmt"

	"github.com/rifqimalik/cashlens-backend/internal/logger"
	"github.com/rifqimalik/cashlens-backend/internal/repository"
)

// WinBackService handles win-back campaign logic
type WinBackService interface {
	RunWinBackCampaign(ctx context.Context) (int, error)
}

type winBackService struct {
	winBackRepo   repository.WinBackRepository
	chatRepo      repository.ChatLinkRepository
	telegramToken string
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
		log:           logger.GetDefault().With("component", "win_back_service"),
	}
}

// RunWinBackCampaign sends win-back messages to eligible users
// Returns the number of users who were sent a message
func (s *winBackService) RunWinBackCampaign(ctx context.Context) (int, error) {
	// Get users whose subscription expired 7 days ago
	userIDs, err := s.winBackRepo.GetUsersEligibleForWinBack(ctx, 7, 100)
	if err != nil {
		return 0, fmt.Errorf("failed to get eligible users: %w", err)
	}

	if len(userIDs) == 0 {
		s.log.Info("No users eligible for win-back campaign")
		return 0, nil
	}

	sent := 0
	for _, userID := range userIDs {
		// Get user's Telegram chat ID
		chatLink, err := s.chatRepo.GetByUserID(ctx, userID, "telegram")
		if err != nil {
			s.log.Warn("User has no Telegram link, skipping", "user_id", userID)
			continue
		}

		// TODO: Send Telegram message with 50% discount offer
		// Message template:
		// "Kamu melewatkan CashLens Premium! 🎉 Dapatkan diskon 50% dengan kode WINBACK50. Upgrade sekarang: https://cashlens.app/upgrade"

		// For now, just log and mark as sent
		s.log.Info("Would send win-back message",
			"user_id", userID,
			"chat_id", chatLink.ChatID,
		)

		// Mark win-back as sent (prevents duplicate messages)
		if err := s.winBackRepo.MarkWinBackSent(ctx, userID); err != nil {
			s.log.Error("Failed to mark win-back sent", "user_id", userID, "error", err)
			continue
		}

		sent++
	}

	s.log.Info("Win-back campaign completed", "users_sent", sent, "total_eligible", len(userIDs))
	return sent, nil
}
