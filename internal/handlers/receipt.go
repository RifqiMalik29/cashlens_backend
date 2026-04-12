package handlers

import (
	"encoding/json"
	"net/http"
)

type ReceiptHandler struct {
	geminiAPIKey string
}

func NewReceiptHandler(geminiAPIKey string) *ReceiptHandler {
	return &ReceiptHandler{geminiAPIKey: geminiAPIKey}
}

func (h *ReceiptHandler) ScanReceipt(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement Gemini API integration
	// 1. Parse multipart file from request
	// 2. Send to Gemini Vision API
	// 3. Extract: amount, date, merchant, line items
	// 4. Return structured data
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]any{
		"message": "Receipt scanner not yet implemented - requires Gemini API integration",
		"setup": "Add GEMINI_API_KEY to .env and implement Gemini SDK calls",
	})
}
