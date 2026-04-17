package service

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5" // New import
	"github.com/rifqimalik/cashlens-backend/internal/models"
	"github.com/rifqimalik/cashlens-backend/internal/repository"
)

// TrialEligibilityService defines the interface for managing free trial eligibility.
type TrialEligibilityService interface {
	CheckAndSetTrial(user *models.User, newDeviceID *string) (bool, error)
}

// trialEligibilityService implements TrialEligibilityService.
type trialEligibilityService struct {
	userRepo repository.UserRepository
}

// NewTrialEligibilityService creates a new TrialEligibilityService.
func NewTrialEligibilityService(userRepo repository.UserRepository) TrialEligibilityService {
	return &trialEligibilityService{
		userRepo: userRepo,
	}
}

// CheckAndSetTrial performs the core eligibility logic and sets trial status for a user.
// It returns true if a trial was granted/is active, false otherwise.
func (s *trialEligibilityService) CheckAndSetTrial(user *models.User, newDeviceID *string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// If the user already has an active trial, return true
	if user.TrialStatus == "active" && user.TrialEndAt != nil && user.TrialEndAt.After(time.Now()) {
		return true, nil
	}

	// Check if this email has ever had a trial (active, expired, or denied)
	existingUserByEmail, err := s.userRepo.GetByEmail(ctx, user.Email) // Use GetByEmail from updated repo
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return false, err // Propagate actual errors
	}
	emailHasHadTrial := existingUserByEmail != nil && (existingUserByEmail.TrialStatus == "active" || existingUserByEmail.TrialStatus == "expired" || existingUserByEmail.TrialStatus == "denied")

	// Check if this device ID has ever had a trial (active, expired, or denied)
	var deviceHasHadTrial bool
	if newDeviceID != nil && *newDeviceID != "" {
		usersByDeviceID, err := s.userRepo.GetByDeviceID(ctx, *newDeviceID) // Use GetByDeviceID from updated repo
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {                   // GetByDeviceID returns slice, so ErrNotFound not expected
			return false, err // Propagate actual errors
		}
		for _, u := range usersByDeviceID {
			if u.TrialStatus == "active" || u.TrialStatus == "expired" || u.TrialStatus == "denied" {
				deviceHasHadTrial = true
				break
			}
		}
	}

	if emailHasHadTrial || deviceHasHadTrial {
		// If either the email or the device has previously had a trial, deny a new one
		user.TrialStatus = "denied"
		if newDeviceID != nil && user.DeviceID == nil { // Associate device ID even if denied
			user.DeviceID = newDeviceID
		}
		if err := s.userRepo.Update(ctx, user); err != nil {
			return false, err
		}
		return false, nil
	}

	// Grant new trial
	now := time.Now().UTC()
	user.TrialStartAt = &now
	trialEnd := now.Add(7 * 24 * time.Hour)
	user.TrialEndAt = &trialEnd
	user.TrialStatus = "active"
	if newDeviceID != nil {
		user.DeviceID = newDeviceID
	}
	if err := s.userRepo.Update(ctx, user); err != nil {
		return false, err
	}

	return true, nil
}
