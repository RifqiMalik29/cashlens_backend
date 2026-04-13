# Expiry Reminder Design

**Date:** 2026-04-13  
**Status:** Approved

## Overview

Send subscription expiry reminders to premium users via Telegram and Expo Push Notifications — 3 days before and 1 day before their subscription expires. Both channels are independent: Telegram only if the user has linked their account, push only if they have a stored token.

## Database

Migration `00018_add_reminder_fields_to_users.sql` adds three columns to `users`:

```sql
ALTER TABLE users ADD COLUMN IF NOT EXISTS reminder_3d_sent_at TIMESTAMPTZ;
ALTER TABLE users ADD COLUMN IF NOT EXISTS reminder_1d_sent_at TIMESTAMPTZ;
ALTER TABLE users ADD COLUMN IF NOT EXISTS expo_push_token VARCHAR(255);
```

- `reminder_3d_sent_at` — set after 3-day reminder is sent, prevents duplicates
- `reminder_1d_sent_at` — set after 1-day reminder is sent, prevents duplicates
- `expo_push_token` — stored by mobile on app start or token refresh

## Components

### ReminderRepository (`internal/repository/reminder_repository.go`)

```go
type ReminderUser struct {
    ID            uuid.UUID
    Language      string
    ExpoPushToken string
}

type ReminderRepository interface {
    GetUsersEligibleForReminder(ctx context.Context, daysBeforeExpiry int) ([]ReminderUser, error)
    MarkReminderSent(ctx context.Context, userID uuid.UUID, daysBeforeExpiry int) error
}
```

Query for `GetUsersEligibleForReminder(ctx, N)`:
```sql
SELECT id, language, COALESCE(expo_push_token, '') FROM users
WHERE subscription_tier = 'premium'
  AND subscription_expires_at >= NOW() + (N days)
  AND subscription_expires_at < NOW() + (N+1 days)
  AND reminder_Nd_sent_at IS NULL
```

`MarkReminderSent` sets `reminder_3d_sent_at` or `reminder_1d_sent_at` based on `daysBeforeExpiry`.

### ExpiryReminderService (`internal/service/expiry_reminder_service.go`)

```go
type ExpiryReminderService interface {
    RunReminders(ctx context.Context, daysBeforeExpiry int) (int, error)
}
```

For each eligible user:
1. Try Telegram — call `chatRepo.GetByUserID(ctx, userID, "telegram")`, send if found
2. Try Expo push — call Expo API if `ExpoPushToken` non-empty
3. Mark reminder sent only if at least one channel succeeded

Expo push API: `POST https://exp.host/push/send` with JSON body:
```json
{
  "to": "ExponentPushToken[...]",
  "title": "...",
  "body": "..."
}
```

### UserRepository additions

- `UpdatePushToken(ctx, userID, token string) error` — stores or clears the Expo push token
- `GetByID` and `GetByEmail` must scan the new `expo_push_token` column (COALESCE to empty string)

### Auth additions

- `AuthService.UpdatePushToken(ctx, userID, token string) error`
- `AuthHandler.UpdatePushToken` — `PATCH /auth/push-token`
  - Body: `{"push_token": "ExponentPushToken[...]"}`
  - Requires auth
  - Clears token if empty string is sent (user logout)

### Model additions (`internal/models/user.go`)

```go
// On User struct
ExpoPushToken string `json:"-"` // never expose in API responses

// New request type
type UpdatePushTokenRequest struct {
    PushToken string `json:"push_token" validate:"required"`
}
```

## Messages

### 3-day reminder

| Language | Telegram | Push title | Push body |
|---|---|---|---|
| `id` | "Hei! Langganan CashLens Premium kamu akan berakhir dalam 3 hari 😢\n\nJangan sampai ketinggalan fitur AI scan struk & analisis keuangan otomatis.\n\n🎉 Perpanjang sekarang dan nikmati terus semua fitur premium!\n\nPerbarui langganan: https://cashlens.app/upgrade" | "Langganan Hampir Berakhir 😢" | "Premium kamu berakhir dalam 3 hari! Perpanjang sekarang." |
| `en` | "Hey! Your CashLens Premium subscription expires in 3 days 😢\n\nDon't miss out on AI receipt scanning & automatic financial analysis.\n\n🎉 Renew now and keep enjoying all premium features!\n\nRenew here: https://cashlens.app/upgrade" | "Subscription Expiring Soon 😢" | "Your Premium expires in 3 days! Renew now." |

### 1-day reminder

| Language | Telegram | Push title | Push body |
|---|---|---|---|
| `id` | "⚠️ Langganan CashLens Premium kamu berakhir BESOK!\n\nSegera perpanjang agar tidak kehilangan akses ke AI scan struk & analisis keuangan otomatis.\n\n🎉 Perpanjang sekarang: https://cashlens.app/upgrade" | "Langganan Berakhir Besok! ⚠️" | "Premium kamu berakhir besok! Jangan sampai kehabisan." |
| `en` | "⚠️ Your CashLens Premium subscription expires TOMORROW!\n\nRenew now to keep access to AI receipt scanning & automatic financial analysis.\n\n🎉 Renew now: https://cashlens.app/upgrade" | "Subscription Expires Tomorrow! ⚠️" | "Your Premium expires tomorrow! Don't lose access." |

## Scheduling (`cmd/server/main.go`)

Two goroutines following the same pattern as the win-back scheduler:

```go
// 3-day reminder — runs daily
go func() {
    ticker := time.NewTicker(24 * time.Hour)
    defer ticker.Stop()
    time.Sleep(1 * time.Hour)
    for range ticker.C {
        expiryReminderService.RunReminders(ctx, 3)
    }
}()

// 1-day reminder — runs daily
go func() {
    ticker := time.NewTicker(24 * time.Hour)
    defer ticker.Stop()
    time.Sleep(2 * time.Hour)
    for range ticker.C {
        expiryReminderService.RunReminders(ctx, 1)
    }
}()
```

## Error Handling

- Telegram failure for a user → log, continue to push attempt
- Push failure for a user → log, still mark sent if Telegram succeeded
- If both fail → do NOT mark sent (will retry next day)
- Expo API non-200 → log error with status code

## Mobile Integration

- On app start / token refresh: `PATCH /auth/push-token` with `{"push_token": "ExponentPushToken[...]"}`
- On logout: `PATCH /auth/push-token` with `{"push_token": ""}` to clear the token. The logout endpoint itself does not clear it.
