package models

import (
	"time"

	"github.com/google/uuid"
)

// User represents a user in the system.
func StringPtr(s string) *string {
	return &s
}

type User struct {
	ID                    uuid.UUID  `json:"id"`
	Email                 string     `json:"email"`
	PasswordHash          string     `json:"-"`                                 // Never expose in JSON
	Name                  *string    `json:"name,omitempty"`                    // From current UserRepository
	Language              string     `json:"language"`                          // From current UserRepository
	ExpoPushToken         *string    `json:"-"`                                 // From current UserRepository, nullable
	SubscriptionTier      string     `json:"subscription_tier"`                 // From current UserRepository
	SubscriptionExpiry    *time.Time `json:"subscription_expires_at,omitempty"` // From current UserRepository, nullable
	IsFounder             bool       `json:"is_founder"`                        // From current UserRepository
	GoogleID              *string    `json:"google_id,omitempty"`
	AuthProvider          string     `json:"auth_provider"`
	IsConfirmed           bool       `json:"is_confirmed"`                      // From current UserRepository
	ConfirmationToken     *string    `json:"-"`                                 // From current UserRepository, nullable
	ConfirmationExpiresAt *time.Time `json:"-"`                                 // From current UserRepository, nullable
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
	DeviceID              *string    `json:"device_id,omitempty"`      // New: Stores unique device identifier from React Native
	TrialStartAt          *time.Time `json:"trial_start_at,omitempty"` // New: Timestamp when the free trial started
	TrialEndAt            *time.Time `json:"trial_end_at,omitempty"`   // New: Timestamp when the free trial ends (start_at + 7 days)
	TrialStatus           string     `json:"trial_status,omitempty"`   // New: 'inactive', 'active', 'expired', 'denied'
}

type CreateUserRequest struct {
	Email    string  `json:"email" validate:"required,email"`
	Password string  `json:"password" validate:"required,min=8"`
	Name     string  `json:"name,omitempty" validate:"max=100"`
	Language string  `json:"language,omitempty" validate:"omitempty,oneof=id en"`
	DeviceID *string `json:"device_id,omitempty"` // New: Optional device ID from frontend
}

type UpdateLanguageRequest struct {
	Language string `json:"language" validate:"required,oneof=id en"`
}

type UpdatePushTokenRequest struct {
	PushToken string `json:"push_token"`
}

type LoginRequest struct {
	Email    string  `json:"email" validate:"required,email"`
	Password string  `json:"password" validate:"required"`
	DeviceID *string `json:"device_id,omitempty"` // New: Optional device ID from frontend
}

type GoogleLoginRequest struct {
	IDToken  string  `json:"id_token" validate:"required"`
	DeviceID *string `json:"device_id,omitempty"`
}

type AuthResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	User         User   `json:"user"`
}
