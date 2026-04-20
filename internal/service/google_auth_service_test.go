package service

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rifqimalik/cashlens-backend/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockGoogleUserRepo struct {
	mock.Mock
}

func (m *mockGoogleUserRepo) Create(ctx context.Context, user *models.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}
func (m *mockGoogleUserRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}
func (m *mockGoogleUserRepo) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}
func (m *mockGoogleUserRepo) GetByDeviceID(ctx context.Context, deviceID string) ([]models.User, error) {
	args := m.Called(ctx, deviceID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.User), args.Error(1)
}
func (m *mockGoogleUserRepo) GetByGoogleID(ctx context.Context, googleID string) (*models.User, error) {
	args := m.Called(ctx, googleID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}
func (m *mockGoogleUserRepo) Update(ctx context.Context, user *models.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}
func (m *mockGoogleUserRepo) UpdateSubscription(ctx context.Context, userID uuid.UUID, tier string, expiresAt *time.Time) error {
	args := m.Called(ctx, userID, tier, expiresAt)
	return args.Error(0)
}
func (m *mockGoogleUserRepo) UpdateFounder(ctx context.Context, userID uuid.UUID, isFounder bool) error {
	args := m.Called(ctx, userID, isFounder)
	return args.Error(0)
}
func (m *mockGoogleUserRepo) UpdateLanguage(ctx context.Context, userID uuid.UUID, language string) error {
	args := m.Called(ctx, userID, language)
	return args.Error(0)
}
func (m *mockGoogleUserRepo) UpdatePushToken(ctx context.Context, userID uuid.UUID, token string) error {
	args := m.Called(ctx, userID, token)
	return args.Error(0)
}
func (m *mockGoogleUserRepo) UpdateGoogleID(ctx context.Context, userID uuid.UUID, googleID string) error {
	args := m.Called(ctx, userID, googleID)
	return args.Error(0)
}
func (m *mockGoogleUserRepo) GetByConfirmationToken(ctx context.Context, token string) (*models.User, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}
func (m *mockGoogleUserRepo) UpdateConfirmationStatus(ctx context.Context, userID uuid.UUID, isConfirmed bool) error {
	args := m.Called(ctx, userID, isConfirmed)
	return args.Error(0)
}
func (m *mockGoogleUserRepo) UpdateConfirmationToken(ctx context.Context, userID uuid.UUID, token string, expiresAt time.Time) error {
	args := m.Called(ctx, userID, token, expiresAt)
	return args.Error(0)
}
func (m *mockGoogleUserRepo) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}
func (m *mockGoogleUserRepo) GetExpiredTrialUsers(ctx context.Context) ([]*models.User, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.User), args.Error(1)
}

func newTestGoogleAuthService(repo *mockGoogleUserRepo, googleServer *httptest.Server, clientID string) GoogleAuthService {
	mockSeeding := &MockCategorySeedingService{}
	mockSeeding.On("SeedDefaultCategories", mock.Anything, mock.Anything).Return(nil)
	mockTrialEligibility := &MockTrialEligibilityService{}
	mockTrialEligibility.On("CheckAndSetTrial", mock.Anything, mock.Anything).Return(true, nil)
	return NewGoogleAuthService(
		repo,
		mockSeeding,
		mockTrialEligibility,
		"test-jwt-secret",
		24*time.Hour,
		googleServer.URL,
		clientID,
	)
}

func makeGoogleTokenInfoServer(payload string, status int) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		fmt.Fprint(w, payload)
	}))
}

func TestGoogleAuthService_LoginWithGoogle_NewUser(t *testing.T) {
	googleServer := makeGoogleTokenInfoServer(`{
		"sub": "google-123",
		"email": "new@example.com",
		"name": "New User",
		"aud": "test-client-id",
		"exp": "9999999999"
	}`, http.StatusOK)
	defer googleServer.Close()

	repo := &mockGoogleUserRepo{}
	repo.On("GetByGoogleID", mock.Anything, "google-123").Return(nil, fmt.Errorf("user not found"))
	repo.On("GetByEmail", mock.Anything, "new@example.com").Return(nil, fmt.Errorf("user not found"))
	repo.On("Create", mock.Anything, mock.AnythingOfType("*models.User")).Return(nil)
	repo.On("Update", mock.Anything, mock.AnythingOfType("*models.User")).Return(nil)
	repo.On("GetByDeviceID", mock.Anything, mock.Anything).Return([]models.User{}, nil)

	svc := newTestGoogleAuthService(repo, googleServer, "test-client-id")
	res, err := svc.LoginWithGoogle(context.Background(), "any-token", nil)

	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.NotEmpty(t, res.AccessToken)
	assert.Equal(t, "new@example.com", res.User.Email)
	assert.Equal(t, "google", res.User.AuthProvider)
	assert.True(t, res.User.IsConfirmed)
}

func TestGoogleAuthService_LoginWithGoogle_ExistingGoogleUser(t *testing.T) {
	googleServer := makeGoogleTokenInfoServer(`{
		"sub": "google-456",
		"email": "existing@example.com",
		"name": "Existing User",
		"aud": "test-client-id",
		"exp": "9999999999"
	}`, http.StatusOK)
	defer googleServer.Close()

	existingGoogleID := "google-456"
	existingUser := &models.User{
		ID:           uuid.New(),
		Email:        "existing@example.com",
		GoogleID:     &existingGoogleID,
		AuthProvider: "google",
		IsConfirmed:  true,
		Language:     "id",
	}

	repo := &mockGoogleUserRepo{}
	repo.On("GetByGoogleID", mock.Anything, "google-456").Return(existingUser, nil)

	svc := newTestGoogleAuthService(repo, googleServer, "test-client-id")
	res, err := svc.LoginWithGoogle(context.Background(), "any-token", nil)

	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.Equal(t, existingUser.ID, res.User.ID)
}

func TestGoogleAuthService_LoginWithGoogle_MergeExistingEmailUser(t *testing.T) {
	googleServer := makeGoogleTokenInfoServer(`{
		"sub": "google-789",
		"email": "emailuser@example.com",
		"name": "Email User",
		"aud": "test-client-id",
		"exp": "9999999999"
	}`, http.StatusOK)
	defer googleServer.Close()

	existingUser := &models.User{
		ID:           uuid.New(),
		Email:        "emailuser@example.com",
		AuthProvider: "email",
		IsConfirmed:  true,
		Language:     "id",
	}

	repo := &mockGoogleUserRepo{}
	repo.On("GetByGoogleID", mock.Anything, "google-789").Return(nil, fmt.Errorf("user not found"))
	repo.On("GetByEmail", mock.Anything, "emailuser@example.com").Return(existingUser, nil)
	repo.On("UpdateGoogleID", mock.Anything, existingUser.ID, "google-789").Return(nil)

	svc := newTestGoogleAuthService(repo, googleServer, "test-client-id")
	res, err := svc.LoginWithGoogle(context.Background(), "any-token", nil)

	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.Equal(t, existingUser.ID, res.User.ID)
	repo.AssertCalled(t, "UpdateGoogleID", mock.Anything, existingUser.ID, "google-789")
}

func TestGoogleAuthService_LoginWithGoogle_InvalidToken(t *testing.T) {
	googleServer := makeGoogleTokenInfoServer(`{"error": "invalid_token"}`, http.StatusBadRequest)
	defer googleServer.Close()

	repo := &mockGoogleUserRepo{}
	svc := newTestGoogleAuthService(repo, googleServer, "test-client-id")
	res, err := svc.LoginWithGoogle(context.Background(), "bad-token", nil)

	assert.Error(t, err)
	assert.Nil(t, res)
}

func TestGoogleAuthService_LoginWithGoogle_AudMismatch(t *testing.T) {
	googleServer := makeGoogleTokenInfoServer(`{
		"sub": "google-999",
		"email": "user@example.com",
		"name": "User",
		"aud": "wrong-client-id",
		"exp": "9999999999"
	}`, http.StatusOK)
	defer googleServer.Close()

	repo := &mockGoogleUserRepo{}
	svc := newTestGoogleAuthService(repo, googleServer, "test-client-id")
	res, err := svc.LoginWithGoogle(context.Background(), "any-token", nil)

	assert.Error(t, err)
	assert.Nil(t, res)
}
