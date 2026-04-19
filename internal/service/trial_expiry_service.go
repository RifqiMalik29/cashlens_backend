package service

import (
	"context"
	"log/slog"

	"github.com/rifqimalik/cashlens-backend/internal/models"
	"github.com/rifqimalik/cashlens-backend/internal/pkg/mailer"
)

type TrialExpiryService interface {
	ExpireTrials(ctx context.Context) (int, error)
}

// trialUserRepository is the subset of UserRepository needed by TrialExpiryService.
type trialUserRepository interface {
	GetExpiredTrialUsers(ctx context.Context) ([]*models.User, error)
	Update(ctx context.Context, user *models.User) error
}

type trialExpiryService struct {
	userRepo trialUserRepository
	mailer   mailer.Mailer
}

func NewTrialExpiryService(userRepo trialUserRepository, mailer mailer.Mailer) TrialExpiryService {
	return &trialExpiryService{
		userRepo: userRepo,
		mailer:   mailer,
	}
}

func (s *trialExpiryService) ExpireTrials(ctx context.Context) (int, error) {
	users, err := s.userRepo.GetExpiredTrialUsers(ctx)
	if err != nil {
		return 0, err
	}

	count := 0
	for _, user := range users {
		if err := s.expireUser(ctx, user); err != nil {
			slog.Error("Failed to expire trial for user", "user_id", user.ID, "error", err)
		} else {
			count++
		}
	}
	return count, nil
}

func (s *trialExpiryService) expireUser(ctx context.Context, user *models.User) error {
	user.TrialStatus = "expired"
	if err := s.userRepo.Update(ctx, user); err != nil {
		return err
	}
	if err := s.mailer.SendTrialExpiredEmail(user.Email); err != nil {
		slog.Error("Failed to send trial expired email", "email", user.Email, "error", err)
	}
	return nil
}
