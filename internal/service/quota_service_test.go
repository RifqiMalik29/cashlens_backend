package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rifqimalik/cashlens-backend/internal/models"
	"github.com/rifqimalik/cashlens-backend/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockQuotaRepo struct{ mock.Mock }

func (m *MockQuotaRepo) IncrementTransactionsIfUnderLimit(ctx context.Context, userID uuid.UUID, month, year, limit int) (bool, error) {
	args := m.Called(ctx, userID, month, year, limit)
	return args.Bool(0), args.Error(1)
}

func (m *MockQuotaRepo) IncrementScansIfUnderLimit(ctx context.Context, userID uuid.UUID, month, year, limit int) (bool, error) {
	args := m.Called(ctx, userID, month, year, limit)
	return args.Bool(0), args.Error(1)
}

func (m *MockQuotaRepo) GetOrCreate(ctx context.Context, userID uuid.UUID, month, year int) (*models.UserQuota, error) {
	args := m.Called(ctx, userID, month, year)
	return args.Get(0).(*models.UserQuota), args.Error(1)
}

type MockUserRepoForQuota struct{ mock.Mock }

// GetByID retrieves a user by their ID.
func (m *MockUserRepoForQuota) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

// Create inserts a new user.
func (m *MockUserRepoForQuota) Create(ctx context.Context, user *models.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

// GetByEmail retrieves a user by their email.
func (m *MockUserRepoForQuota) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

// GetByDeviceID retrieves users by device ID.
func (m *MockUserRepoForQuota) GetByDeviceID(ctx context.Context, deviceID string) ([]models.User, error) {
	args := m.Called(ctx, deviceID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.User), args.Error(1)
}

// Update updates a user.
func (m *MockUserRepoForQuota) Update(ctx context.Context, user *models.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

// UpdateSubscription updates a user's subscription.
func (m *MockUserRepoForQuota) UpdateSubscription(ctx context.Context, userID uuid.UUID, tier string, expiresAt *time.Time) error {
	args := m.Called(ctx, userID, tier, expiresAt)
	return args.Error(0)
}

// UpdateFounder updates a user's founder status.
func (m *MockUserRepoForQuota) UpdateFounder(ctx context.Context, userID uuid.UUID, isFounder bool) error {
	args := m.Called(ctx, userID, isFounder)
	return args.Error(0)
}

// UpdateLanguage updates a user's language preference.
func (m *MockUserRepoForQuota) UpdateLanguage(ctx context.Context, userID uuid.UUID, language string) error {
	args := m.Called(ctx, userID, language)
	return args.Error(0)
}

// UpdatePushToken updates a user's push token.
func (m *MockUserRepoForQuota) UpdatePushToken(ctx context.Context, userID uuid.UUID, token string) error {
	args := m.Called(ctx, userID, token)
	return args.Error(0)
}

// GetByConfirmationToken retrieves a user by confirmation token.
func (m *MockUserRepoForQuota) GetByConfirmationToken(ctx context.Context, token string) (*models.User, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

// UpdateConfirmationStatus updates a user's confirmation status.
func (m *MockUserRepoForQuota) UpdateConfirmationStatus(ctx context.Context, userID uuid.UUID, isConfirmed bool) error {
	args := m.Called(ctx, userID, isConfirmed)
	return args.Error(0)
}

// UpdateConfirmationToken updates a user's confirmation token.
func (m *MockUserRepoForQuota) UpdateConfirmationToken(ctx context.Context, userID uuid.UUID, token string, expiresAt time.Time) error {
	args := m.Called(ctx, userID, token, expiresAt)
	return args.Error(0)
}

// Delete deletes a user.
func (m *MockUserRepoForQuota) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// GetExpiredTrialUsers retrieves users with expired trials.
func (m *MockUserRepoForQuota) GetExpiredTrialUsers(ctx context.Context) ([]*models.User, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.User), args.Error(1)
}

// GetByGoogleID retrieves a user by their Google ID.
func (m *MockUserRepoForQuota) GetByGoogleID(ctx context.Context, googleID string) (*models.User, error) {
	args := m.Called(ctx, googleID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

// UpdateGoogleID updates a user's Google ID.
func (m *MockUserRepoForQuota) UpdateGoogleID(ctx context.Context, userID uuid.UUID, googleID string) error {
	args := m.Called(ctx, userID, googleID)
	return args.Error(0)
}

func TestQuotaService_ActiveTrial_BypassesTransactionLimit(t *testing.T) {
	quotaRepo := new(MockQuotaRepo)
	userRepo := new(MockUserRepoForQuota)

	trialEnd := time.Now().Add(24 * time.Hour)
	userID := uuid.New()
	user := &models.User{
		ID:               userID,
		SubscriptionTier: "free",
		TrialStatus:      "active",
		TrialEndAt:       &trialEnd,
	}
	userRepo.On("GetByID", mock.Anything, userID).Return(user, nil)

	svc := service.NewQuotaService(quotaRepo, userRepo)
	err := svc.CheckAndIncrementTransactionQuota(context.Background(), userID)

	assert.NoError(t, err)
	quotaRepo.AssertNotCalled(t, "IncrementTransactionsIfUnderLimit")
	userRepo.AssertExpectations(t)
}

func TestQuotaService_ExpiredTrial_EnforcesLimit(t *testing.T) {
	quotaRepo := new(MockQuotaRepo)
	userRepo := new(MockUserRepoForQuota)

	trialEnd := time.Now().Add(-1 * time.Hour)
	userID := uuid.New()
	user := &models.User{
		ID:               userID,
		SubscriptionTier: "free",
		TrialStatus:      "expired",
		TrialEndAt:       &trialEnd,
	}
	userRepo.On("GetByID", mock.Anything, userID).Return(user, nil)
	quotaRepo.On("IncrementTransactionsIfUnderLimit", mock.Anything, userID, mock.Anything, mock.Anything, models.FreeTierLimits.MaxTransactionsPerMonth).Return(true, nil)

	svc := service.NewQuotaService(quotaRepo, userRepo)
	err := svc.CheckAndIncrementTransactionQuota(context.Background(), userID)

	assert.NoError(t, err)
	quotaRepo.AssertExpectations(t)
}

func TestQuotaService_ActiveTrial_BypassesScanLimit(t *testing.T) {
	quotaRepo := new(MockQuotaRepo)
	userRepo := new(MockUserRepoForQuota)

	trialEnd := time.Now().Add(24 * time.Hour)
	userID := uuid.New()
	user := &models.User{
		ID:               userID,
		SubscriptionTier: "free",
		TrialStatus:      "active",
		TrialEndAt:       &trialEnd,
	}
	userRepo.On("GetByID", mock.Anything, userID).Return(user, nil)

	svc := service.NewQuotaService(quotaRepo, userRepo)
	err := svc.CheckAndIncrementScanQuota(context.Background(), userID)

	assert.NoError(t, err)
	quotaRepo.AssertNotCalled(t, "IncrementScansIfUnderLimit")
	userRepo.AssertExpectations(t)
}

func TestQuotaService_ExpiredTrial_EnforcesScanLimit(t *testing.T) {
	quotaRepo := new(MockQuotaRepo)
	userRepo := new(MockUserRepoForQuota)

	trialEnd := time.Now().Add(-1 * time.Hour)
	userID := uuid.New()
	user := &models.User{
		ID:               userID,
		SubscriptionTier: "free",
		TrialStatus:      "expired",
		TrialEndAt:       &trialEnd,
	}
	userRepo.On("GetByID", mock.Anything, userID).Return(user, nil)
	quotaRepo.On("IncrementScansIfUnderLimit", mock.Anything, userID, mock.Anything, mock.Anything, models.FreeTierLimits.MaxScansPerMonth).Return(true, nil)

	svc := service.NewQuotaService(quotaRepo, userRepo)
	err := svc.CheckAndIncrementScanQuota(context.Background(), userID)

	assert.NoError(t, err)
	quotaRepo.AssertExpectations(t)
}
