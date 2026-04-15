package models

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID                    uuid.UUID  `json:"id"`
	Email                 string     `json:"email"`
	PasswordHash          string     `json:"-"` // Never expose in JSON
	Name                  *string    `json:"name,omitempty"`
	Language              string     `json:"language"`
	ExpoPushToken         string     `json:"-"` // never expose in API responses
	SubscriptionTier      string     `json:"subscription_tier"`
	SubscriptionExpiry    *time.Time `json:"subscription_expires_at,omitempty"`
	IsFounder             bool       `json:"is_founder"`
	IsConfirmed           bool       `json:"is_confirmed"`
	ConfirmationToken     *string    `json:"-"` // never expose in API responses
	ConfirmationExpiresAt *time.Time `json:"-"` // never expose in API responses
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
}

type CreateUserRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
	Name     string `json:"name,omitempty" validate:"max=100"`
	Language string `json:"language,omitempty" validate:"omitempty,oneof=id en"`
}

type UpdateLanguageRequest struct {
	Language string `json:"language" validate:"required,oneof=id en"`
}

type UpdatePushTokenRequest struct {
	PushToken string `json:"push_token"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type AuthResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	User         User   `json:"user"`
}
