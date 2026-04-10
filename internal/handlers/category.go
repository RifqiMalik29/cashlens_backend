package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
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
	// TODO: Implement
	var req models.CreateCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid Request Body", http.StatusBadRequest)
		return
	}

	id, ok := r.Context().Value(middleware.UserIDKey).(*uuid.UUID)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	res, err := h.categoryService.Create(r.Context(), *id, req)
	if err != nil {
		http.Error(w, "Create Category Failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(res)
}

func (h *CategoryHandler) List(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement
	id, ok := r.Context().Value(middleware.UserIDKey).(*uuid.UUID)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	res, err := h.categoryService.ListUserCategories(r.Context(), *id)
	if err != nil {
		http.Error(w, "Get Categories Failed", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(res)
}
