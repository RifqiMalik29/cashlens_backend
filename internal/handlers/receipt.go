package handlers

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"
	apperrors "github.com/rifqimalik/cashlens-backend/internal/errors"
	"github.com/rifqimalik/cashlens-backend/internal/middleware"
	"github.com/rifqimalik/cashlens-backend/internal/models"
	"github.com/rifqimalik/cashlens-backend/internal/pkg/gemini"
	"github.com/rifqimalik/cashlens-backend/internal/repository"
)

type ReceiptHandler struct {
	geminiAPIKey   string
	categoryRepo   repository.CategoryRepository
}

func NewReceiptHandler(geminiAPIKey string, categoryRepo repository.CategoryRepository) *ReceiptHandler {
	return &ReceiptHandler{
		geminiAPIKey:   geminiAPIKey,
		categoryRepo:   categoryRepo,
	}
}

func (h *ReceiptHandler) ScanReceipt(w http.ResponseWriter, r *http.Request) {
	if h.geminiAPIKey == "" {
		apperrors.WriteJSONError(w, "Gemini API key not configured", http.StatusNotImplemented)
		return
	}

	// Parse multipart form (max 10MB)
	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		apperrors.WriteJSONError(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("image")
	if err != nil {
		apperrors.WriteJSONError(w, "Image file is required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Validate file type
	if header.Header.Get("Content-Type") != "image/jpeg" && header.Header.Get("Content-Type") != "image/png" {
		apperrors.WriteJSONError(w, "Only JPEG and PNG images are supported", http.StatusBadRequest)
		return
	}

	// Read file
	imageData, err := io.ReadAll(file)
	if err != nil {
		apperrors.WriteJSONError(w, "Failed to read image", http.StatusInternalServerError)
		return
	}

	// Extract user ID from context
	userID, ok := r.Context().Value(middleware.UserIDKey).(*uuid.UUID)
	if !ok {
		apperrors.WriteJSONError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Fetch user's categories for dynamic Gemini prompt
	categories, err := h.categoryRepo.ListByUserID(r.Context(), *userID)
	if err != nil {
		categories = []*models.Category{} // fallback gracefully
	}

	// Call Gemini API
	result, err := h.callGeminiVision(imageData, categories)
	if err != nil {
		apperrors.WriteJSONError(w, fmt.Sprintf("Failed to scan receipt: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"data": result,
	})
}

func (h *ReceiptHandler) callGeminiVision(imageData []byte, categories []*models.Category) (map[string]any, error) {
	// Encode image to base64
	base64Image := base64.StdEncoding.EncodeToString(imageData)

	// Determine MIME type from magic bytes
	mimeType := "image/jpeg" // default
	if bytes.HasPrefix(imageData, []byte{0x89, 0x50, 0x4E, 0x47}) {
		mimeType = "image/png"
	}

	// Build category list for prompt
	var catBuilder strings.Builder
	for _, c := range categories {
		catBuilder.WriteString(fmt.Sprintf("- %s (%s)\n", c.Name, c.Type))
	}
	catList := catBuilder.String()

	// Build request
	prompt := fmt.Sprintf(`You are a high-precision receipt parsing engine. Analyze this image and return ONLY a JSON object.

Structure:
{
  "amount": <number: actual total paid for items>,
  "currency": "IDR",
  "date": "YYYY-MM-DD",
  "merchant": "<string: brand name from top of receipt>",
  "category": "<string: exact category name from the list below>",
  "items": [{"name": string, "price": number}],
  "confidence": <number: 0-100>
}

Merchant Extraction Rules:
- The merchant is usually at the VERY TOP.
- Stylized fonts can be tricky. Look at the item list to confirm if the merchant name appears there too.
- Clean the name: Remove addresses, phone numbers, and slogans.

Available Categories (choose the most fitting name exactly as written):
%s

Category Selection Rules:
- Match based on the items purchased, not the store name.
- Food, drinks, restaurants → food category
- Stationery, office supplies, crafts → shopping category
- Fuel, parking, rideshare → transport category
- If unsure, pick the closest match. Only use "Other" if nothing fits.

Anti-Hallucination Rules:
- IGNORE "Tunai", "Cash", or "Bayar" lines when picking the "amount".
- IGNORE "Kembalian" or "Change".
- The "amount" must equal the sum of item prices if available.
- Return ONLY the JSON object, no markdown, no explanation.`, catList)

	requestBody := gemini.GeminiRequest{
		Contents: []gemini.GeminiContent{
			{
				Parts: []gemini.GeminiPart{
					{Text: prompt},
					{InlineData: &gemini.GeminiImageData{MimeType: mimeType, Data: base64Image}},
				},
			},
		},
		GenerationConfig: &gemini.GeminiGenerationConfig{
			ResponseMimeType: "application/json",
			Temperature:      0.1,
		},
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Call Gemini API
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/gemini-3.1-flash-lite-preview:generateContent?key=%s", h.geminiAPIKey)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to call Gemini API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Gemini API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse response
	var geminiResp gemini.GeminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&geminiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("empty response from Gemini API")
	}

	// With JSON response mode, no markdown cleaning needed
	responseText := geminiResp.Candidates[0].Content.Parts[0].Text

	// Parse the extracted JSON
	var result map[string]any
	if err := json.Unmarshal([]byte(responseText), &result); err != nil {
		return nil, fmt.Errorf("failed to parse receipt data: %w", err)
	}

	return result, nil
}
