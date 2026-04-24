package telegram

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rifqimalik/cashlens-backend/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestFilterFixedCategories(t *testing.T) {
	cats := []*models.Category{
		{ID: uuid.New(), Name: "Makanan & Minuman"},
		{ID: uuid.New(), Name: "Transportasi"},
		{ID: uuid.New(), Name: "Custom Category"},
		{ID: uuid.New(), Name: "Lainnya"},
		{ID: uuid.New(), Name: "Another Custom"},
	}

	result := filterFixedCategories(cats)

	assert.Len(t, result, 3)
	names := make([]string, len(result))
	for i, c := range result {
		names[i] = c.Name
	}
	assert.Contains(t, names, "Makanan & Minuman")
	assert.Contains(t, names, "Transportasi")
	assert.Contains(t, names, "Lainnya")
	assert.NotContains(t, names, "Custom Category")
}

// --- Mocks ---

type mockUserRepo struct{ mock.Mock }

func (m *mockUserRepo) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

// Stub out the rest of UserRepository to satisfy interface.
func (m *mockUserRepo) Create(ctx context.Context, user *models.User) error {
	return m.Called(ctx, user).Error(0)
}
func (m *mockUserRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}
func (m *mockUserRepo) GetByGoogleID(ctx context.Context, googleID string) (*models.User, error) {
	args := m.Called(ctx, googleID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}
func (m *mockUserRepo) UpdateGoogleID(ctx context.Context, userID uuid.UUID, googleID string) error {
	return m.Called(ctx, userID, googleID).Error(0)
}
func (m *mockUserRepo) GetByDeviceID(ctx context.Context, deviceID string) ([]models.User, error) {
	args := m.Called(ctx, deviceID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.User), args.Error(1)
}
func (m *mockUserRepo) Update(ctx context.Context, user *models.User) error {
	return m.Called(ctx, user).Error(0)
}
func (m *mockUserRepo) UpdateSubscription(ctx context.Context, userID uuid.UUID, tier string, expiresAt *time.Time) error {
	return m.Called(ctx, userID, tier, expiresAt).Error(0)
}
func (m *mockUserRepo) UpdateFounder(ctx context.Context, userID uuid.UUID, isFounder bool) error {
	return m.Called(ctx, userID, isFounder).Error(0)
}
func (m *mockUserRepo) UpdateLanguage(ctx context.Context, userID uuid.UUID, language string) error {
	return m.Called(ctx, userID, language).Error(0)
}
func (m *mockUserRepo) UpdatePushToken(ctx context.Context, userID uuid.UUID, token string) error {
	return m.Called(ctx, userID, token).Error(0)
}
func (m *mockUserRepo) GetByConfirmationToken(ctx context.Context, token string) (*models.User, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}
func (m *mockUserRepo) UpdateConfirmationStatus(ctx context.Context, userID uuid.UUID, isConfirmed bool) error {
	return m.Called(ctx, userID, isConfirmed).Error(0)
}
func (m *mockUserRepo) UpdateConfirmationToken(ctx context.Context, userID uuid.UUID, token string, expiresAt time.Time) error {
	return m.Called(ctx, userID, token, expiresAt).Error(0)
}
func (m *mockUserRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}
func (m *mockUserRepo) GetExpiredTrialUsers(ctx context.Context) ([]*models.User, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.User), args.Error(1)
}

type mockChatRepo struct{ mock.Mock }

func (m *mockChatRepo) Create(ctx context.Context, link *models.UserChatLink) error {
	return m.Called(ctx, link).Error(0)
}
func (m *mockChatRepo) Upsert(ctx context.Context, link *models.UserChatLink) error {
	return m.Called(ctx, link).Error(0)
}
func (m *mockChatRepo) GetByChatID(ctx context.Context, chatID string, platform string) (*models.UserChatLink, error) {
	args := m.Called(ctx, chatID, platform)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.UserChatLink), args.Error(1)
}
func (m *mockChatRepo) GetByUserID(ctx context.Context, userID uuid.UUID, platform string) (*models.UserChatLink, error) {
	args := m.Called(ctx, userID, platform)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.UserChatLink), args.Error(1)
}
func (m *mockChatRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}

// sentMessages captures what the bot sends back to a chat.
type sentMessages struct {
	messages []string
}

func newBotForTest(userRepo *mockUserRepo, chatRepo *mockChatRepo) (*BotService, *sentMessages) {
	sent := &sentMessages{}
	bot := &BotService{
		userRepo:   userRepo,
		chatRepo:   chatRepo,
		httpClient: nil, // not needed; sendReply is overridden via spy below
	}
	// Override sendReply to capture output without calling Telegram API.
	bot.replySpy = func(chatID int64, text string) {
		sent.messages = append(sent.messages, text)
	}
	return bot, sent
}

// --- Tests ---

// TestHandleLink_EmailNotFound verifies the bot replies with an error when
// the email doesn't exist in the database.
func TestHandleLink_EmailNotFound(t *testing.T) {
	userRepo := &mockUserRepo{}
	chatRepo := &mockChatRepo{}

	userRepo.On("GetByEmail", mock.Anything, "unknown@example.com").
		Return(nil, fmt.Errorf("not found"))

	bot, sent := newBotForTest(userRepo, chatRepo)
	bot.handleLink(100, "unknown@example.com", nil)

	assert.Len(t, sent.messages, 1)
	assert.Contains(t, sent.messages[0], "not found")
	chatRepo.AssertNotCalled(t, "Upsert")
}

// TestHandleLink_Success_NewLink verifies the bot links a new account and
// replies with a welcome message.
func TestHandleLink_Success_NewLink(t *testing.T) {
	userRepo := &mockUserRepo{}
	chatRepo := &mockChatRepo{}

	name := "Malik"
	user := &models.User{ID: uuid.New(), Name: &name}
	userRepo.On("GetByEmail", mock.Anything, "malik@example.com").Return(user, nil)
	chatRepo.On("Upsert", mock.Anything, mock.AnythingOfType("*models.UserChatLink")).Return(nil)

	bot, sent := newBotForTest(userRepo, chatRepo)
	bot.handleLink(100, "malik@example.com", nil)

	assert.Len(t, sent.messages, 1)
	assert.Contains(t, sent.messages[0], "Account Linked")
	assert.Contains(t, sent.messages[0], "Malik")
	chatRepo.AssertCalled(t, "Upsert", mock.Anything, mock.AnythingOfType("*models.UserChatLink"))
}

// TestHandleLink_AlreadyLinked verifies re-linking the same chat to the same
// account succeeds (upsert, not insert) and gives a welcome message.
func TestHandleLink_AlreadyLinked(t *testing.T) {
	userRepo := &mockUserRepo{}
	chatRepo := &mockChatRepo{}

	name := "Malik"
	user := &models.User{ID: uuid.New(), Name: &name}
	userRepo.On("GetByEmail", mock.Anything, "malik@example.com").Return(user, nil)
	// Upsert always succeeds even when row already exists.
	chatRepo.On("Upsert", mock.Anything, mock.AnythingOfType("*models.UserChatLink")).Return(nil)

	bot, sent := newBotForTest(userRepo, chatRepo)
	bot.handleLink(100, "malik@example.com", nil)
	bot.handleLink(100, "malik@example.com", nil) // second time

	assert.Equal(t, 2, len(sent.messages))
	assert.Contains(t, sent.messages[1], "Account Linked")
}
