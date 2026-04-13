package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rifqimalik/cashlens-backend/internal/errors"
	"github.com/rifqimalik/cashlens-backend/internal/models"
	"github.com/rifqimalik/cashlens-backend/internal/repository"
)

type QuotaService interface {
	CheckAndIncrementTransactionQuota(ctx context.Context, userID uuid.UUID) error
	CheckAndIncrementScanQuota(ctx context.Context, userID uuid.UUID) error
	GetCurrentUsage(ctx context.Context, userID uuid.UUID) (*models.UserQuota, error)
}

type quotaService struct {
	quotaRepo repository.QuotaRepository
	userRepo  repository.UserRepository
}

func NewQuotaService(quotaRepo repository.QuotaRepository, userRepo repository.UserRepository) QuotaService {
	return &quotaService{
		quotaRepo: quotaRepo,
		userRepo:  userRepo,
	}
}

func (s *quotaService) CheckAndIncrementTransactionQuota(ctx context.Context, userID uuid.UUID) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	// Premium users with active subscription have unlimited access
	if user.SubscriptionTier == "premium" &&
		(user.SubscriptionExpiry == nil || user.SubscriptionExpiry.After(time.Now())) {
		return nil
	}

	now := time.Now()
	month, year := int(now.Month()), now.Year()

	// Atomic check + increment: prevents TOCTOU race condition
	success, err := s.quotaRepo.IncrementTransactionsIfUnderLimit(ctx, userID, month, year, models.FreeTierLimits.MaxTransactionsPerMonth)
	if err != nil {
		return fmt.Errorf("failed to update quota: %w", err)
	}

	if !success {
		// Get current usage for error message
		quota, _ := s.quotaRepo.GetOrCreate(ctx, userID, month, year)
		used := 0
		if quota != nil {
			used = quota.TransactionsUsed
		}
		return errors.NewForbidden(
			fmt.Sprintf("Monthly transaction limit reached (%d/%d). Upgrade to premium for unlimited transactions.",
				used, models.FreeTierLimits.MaxTransactionsPerMonth),
		)
	}

	return nil
}

func (s *quotaService) CheckAndIncrementScanQuota(ctx context.Context, userID uuid.UUID) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	// Premium users with active subscription have unlimited access
	if user.SubscriptionTier == "premium" &&
		(user.SubscriptionExpiry == nil || user.SubscriptionExpiry.After(time.Now())) {
		return nil
	}

	now := time.Now()
	month, year := int(now.Month()), now.Year()

	// Atomic check + increment: prevents TOCTOU race condition
	success, err := s.quotaRepo.IncrementScansIfUnderLimit(ctx, userID, month, year, models.FreeTierLimits.MaxScansPerMonth)
	if err != nil {
		return fmt.Errorf("failed to update quota: %w", err)
	}

	if !success {
		// Get current usage for error message
		quota, _ := s.quotaRepo.GetOrCreate(ctx, userID, month, year)
		used := 0
		if quota != nil {
			used = quota.ScansUsed
		}
		return errors.NewForbidden(
			fmt.Sprintf("Monthly scan limit reached (%d/%d). Upgrade to premium for unlimited scans.",
				used, models.FreeTierLimits.MaxScansPerMonth),
		)
	}

	return nil
}

func (s *quotaService) GetCurrentUsage(ctx context.Context, userID uuid.UUID) (*models.UserQuota, error) {
	now := time.Now()
	month, year := int(now.Month()), now.Year()

	return s.quotaRepo.GetOrCreate(ctx, userID, month, year)
}
