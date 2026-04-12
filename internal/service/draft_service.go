package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rifqimalik/cashlens-backend/internal/errors"
	"github.com/rifqimalik/cashlens-backend/internal/models"
	"github.com/rifqimalik/cashlens-backend/internal/repository"
)

type DraftService interface {
	Create(ctx context.Context, userID uuid.UUID, req models.CreateDraftRequest) (*models.DraftTransaction, error)
	Get(ctx context.Context, id, userID uuid.UUID) (*models.DraftTransaction, error)
	List(ctx context.Context, userID uuid.UUID, status models.DraftStatus) ([]*models.DraftTransaction, error)
	Confirm(ctx context.Context, draftID, userID uuid.UUID, req models.ConfirmDraftRequest) (*models.Transaction, error)
	Delete(ctx context.Context, id, userID uuid.UUID) error
}

type draftService struct {
	draftRepo     repository.DraftRepository
	transactionRepo repository.TransactionRepository
}

func NewDraftService(draftRepo repository.DraftRepository, transactionRepo repository.TransactionRepository) DraftService {
	return &draftService{
		draftRepo:     draftRepo,
		transactionRepo: transactionRepo,
	}
}

func (s *draftService) Create(ctx context.Context, userID uuid.UUID, req models.CreateDraftRequest) (*models.DraftTransaction, error) {
	if req.Source == "" {
		return nil, errors.NewBadRequest("Source is required")
	}

	draft := &models.DraftTransaction{
		ID:              uuid.New(),
		UserID:          userID,
		CategoryID:      req.CategoryID,
		Amount:          req.Amount,
		Description:     req.Description,
		TransactionDate: req.TransactionDate,
		Source:          req.Source,
		RawData:         req.RawData,
		Status:          models.DraftStatusPending,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	err := s.draftRepo.Create(ctx, draft)
	if err != nil {
		return nil, fmt.Errorf("failed to create draft: %w", err)
	}

	return draft, nil
}

func (s *draftService) Get(ctx context.Context, id, userID uuid.UUID) (*models.DraftTransaction, error) {
	draft, err := s.draftRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get draft: %w", err)
	}

	if draft.UserID != userID {
		return nil, errors.NewForbidden("Forbidden access")
	}

	return draft, nil
}

func (s *draftService) List(ctx context.Context, userID uuid.UUID, status models.DraftStatus) ([]*models.DraftTransaction, error) {
	if status == "" {
		status = models.DraftStatusPending
	}

	drafts, err := s.draftRepo.ListByUserID(ctx, userID, status)
	if err != nil {
		return nil, fmt.Errorf("failed to list drafts: %w", err)
	}

	return drafts, nil
}

func (s *draftService) Confirm(ctx context.Context, draftID, userID uuid.UUID, req models.ConfirmDraftRequest) (*models.Transaction, error) {
	draft, err := s.draftRepo.GetByID(ctx, draftID)
	if err != nil {
		return nil, fmt.Errorf("failed to get draft: %w", err)
	}

	if draft.UserID != userID {
		return nil, errors.NewForbidden("Forbidden access")
	}

	if draft.Status != models.DraftStatusPending {
		return nil, errors.NewBadRequest("Draft is not in pending status")
	}

	// Create actual transaction
	transaction := &models.Transaction{
		ID:              uuid.New(),
		UserID:          userID,
		CategoryID:      req.CategoryID,
		Amount:          req.Amount,
		Description:     &req.Description,
		TransactionDate: req.TransactionDate,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	err = s.transactionRepo.Create(ctx, transaction)
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	// Mark draft as confirmed
	err = s.draftRepo.Confirm(ctx, draftID, transaction.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to confirm draft: %w", err)
	}

	return transaction, nil
}

func (s *draftService) Delete(ctx context.Context, id, userID uuid.UUID) error {
	draft, err := s.draftRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get draft: %w", err)
	}

	if draft.UserID != userID {
		return errors.NewForbidden("Forbidden access")
	}

	err = s.draftRepo.Delete(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete draft: %w", err)
	}

	return nil
}
