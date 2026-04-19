package service_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rifqimalik/cashlens-backend/internal/models"
	"github.com/rifqimalik/cashlens-backend/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockUserRepoForExpiry struct{ mock.Mock }

func (m *MockUserRepoForExpiry) GetExpiredTrialUsers(ctx context.Context) ([]*models.User, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*models.User), args.Error(1)
}
func (m *MockUserRepoForExpiry) Update(ctx context.Context, user *models.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

type MockMailerForExpiry struct{ mock.Mock }

func (m *MockMailerForExpiry) SendConfirmationEmail(to, token string) error {
	args := m.Called(to, token)
	return args.Error(0)
}
func (m *MockMailerForExpiry) SendTrialExpiredEmail(to string) error {
	args := m.Called(to)
	return args.Error(0)
}

func TestTrialExpiryService_ExpireTrials_UpdatesStatusAndSendsEmail(t *testing.T) {
	userRepo := new(MockUserRepoForExpiry)
	mailerMock := new(MockMailerForExpiry)

	trialEnd := time.Now().Add(-1 * time.Hour)
	user := &models.User{
		ID:          uuid.New(),
		Email:       "user@example.com",
		TrialStatus: "active",
		TrialEndAt:  &trialEnd,
	}

	userRepo.On("GetExpiredTrialUsers", mock.Anything).Return([]*models.User{user}, nil)
	userRepo.On("Update", mock.Anything, mock.MatchedBy(func(u *models.User) bool {
		return u.TrialStatus == "expired"
	})).Return(nil)
	mailerMock.On("SendTrialExpiredEmail", "user@example.com").Return(nil)

	svc := service.NewTrialExpiryService(userRepo, mailerMock)
	count, err := svc.ExpireTrials(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, 1, count)
	assert.Equal(t, "expired", user.TrialStatus)
	userRepo.AssertExpectations(t)
	mailerMock.AssertExpectations(t)
}

func TestTrialExpiryService_ExpireTrials_NoExpiredUsers(t *testing.T) {
	userRepo := new(MockUserRepoForExpiry)
	mailerMock := new(MockMailerForExpiry)

	userRepo.On("GetExpiredTrialUsers", mock.Anything).Return([]*models.User{}, nil)

	svc := service.NewTrialExpiryService(userRepo, mailerMock)
	count, err := svc.ExpireTrials(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, 0, count)
	mailerMock.AssertNotCalled(t, "SendTrialExpiredEmail")
}

func TestTrialExpiryService_ExpireTrials_PerUserErrorDoesNotAbort(t *testing.T) {
	userRepo := new(MockUserRepoForExpiry)
	mailerMock := new(MockMailerForExpiry)

	trialEnd := time.Now().Add(-1 * time.Hour)
	user1 := &models.User{ID: uuid.New(), Email: "a@example.com", TrialStatus: "active", TrialEndAt: &trialEnd}
	user2 := &models.User{ID: uuid.New(), Email: "b@example.com", TrialStatus: "active", TrialEndAt: &trialEnd}

	userRepo.On("GetExpiredTrialUsers", mock.Anything).Return([]*models.User{user1, user2}, nil)
	userRepo.On("Update", mock.Anything, mock.MatchedBy(func(u *models.User) bool { return u.Email == "a@example.com" })).Return(fmt.Errorf("db error"))
	userRepo.On("Update", mock.Anything, mock.MatchedBy(func(u *models.User) bool { return u.Email == "b@example.com" })).Return(nil)
	mailerMock.On("SendTrialExpiredEmail", "b@example.com").Return(nil)

	svc := service.NewTrialExpiryService(userRepo, mailerMock)
	count, err := svc.ExpireTrials(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, 1, count)
	mailerMock.AssertCalled(t, "SendTrialExpiredEmail", "b@example.com")
	mailerMock.AssertNotCalled(t, "SendTrialExpiredEmail", "a@example.com")
}
