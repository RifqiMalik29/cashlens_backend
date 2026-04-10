package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/rifqimalik/cashlens-backend/internal/service"
)

type TransactionHandler struct {
	transactionService service.TransactionService
}

func NewTransactionHandler(transactionService service.TransactionService) *TransactionHandler {
	return &TransactionHandler{transactionService: transactionService}
}

func (h *TransactionHandler) Create(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "create transaction - to be implemented",
	})
}

func (h *TransactionHandler) List(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "list transactions - to be implemented",
	})
}
