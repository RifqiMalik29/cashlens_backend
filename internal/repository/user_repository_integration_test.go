package repository_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rifqimalik/cashlens-backend/internal/models"
	"github.com/rifqimalik/cashlens-backend/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testDB *pgxpool.Pool

func TestMain(m *testing.M) {
	// Setup test database
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://user:password@localhost:5432/cashlens_test?sslmode=disable"
		fmt.Println("DATABASE_URL not set, using default for testing:", databaseURL)
	}

	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to parse database config: %v\n", err)
		os.Exit(1)
	}

	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
	testDB = pool
	// Defer closing the pool until TestMain exits
	defer testDB.Close()

	// For integration tests, ensure the table exists with the correct schema
	_, err = testDB.Exec(context.Background(), `
		DROP TABLE IF EXISTS users CASCADE;
		CREATE TABLE users (
			id UUID PRIMARY KEY,
			email VARCHAR(255) UNIQUE NOT NULL,
			password_hash VARCHAR(255) NOT NULL,
			name VARCHAR(255),
			language VARCHAR(10) NOT NULL DEFAULT 'en',
			expo_push_token VARCHAR(255),
			subscription_tier VARCHAR(50) NOT NULL DEFAULT 'free',
			subscription_expires_at TIMESTAMPTZ,
			is_founder BOOLEAN NOT NULL DEFAULT FALSE,
			is_confirmed BOOLEAN NOT NULL DEFAULT FALSE,
			confirmation_token VARCHAR(255),
			confirmation_expires_at TIMESTAMPTZ,
			device_id VARCHAR(255),
			trial_start_at TIMESTAMPTZ,
			trial_end_at TIMESTAMPTZ,
			trial_status VARCHAR(50) NOT NULL DEFAULT 'inactive',
			created_at TIMESTAMPTZ NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL
		);
	`)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to set up test users table: %v\n", err)
		os.Exit(1)
	}

	exitCode := m.Run()
	os.Exit(exitCode)
}

func clearUsersTable(t *testing.T) {
	_, err := testDB.Exec(context.Background(), "DELETE FROM users")
	require.NoError(t, err)
}

func TestUserRepository_FindByDeviceID(t *testing.T) {
	clearUsersTable(t)
	repo := repository.NewUserRepository(testDB)
	ctx := context.Background()

	// Create a user with a device ID
	userID1 := uuid.New()
	deviceID1 := "test_device_id_1"
	now := time.Now().UTC().Truncate(time.Millisecond) // Truncate for comparison
	user1 := &models.User{
		ID:                    userID1,
		Email:                 "test1@example.com",
		PasswordHash:          "hashedpassword",
		Name:                  models.StringPtr("Test User 1"),
		Language:              "en",
		ExpoPushToken:         models.StringPtr("expo_token_1"),
		SubscriptionTier:      "free",
		SubscriptionExpiry:    nil,
		IsFounder:             false,
		IsConfirmed:           true,
		ConfirmationToken:     nil,
		ConfirmationExpiresAt: nil,
		DeviceID:              &deviceID1,                                                          // New field
		TrialStartAt:          &now,                                                                // New field
		TrialEndAt:            func() *time.Time { t := now.Add(7 * 24 * time.Hour); return &t }(), // New field
		TrialStatus:           "active",                                                            // New field
		CreatedAt:             now,
		UpdatedAt:             now,
	}
	err := repo.Create(ctx, user1)
	require.NoError(t, err)

	// Create another user on the same device
	userID2 := uuid.New()
	user2 := &models.User{
		ID:                    userID2,
		Email:                 "test2@example.com",
		PasswordHash:          "hashedpassword2",
		Name:                  models.StringPtr("Test User 2"),
		Language:              "en",
		ExpoPushToken:         models.StringPtr("expo_token_2"),
		SubscriptionTier:      "free",
		SubscriptionExpiry:    nil,
		IsFounder:             false,
		IsConfirmed:           true,
		ConfirmationToken:     nil,
		ConfirmationExpiresAt: nil,
		DeviceID:              &deviceID1, // Same device ID
		TrialStartAt:          &now,
		TrialEndAt:            func() *time.Time { t := now.Add(7 * 24 * time.Hour); return &t }(),
		TrialStatus:           "active",
		CreatedAt:             now,
		UpdatedAt:             now,
	}
	err = repo.Create(ctx, user2)
	require.NoError(t, err)

	// Create a user on a different device
	userID3 := uuid.New()
	deviceID3 := "test_device_id_3"
	user3 := &models.User{
		ID:                    userID3,
		Email:                 "test3@example.com",
		PasswordHash:          "hashedpassword3",
		Name:                  models.StringPtr("Test User 3"),
		Language:              "en",
		ExpoPushToken:         models.StringPtr("expo_token_3"),
		SubscriptionTier:      "free",
		SubscriptionExpiry:    nil,
		IsFounder:             false,
		IsConfirmed:           true,
		ConfirmationToken:     nil,
		ConfirmationExpiresAt: nil,
		DeviceID:              &deviceID3,
		TrialStartAt:          &now,
		TrialEndAt:            func() *time.Time { t := now.Add(7 * 24 * time.Hour); return &t }(),
		TrialStatus:           "active",
		CreatedAt:             now,
		UpdatedAt:             now,
	}
	err = repo.Create(ctx, user3)
	require.NoError(t, err)

	// Find users by deviceID1
	foundUsers, err := repo.GetByDeviceID(ctx, deviceID1)
	assert.NoError(t, err)
	assert.Len(t, foundUsers, 2)

	// Check if both user1 and user2 are in foundUsers
	var foundIDs []uuid.UUID
	for _, u := range foundUsers {
		foundIDs = append(foundIDs, u.ID)
	}
	assert.Contains(t, foundIDs, user1.ID)
	assert.Contains(t, foundIDs, user2.ID)

	// Find users by non-existent device ID
	foundUsers, err = repo.GetByDeviceID(ctx, "non_existent_device")
	assert.NoError(t, err)
	assert.Len(t, foundUsers, 0)
}

func TestUserRepository_CreateAndUpdateWithTrialFields(t *testing.T) {
	clearUsersTable(t)
	repo := repository.NewUserRepository(testDB)
	ctx := context.Background()

	// Test Create
	userID := uuid.New()
	deviceID := "new_test_device"
	now := time.Now().UTC().Truncate(time.Millisecond)
	trialEnd := now.Add(7 * 24 * time.Hour).Truncate(time.Millisecond)

	user := &models.User{
		ID:                    userID,
		Email:                 "create@example.com",
		PasswordHash:          "hashed",
		Name:                  models.StringPtr("Create Test"),
		Language:              "en",
		ExpoPushToken:         models.StringPtr("create_token"),
		SubscriptionTier:      "pro",
		SubscriptionExpiry:    &trialEnd,
		IsFounder:             true,
		IsConfirmed:           false,
		ConfirmationToken:     models.StringPtr("confirm_token"),
		ConfirmationExpiresAt: &trialEnd, // using trialEnd as a dummy expiry for confirmation
		DeviceID:              &deviceID,
		TrialStartAt:          &now,
		TrialEndAt:            &trialEnd,
		TrialStatus:           "active",
		CreatedAt:             now,
		UpdatedAt:             now,
	}

	err := repo.Create(ctx, user)
	require.NoError(t, err)

	// Verify created user
	fetchedUser, err := repo.GetByID(ctx, userID)
	require.NoError(t, err)
	assert.NotNil(t, fetchedUser.DeviceID)
	assert.Equal(t, deviceID, *fetchedUser.DeviceID)
	assert.NotNil(t, fetchedUser.TrialStartAt)
	assert.WithinDuration(t, now, *fetchedUser.TrialStartAt, time.Millisecond) // Compare with millisecond precision
	assert.NotNil(t, fetchedUser.TrialEndAt)
	assert.WithinDuration(t, trialEnd, *fetchedUser.TrialEndAt, time.Millisecond) // Compare with millisecond precision
	assert.Equal(t, "active", fetchedUser.TrialStatus)

	// Test Update
	updatedDeviceID := "updated_device"
	updatedTrialStatus := "expired"
	fetchedUser.DeviceID = &updatedDeviceID
	fetchedUser.TrialStatus = updatedTrialStatus
	fetchedUser.Name = models.StringPtr("Updated Name")
	fetchedUser.ExpoPushToken = models.StringPtr("updated_expo")

	err = repo.Update(ctx, fetchedUser)
	require.NoError(t, err)

	// Verify updated user
	reFetchedUser, err := repo.GetByID(ctx, userID)
	require.NoError(t, err)
	assert.NotNil(t, reFetchedUser.DeviceID)
	assert.Equal(t, updatedDeviceID, *reFetchedUser.DeviceID)
	assert.Equal(t, updatedTrialStatus, reFetchedUser.TrialStatus)
	assert.Equal(t, "Updated Name", *reFetchedUser.Name)
	assert.Equal(t, "updated_expo", *reFetchedUser.ExpoPushToken)
	assert.True(t, reFetchedUser.UpdatedAt.After(fetchedUser.UpdatedAt), "UpdatedAt should be after the previous UpdatedAt")
	assert.WithinDuration(t, time.Now(), reFetchedUser.UpdatedAt, 5*time.Second) // Check if UpdatedAt is recent
}
