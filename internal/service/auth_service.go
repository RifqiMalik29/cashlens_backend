package service

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/rifqimalik/cashlens-backend/internal/models"
	"github.com/rifqimalik/cashlens-backend/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

type AuthService interface {
	Register(ctx context.Context, req models.CreateUserRequest) (*models.AuthResponse, error)
	Login(ctx context.Context, req models.LoginRequest) (*models.AuthResponse, error)
	ValidateToken(tokenString string) (*uuid.UUID, error)
	GetMe(ctx context.Context, userID uuid.UUID) (*models.User, error)
	GetTelegramStatus(ctx context.Context, userID uuid.UUID) (map[string]any, error)
}

type authService struct {
	userRepo      repository.UserRepository
	chatRepo      repository.ChatLinkRepository
	jwtSecret     string
	jwtExpiration time.Duration
}

func NewAuthService(userRepo repository.UserRepository, chatRepo repository.ChatLinkRepository, jwtSecret string, jwtExpiration time.Duration) AuthService {
	return &authService{
		userRepo:      userRepo,
		chatRepo:      chatRepo,
		jwtSecret:     jwtSecret,
		jwtExpiration: jwtExpiration,
	}
}

var emailRegex = regexp.MustCompile(`^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,4}$`)

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

	// Create user
	user = &models.User{
		ID:           uuid.New(),
		Email:        req.Email,
		PasswordHash: p,
		Name:         &req.Name,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	err = s.userRepo.Create(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("Register failed: %w", err)
	}

	// Generate JWT token
	token, err := generateToken(user.ID, s.jwtSecret, s.jwtExpiration)
	if err != nil {
		return nil, fmt.Errorf("Token failed to produce: %w", err)
	}

	return &models.AuthResponse{
		AccessToken: token,
		User:        *user,
	}, nil
}

func (s *authService) Login(ctx context.Context, req models.LoginRequest) (*models.AuthResponse, error) {
	res, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil, fmt.Errorf("User not found: %w", err)
	}

	// Verify password
	status := checkPasswordHash(req.Password, res.PasswordHash)
	if !status {
		return nil, fmt.Errorf("Invalid email or password")
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

func (s *authService) GetTelegramStatus(ctx context.Context, userID uuid.UUID) (map[string]any, error) {
	link, err := s.chatRepo.GetByUserID(ctx, userID, "telegram")
	if err != nil {
		return map[string]any{
			"is_linked": false,
		}, nil
	}

	return map[string]any{
		"is_linked": link.IsActive,
		"chat_id":   link.ChatID,
	}, nil
}

// Helper methods (to be implemented)
func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
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

func isEmailValid(e string) bool {
	return emailRegex.MatchString(e)
}
