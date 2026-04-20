package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/rifqimalik/cashlens-backend/internal/models"
	"github.com/rifqimalik/cashlens-backend/internal/repository"
)

type GoogleAuthService interface {
	LoginWithGoogle(ctx context.Context, idToken string, deviceID *string) (*models.AuthResponse, error)
}

type googleAuthService struct {
	userRepo                repository.UserRepository
	categorySeedingService  CategorySeedingService
	trialEligibilityService TrialEligibilityService
	jwtSecret               string
	jwtExpiration           time.Duration
	googleTokenInfoBaseURL  string
	googleClientID          string
}

func NewGoogleAuthService(
	userRepo repository.UserRepository,
	categorySeedingService CategorySeedingService,
	trialEligibilityService TrialEligibilityService,
	jwtSecret string,
	jwtExpiration time.Duration,
	googleTokenInfoBaseURL string,
	googleClientID string,
) GoogleAuthService {
	return &googleAuthService{
		userRepo:                userRepo,
		categorySeedingService:  categorySeedingService,
		trialEligibilityService: trialEligibilityService,
		jwtSecret:               jwtSecret,
		jwtExpiration:           jwtExpiration,
		googleTokenInfoBaseURL:  googleTokenInfoBaseURL,
		googleClientID:          googleClientID,
	}
}

type googleTokenClaims struct {
	Sub   string `json:"sub"`
	Email string `json:"email"`
	Name  string `json:"name"`
	Aud   string `json:"aud"`
	Exp   string `json:"exp"`
}

func (s *googleAuthService) verifyGoogleToken(idToken string) (*googleTokenClaims, error) {
	url := fmt.Sprintf("%s/tokeninfo?id_token=%s", s.googleTokenInfoBaseURL, idToken)
	resp, err := http.Get(url) //nolint:gosec
	if err != nil {
		return nil, fmt.Errorf("failed to contact Google: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("invalid Google token")
	}

	var claims googleTokenClaims
	if err := json.NewDecoder(resp.Body).Decode(&claims); err != nil {
		return nil, fmt.Errorf("failed to parse Google token response: %w", err)
	}

	if claims.Exp != "" {
		expUnix, err := strconv.ParseInt(claims.Exp, 10, 64)
		if err == nil && time.Now().Unix() > expUnix {
			return nil, fmt.Errorf("invalid Google token")
		}
	}

	if s.googleClientID != "" && claims.Aud != s.googleClientID {
		return nil, fmt.Errorf("invalid Google token")
	}

	if claims.Sub == "" {
		return nil, fmt.Errorf("invalid Google token")
	}

	if claims.Email == "" {
		return nil, fmt.Errorf("Google account has no email")
	}

	return &claims, nil
}

func (s *googleAuthService) LoginWithGoogle(ctx context.Context, idToken string, deviceID *string) (*models.AuthResponse, error) {
	claims, err := s.verifyGoogleToken(idToken)
	if err != nil {
		return nil, err
	}

	// 1. Find by google_id
	user, err := s.userRepo.GetByGoogleID(ctx, claims.Sub)
	if err == nil {
		accessToken, err := generateTokenWithExpiry(user.ID, s.jwtSecret, s.jwtExpiration)
		if err != nil {
			return nil, fmt.Errorf("failed to generate token: %w", err)
		}
		return &models.AuthResponse{AccessToken: accessToken, User: *user}, nil
	}

	// 2. Find by email (auto-merge existing email/password user)
	user, err = s.userRepo.GetByEmail(ctx, claims.Email)
	if err == nil {
		// Link Google ID to existing account. auth_provider remains "email" intentionally —
		// it reflects the primary signup method, not the current login method.
		if err := s.userRepo.UpdateGoogleID(ctx, user.ID, claims.Sub); err != nil {
			return nil, fmt.Errorf("failed to link Google account: %w", err)
		}
		accessToken, err := generateTokenWithExpiry(user.ID, s.jwtSecret, s.jwtExpiration)
		if err != nil {
			return nil, fmt.Errorf("failed to generate token: %w", err)
		}
		return &models.AuthResponse{AccessToken: accessToken, User: *user}, nil
	}

	// 3. New user
	googleID := claims.Sub
	authProvider := "google"
	name := claims.Name
	newUser := &models.User{
		ID:               uuid.New(),
		Email:            claims.Email,
		Name:             &name,
		Language:         "id",
		AuthProvider:     authProvider,
		GoogleID:         &googleID,
		IsConfirmed:      true,
		SubscriptionTier: "free",
		TrialStatus:      "inactive",
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	if err := s.userRepo.Create(ctx, newUser); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	if _, err := s.trialEligibilityService.CheckAndSetTrial(newUser, deviceID); err != nil {
		slog.Error("Failed to check trial for Google user", "user_id", newUser.ID, "error", err)
	}

	if err := s.categorySeedingService.SeedDefaultCategories(ctx, newUser.ID); err != nil {
		slog.Error("Failed to seed categories for Google user", "user_id", newUser.ID, "error", err)
	}

	accessToken, err := generateTokenWithExpiry(newUser.ID, s.jwtSecret, s.jwtExpiration)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	return &models.AuthResponse{AccessToken: accessToken, User: *newUser}, nil
}
