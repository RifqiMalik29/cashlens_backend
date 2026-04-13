# Expiry Reminder Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Send subscription expiry reminders to premium users via Telegram and Expo Push Notifications, 3 days and 1 day before their subscription expires.

**Architecture:** A new `ExpiryReminderService` follows the same pattern as `WinBackService` — a daily goroutine queries eligible users from a `ReminderRepository`, sends messages on both channels independently, and marks sent to prevent duplicates. Push tokens are stored on the user record via a new `PATCH /auth/push-token` endpoint.

**Tech Stack:** Go, pgx/v5, Expo Push API (`https://exp.host/push/send`), Telegram Bot API, chi router.

---

## File Map

| File | Action | Responsibility |
|---|---|---|
| `migrations/00018_add_reminder_fields_to_users.sql` | Create | Add `reminder_3d_sent_at`, `reminder_1d_sent_at`, `expo_push_token` columns |
| `internal/models/user.go` | Modify | Add `ExpoPushToken` field + `UpdatePushTokenRequest` type |
| `internal/repository/user_repository.go` | Modify | Add `UpdatePushToken`, scan `expo_push_token` in `GetByID`/`GetByEmail` |
| `internal/repository/reminder_repository.go` | Create | `GetUsersEligibleForReminder`, `MarkReminderSent` |
| `internal/service/expiry_reminder_service.go` | Create | `RunReminders` — query users, send Telegram + Expo push, mark sent |
| `internal/service/auth_service.go` | Modify | Add `UpdatePushToken` to interface + implementation |
| `internal/handlers/auth.go` | Modify | Add `UpdatePushToken` handler |
| `cmd/server/main.go` | Modify | Wire `ReminderRepository`, `ExpiryReminderService`, two scheduler goroutines, new route |

---

### Task 1: Migration

**Files:**
- Create: `migrations/00018_add_reminder_fields_to_users.sql`

- [ ] **Step 1: Create migration file**

```sql
-- +goose Up
-- +goose StatementBegin
ALTER TABLE users ADD COLUMN IF NOT EXISTS reminder_3d_sent_at TIMESTAMPTZ;
ALTER TABLE users ADD COLUMN IF NOT EXISTS reminder_1d_sent_at TIMESTAMPTZ;
ALTER TABLE users ADD COLUMN IF NOT EXISTS expo_push_token VARCHAR(255);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE users DROP COLUMN IF EXISTS reminder_3d_sent_at;
ALTER TABLE users DROP COLUMN IF EXISTS reminder_1d_sent_at;
ALTER TABLE users DROP COLUMN IF EXISTS expo_push_token;
-- +goose StatementEnd
```

- [ ] **Step 2: Verify build still passes**

```bash
go build ./...
```

Expected: no output (clean build).

- [ ] **Step 3: Commit**

```bash
git add migrations/00018_add_reminder_fields_to_users.sql
git commit -m "feat: add reminder and push token columns to users table"
```

---

### Task 2: Model update

**Files:**
- Modify: `internal/models/user.go`

- [ ] **Step 1: Add `ExpoPushToken` to `User` struct**

In `internal/models/user.go`, add the field after `Language`:

```go
type User struct {
	ID                 uuid.UUID  `json:"id"`
	Email              string     `json:"email"`
	PasswordHash       string     `json:"-"`
	Name               *string    `json:"name,omitempty"`
	Language           string     `json:"language"`
	ExpoPushToken      string     `json:"-"` // never expose in API responses
	SubscriptionTier   string     `json:"subscription_tier"`
	SubscriptionExpiry *time.Time `json:"subscription_expires_at,omitempty"`
	IsFounder          bool       `json:"is_founder"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}
```

- [ ] **Step 2: Add `UpdatePushTokenRequest` type**

Add after `UpdateLanguageRequest`:

```go
type UpdatePushTokenRequest struct {
	PushToken string `json:"push_token"`
}
```

Note: no `validate:"required"` — empty string is valid (clears the token on logout).

- [ ] **Step 3: Verify build**

```bash
go build ./...
```

Expected: no output.

- [ ] **Step 4: Commit**

```bash
git add internal/models/user.go
git commit -m "feat: add ExpoPushToken to User model and UpdatePushTokenRequest"
```

---

### Task 3: UserRepository — scan push token + UpdatePushToken

**Files:**
- Modify: `internal/repository/user_repository.go`

- [ ] **Step 1: Add `UpdatePushToken` to the interface**

In `internal/repository/user_repository.go`, add to the `UserRepository` interface:

```go
UpdatePushToken(ctx context.Context, userID uuid.UUID, token string) error
```

Full updated interface:

```go
type UserRepository interface {
	Create(ctx context.Context, user *models.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	GetByEmail(ctx context.Context, email string) (*models.User, error)
	Update(ctx context.Context, user *models.User) error
	UpdateSubscription(ctx context.Context, userID uuid.UUID, tier string, expiresAt *time.Time) error
	UpdateFounder(ctx context.Context, userID uuid.UUID, isFounder bool) error
	UpdateLanguage(ctx context.Context, userID uuid.UUID, language string) error
	UpdatePushToken(ctx context.Context, userID uuid.UUID, token string) error
	Delete(ctx context.Context, id uuid.UUID) error
}
```

- [ ] **Step 2: Update `GetByID` to scan `expo_push_token`**

Replace the existing `GetByID` method:

```go
func (r *userRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	user := &models.User{}
	query := `
		SELECT id, email, password_hash, name, language, COALESCE(expo_push_token, ''), subscription_tier, subscription_expires_at, is_founder, created_at, updated_at
		FROM users WHERE id = $1
	`
	err := r.db.QueryRow(ctx, query, id).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.Name, &user.Language, &user.ExpoPushToken,
		&user.SubscriptionTier, &user.SubscriptionExpiry, &user.IsFounder,
		&user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return user, nil
}
```

- [ ] **Step 3: Update `GetByEmail` to scan `expo_push_token`**

Replace the existing `GetByEmail` method:

```go
func (r *userRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	user := &models.User{}
	query := `
		SELECT id, email, password_hash, name, language, COALESCE(expo_push_token, ''), subscription_tier, subscription_expires_at, is_founder, created_at, updated_at
		FROM users WHERE email = $1
	`
	err := r.db.QueryRow(ctx, query, email).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.Name, &user.Language, &user.ExpoPushToken,
		&user.SubscriptionTier, &user.SubscriptionExpiry, &user.IsFounder,
		&user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}
	return user, nil
}
```

- [ ] **Step 4: Add `UpdatePushToken` implementation**

Add after `UpdateLanguage`:

```go
func (r *userRepository) UpdatePushToken(ctx context.Context, userID uuid.UUID, token string) error {
	var query string
	var err error
	if token == "" {
		query = `UPDATE users SET expo_push_token = NULL, updated_at = NOW() WHERE id = $1`
		_, err = r.db.Exec(ctx, query, userID)
	} else {
		query = `UPDATE users SET expo_push_token = $2, updated_at = NOW() WHERE id = $1`
		_, err = r.db.Exec(ctx, query, userID, token)
	}
	if err != nil {
		return fmt.Errorf("failed to update push token: %w", err)
	}
	return nil
}
```

- [ ] **Step 5: Verify build**

```bash
go build ./...
```

Expected: no output.

- [ ] **Step 6: Commit**

```bash
git add internal/repository/user_repository.go
git commit -m "feat: add UpdatePushToken to UserRepository, scan expo_push_token in queries"
```

---

### Task 4: ReminderRepository

**Files:**
- Create: `internal/repository/reminder_repository.go`

- [ ] **Step 1: Create the file**

```go
package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ReminderUser holds the minimal user data needed for expiry reminders.
type ReminderUser struct {
	ID            uuid.UUID
	Language      string
	ExpoPushToken string
}

type ReminderRepository interface {
	GetUsersEligibleForReminder(ctx context.Context, daysBeforeExpiry int) ([]ReminderUser, error)
	MarkReminderSent(ctx context.Context, userID uuid.UUID, daysBeforeExpiry int) error
}

type reminderRepository struct {
	pool *pgxpool.Pool
}

func NewReminderRepository(pool *pgxpool.Pool) ReminderRepository {
	return &reminderRepository{pool: pool}
}

// GetUsersEligibleForReminder returns premium users whose subscription expires
// in exactly N days (within a 1-day window) and haven't been sent this reminder yet.
func (r *reminderRepository) GetUsersEligibleForReminder(ctx context.Context, daysBeforeExpiry int) ([]ReminderUser, error) {
	var sentAtColumn string
	switch daysBeforeExpiry {
	case 3:
		sentAtColumn = "reminder_3d_sent_at"
	case 1:
		sentAtColumn = "reminder_1d_sent_at"
	default:
		return nil, fmt.Errorf("unsupported daysBeforeExpiry: %d (must be 1 or 3)", daysBeforeExpiry)
	}

	query := fmt.Sprintf(`
		SELECT id, language, COALESCE(expo_push_token, '')
		FROM users
		WHERE subscription_tier = 'premium'
		  AND subscription_expires_at >= NOW() + ($1 || ' days')::INTERVAL
		  AND subscription_expires_at <  NOW() + ($2 || ' days')::INTERVAL
		  AND %s IS NULL
	`, sentAtColumn)

	rows, err := r.pool.Query(ctx, query, daysBeforeExpiry, daysBeforeExpiry+1)
	if err != nil {
		return nil, fmt.Errorf("failed to query eligible users: %w", err)
	}
	defer rows.Close()

	var users []ReminderUser
	for rows.Next() {
		var u ReminderUser
		if err := rows.Scan(&u.ID, &u.Language, &u.ExpoPushToken); err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}
	return users, nil
}

// MarkReminderSent sets reminder_3d_sent_at or reminder_1d_sent_at to NOW().
func (r *reminderRepository) MarkReminderSent(ctx context.Context, userID uuid.UUID, daysBeforeExpiry int) error {
	var query string
	switch daysBeforeExpiry {
	case 3:
		query = `UPDATE users SET reminder_3d_sent_at = NOW(), updated_at = NOW() WHERE id = $1`
	case 1:
		query = `UPDATE users SET reminder_1d_sent_at = NOW(), updated_at = NOW() WHERE id = $1`
	default:
		return fmt.Errorf("unsupported daysBeforeExpiry: %d (must be 1 or 3)", daysBeforeExpiry)
	}
	_, err := r.pool.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to mark reminder sent: %w", err)
	}
	return nil
}
```

- [ ] **Step 2: Verify build**

```bash
go build ./...
```

Expected: no output.

- [ ] **Step 3: Commit**

```bash
git add internal/repository/reminder_repository.go
git commit -m "feat: add ReminderRepository with eligible user query and mark sent"
```

---

### Task 5: ExpiryReminderService

**Files:**
- Create: `internal/service/expiry_reminder_service.go`

- [ ] **Step 1: Create the file**

```go
package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/rifqimalik/cashlens-backend/internal/logger"
	"github.com/rifqimalik/cashlens-backend/internal/repository"
)

type reminderMessages struct {
	telegram  string
	pushTitle string
	pushBody  string
}

var expiryMessages = map[int]map[string]reminderMessages{
	3: {
		"id": {
			telegram:  "Hei! Langganan CashLens Premium kamu akan berakhir dalam 3 hari 😢\n\nJangan sampai ketinggalan fitur AI scan struk & analisis keuangan otomatis.\n\n🎉 Perpanjang sekarang dan nikmati terus semua fitur premium!\n\nPerbarui langganan: https://cashlens.app/upgrade",
			pushTitle: "Langganan Hampir Berakhir 😢",
			pushBody:  "Premium kamu berakhir dalam 3 hari! Perpanjang sekarang.",
		},
		"en": {
			telegram:  "Hey! Your CashLens Premium subscription expires in 3 days 😢\n\nDon't miss out on AI receipt scanning & automatic financial analysis.\n\n🎉 Renew now and keep enjoying all premium features!\n\nRenew here: https://cashlens.app/upgrade",
			pushTitle: "Subscription Expiring Soon 😢",
			pushBody:  "Your Premium expires in 3 days! Renew now.",
		},
	},
	1: {
		"id": {
			telegram:  "⚠️ Langganan CashLens Premium kamu berakhir BESOK!\n\nSegera perpanjang agar tidak kehilangan akses ke AI scan struk & analisis keuangan otomatis.\n\n🎉 Perpanjang sekarang: https://cashlens.app/upgrade",
			pushTitle: "Langganan Berakhir Besok! ⚠️",
			pushBody:  "Premium kamu berakhir besok! Jangan sampai kehabisan.",
		},
		"en": {
			telegram:  "⚠️ Your CashLens Premium subscription expires TOMORROW!\n\nRenew now to keep access to AI receipt scanning & automatic financial analysis.\n\n🎉 Renew now: https://cashlens.app/upgrade",
			pushTitle: "Subscription Expires Tomorrow! ⚠️",
			pushBody:  "Your Premium expires tomorrow! Don't lose access.",
		},
	},
}

type ExpiryReminderService interface {
	RunReminders(ctx context.Context, daysBeforeExpiry int) (int, error)
}

type expiryReminderService struct {
	reminderRepo  repository.ReminderRepository
	chatRepo      repository.ChatLinkRepository
	telegramToken string
	httpClient    *http.Client
	log           *logger.Logger
}

func NewExpiryReminderService(
	reminderRepo repository.ReminderRepository,
	chatRepo repository.ChatLinkRepository,
	telegramToken string,
) ExpiryReminderService {
	return &expiryReminderService{
		reminderRepo:  reminderRepo,
		chatRepo:      chatRepo,
		telegramToken: telegramToken,
		httpClient:    &http.Client{Timeout: 10 * time.Second},
		log:           logger.GetDefault().With("component", "expiry_reminder_service"),
	}
}

// RunReminders sends expiry reminders to eligible users daysBeforeExpiry days before their subscription ends.
// Returns the number of users successfully notified on at least one channel.
func (s *expiryReminderService) RunReminders(ctx context.Context, daysBeforeExpiry int) (int, error) {
	users, err := s.reminderRepo.GetUsersEligibleForReminder(ctx, daysBeforeExpiry)
	if err != nil {
		return 0, fmt.Errorf("failed to get eligible users: %w", err)
	}

	if len(users) == 0 {
		s.log.Info("No users eligible for expiry reminder", "days_before", daysBeforeExpiry)
		return 0, nil
	}

	sent := 0
	for _, u := range users {
		msgs, ok := expiryMessages[daysBeforeExpiry][u.Language]
		if !ok {
			msgs = expiryMessages[daysBeforeExpiry]["en"]
		}

		atLeastOne := false

		// Try Telegram
		chatLink, err := s.chatRepo.GetByUserID(ctx, u.ID, "telegram")
		if err == nil {
			if err := s.sendTelegramMessage(chatLink.ChatID, msgs.telegram); err != nil {
				s.log.Error("Failed to send reminder via Telegram", "user_id", u.ID, "error", err)
			} else {
				s.log.Info("Reminder sent via Telegram", "user_id", u.ID, "days_before", daysBeforeExpiry)
				atLeastOne = true
			}
		}

		// Try Expo push
		if u.ExpoPushToken != "" {
			if err := s.sendExpoPush(u.ExpoPushToken, msgs.pushTitle, msgs.pushBody); err != nil {
				s.log.Error("Failed to send reminder via Expo push", "user_id", u.ID, "error", err)
			} else {
				s.log.Info("Reminder sent via Expo push", "user_id", u.ID, "days_before", daysBeforeExpiry)
				atLeastOne = true
			}
		}

		if !atLeastOne {
			s.log.Warn("No channel succeeded for user, will retry next run", "user_id", u.ID)
			continue
		}

		if err := s.reminderRepo.MarkReminderSent(ctx, u.ID, daysBeforeExpiry); err != nil {
			s.log.Error("Failed to mark reminder sent", "user_id", u.ID, "error", err)
			continue
		}

		sent++
	}

	s.log.Info("Expiry reminder run completed", "days_before", daysBeforeExpiry, "users_sent", sent, "total_eligible", len(users))
	return sent, nil
}

func (s *expiryReminderService) sendTelegramMessage(chatID string, text string) error {
	payload := map[string]any{
		"chat_id":    chatID,
		"text":       text,
		"parse_mode": "Markdown",
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", s.telegramToken)
	resp, err := s.httpClient.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to call Telegram API: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Telegram API returned status %d", resp.StatusCode)
	}
	return nil
}

func (s *expiryReminderService) sendExpoPush(token, title, body string) error {
	payload := map[string]any{
		"to":    token,
		"title": title,
		"body":  body,
	}
	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal push payload: %w", err)
	}
	resp, err := s.httpClient.Post("https://exp.host/push/send", "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to call Expo push API: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Expo push API returned status %d", resp.StatusCode)
	}
	return nil
}
```

- [ ] **Step 2: Verify build**

```bash
go build ./...
```

Expected: no output.

- [ ] **Step 3: Commit**

```bash
git add internal/service/expiry_reminder_service.go
git commit -m "feat: add ExpiryReminderService with Telegram and Expo push support"
```

---

### Task 6: Auth service and handler — UpdatePushToken

**Files:**
- Modify: `internal/service/auth_service.go`
- Modify: `internal/handlers/auth.go`

- [ ] **Step 1: Add `UpdatePushToken` to AuthService interface**

In `internal/service/auth_service.go`, add to the interface:

```go
type AuthService interface {
	Register(ctx context.Context, req models.CreateUserRequest) (*models.AuthResponse, error)
	Login(ctx context.Context, req models.LoginRequest) (*models.AuthResponse, error)
	ValidateToken(tokenString string) (*uuid.UUID, error)
	GetMe(ctx context.Context, userID uuid.UUID) (*models.User, error)
	UpdateLanguage(ctx context.Context, userID uuid.UUID, language string) error
	UpdatePushToken(ctx context.Context, userID uuid.UUID, token string) error
	GetTelegramStatus(ctx context.Context, userID uuid.UUID) (map[string]any, error)
	UnlinkTelegram(ctx context.Context, userID uuid.UUID) error
}
```

- [ ] **Step 2: Add implementation after `UpdateLanguage`**

```go
func (s *authService) UpdatePushToken(ctx context.Context, userID uuid.UUID, token string) error {
	return s.userRepo.UpdatePushToken(ctx, userID, token)
}
```

- [ ] **Step 3: Add handler to `internal/handlers/auth.go`**

Add after `UpdateLanguage` handler:

```go
func (h *AuthHandler) UpdatePushToken(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(*uuid.UUID)
	if !ok {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Unauthorized"})
		return
	}

	var req models.UpdatePushTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Invalid request body"})
		return
	}

	if err := h.authService.UpdatePushToken(r.Context(), *userID, req.PushToken); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Failed to update push token"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"message": "Push token updated successfully",
	})
}
```

- [ ] **Step 4: Verify build**

```bash
go build ./...
```

Expected: no output.

- [ ] **Step 5: Commit**

```bash
git add internal/service/auth_service.go internal/handlers/auth.go
git commit -m "feat: add UpdatePushToken to AuthService and AuthHandler"
```

---

### Task 7: Wire everything in main.go

**Files:**
- Modify: `cmd/server/main.go`

- [ ] **Step 1: Add `reminderRepo` after `winBackRepo`**

In the repositories block (around line 75), add:

```go
reminderRepo := repository.NewReminderRepository(db.Pool)
```

- [ ] **Step 2: Add `expiryReminderService` after `winBackService`**

```go
expiryReminderService := service.NewExpiryReminderService(reminderRepo, chatRepo, cfg.Telegram.BotToken)
```

- [ ] **Step 3: Add the two scheduler goroutines after the win-back scheduler block**

```go
// Start 3-day expiry reminder scheduler (runs daily)
go func() {
    ticker := time.NewTicker(24 * time.Hour)
    defer ticker.Stop()
    time.Sleep(90 * time.Minute)
    for range ticker.C {
        count, err := expiryReminderService.RunReminders(context.Background(), 3)
        if err != nil {
            log.Error("3-day expiry reminder failed", "error", err)
        } else if count > 0 {
            log.Info("3-day expiry reminders sent", "users_sent", count)
        }
    }
}()
log.Info("3-day expiry reminder scheduler started")

// Start 1-day expiry reminder scheduler (runs daily)
go func() {
    ticker := time.NewTicker(24 * time.Hour)
    defer ticker.Stop()
    time.Sleep(2 * time.Hour)
    for range ticker.C {
        count, err := expiryReminderService.RunReminders(context.Background(), 1)
        if err != nil {
            log.Error("1-day expiry reminder failed", "error", err)
        } else if count > 0 {
            log.Info("1-day expiry reminders sent", "users_sent", count)
        }
    }
}()
log.Info("1-day expiry reminder scheduler started")
```

- [ ] **Step 4: Register `PATCH /auth/push-token` route**

In the authenticated routes block, after `PATCH /auth/language`:

```go
r.Patch("/auth/push-token", authHandler.UpdatePushToken)
```

- [ ] **Step 5: Verify build**

```bash
go build ./...
```

Expected: no output.

- [ ] **Step 6: Commit**

```bash
git add cmd/server/main.go
git commit -m "feat: wire ExpiryReminderService and push token route in main"
```

---

### Task 8: Push to production

- [ ] **Step 1: Push development branch**

```bash
git push origin development
```

- [ ] **Step 2: Merge to main and push**

```bash
git checkout main
git merge development --no-edit
git push origin main
git checkout development
```

Railway will auto-run migration `00018` on deploy, adding the three columns.

---

## Self-Review

**Spec coverage:**
- ✅ Migration with 3 columns
- ✅ `ReminderRepository.GetUsersEligibleForReminder` + `MarkReminderSent`
- ✅ `ExpiryReminderService.RunReminders` with Telegram + Expo push
- ✅ Independent channels — each tried regardless of the other
- ✅ Mark sent only if at least one channel succeeded
- ✅ Both 3-day and 1-day messages in `id` and `en`
- ✅ `UpdatePushToken` on UserRepository, AuthService, AuthHandler
- ✅ `PATCH /auth/push-token` route registered
- ✅ Two daily scheduler goroutines in main.go
- ✅ Empty token clears to NULL in DB

**Type consistency check:**
- `ReminderUser.ExpoPushToken` (string) — used correctly in service
- `MarkReminderSent(ctx, userID, daysBeforeExpiry int)` — matches repository interface and service call sites
- `expiryMessages[daysBeforeExpiry][u.Language]` — keys `3`/`1` and `"id"`/`"en"` match map definition

**No placeholders found.**
