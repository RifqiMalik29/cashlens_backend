package handlers

import (
	"encoding/json"
	"net/http"

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
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "create category - to be implemented",
	})
}

func (h *CategoryHandler) List(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "list categories - to be implemented",
	})
}
