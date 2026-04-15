package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	apperrors "github.com/rifqimalik/cashlens-backend/internal/errors"
	"github.com/rifqimalik/cashlens-backend/internal/middleware"
	"github.com/rifqimalik/cashlens-backend/internal/models"
	"github.com/rifqimalik/cashlens-backend/internal/pkg/validator"
	"github.com/rifqimalik/cashlens-backend/internal/service"
)

type TransactionHandler struct {
	transactionService service.TransactionService
}

func NewTransactionHandler(transactionService service.TransactionService) *TransactionHandler {
	return &TransactionHandler{transactionService: transactionService}
}

func (h *TransactionHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req models.CreateTransactionRequest
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

	res, err := h.transactionService.Create(r.Context(), *userID, req)
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

func (h *TransactionHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(*uuid.UUID)
	if !ok {
		apperrors.WriteJSONError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	res, err := h.transactionService.List(r.Context(), *userID, limit, offset)
	if err != nil {
		apperrors.WriteJSONError(w, "Failed to list transactions", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"data": res,
	})
}

func (h *TransactionHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(*uuid.UUID)
	if !ok {
		apperrors.WriteJSONError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	idStr := chi.URLParam(r, "id")
	transactionID, err := uuid.Parse(idStr)
	if err != nil {
		apperrors.WriteJSONError(w, "Invalid transaction ID", http.StatusBadRequest)
		return
	}

	res, err := h.transactionService.Get(r.Context(), transactionID, *userID)
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

func (h *TransactionHandler) ListByDateRange(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(*uuid.UUID)
	if !ok {
		apperrors.WriteJSONError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	startStr := r.URL.Query().Get("start")
	endStr := r.URL.Query().Get("end")

	if startStr == "" || endStr == "" {
		apperrors.WriteJSONError(w, "Start and end date query parameters are required", http.StatusBadRequest)
		return
	}

	start, err := time.Parse("2006-01-02", startStr)
	if err != nil {
		apperrors.WriteJSONError(w, "Invalid start date format (expected YYYY-MM-DD)", http.StatusBadRequest)
		return
	}

	end, err := time.Parse("2006-01-02", endStr)
	if err != nil {
		apperrors.WriteJSONError(w, "Invalid end date format (expected YYYY-MM-DD)", http.StatusBadRequest)
		return
	}

	res, err := h.transactionService.ListByDateRange(r.Context(), *userID, start, end)
	if err != nil {
		apperrors.WriteJSONError(w, "Failed to list transactions", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"data": res,
	})
}

func (h *TransactionHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(*uuid.UUID)
	if !ok {
		apperrors.WriteJSONError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	idStr := chi.URLParam(r, "id")
	transactionID, err := uuid.Parse(idStr)
	if err != nil {
		apperrors.WriteJSONError(w, "Invalid transaction ID", http.StatusBadRequest)
		return
	}

	var req models.UpdateTransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apperrors.WriteJSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if validationErrors := validator.ValidateStruct(&req); validationErrors != nil {
		apperrors.WriteJSONError(w, "Validation failed", http.StatusBadRequest, validationErrors)
		return
	}

	res, err := h.transactionService.Update(r.Context(), transactionID, *userID, req)
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
		"data":    res,
		"message": "Transaction updated successfully",
	})
}

func (h *TransactionHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(*uuid.UUID)
	if !ok {
		apperrors.WriteJSONError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	idStr := chi.URLParam(r, "id")
	transactionID, err := uuid.Parse(idStr)
	if err != nil {
		apperrors.WriteJSONError(w, "Invalid transaction ID", http.StatusBadRequest)
		return
	}

	err = h.transactionService.Delete(r.Context(), transactionID, *userID)
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
		"message": "Transaction deleted successfully",
	})
}
