package service

import (
	"context"
	"fmt"
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
}

type authService struct {
	userRepo      repository.UserRepository
	jwtSecret     string
	jwtExpiration time.Duration
}

func NewAuthService(userRepo repository.UserRepository, jwtSecret string, jwtExpiration time.Duration) AuthService {
	return &authService{
		userRepo:      userRepo,
		jwtSecret:     jwtSecret,
		jwtExpiration: jwtExpiration,
	}
}

func (s *authService) Register(ctx context.Context, req models.CreateUserRequest) (*models.AuthResponse, error) {
	// TODO: Implement validation
	// TODO: Check if email already exists
	// TODO: Hash password
	// TODO: Create user
	// TODO: Generate JWT token
	return nil, fmt.Errorf("not implemented")
}

func (s *authService) Login(ctx context.Context, req models.LoginRequest) (*models.AuthResponse, error) {
	// TODO: Implement login logic
	// TODO: Verify password
	// TODO: Generate JWT token
	return nil, fmt.Errorf("not implemented")
}

func (s *authService) ValidateToken(tokenString string) (*uuid.UUID, error) {
	// TODO: Implement JWT validation
	return nil, fmt.Errorf("not implemented")
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
