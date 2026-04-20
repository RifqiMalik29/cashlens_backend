package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rifqimalik/cashlens-backend/internal/models"
	"github.com/rifqimalik/cashlens-backend/internal/repository"
)

// RevenueCatService handles the business logic for RevenueCat webhooks
type RevenueCatService struct {
	userRepo  repository.UserRepository
	eventRepo repository.SubscriptionEventRepository
}

// NewRevenueCatService creates a new instance of RevenueCatService
func NewRevenueCatService(userRepo repository.UserRepository, eventRepo repository.SubscriptionEventRepository) *RevenueCatService {
	return &RevenueCatService{
		userRepo:  userRepo,
		eventRepo: eventRepo,
	}
}

// ProcessWebhook processes a RevenueCat webhook event
func (s *RevenueCatService) ProcessWebhook(ctx context.Context, event *models.RevenueCatEvent) error {
	userID, err := uuid.Parse(event.AppUserID)
	if err != nil {
		return fmt.Errorf("failed to parse user ID from webhook: %w", err)
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user for webhook processing: %w", err)
	}

	// Default to "premium" if entitlements are present, could be made more granular
	var tier string
	if len(event.EntitlementIDs) > 0 {
		// You could have a mapping from entitlement ID to tier name
		tier = "premium"
	} else {
		tier = "free"
	}

	var newExpiry *time.Time
	if event.ExpiresAtMS != nil && *event.ExpiresAtMS > 0 {
		expiry := time.Unix(0, *event.ExpiresAtMS*int64(time.Millisecond))
		newExpiry = &expiry
	}

	// Update user subscription status based on the event type
	switch event.Type {
	case "INITIAL_PURCHASE", "RENEWAL", "UNCANCELLATION", "PRODUCT_CHANGE":
		user.SubscriptionTier = tier
		user.SubscriptionExpiry = newExpiry
		if err := s.userRepo.Update(ctx, user); err != nil {
			return fmt.Errorf("failed to update user subscription on '%s' event: %w", event.Type, err)
		}

	case "CANCELLATION":
		// The subscription is still valid until the expiration date.
		// We can simply log this event. The `EXPIRATION` event will handle the downgrade.
		// If the cancellation is for a reason that requires immediate revocation, handle it here.
		if event.CancelReason == "CUSTOMER_SERVICE" || event.CancelReason == "BILLING_ERROR" {
			user.SubscriptionTier = "free"
			user.SubscriptionExpiry = nil // Revoke immediately
			if err := s.userRepo.Update(ctx, user); err != nil {
				return fmt.Errorf("failed to immediately revoke subscription on cancellation: %w", err)
			}
		}

	case "EXPIRATION":
		user.SubscriptionTier = "free"
		user.SubscriptionExpiry = nil
		if err := s.userRepo.Update(ctx, user); err != nil {
			return fmt.Errorf("failed to downgrade user subscription on expiration: %w", err)
		}

	case "TEST":
		// This is a test event from RevenueCat, log it and do nothing.
		fmt.Printf("Received RevenueCat test event for user %s
", userID)
		return nil
	}

	// Record the event in the database
	subEvent := &models.SubscriptionEvent{
		ID:                uuid.New(),
		UserID:            userID,
		EventType:         event.Type,
		Plan:              safeString(event.ProductID),
		PricePaid:         safeFloat64(event.Price),
		ExternalInvoiceID: safeString(event.TransactionID), // Using TransactionID as the unique identifier
		CancelReason:      safeString(event.CancelReason),
		CreatedAt:         time.Now().UTC(),
	}

	if err := s.eventRepo.Create(ctx, subEvent); err != nil {
		return fmt.Errorf("failed to record subscription event: %w", err)
	}

	return nil
}

// safeString is a helper to dereference a string pointer or return an empty string
func safeString(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// safeFloat64 is a helper to dereference a float64 pointer or return 0
func safeFloat64(p *float64) float64 {
	if p == nil {
		return 0
	}
	return *p
}
