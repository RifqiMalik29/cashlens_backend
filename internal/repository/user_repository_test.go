package repository

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rifqimalik/cashlens-backend/internal/models"
)

func TestUserRepository(t *testing.T) {
	// TODO: Implement integration tests with test database
	t.Skip("Integration tests require database setup")
}

func TestUserRepository_GetExpiredTrialUsers(t *testing.T) {
	// This is an integration test — requires a real DB.
	// Skip if no DATABASE_URL set.
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL not set")
	}

	// Connect to the test database
	ctx := context.Background()
	db, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	repo := NewUserRepository(db)

	// Setup: create a user with trial_status='active' and trial_end_at in the past
	userWithExpiredTrial := &models.User{
		ID:                     uuid.New(),
		Email:                  "expired-trial@test.com",
		PasswordHash:           "hash1",
		Name:                   "Expired Trial User",
		Language:               "en",
		ExpoPushToken:          "",
		SubscriptionTier:       "free",
		SubscriptionExpiry:     nil,
		IsFounder:              false,
		IsConfirmed:            true,
		ConfirmationToken:      "",
		ConfirmationExpiresAt:  nil,
		DeviceID:               "",
		TrialStartAt:           ptrTime(time.Now().AddDate(0, 0, -30)),
		TrialEndAt:             ptrTime(time.Now().AddDate(0, 0, -1)), // Expired 1 day ago
		TrialStatus:            "active",
		CreatedAt:              time.Now(),
		UpdatedAt:              time.Now(),
	}

	err = repo.Create(ctx, userWithExpiredTrial)
	if err != nil {
		t.Fatalf("failed to create user with expired trial: %v", err)
	}

	// Create a user with trial_status='active' and future trial_end_at
	userWithFutureTrial := &models.User{
		ID:                     uuid.New(),
		Email:                  "future-trial@test.com",
		PasswordHash:           "hash2",
		Name:                   "Future Trial User",
		Language:               "en",
		ExpoPushToken:          "",
		SubscriptionTier:       "free",
		SubscriptionExpiry:     nil,
		IsFounder:              false,
		IsConfirmed:            true,
		ConfirmationToken:      "",
		ConfirmationExpiresAt:  nil,
		DeviceID:               "",
		TrialStartAt:           ptrTime(time.Now()),
		TrialEndAt:             ptrTime(time.Now().AddDate(0, 0, 30)), // Expires in 30 days
		TrialStatus:            "active",
		CreatedAt:              time.Now(),
		UpdatedAt:              time.Now(),
	}

	err = repo.Create(ctx, userWithFutureTrial)
	if err != nil {
		t.Fatalf("failed to create user with future trial: %v", err)
	}

	// Call GetExpiredTrialUsers
	users, err := repo.GetExpiredTrialUsers(ctx)
	if err != nil {
		t.Fatalf("GetExpiredTrialUsers failed: %v", err)
	}

	// Assert: returned slice contains the expired trial user
	found := false
	for _, user := range users {
		if user.ID == userWithExpiredTrial.ID {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected expired trial user to be in results, but was not found")
	}

	// Assert: user with trial_status='active' and future trial_end_at is NOT returned
	for _, user := range users {
		if user.ID == userWithFutureTrial.ID {
			t.Errorf("user with future trial end date should not be in expired trial users")
		}
	}

	// Cleanup
	repo.Delete(ctx, userWithExpiredTrial.ID)
	repo.Delete(ctx, userWithFutureTrial.ID)
}

// Helper function to create a pointer to a time.Time
func ptrTime(t time.Time) *time.Time {
	return &t
}
