package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rifqimalik/cashlens-backend/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockUserRepository is a mock of repository.UserRepository
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) Create(ctx context.Context, user *models.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) GetByDeviceID(ctx context.Context, deviceID string) ([]models.User, error) {
	args := m.Called(ctx, deviceID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.User), args.Error(1)
}

func (m *MockUserRepository) Update(ctx context.Context, user *models.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) UpdateSubscription(ctx context.Context, userID uuid.UUID, tier string, expiresAt *time.Time) error {
	args := m.Called(ctx, userID, tier, expiresAt)
	return args.Error(0)
}

func (m *MockUserRepository) UpdateFounder(ctx context.Context, userID uuid.UUID, isFounder bool) error {
	args := m.Called(ctx, userID, isFounder)
	return args.Error(0)
}

func (m *MockUserRepository) UpdateLanguage(ctx context.Context, userID uuid.UUID, language string) error {
	args := m.Called(ctx, userID, language)
	return args.Error(0)
}

func (m *MockUserRepository) UpdatePushToken(ctx context.Context, userID uuid.UUID, token string) error {
	args := m.Called(ctx, userID, token)
	return args.Error(0)
}

func (m *MockUserRepository) GetByConfirmationToken(ctx context.Context, token string) (*models.User, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) UpdateConfirmationStatus(ctx context.Context, userID uuid.UUID, isConfirmed bool) error {
	args := m.Called(ctx, userID, isConfirmed)
	return args.Error(0)
}

func (m *MockUserRepository) UpdateConfirmationToken(ctx context.Context, userID uuid.UUID, token string, expiresAt time.Time) error {
	args := m.Called(ctx, userID, token, expiresAt)
	return args.Error(0)
}

func (m *MockUserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockUserRepository) GetExpiredTrialUsers(ctx context.Context) ([]*models.User, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.User), args.Error(1)
}

// MockTrialEligibilityService is a mock implementation of TrialEligibilityService.
type MockTrialEligibilityService struct {
	mock.Mock
}

func (m *MockTrialEligibilityService) CheckAndSetTrial(user *models.User, newDeviceID *string) (bool, error) {
	args := m.Called(user, newDeviceID)
	return args.Bool(0), args.Error(1)
}

// MockCategorySeedingService
type MockCategorySeedingService struct {
	mock.Mock
}

func (m *MockCategorySeedingService) SeedDefaultCategories(ctx context.Context, userID uuid.UUID) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

// MockMailer
type MockMailer struct {
	mock.Mock
}

func (m *MockMailer) SendConfirmationEmail(to, token string) error {
	args := m.Called(to, token)
	return args.Error(0)
}

func (m *MockMailer) SendTrialExpiredEmail(to string) error {
	args := m.Called(to)
	return args.Error(0)
}

func TestAuthService_Register_WithTrial(t *testing.T) {
	ctx := context.Background()
	email := "newuser@example.com"
	password := "password123"
	name := "New User"
	lang := "en"
	deviceID := "test_device_id"

	userRepo := new(MockUserRepository)
	seedingService := new(MockCategorySeedingService)
	mailer := new(MockMailer)
	trialService := new(MockTrialEligibilityService) // Mock trial service

	s := NewAuthService(userRepo, seedingService, nil, mailer, "secret", time.Hour, trialService)

	// Mock expectations
	userRepo.On("GetByEmail", ctx, email).Return(nil, fmt.Errorf("user not found")) // User does not exist
	userRepo.On("Create", ctx, mock.AnythingOfType("*models.User")).Return(nil)
	seedingService.On("SeedDefaultCategories", ctx, mock.AnythingOfType("uuid.UUID")).Return(nil)
	mailer.On("SendConfirmationEmail", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
	trialService.On("CheckAndSetTrial", mock.AnythingOfType("*models.User"), &deviceID).Return(true, nil).Run(func(args mock.Arguments) {
		userArg := args.Get(0).(*models.User)
		deviceArg := args.Get(1).(*string)
		now := time.Now().UTC()
		trialEnd := now.Add(7 * 24 * time.Hour)
		userArg.DeviceID = deviceArg
		userArg.TrialStartAt = &now
		userArg.TrialEndAt = &trialEnd
		userArg.TrialStatus = "active"
	})

	req := models.CreateUserRequest{
		Email:    email,
		Password: password,
		Name:     name,
		Language: lang,
		DeviceID: &deviceID,
	}

	authResponse, err := s.Register(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, authResponse)
	assert.NotNil(t, authResponse.User.DeviceID)
	assert.Equal(t, deviceID, *authResponse.User.DeviceID)
	assert.Equal(t, "active", authResponse.User.TrialStatus) // Assuming trial is active

	userRepo.AssertExpectations(t)
	seedingService.AssertExpectations(t)
	mailer.AssertExpectations(t)
	trialService.AssertExpectations(t) // Verify trial service was called
}

func TestAuthService_Login_WithTrial(t *testing.T) {
	ctx := context.Background()
	email := "user@example.com"
	password := "password123"
	hashedPassword, _ := hashPassword(password)
	deviceID := "login_device_id"
	now := time.Now().UTC()

	userRepo := new(MockUserRepository)
	seedingService := new(MockCategorySeedingService)
	mailer := new(MockMailer)
	trialService := new(MockTrialEligibilityService)

	s := NewAuthService(userRepo, seedingService, nil, mailer, "secret", time.Hour, trialService)

	// User object for GetByEmail
	confirmedUser := &models.User{
		ID:           uuid.New(),
		Email:        email,
		PasswordHash: hashedPassword,
		IsConfirmed:  true,
		CreatedAt:    now.Add(-time.Hour),
		UpdatedAt:    now.Add(-time.Hour),
	}

	// Mock expectations
	userRepo.On("GetByEmail", ctx, email).Return(confirmedUser, nil)
	// Expect CheckAndSetTrial to be called when deviceID is provided and user's deviceID is nil
	trialService.On("CheckAndSetTrial", confirmedUser, &deviceID).Return(true, nil).Run(func(args mock.Arguments) {
		userArg := args.Get(0).(*models.User)
		deviceArg := args.Get(1).(*string)
		userArg.DeviceID = deviceArg
	})

	req := models.LoginRequest{
		Email:    email,
		Password: password,
		DeviceID: &deviceID,
	}

	authResponse, err := s.Login(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, authResponse)
	assert.NotNil(t, authResponse.User.DeviceID)
	assert.Equal(t, deviceID, *authResponse.User.DeviceID)

	userRepo.AssertExpectations(t)
	trialService.AssertCalled(t, "CheckAndSetTrial", confirmedUser, &deviceID) // Ensure it was called
}

func TestAuthService_Login_StrictConfirmation(t *testing.T) {
	ctx := context.Background()
	email := "test@example.com"
	password := "password123"
	hashedPassword, _ := hashPassword(password)

	t.Run("Login fails if not confirmed and resends OTP", func(t *testing.T) {
		userRepo := new(MockUserRepository)
		seedingService := new(MockCategorySeedingService)
		mailer := new(MockMailer)
		s := NewAuthService(userRepo, seedingService, nil, mailer, "secret", time.Hour, nil) // Add nil for trialEligibilityService

		unconfirmedUser := &models.User{
			ID:           uuid.New(),
			Email:        email,
			PasswordHash: hashedPassword,
			IsConfirmed:  false,
		}

		userRepo.On("GetByEmail", ctx, email).Return(unconfirmedUser, nil)
		userRepo.On("UpdateConfirmationToken", ctx, unconfirmedUser.ID, mock.AnythingOfType("string"), mock.AnythingOfType("time.Time")).Return(nil)
		mailer.On("SendConfirmationEmail", email, mock.AnythingOfType("string")).Return(nil)

		res, err := s.Login(ctx, models.LoginRequest{
			Email:    email,
			Password: password,
		})

		assert.Error(t, err)
		var notConfirmed *ErrEmailNotConfirmed
		assert.ErrorAs(t, err, &notConfirmed)
		assert.Equal(t, email, notConfirmed.Email)
		assert.Nil(t, res)
		userRepo.AssertExpectations(t)
	})

	t.Run("Login succeeds if confirmed", func(t *testing.T) {
		userRepo := new(MockUserRepository)
		seedingService := new(MockCategorySeedingService)
		mailer := new(MockMailer)
		s := NewAuthService(userRepo, seedingService, nil, mailer, "secret", time.Hour, nil) // Add nil for trialEligibilityService

		confirmedUser := &models.User{
			ID:           uuid.New(),
			Email:        email,
			PasswordHash: hashedPassword,
			IsConfirmed:  true,
		}

		userRepo.On("GetByEmail", ctx, email).Return(confirmedUser, nil)

		res, err := s.Login(ctx, models.LoginRequest{
			Email:    email,
			Password: password,
		})

		assert.NoError(t, err)
		if assert.NotNil(t, res) {
			assert.NotEmpty(t, res.AccessToken)
		}
		userRepo.AssertExpectations(t)
	})
}
