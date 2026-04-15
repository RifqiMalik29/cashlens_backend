package service

import (
	"context"
	"crypto/rand"
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

// ErrEmailNotConfirmed is returned when a user tries to login without confirming their email.
// It carries the email so the handler can include it in the response.
type ErrEmailNotConfirmed struct {
	Email string
}

func (e *ErrEmailNotConfirmed) Error() string {
	return "EMAIL_NOT_CONFIRMED"
}

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
	DeleteAccount(ctx context.Context, userID uuid.UUID) error
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
	// Check if user exists
	_, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err == nil {
		return nil, fmt.Errorf("email already registered")
	}

	hashedPassword, err := hashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

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
	user := &models.User{
		ID:                    uuid.New(),
		Email:                 req.Email,
		PasswordHash:          hashedPassword,
		Name:                  &req.Name,
		Language:              lang,
		SubscriptionTier:      "free",
		IsConfirmed:           false,
		ConfirmationToken:     &token,
		ConfirmationExpiresAt: &expiresAt,
		CreatedAt:             time.Now(),
		UpdatedAt:             time.Now(),
	}

	err = s.userRepo.Create(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Seed default categories
	if err := s.categorySeedingService.SeedDefaultCategories(ctx, user.ID); err != nil {
		slog.Error("Failed to seed categories for new user", "user_id", user.ID, "error", err)
	}

	// Send confirmation email asynchronously
	go func() {
		if err := s.mailer.SendConfirmationEmail(user.Email, token); err != nil {
			slog.Error("Failed to send confirmation email", "email", user.Email, "error", err)
		}
	}()

	// Generate JWT
	accessToken, err := s.generateToken(user.ID, s.jwtExpiration, s.jwtSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	return &models.AuthResponse{
		AccessToken: accessToken,
		User:        *user,
	}, nil
}

func (s *authService) Login(ctx context.Context, req models.LoginRequest) (*models.AuthResponse, error) {
	user, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil, fmt.Errorf("invalid email or password")
	}

	if !checkPasswordHash(req.Password, user.PasswordHash) {
		return nil, fmt.Errorf("invalid email or password")
	}

	// Check if email is confirmed
	if !user.IsConfirmed {
		if err := s.sendConfirmationOTP(ctx, user); err != nil {
			slog.Error("Failed to resend OTP on login", "email", user.Email, "error", err)
		}
		return nil, &ErrEmailNotConfirmed{Email: user.Email}
	}

	// Generate JWT
	token, err := s.generateToken(user.ID, s.jwtExpiration, s.jwtSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	return &models.AuthResponse{
		AccessToken: token,
		User:        *user,
	}, nil
}

func (s *authService) ValidateToken(tokenString string) (*uuid.UUID, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.jwtSecret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		userIDStr, ok := claims["user_id"].(string)
		if !ok {
			return nil, fmt.Errorf("invalid token claims")
		}

		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			return nil, fmt.Errorf("invalid user id in token")
		}

		return &userID, nil
	}

	return nil, fmt.Errorf("invalid token")
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
		return map[string]any{
			"is_linked": false,
		}, nil
	}

	return map[string]any{
		"is_linked": true,
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

func (s *authService) DeleteAccount(ctx context.Context, userID uuid.UUID) error {
	slog.Warn("Deleting account and all associated data", "user_id", userID.String())
	return s.userRepo.Delete(ctx, userID)
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

	return s.sendConfirmationOTP(ctx, user)
}

// sendConfirmationOTP generates a new OTP, persists it, and dispatches the confirmation email.
// It returns an error if OTP generation or DB persistence fails.
func (s *authService) sendConfirmationOTP(ctx context.Context, user *models.User) error {
	token, err := generateOTP()
	if err != nil {
		return fmt.Errorf("failed to generate verification code: %w", err)
	}
	expiresAt := time.Now().Add(10 * time.Minute)

	if err := s.userRepo.UpdateConfirmationToken(ctx, user.ID, token, expiresAt); err != nil {
		return fmt.Errorf("failed to update verification code: %w", err)
	}

	email := user.Email
	go func() {
		if err := s.mailer.SendConfirmationEmail(email, token); err != nil {
			slog.Error("Failed to send confirmation email", "email", email, "error", err)
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

func (s *authService) generateToken(userID uuid.UUID, expiration time.Duration, secret string) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID.String(),
		"exp":     time.Now().Add(expiration).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}
