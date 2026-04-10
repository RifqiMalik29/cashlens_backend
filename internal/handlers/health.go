package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/rifqimalik/cashlens-backend/internal/database"
	"github.com/rifqimalik/cashlens-backend/internal/models"
)

type HealthHandler struct {
	db *database.Database
}

func NewHealthHandler(db *database.Database) *HealthHandler {
	return &HealthHandler{db: db}
}

func (h *HealthHandler) Check(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	response := models.HealthResponse{
		Status: "ok",
	}

	// Check database health
	if err := h.db.Health(ctx); err != nil {
		response.Database = "unhealthy: " + err.Error()
		w.WriteHeader(http.StatusServiceUnavailable)
	} else {
		response.Database = "healthy"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
