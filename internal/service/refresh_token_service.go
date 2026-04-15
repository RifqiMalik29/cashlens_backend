package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/rifqimalik/cashlens-backend/internal/logger"
	"github.com/rifqimalik/cashlens-backend/internal/models"
	"github.com/rifqimalik/cashlens-backend/internal/repository"
)

type RefreshTokenService interface {
	GenerateRefreshToken(ctx context.Context, userID uuid.UUID, ip, userAgent string) (*models.RefreshToken, error)
	RefreshAccessToken(ctx context.Context, refreshToken string) (*models.RefreshTokenResponse, error)
	RevokeToken(ctx context.Context, token string) error
	RevokeAllUserTokens(ctx context.Context, userID uuid.UUID) error
	CleanupExpiredTokens(ctx context.Context) (int64, error)
}

type refreshTokenService struct {
	tokenRepo      repository.RefreshTokenRepository
	userRepo       repository.UserRepository
	jwtSecret      string
	accessExpiry   time.Duration
	refreshExpiry  time.Duration
	maxReuseWindow time.Duration
	log            *logger.Logger
}

func NewRefreshTokenService(
	tokenRepo repository.RefreshTokenRepository,
	userRepo repository.UserRepository,
	jwtSecret string,
	accessExpiry time.Duration,
	refreshExpiry time.Duration,
	maxReuseWindow time.Duration,
) RefreshTokenService {
	return &refreshTokenService{
		tokenRepo:      tokenRepo,
		userRepo:       userRepo,
		jwtSecret:      jwtSecret,
		accessExpiry:   accessExpiry,
		refreshExpiry:  refreshExpiry,
		maxReuseWindow: maxReuseWindow,
		log:            logger.GetDefault().With("component", "refresh_token_service"),
	}
}

// GenerateRefreshToken creates a new refresh token for the user
func (s *refreshTokenService) GenerateRefreshToken(ctx context.Context, userID uuid.UUID, ip, userAgent string) (*models.RefreshToken, error) {
	// Generate cryptographically secure random token
	tokenString, err := generateSecureToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	token := &models.RefreshToken{
		ID:        uuid.New(),
		UserID:    userID,
		Token:     tokenString,
		ExpiresAt: time.Now().Add(s.refreshExpiry),
		IPAddress: ip,
		UserAgent: userAgent,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err = s.tokenRepo.Create(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("failed to store refresh token: %w", err)
	}

	s.log.Info("Refresh token generated", "user_id", userID, "token_id", token.ID)
	return token, nil
}

// RefreshAccessToken validates the refresh token and returns new access and refresh tokens
func (s *refreshTokenService) RefreshAccessToken(ctx context.Context, refreshToken string) (*models.RefreshTokenResponse, error) {
	log := s.log.With("operation", "refresh_token")

	// Get the token from database
	storedToken, err := s.tokenRepo.GetByToken(ctx, refreshToken)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token")
	}

	// Check if token is revoked
	if storedToken.RevokedAt != nil {
		// SECURITY: Token reuse detected - this might be a compromised token
		// Revoke the entire token family
		s.revokeTokenFamily(ctx, refreshToken, storedToken.UserID)
		log.Error("Potential token compromise detected - token family revoked",
			"user_id", storedToken.UserID,
			"token_id", storedToken.ID,
		)
		return nil, fmt.Errorf("token has been revoked")
	}

	// Check if token is expired
	if time.Now().After(storedToken.ExpiresAt) {
		// Token expired - revoke it
		s.tokenRepo.Revoke(ctx, storedToken.ID)
		log.Info("Expired refresh token revoked", "token_id", storedToken.ID)
		return nil, fmt.Errorf("refresh token expired")
	}

	// Verify user still exists
	_, err = s.userRepo.GetByID(ctx, storedToken.UserID)
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}

	// Generate new access token
	accessToken, err := generateTokenWithExpiry(storedToken.UserID, s.jwtSecret, s.accessExpiry)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	// TOKEN ROTATION: Generate new refresh token
	newRefreshToken, err := s.GenerateRefreshToken(ctx, storedToken.UserID, storedToken.IPAddress, storedToken.UserAgent)
	if err != nil {
		return nil, fmt.Errorf("failed to generate new refresh token: %w", err)
	}

	// Mark old token as replaced (for rotation tracking)
	s.tokenRepo.MarkReplaced(ctx, storedToken.ID, newRefreshToken.ID)

	// Revoke old token
	s.tokenRepo.Revoke(ctx, storedToken.ID)

	log.Info("Tokens refreshed successfully",
		"user_id", storedToken.UserID,
		"old_token_id", storedToken.ID,
		"new_token_id", newRefreshToken.ID,
	)

	return &models.RefreshTokenResponse{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken.Token,
		TokenType:    "Bearer",
		ExpiresIn:    int64(s.accessExpiry.Seconds()),
	}, nil
}

// RevokeToken revokes a specific refresh token
func (s *refreshTokenService) RevokeToken(ctx context.Context, token string) error {
	storedToken, err := s.tokenRepo.GetByToken(ctx, token)
	if err != nil {
		return fmt.Errorf("token not found")
	}

	err = s.tokenRepo.Revoke(ctx, storedToken.ID)
	if err != nil {
		return fmt.Errorf("failed to revoke token: %w", err)
	}

	s.log.Info("Refresh token revoked", "token_id", storedToken.ID, "user_id", storedToken.UserID)
	return nil
}

// RevokeAllUserTokens revokes all active refresh tokens for a user
func (s *refreshTokenService) RevokeAllUserTokens(ctx context.Context, userID uuid.UUID) error {
	err := s.tokenRepo.RevokeByUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to revoke user tokens: %w", err)
	}

	s.log.Info("All user refresh tokens revoked", "user_id", userID)
	return nil
}

// CleanupExpiredTokens removes expired and long-revoked tokens from database
func (s *refreshTokenService) CleanupExpiredTokens(ctx context.Context) (int64, error) {
	// Delete tokens that expired or were revoked more than 30 days ago
	cleanupBefore := time.Now().Add(-30 * 24 * time.Hour)
	deleted, err := s.tokenRepo.DeleteExpired(ctx, cleanupBefore)
	if err != nil {
		return 0, fmt.Errorf("cleanup failed: %w", err)
	}

	if deleted > 0 {
		s.log.Info("Cleaned up expired refresh tokens", "deleted_count", deleted)
	}

	return deleted, nil
}

// revokeTokenFamily revokes all tokens in the rotation chain (security feature)
func (s *refreshTokenService) revokeTokenFamily(ctx context.Context, tokenString string, userID uuid.UUID) {
	log := s.log.With("operation", "revoke_token_family")

	// Revoke all tokens for this user (safety measure)
	err := s.tokenRepo.RevokeByUserID(ctx, userID)
	if err != nil {
		log.Error("Failed to revoke token family", "error", err, "user_id", userID)
	} else {
		log.Warn("Token family revoked", "user_id", userID)
	}
}

// generateSecureToken creates a cryptographically secure random token
func generateSecureToken() (string, error) {
	bytes := make([]byte, 64)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// generateTokenWithExpiry creates a JWT token with custom expiry
func generateTokenWithExpiry(userID uuid.UUID, secret string, expiration time.Duration) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID.String(),
		"exp":     time.Now().Add(expiration).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}
