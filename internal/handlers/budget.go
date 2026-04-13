package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	apperrors "github.com/rifqimalik/cashlens-backend/internal/errors"
	"github.com/rifqimalik/cashlens-backend/internal/middleware"
	"github.com/rifqimalik/cashlens-backend/internal/models"
	"github.com/rifqimalik/cashlens-backend/internal/pkg/validator"
	"github.com/rifqimalik/cashlens-backend/internal/service"
)

type BudgetHandler struct {
	budgetService service.BudgetService
}

func NewBudgetHandler(budgetService service.BudgetService) *BudgetHandler {
	return &BudgetHandler{budgetService: budgetService}
}

func (h *BudgetHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req models.CreateBudgetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apperrors.WriteJSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if validationErrors := validator.ValidateStruct(&req); validationErrors != nil {
		apperrors.WriteJSONError(w, "Validation failed", http.StatusBadRequest, validationErrors)
		return
	}

	userID, ok := r.Context().Value(middleware.UserIDKey).(*uuid.UUID)
	if !ok {
		apperrors.WriteJSONError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	res, err := h.budgetService.Create(r.Context(), *userID, req)
	if err != nil {
		var appErr *apperrors.AppError
		if errors.As(err, &appErr) {
			apperrors.WriteAppError(w, appErr)
			return
		}
		apperrors.WriteJSONError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{
		"data": res,
	})
}

func (h *BudgetHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(*uuid.UUID)
	if !ok {
		apperrors.WriteJSONError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	res, err := h.budgetService.List(r.Context(), *userID)
	if err != nil {
		apperrors.WriteJSONError(w, "Failed to list budgets", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"data": res,
	})
}

func (h *BudgetHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(*uuid.UUID)
	if !ok {
		apperrors.WriteJSONError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	idStr := chi.URLParam(r, "id")
	budgetID, err := uuid.Parse(idStr)
	if err != nil {
		apperrors.WriteJSONError(w, "Invalid budget ID", http.StatusBadRequest)
		return
	}

	res, err := h.budgetService.Get(r.Context(), budgetID, *userID)
	if err != nil {
		var appErr *apperrors.AppError
		if errors.As(err, &appErr) {
			apperrors.WriteAppError(w, appErr)
			return
		}
		apperrors.WriteJSONError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"data": res,
	})
}

func (h *BudgetHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(*uuid.UUID)
	if !ok {
		apperrors.WriteJSONError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	idStr := chi.URLParam(r, "id")
	budgetID, err := uuid.Parse(idStr)
	if err != nil {
		apperrors.WriteJSONError(w, "Invalid budget ID", http.StatusBadRequest)
		return
	}

	var req models.UpdateBudgetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apperrors.WriteJSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if validationErrors := validator.ValidateStruct(&req); validationErrors != nil {
		apperrors.WriteJSONError(w, "Validation failed", http.StatusBadRequest, validationErrors)
		return
	}

	err = h.budgetService.Update(r.Context(), budgetID, *userID, req)
	if err != nil {
		var appErr *apperrors.AppError
		if errors.As(err, &appErr) {
			apperrors.WriteAppError(w, appErr)
			return
		}
		apperrors.WriteJSONError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"message": "Budget updated successfully",
	})
}

func (h *BudgetHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(*uuid.UUID)
	if !ok {
		apperrors.WriteJSONError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	idStr := chi.URLParam(r, "id")
	budgetID, err := uuid.Parse(idStr)
	if err != nil {
		apperrors.WriteJSONError(w, "Invalid budget ID", http.StatusBadRequest)
		return
	}

	err = h.budgetService.Delete(r.Context(), budgetID, *userID)
	if err != nil {
		var appErr *apperrors.AppError
		if errors.As(err, &appErr) {
			apperrors.WriteAppError(w, appErr)
			return
		}
		apperrors.WriteJSONError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"message": "Budget deleted successfully",
	})
}
