package service

import (
	"context"
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

func TestAuthService_Login_StrictConfirmation(t *testing.T) {
	ctx := context.Background()
	email := "test@example.com"
	password := "password123"
	hashedPassword, _ := hashPassword(password)

	t.Run("Login fails if not confirmed and resends OTP", func(t *testing.T) {
		userRepo := new(MockUserRepository)
		seedingService := new(MockCategorySeedingService)
		mailer := new(MockMailer)
		s := NewAuthService(userRepo, seedingService, nil, mailer, "secret", time.Hour)

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
		s := NewAuthService(userRepo, seedingService, nil, mailer, "secret", time.Hour)

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
