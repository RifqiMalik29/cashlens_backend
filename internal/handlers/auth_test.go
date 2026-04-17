package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rifqimalik/cashlens-backend/internal/config"
	"github.com/rifqimalik/cashlens-backend/internal/handlers"
	"github.com/rifqimalik/cashlens-backend/internal/middleware"
	"github.com/rifqimalik/cashlens-backend/internal/models"
	"github.com/rifqimalik/cashlens-backend/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockAuthService is a mock implementation of service.AuthService.
type MockAuthService struct {
	mock.Mock
}

func (m *MockAuthService) Register(ctx context.Context, req models.CreateUserRequest) (*models.AuthResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.AuthResponse), args.Error(1)
}

func (m *MockAuthService) Login(ctx context.Context, req models.LoginRequest) (*models.AuthResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.AuthResponse), args.Error(1)
}

func (m *MockAuthService) ConfirmEmail(ctx context.Context, email, otp string) error {
	args := m.Called(ctx, email, otp)
	return args.Error(0)
}

func (m *MockAuthService) ResendConfirmation(ctx context.Context, email string) error {
	args := m.Called(ctx, email)
	return args.Error(0)
}

func (m *MockAuthService) ValidateToken(tokenString string) (*uuid.UUID, error) {
	args := m.Called(tokenString)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*uuid.UUID), args.Error(1)
}

func (m *MockAuthService) GetMe(ctx context.Context, userID uuid.UUID) (*models.User, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockAuthService) UpdateLanguage(ctx context.Context, userID uuid.UUID, language string) error {
	args := m.Called(ctx, userID, language)
	return args.Error(0)
}

func (m *MockAuthService) UpdatePushToken(ctx context.Context, userID uuid.UUID, token string) error {
	args := m.Called(ctx, userID, token)
	return args.Error(0)
}

func (m *MockAuthService) GetTelegramStatus(ctx context.Context, userID uuid.UUID) (map[string]any, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(map[string]any), args.Error(1)
}

func (m *MockAuthService) UnlinkTelegram(ctx context.Context, userID uuid.UUID) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockAuthService) DeleteAccount(ctx context.Context, userID uuid.UUID) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

// MockRefreshTokenService is a mock implementation of service.RefreshTokenService.
type MockRefreshTokenService struct {
	mock.Mock
}

func (m *MockRefreshTokenService) GenerateRefreshToken(ctx context.Context, userID uuid.UUID, ip, userAgent string) (*models.RefreshToken, error) {
	args := m.Called(ctx, userID, ip, userAgent)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.RefreshToken), args.Error(1)
}

func (m *MockRefreshTokenService) RefreshAccessToken(ctx context.Context, refreshToken string) (*models.RefreshTokenResponse, error) {
	args := m.Called(ctx, refreshToken)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.RefreshTokenResponse), args.Error(1)
}

func (m *MockRefreshTokenService) RevokeAllUserTokens(ctx context.Context, userID uuid.UUID) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockRefreshTokenService) CleanupExpiredTokens(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockRefreshTokenService) RevokeToken(ctx context.Context, token string) error {
	args := m.Called(ctx, token)
	return args.Error(0)
}

func TestAuthHandler_Register(t *testing.T) {
	mockAuthService := new(MockAuthService)
	mockRefreshTokenService := new(MockRefreshTokenService)
	cfg := &config.Config{} // Use a dummy config

	handler := handlers.NewAuthHandler(mockAuthService, mockRefreshTokenService, cfg)

	t.Run("successful registration with device ID", func(t *testing.T) {
		email := "test@example.com"
		password := "password123"
		name := "Test User"
		lang := "en"
		deviceID := "test_device_id_123"

		reqBody := models.CreateUserRequest{
			Email:    email,
			Password: password,
			Name:     name,
			Language: lang,
			DeviceID: &deviceID,
		}

		user := models.User{
			ID:           uuid.New(),
			Email:        email,
			PasswordHash: "hashedpassword",
			Name:         &name,
			Language:     lang,
			DeviceID:     &deviceID,
			TrialStatus:  "active",
		}

		mockAuthService.On("Register", mock.Anything, reqBody).Return(&models.AuthResponse{
			AccessToken: "dummy_access_token",
			User:        user,
		}, nil).Once()

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		// Use chi router to handle context correctly
		r := chi.NewRouter()
		r.Post("/api/v1/auth/register", handler.Register)
		r.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusCreated, rr.Code)
		var responseBody map[string]interface{}
		json.NewDecoder(rr.Body).Decode(&responseBody)

		assert.Equal(t, "Registration successful. Please check your email to confirm your account.", responseBody["message"])
		data, ok := responseBody["data"].(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, "dummy_access_token", data["access_token"]) // Corrected key
		userData, ok := data["user"].(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, deviceID, userData["device_id"])       // Corrected key
		assert.Equal(t, "active", userData["trial_status"]) // Corrected key

		mockAuthService.AssertExpectations(t)
	})

	t.Run("registration fails due to existing email", func(t *testing.T) {
		email := "existing@example.com"
		password := "password123"
		name := "Test User"

		reqBody := models.CreateUserRequest{
			Email:    email,
			Password: password,
			Name:     name,
		}

		mockAuthService.On("Register", mock.Anything, reqBody).Return(nil, fmt.Errorf("email already registered")).Once()

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		r := chi.NewRouter()
		r.Post("/api/v1/auth/register", handler.Register)
		r.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
		var responseBody handlers.ErrorResponse
		json.NewDecoder(rr.Body).Decode(&responseBody)
		assert.Equal(t, "email already registered", responseBody.Error)

		mockAuthService.AssertExpectations(t)
	})

	t.Run("registration fails due to invalid request body", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString("invalid json"))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		r := chi.NewRouter()
		r.Post("/api/v1/auth/register", handler.Register)
		r.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
		var responseBody handlers.ErrorResponse
		json.NewDecoder(rr.Body).Decode(&responseBody)
		assert.Equal(t, "Invalid request body", responseBody.Error)
	})

	t.Run("registration fails due to validation error", func(t *testing.T) {
		reqBody := models.CreateUserRequest{
			Email:    "invalid-email",
			Password: "short",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		r := chi.NewRouter()
		r.Post("/api/v1/auth/register", handler.Register)
		r.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
		var responseBody handlers.ErrorResponse
		json.NewDecoder(rr.Body).Decode(&responseBody)
		assert.Equal(t, "Validation failed", responseBody.Error)
		assert.Contains(t, responseBody.Details, "email")
		assert.Contains(t, responseBody.Details, "password")
	})
}

func TestAuthHandler_Login(t *testing.T) {
	mockAuthService := new(MockAuthService)
	mockRefreshTokenService := new(MockRefreshTokenService)
	cfg := &config.Config{}

	handler := handlers.NewAuthHandler(mockAuthService, mockRefreshTokenService, cfg)

	t.Run("successful login returns access and refresh token", func(t *testing.T) {
		email := "test@example.com"
		password := "password123"
		userID := uuid.New()
		deviceID := "device_abc"

		reqBody := models.LoginRequest{
			Email:    email,
			Password: password,
			DeviceID: &deviceID,
		}

		user := models.User{ID: userID, Email: email}
		authRes := &models.AuthResponse{AccessToken: "access_tok", User: user}
		refreshTok := &models.RefreshToken{Token: "refresh_tok"}

		mockAuthService.On("Login", mock.Anything, reqBody).Return(authRes, nil).Once()
		mockRefreshTokenService.On("GenerateRefreshToken", mock.Anything, userID, mock.Anything, mock.Anything).Return(refreshTok, nil).Once()

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		r := chi.NewRouter()
		r.Post("/api/v1/auth/login", handler.Login)
		r.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		var responseBody map[string]interface{}
		json.NewDecoder(rr.Body).Decode(&responseBody)
		data := responseBody["data"].(map[string]interface{})
		assert.Equal(t, "access_tok", data["access_token"])
		assert.Equal(t, "refresh_tok", data["refresh_token"])

		mockAuthService.AssertExpectations(t)
		mockRefreshTokenService.AssertExpectations(t)
	})

	t.Run("login fails with wrong credentials returns 401", func(t *testing.T) {
		reqBody := models.LoginRequest{Email: "test@example.com", Password: "wrong"}

		mockAuthService.On("Login", mock.Anything, reqBody).Return(nil, fmt.Errorf("invalid credentials")).Once()

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		r := chi.NewRouter()
		r.Post("/api/v1/auth/login", handler.Login)
		r.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
		var responseBody handlers.ErrorResponse
		json.NewDecoder(rr.Body).Decode(&responseBody)
		assert.Equal(t, "invalid credentials", responseBody.Error)

		mockAuthService.AssertExpectations(t)
	})

	t.Run("login fails with unconfirmed email returns 403", func(t *testing.T) {
		reqBody := models.LoginRequest{Email: "unconfirmed@example.com", Password: "password123"}

		mockAuthService.On("Login", mock.Anything, reqBody).Return(nil, &service.ErrEmailNotConfirmed{Email: "unconfirmed@example.com"}).Once()

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		r := chi.NewRouter()
		r.Post("/api/v1/auth/login", handler.Login)
		r.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusForbidden, rr.Code)
		var responseBody map[string]interface{}
		json.NewDecoder(rr.Body).Decode(&responseBody)
		assert.Equal(t, true, responseBody["requires_confirmation"])
		assert.Equal(t, "unconfirmed@example.com", responseBody["email"])

		mockAuthService.AssertExpectations(t)
	})

	t.Run("login fails due to invalid request body", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString("bad json"))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		r := chi.NewRouter()
		r.Post("/api/v1/auth/login", handler.Login)
		r.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
		var responseBody handlers.ErrorResponse
		json.NewDecoder(rr.Body).Decode(&responseBody)
		assert.Equal(t, "Invalid request body", responseBody.Error)
	})
}

func TestAuthHandler_ConfirmEmail(t *testing.T) {
	mockAuthService := new(MockAuthService)
	mockRefreshTokenService := new(MockRefreshTokenService)
	cfg := &config.Config{}

	handler := handlers.NewAuthHandler(mockAuthService, mockRefreshTokenService, cfg)

	t.Run("successful email confirmation returns 200", func(t *testing.T) {
		mockAuthService.On("ConfirmEmail", mock.Anything, "test@example.com", "123456").Return(nil).Once()

		body, _ := json.Marshal(map[string]string{"email": "test@example.com", "otp": "123456"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/confirm-email", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		r := chi.NewRouter()
		r.Post("/api/v1/auth/confirm-email", handler.ConfirmEmail)
		r.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		var responseBody map[string]interface{}
		json.NewDecoder(rr.Body).Decode(&responseBody)
		assert.Equal(t, "Email confirmed successfully. You can now log in.", responseBody["message"])

		mockAuthService.AssertExpectations(t)
	})

	t.Run("confirm email fails returns 400", func(t *testing.T) {
		mockAuthService.On("ConfirmEmail", mock.Anything, "test@example.com", "000000").Return(fmt.Errorf("invalid OTP")).Once()

		body, _ := json.Marshal(map[string]string{"email": "test@example.com", "otp": "000000"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/confirm-email", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		r := chi.NewRouter()
		r.Post("/api/v1/auth/confirm-email", handler.ConfirmEmail)
		r.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
		var responseBody handlers.ErrorResponse
		json.NewDecoder(rr.Body).Decode(&responseBody)
		assert.Equal(t, "invalid OTP", responseBody.Error)

		mockAuthService.AssertExpectations(t)
	})
}

func TestAuthHandler_ResendConfirmation(t *testing.T) {
	mockAuthService := new(MockAuthService)
	mockRefreshTokenService := new(MockRefreshTokenService)
	cfg := &config.Config{}

	handler := handlers.NewAuthHandler(mockAuthService, mockRefreshTokenService, cfg)

	t.Run("successful resend returns 200", func(t *testing.T) {
		mockAuthService.On("ResendConfirmation", mock.Anything, "test@example.com").Return(nil).Once()

		body, _ := json.Marshal(map[string]string{"email": "test@example.com"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/resend-confirmation", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		r := chi.NewRouter()
		r.Post("/api/v1/auth/resend-confirmation", handler.ResendConfirmation)
		r.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		var responseBody map[string]interface{}
		json.NewDecoder(rr.Body).Decode(&responseBody)
		assert.Equal(t, "Confirmation email resent successfully.", responseBody["message"])

		mockAuthService.AssertExpectations(t)
	})

	t.Run("resend fails returns 400", func(t *testing.T) {
		mockAuthService.On("ResendConfirmation", mock.Anything, "unknown@example.com").Return(fmt.Errorf("user not found")).Once()

		body, _ := json.Marshal(map[string]string{"email": "unknown@example.com"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/resend-confirmation", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		r := chi.NewRouter()
		r.Post("/api/v1/auth/resend-confirmation", handler.ResendConfirmation)
		r.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
		var responseBody handlers.ErrorResponse
		json.NewDecoder(rr.Body).Decode(&responseBody)
		assert.Equal(t, "user not found", responseBody.Error)

		mockAuthService.AssertExpectations(t)
	})
}

// helper: inject userID into request context (simulates auth middleware)
func withUserID(req *http.Request, userID *uuid.UUID) *http.Request {
	ctx := context.WithValue(req.Context(), middleware.UserIDKey, userID)
	return req.WithContext(ctx)
}

func TestAuthHandler_GetMe(t *testing.T) {
	mockAuthService := new(MockAuthService)
	mockRefreshTokenService := new(MockRefreshTokenService)
	cfg := &config.Config{}

	handler := handlers.NewAuthHandler(mockAuthService, mockRefreshTokenService, cfg)

	t.Run("returns user when authenticated", func(t *testing.T) {
		userID := uuid.New()
		name := "Test User"
		user := &models.User{ID: userID, Email: "test@example.com", Name: &name}

		mockAuthService.On("GetMe", mock.Anything, userID).Return(user, nil).Once()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
		req = withUserID(req, &userID)
		rr := httptest.NewRecorder()

		r := chi.NewRouter()
		r.Get("/api/v1/auth/me", handler.GetMe)
		r.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		var responseBody map[string]interface{}
		json.NewDecoder(rr.Body).Decode(&responseBody)
		data := responseBody["data"].(map[string]interface{})
		assert.Equal(t, "test@example.com", data["email"])

		mockAuthService.AssertExpectations(t)
	})

	t.Run("returns 401 when no user in context", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
		rr := httptest.NewRecorder()

		r := chi.NewRouter()
		r.Get("/api/v1/auth/me", handler.GetMe)
		r.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	t.Run("returns 401 when user not found in service", func(t *testing.T) {
		userID := uuid.New()

		mockAuthService.On("GetMe", mock.Anything, userID).Return(nil, fmt.Errorf("not found")).Once()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
		req = withUserID(req, &userID)
		rr := httptest.NewRecorder()

		r := chi.NewRouter()
		r.Get("/api/v1/auth/me", handler.GetMe)
		r.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
		mockAuthService.AssertExpectations(t)
	})
}

func TestAuthHandler_Logout(t *testing.T) {
	mockAuthService := new(MockAuthService)
	mockRefreshTokenService := new(MockRefreshTokenService)
	cfg := &config.Config{}

	handler := handlers.NewAuthHandler(mockAuthService, mockRefreshTokenService, cfg)

	t.Run("successful logout returns 200", func(t *testing.T) {
		userID := uuid.New()

		mockRefreshTokenService.On("RevokeAllUserTokens", mock.Anything, userID).Return(nil).Once()

		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
		req = withUserID(req, &userID)
		rr := httptest.NewRecorder()

		r := chi.NewRouter()
		r.Post("/api/v1/auth/logout", handler.Logout)
		r.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		var responseBody map[string]interface{}
		json.NewDecoder(rr.Body).Decode(&responseBody)
		assert.Equal(t, "Logged out successfully", responseBody["message"])

		mockRefreshTokenService.AssertExpectations(t)
	})

	t.Run("logout fails returns 500", func(t *testing.T) {
		userID := uuid.New()

		mockRefreshTokenService.On("RevokeAllUserTokens", mock.Anything, userID).Return(fmt.Errorf("db error")).Once()

		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
		req = withUserID(req, &userID)
		rr := httptest.NewRecorder()

		r := chi.NewRouter()
		r.Post("/api/v1/auth/logout", handler.Logout)
		r.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		mockRefreshTokenService.AssertExpectations(t)
	})

	t.Run("logout returns 401 when unauthenticated", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
		rr := httptest.NewRecorder()

		r := chi.NewRouter()
		r.Post("/api/v1/auth/logout", handler.Logout)
		r.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})
}

func TestAuthHandler_Refresh(t *testing.T) {
	mockAuthService := new(MockAuthService)
	mockRefreshTokenService := new(MockRefreshTokenService)
	cfg := &config.Config{}

	handler := handlers.NewAuthHandler(mockAuthService, mockRefreshTokenService, cfg)

	t.Run("successful refresh returns new tokens", func(t *testing.T) {
		refreshRes := &models.RefreshTokenResponse{AccessToken: "new_access", RefreshToken: "new_refresh"}

		mockRefreshTokenService.On("RefreshAccessToken", mock.Anything, "valid_refresh_tok").Return(refreshRes, nil).Once()

		body, _ := json.Marshal(models.RefreshTokenRequest{RefreshToken: "valid_refresh_tok"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		r := chi.NewRouter()
		r.Post("/api/v1/auth/refresh", handler.Refresh)
		r.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		var responseBody map[string]interface{}
		json.NewDecoder(rr.Body).Decode(&responseBody)
		data := responseBody["data"].(map[string]interface{})
		assert.Equal(t, "new_access", data["access_token"])
		assert.Equal(t, "new_refresh", data["refresh_token"])

		mockRefreshTokenService.AssertExpectations(t)
	})

	t.Run("refresh fails with invalid token returns 401", func(t *testing.T) {
		mockRefreshTokenService.On("RefreshAccessToken", mock.Anything, "expired_tok").Return(nil, fmt.Errorf("token expired")).Once()

		body, _ := json.Marshal(models.RefreshTokenRequest{RefreshToken: "expired_tok"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		r := chi.NewRouter()
		r.Post("/api/v1/auth/refresh", handler.Refresh)
		r.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
		var responseBody handlers.ErrorResponse
		json.NewDecoder(rr.Body).Decode(&responseBody)
		assert.Equal(t, "token expired", responseBody.Error)

		mockRefreshTokenService.AssertExpectations(t)
	})
}

func TestAuthHandler_DeleteMe(t *testing.T) {
	mockAuthService := new(MockAuthService)
	mockRefreshTokenService := new(MockRefreshTokenService)
	cfg := &config.Config{}

	handler := handlers.NewAuthHandler(mockAuthService, mockRefreshTokenService, cfg)

	t.Run("successful account deletion returns 200", func(t *testing.T) {
		userID := uuid.New()

		mockAuthService.On("DeleteAccount", mock.Anything, userID).Return(nil).Once()

		req := httptest.NewRequest(http.MethodDelete, "/api/v1/auth/me", nil)
		req = withUserID(req, &userID)
		rr := httptest.NewRecorder()

		r := chi.NewRouter()
		r.Delete("/api/v1/auth/me", handler.DeleteMe)
		r.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		var responseBody map[string]interface{}
		json.NewDecoder(rr.Body).Decode(&responseBody)
		assert.Equal(t, "Account and all associated data have been permanently deleted.", responseBody["message"])

		mockAuthService.AssertExpectations(t)
	})

	t.Run("delete fails returns 500", func(t *testing.T) {
		userID := uuid.New()

		mockAuthService.On("DeleteAccount", mock.Anything, userID).Return(fmt.Errorf("db error")).Once()

		req := httptest.NewRequest(http.MethodDelete, "/api/v1/auth/me", nil)
		req = withUserID(req, &userID)
		rr := httptest.NewRecorder()

		r := chi.NewRouter()
		r.Delete("/api/v1/auth/me", handler.DeleteMe)
		r.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		mockAuthService.AssertExpectations(t)
	})
}
