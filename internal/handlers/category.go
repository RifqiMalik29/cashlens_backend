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
	"github.com/rifqimalik/cashlens-backend/internal/service"
)

type CategoryHandler struct {
	categoryService service.CategoryService
}

func NewCategoryHandler(categoryService service.CategoryService) *CategoryHandler {
	return &CategoryHandler{categoryService: categoryService}
}

func (h *CategoryHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req models.CreateCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apperrors.WriteJSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	id, ok := r.Context().Value(middleware.UserIDKey).(*uuid.UUID)
	if !ok {
		apperrors.WriteJSONError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	res, err := h.categoryService.Create(r.Context(), *id, req)
	if err != nil {
		apperrors.WriteJSONError(w, "Failed to create category", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{
		"data": res,
	})
}

func (h *CategoryHandler) List(w http.ResponseWriter, r *http.Request) {
	id, ok := r.Context().Value(middleware.UserIDKey).(*uuid.UUID)
	if !ok {
		apperrors.WriteJSONError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	res, err := h.categoryService.ListUserCategories(r.Context(), *id)
	if err != nil {
		apperrors.WriteJSONError(w, "Failed to list categories", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"data": res,
	})
}

func (h *CategoryHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(*uuid.UUID)
	if !ok {
		apperrors.WriteJSONError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	idStr := chi.URLParam(r, "id")
	parseId, err := uuid.Parse(idStr)
	if err != nil {
		apperrors.WriteJSONError(w, "Invalid category ID", http.StatusBadRequest)
		return
	}

	res, err := h.categoryService.Get(r.Context(), parseId, *userID)
	if err != nil {
		apperrors.WriteJSONError(w, "Failed to get category", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"data": res,
	})
}

func (h *CategoryHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := r.Context().Value(middleware.UserIDKey).(*uuid.UUID)
	if !ok {
		apperrors.WriteJSONError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	params := chi.URLParam(r, "id")
	parse, err := uuid.Parse(params)
	if err != nil {
		apperrors.WriteJSONError(w, "Invalid category ID", http.StatusBadRequest)
		return
	}

	err = h.categoryService.Delete(r.Context(), parse, *id)
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
		"message": "Category deleted successfully",
	})
}

func (h *CategoryHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(*uuid.UUID)
	if !ok {
		apperrors.WriteJSONError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	idStr := chi.URLParam(r, "id")
	categoryID, err := uuid.Parse(idStr)
	if err != nil {
		apperrors.WriteJSONError(w, "Invalid category ID", http.StatusBadRequest)
		return
	}

	var req models.UpdateCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apperrors.WriteJSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	err = h.categoryService.Update(r.Context(), categoryID, *userID, req)
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
		"message": "Category updated successfully",
	})
}
