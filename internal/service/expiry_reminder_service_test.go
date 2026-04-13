package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/rifqimalik/cashlens-backend/internal/models"
	"github.com/rifqimalik/cashlens-backend/internal/repository"
)

type MockReminderRepository struct {
	GetUsersFunc     func(ctx context.Context, days int) ([]repository.ReminderUser, error)
	MarkReminderFunc func(ctx context.Context, userID uuid.UUID, days int) error
}

func (m *MockReminderRepository) GetUsersEligibleForReminder(ctx context.Context, days int) ([]repository.ReminderUser, error) {
	return m.GetUsersFunc(ctx, days)
}

func (m *MockReminderRepository) MarkReminderSent(ctx context.Context, userID uuid.UUID, days int) error {
	return m.MarkReminderFunc(ctx, userID, days)
}

type MockChatLinkRepository struct {
	GetByUserIDFunc func(ctx context.Context, userID uuid.UUID, platform string) (*models.UserChatLink, error)
}

func (m *MockChatLinkRepository) GetByUserID(ctx context.Context, userID uuid.UUID, platform string) (*models.UserChatLink, error) {
	return m.GetByUserIDFunc(ctx, userID, platform)
}

func (m *MockChatLinkRepository) Create(ctx context.Context, link *models.UserChatLink) error { return nil }
func (m *MockChatLinkRepository) GetByChatID(ctx context.Context, chatID string, platform string) (*models.UserChatLink, error) {
	return nil, nil
}
func (m *MockChatLinkRepository) Delete(ctx context.Context, id uuid.UUID) error { return nil }

func TestExpiryReminderService_RunReminders(t *testing.T) {
	// Setup mock servers for Telegram and Expo
	tgServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer tgServer.Close()

	expoServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer expoServer.Close()

	userID := uuid.New()
	markSentCalled := false
	
	mockReminderRepo := &MockReminderRepository{
		GetUsersFunc: func(ctx context.Context, days int) ([]repository.ReminderUser, error) {
			return []repository.ReminderUser{
				{
					ID:            userID,
					Language:      "id",
					ExpoPushToken: "ExponentPushToken[123]",
				},
			}, nil
		},
		MarkReminderFunc: func(ctx context.Context, uID uuid.UUID, days int) error {
			markSentCalled = true
			if uID != userID {
				t.Errorf("expected userID %s, got %s", userID, uID)
			}
			return nil
		},
	}

	mockChatRepo := &MockChatLinkRepository{
		GetByUserIDFunc: func(ctx context.Context, uID uuid.UUID, platform string) (*models.UserChatLink, error) {
			return &models.UserChatLink{
				ChatID: "123456",
			}, nil
		},
	}

	svc := NewExpiryReminderService(mockReminderRepo, mockChatRepo, "fake-token")
	s := svc.(*expiryReminderService)
	s.telegramURL = tgServer.URL + "/%s"
	s.expoURL = expoServer.URL

	count, err := svc.RunReminders(context.Background(), 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if count != 1 {
		t.Errorf("expected count 1, got %d", count)
	}

	if !markSentCalled {
		t.Errorf("expected MarkReminderSent to be called")
	}
}

func TestExpiryReminderService_RunReminders_Failure(t *testing.T) {
	// Setup mock servers that fail
	tgServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer tgServer.Close()

	expoServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer expoServer.Close()

	userID := uuid.New()
	markSentCalled := false
	
	mockReminderRepo := &MockReminderRepository{
		GetUsersFunc: func(ctx context.Context, days int) ([]repository.ReminderUser, error) {
			return []repository.ReminderUser{
				{
					ID:            userID,
					Language:      "id",
					ExpoPushToken: "ExponentPushToken[123]",
				},
			}, nil
		},
		MarkReminderFunc: func(ctx context.Context, uID uuid.UUID, days int) error {
			markSentCalled = true
			return nil
		},
	}

	mockChatRepo := &MockChatLinkRepository{
		GetByUserIDFunc: func(ctx context.Context, uID uuid.UUID, platform string) (*models.UserChatLink, error) {
			return &models.UserChatLink{
				ChatID: "123456",
			}, nil
		},
	}

	svc := NewExpiryReminderService(mockReminderRepo, mockChatRepo, "fake-token")
	s := svc.(*expiryReminderService)
	s.telegramURL = tgServer.URL + "/%s"
	s.expoURL = expoServer.URL

	count, err := svc.RunReminders(context.Background(), 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if count != 0 {
		t.Errorf("expected count 0, got %d", count)
	}

	if markSentCalled {
		t.Errorf("expected MarkReminderSent NOT to be called")
	}
}
