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

type DraftHandler struct {
	draftService service.DraftService
}

func NewDraftHandler(draftService service.DraftService) *DraftHandler {
	return &DraftHandler{draftService: draftService}
}

func (h *DraftHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req models.CreateDraftRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid Request Body", http.StatusBadRequest)
		return
	}

	userID, ok := r.Context().Value(middleware.UserIDKey).(*uuid.UUID)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	res, err := h.draftService.Create(r.Context(), *userID, req)
	if err != nil {
		var appErr *apperrors.AppError
		if errors.As(err, &appErr) {
			http.Error(w, appErr.Message, appErr.StatusCode())
			return
		}
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{
		"data": res,
	})
}

func (h *DraftHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(*uuid.UUID)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	status := models.DraftStatus(r.URL.Query().Get("status"))
	
	res, err := h.draftService.List(r.Context(), *userID, status)
	if err != nil {
		http.Error(w, "Failed to list drafts", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"data": res,
	})
}

func (h *DraftHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(*uuid.UUID)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	idStr := chi.URLParam(r, "id")
	draftID, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid Draft ID", http.StatusBadRequest)
		return
	}

	res, err := h.draftService.Get(r.Context(), draftID, *userID)
	if err != nil {
		var appErr *apperrors.AppError
		if errors.As(err, &appErr) {
			http.Error(w, appErr.Message, appErr.StatusCode())
			return
		}
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"data": res,
	})
}

func (h *DraftHandler) Confirm(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(*uuid.UUID)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	idStr := chi.URLParam(r, "id")
	draftID, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid Draft ID", http.StatusBadRequest)
		return
	}

	var req models.ConfirmDraftRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid Request Body", http.StatusBadRequest)
		return
	}

	res, err := h.draftService.Confirm(r.Context(), draftID, *userID, req)
	if err != nil {
		var appErr *apperrors.AppError
		if errors.As(err, &appErr) {
			http.Error(w, appErr.Message, appErr.StatusCode())
			return
		}
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{
		"data": res,
	})
}

func (h *DraftHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(*uuid.UUID)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	idStr := chi.URLParam(r, "id")
	draftID, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid Draft ID", http.StatusBadRequest)
		return
	}

	err = h.draftService.Delete(r.Context(), draftID, *userID)
	if err != nil {
		var appErr *apperrors.AppError
		if errors.As(err, &appErr) {
			http.Error(w, appErr.Message, appErr.StatusCode())
			return
		}
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"message": "Draft deleted successfully",
	})
}
