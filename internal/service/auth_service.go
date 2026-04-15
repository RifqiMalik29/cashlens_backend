package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/rifqimalik/cashlens-backend/internal/models"
	"github.com/rifqimalik/cashlens-backend/internal/pkg/mailer"
	"github.com/rifqimalik/cashlens-backend/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

type AuthService interface {
	Register(ctx context.Context, req models.CreateUserRequest) (*models.AuthResponse, error)
	Login(ctx context.Context, req models.LoginRequest) (*models.AuthResponse, error)
	ConfirmEmail(ctx context.Context, email, otp string) error
	ResendConfirmation(ctx context.Context, email string) error
	ValidateToken(tokenString string) (*uuid.UUID, error)
	GetMe(ctx context.Context, userID uuid.UUID) (*models.User, error)
	UpdateLanguage(ctx context.Context, userID uuid.UUID, language string) error
	UpdatePushToken(ctx context.Context, userID uuid.UUID, token string) error
	GetTelegramStatus(ctx context.Context, userID uuid.UUID) (map[string]any, error)
	UnlinkTelegram(ctx context.Context, userID uuid.UUID) error
}

type authService struct {
	userRepo               repository.UserRepository
	categorySeedingService CategorySeedingService
	chatRepo               repository.ChatLinkRepository
	mailer                 mailer.Mailer
	jwtSecret              string
	jwtExpiration          time.Duration
}

func NewAuthService(
	userRepo repository.UserRepository,
	categorySeedingService CategorySeedingService,
	chatRepo repository.ChatLinkRepository,
	mailer mailer.Mailer,
	jwtSecret string,
	jwtExpiration time.Duration,
) AuthService {
	return &authService{
		userRepo:               userRepo,
		categorySeedingService: categorySeedingService,
		chatRepo:               chatRepo,
		mailer:                 mailer,
		jwtSecret:              jwtSecret,
		jwtExpiration:          jwtExpiration,
	}
}

func (s *authService) Register(ctx context.Context, req models.CreateUserRequest) (*models.AuthResponse, error) {
	// Check if email already exists
	user, err := s.userRepo.GetByEmail(ctx, req.Email)
	if user != nil && err == nil {
		return nil, fmt.Errorf("Email is already registered")
	}

	// Hash password
	p, err := hashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("Failed to hash password: %w", err)
	}

	// Default language to "id" if not provided
	lang := req.Language
	if lang == "" {
		lang = "id"
	}

	// Generate 6-digit OTP
	token, err := generateOTP()
	if err != nil {
		return nil, fmt.Errorf("failed to generate OTP: %w", err)
	}
	expiresAt := time.Now().Add(10 * time.Minute)

	// Create user
	user = &models.User{
		ID:                    uuid.New(),
		Email:                 req.Email,
		PasswordHash:          p,
		Name:                  &req.Name,
		Language:              lang,
		IsConfirmed:           false,
		ConfirmationToken:     &token,
		ConfirmationExpiresAt: &expiresAt,
		CreatedAt:             time.Now(),
		UpdatedAt:             time.Now(),
	}
	err = s.userRepo.Create(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("Register failed: %w", err)
	}

	// Seed default categories for new user (non-fatal: registration still succeeds)
	if err := s.categorySeedingService.SeedDefaultCategories(ctx, user.ID); err != nil {
		slog.Error("Failed to seed default categories for new user", "user_id", user.ID, "error", err)
	}

	// Send confirmation email asynchronously
	go func() {
		if err := s.mailer.SendConfirmationEmail(user.Email, token); err != nil {
			slog.Error("Failed to send confirmation email", "email", user.Email, "error", err)
		}
	}()

	return &models.AuthResponse{
		User: *user,
	}, nil
}

func (s *authService) Login(ctx context.Context, req models.LoginRequest) (*models.AuthResponse, error) {
	res, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		// Return generic error to prevent user enumeration
		return nil, fmt.Errorf("invalid email or password")
	}

	// Verify password
	status := checkPasswordHash(req.Password, res.PasswordHash)
	if !status {
		// Return same generic error for consistency
		return nil, fmt.Errorf("invalid email or password")
	}

	// Check if email is confirmed
	if !res.IsConfirmed {
		return nil, fmt.Errorf("please confirm your email before logging in")
	}

	// Generate JWT token
	token, err := generateToken(res.ID, s.jwtSecret, s.jwtExpiration)
	if err != nil {
		return nil, fmt.Errorf("Token failed to produced: %w", err)
	}

	return &models.AuthResponse{
		AccessToken: token,
		User:        *res,
	}, nil
}

func (s *authService) ValidateToken(tokenString string) (*uuid.UUID, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.jwtSecret), nil
	})

	if err != nil || !token.Valid {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims")
	}
	userIDStr, ok := claims["user_id"].(string)
	if !ok {
		return nil, fmt.Errorf("user_id not found in token")
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid user_id in token: %w", err)
	}

	return &userID, nil
}

func (s *authService) GetMe(ctx context.Context, userID uuid.UUID) (*models.User, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

func (s *authService) UpdateLanguage(ctx context.Context, userID uuid.UUID, language string) error {
	return s.userRepo.UpdateLanguage(ctx, userID, language)
}

func (s *authService) UpdatePushToken(ctx context.Context, userID uuid.UUID, token string) error {
	return s.userRepo.UpdatePushToken(ctx, userID, token)
}

func (s *authService) GetTelegramStatus(ctx context.Context, userID uuid.UUID) (map[string]any, error) {
	link, err := s.chatRepo.GetByUserID(ctx, userID, "telegram")
	if err != nil {
		return map[string]any{"is_linked": false}, nil
	}
	return map[string]any{
		"is_linked": link.IsActive,
		"chat_id":   link.ChatID,
	}, nil
}

func (s *authService) UnlinkTelegram(ctx context.Context, userID uuid.UUID) error {
	link, err := s.chatRepo.GetByUserID(ctx, userID, "telegram")
	if err != nil {
		return fmt.Errorf("no telegram link found for user")
	}
	return s.chatRepo.Delete(ctx, link.ID)
}

func (s *authService) ConfirmEmail(ctx context.Context, email, otp string) error {
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return fmt.Errorf("user not found")
	}

	if user.IsConfirmed {
		return fmt.Errorf("email is already confirmed")
	}

	if user.ConfirmationToken == nil || *user.ConfirmationToken != otp {
		return fmt.Errorf("invalid verification code")
	}

	if user.ConfirmationExpiresAt != nil && time.Now().After(*user.ConfirmationExpiresAt) {
		return fmt.Errorf("verification code has expired")
	}

	return s.userRepo.UpdateConfirmationStatus(ctx, user.ID, true)
}

func (s *authService) ResendConfirmation(ctx context.Context, email string) error {
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return fmt.Errorf("user not found")
	}

	if user.IsConfirmed {
		return fmt.Errorf("email is already confirmed")
	}

	// Generate new 6-digit OTP
	token, err := generateOTP()
	if err != nil {
		return fmt.Errorf("failed to generate verification code: %w", err)
	}
	expiresAt := time.Now().Add(10 * time.Minute)

	// Update user with new token
	err = s.userRepo.UpdateConfirmationToken(ctx, user.ID, token, expiresAt)
	if err != nil {
		return fmt.Errorf("failed to update verification code: %w", err)
	}

	// Send confirmation email asynchronously
	go func() {
		if err := s.mailer.SendConfirmationEmail(user.Email, token); err != nil {
			slog.Error("Failed to send confirmation email", "email", user.Email, "error", err)
		}
	}()

	return nil
}

// Helper methods
func generateOTP() (string, error) {
	b := make([]byte, 3)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	// Generate a 6-digit number (000000-999999)
	otp := fmt.Sprintf("%06d", (int(b[0])<<16|int(b[1])<<8|int(b[2]))%1000000)
	return otp, nil
}

func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	return string(bytes), err
}

func checkPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func generateToken(userID uuid.UUID, secret string, expiration time.Duration) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID.String(),
		"exp":     time.Now().Add(expiration).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}
