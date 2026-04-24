package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/rifqimalik/cashlens-backend/internal/models"
	"github.com/rifqimalik/cashlens-backend/internal/repository"
)

// SubscriptionExpiryCheck automatically downgrades expired premium users to free tier
func SubscriptionExpiryCheck(userRepo repository.UserRepository, eventRepo repository.SubscriptionEventRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, ok := r.Context().Value(UserIDKey).(*uuid.UUID)
			if !ok {
				next.ServeHTTP(w, r)
				return
			}

			// Check and auto-downgrade if needed (non-blocking)
			go func(uid uuid.UUID) {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				user, err := userRepo.GetByID(ctx, uid)
				if err != nil {
					return
				}

				// Only check premium users with an expiry date
				if user.SubscriptionTier != "premium" || user.SubscriptionExpiry == nil {
					return
				}

				// Check if expired
				if time.Now().After(*user.SubscriptionExpiry) {
					// Downgrade to free
					user.SubscriptionTier = "free"
					user.SubscriptionExpiry = nil
					userRepo.Update(ctx, user)

					// Record expiration event
					eventType := "expired"
					event := &models.SubscriptionEvent{
						ID:        uuid.New(),
						UserID:    uid,
						EventType: eventType,
						CreatedAt: time.Now(),
					}
					eventRepo.Create(ctx, event)
				}
			}(*userID)

			next.ServeHTTP(w, r)
		})
	}
}
