package handlers

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	apperrors "github.com/rifqimalik/cashlens-backend/internal/errors"
	"github.com/rifqimalik/cashlens-backend/internal/logger"
	"github.com/rifqimalik/cashlens-backend/internal/middleware"
	"github.com/rifqimalik/cashlens-backend/internal/models"
	"github.com/rifqimalik/cashlens-backend/internal/repository"
	"github.com/rifqimalik/cashlens-backend/internal/service"
)

type ReceiptHandler struct {
	geminiAPIKey   string
	geminiModel    string
	quotaService   service.QuotaService
	categoryRepo   repository.CategoryRepository
}

func NewReceiptHandler(geminiAPIKey, geminiModel string, quotaService service.QuotaService, categoryRepo repository.CategoryRepository) *ReceiptHandler {
	return &ReceiptHandler{
		geminiAPIKey: geminiAPIKey,
		geminiModel:  geminiModel,
		quotaService: quotaService,
		categoryRepo: categoryRepo,
	}
}

func (h *ReceiptHandler) ScanReceipt(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context(), logger.GetDefault())
	log = log.With("component", "receipt_scanner")

	if h.geminiAPIKey == "" {
		apperrors.WriteJSONError(w, "Gemini API key not configured", http.StatusNotImplemented)
		return
	}

	userID, ok := r.Context().Value(middleware.UserIDKey).(*uuid.UUID)
	if !ok {
		apperrors.WriteJSONError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	log.Info("Receipt scan request started", "user_id", userID)

	// Parse multipart form (max 10MB)
	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		log.Error("Failed to parse multipart form", "error", err)
		apperrors.WriteJSONError(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("image")
	if err != nil {
		log.Error("Image file required", "error", err)
		apperrors.WriteJSONError(w, "Image file is required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Validate file type using actual bytes, not client-controlled header
	imageData, err := io.ReadAll(file)
	if err != nil {
		log.Error("Failed to read image file", "error", err)
		apperrors.WriteJSONError(w, "Failed to read image", http.StatusInternalServerError)
		return
	}

	detectedType := http.DetectContentType(imageData)
	log.Info("Image type detected", "mime_type", detectedType, "size_bytes", len(imageData))

	if detectedType != "image/jpeg" && detectedType != "image/png" {
		log.Warn("Unsupported image type", "detected_type", detectedType)
		apperrors.WriteJSONError(w, "Only JPEG and PNG images are supported", http.StatusBadRequest)
		return
	}

	// Fetch user's expense categories to pass into the prompt
	categories, err := h.categoryRepo.ListByUserID(r.Context(), *userID)
	if err != nil {
		log.Error("Failed to fetch user categories", "error", err)
		apperrors.WriteJSONError(w, "Failed to load categories", http.StatusInternalServerError)
		return
	}
	expenseCategories := make([]*models.Category, 0, len(categories))
	for _, c := range categories {
		if c.Type == models.CategoryTypeExpense {
			expenseCategories = append(expenseCategories, c)
		}
	}

	// Atomic quota check + increment (prevents TOCTOU race condition)
	quotaStart := time.Now()
	if err := h.quotaService.CheckAndIncrementScanQuota(r.Context(), *userID); err != nil {
		var appErr *apperrors.AppError
		if errors.As(err, &appErr) {
			log.Warn("Scan quota exceeded", "latency_ms", time.Since(quotaStart).Milliseconds(), "error", appErr.Message)
			apperrors.WriteAppError(w, appErr)
		} else {
			log.Error("Failed to check quota", "latency_ms", time.Since(quotaStart).Milliseconds(), "error", err)
			apperrors.WriteJSONError(w, "Failed to check quota", http.StatusInternalServerError)
		}
		return
	}
	log.Info("Scan quota check passed", "latency_ms", time.Since(quotaStart).Milliseconds())

	// Call Gemini API
	geminiStart := time.Now()
	result, err := h.callGeminiVision(imageData, expenseCategories)
	geminiLatency := time.Since(geminiStart)

	if err != nil {
		log.Error("Gemini API call failed", "latency_ms", geminiLatency.Milliseconds(), "error", err)
		apperrors.WriteJSONError(w, fmt.Sprintf("Failed to scan receipt: %v", err), http.StatusInternalServerError)
		return
	}

	// Extract key info for logging
	amount := 0.0
	merchant := ""
	confidence := 0.0
	if v, ok := result["amount"].(float64); ok {
		amount = v
	}
	if v, ok := result["merchant"].(string); ok {
		merchant = v
	}
	if v, ok := result["confidence"].(float64); ok {
		confidence = v
	}

	itemCount := 0
	if items, ok := result["items"].([]interface{}); ok {
		itemCount = len(items)
	}

	log.Info("Receipt scan completed successfully",
		"latency_ms", geminiLatency.Milliseconds(),
		"merchant", merchant,
		"amount", amount,
		"confidence", confidence,
		"items_count", itemCount,
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"data": result,
	})
}

// Gemini API response structures
type GeminiRequest struct {
	Contents         []GeminiContent         `json:"contents"`
	GenerationConfig *GeminiGenerationConfig `json:"generationConfig,omitempty"`
}

type GeminiGenerationConfig struct {
	ResponseMimeType string  `json:"responseMimeType"`
	Temperature      float64 `json:"temperature"`
}

type GeminiContent struct {
	Parts []GeminiPart `json:"parts"`
}

type GeminiPart struct {
	Text       string           `json:"text,omitempty"`
	InlineData *GeminiImageData `json:"inlineData,omitempty"`
}

type GeminiImageData struct {
	MimeType string `json:"mimeType"`
	Data     string `json:"data"`
}

type GeminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

func (h *ReceiptHandler) callGeminiVision(imageData []byte, categories []*models.Category) (map[string]any, error) {
	// Encode image to base64
	base64Image := base64.StdEncoding.EncodeToString(imageData)

	// Determine MIME type from magic bytes
	mimeType := "image/jpeg" // default
	if bytes.HasPrefix(imageData, []byte{0x89, 0x50, 0x4E, 0x47}) {
		mimeType = "image/png"
	}

	// Build id:name list for the prompt so Gemini returns the UUID directly
	var categoryLines strings.Builder
	fallbackID := ""
	for _, c := range categories {
		categoryLines.WriteString(fmt.Sprintf("- %s (%s)\n", c.ID.String(), c.Name))
		if c.Name == "Lainnya" && fallbackID == "" {
			fallbackID = c.ID.String()
		}
	}
	if len(categories) == 0 {
		categoryLines.WriteString("(no categories available)\n")
	}

	prompt := fmt.Sprintf(`You are a high-precision receipt parsing engine. Analyze this image and return ONLY a JSON object.

Structure:
{
  "amount": <number: actual total paid for items>,
  "currency": "IDR",
  "date": "YYYY-MM-DD",
  "merchant": "<string: brand name from top of receipt>",
  "category_id": "<string: MUST be one of the UUIDs listed below>",
  "items": [{"name": string, "price": number}],
  "confidence": <number: 0-100>
}

Available categories — format is: UUID (Name). Return only the UUID in category_id.
%s
Category Selection Logic:
- Choose "Makanan & Minuman" for food, drinks, restaurants, cafes.
- Choose "Transportasi" for fuel, parking, rideshare, toll.
- Choose "Belanja" for clothes, electronics, department stores, supermarkets.
- Choose "Hiburan" for movies, games, events.
- Choose "Kesehatan" for pharmacy, clinic, hospital.
- Choose "Pendidikan" for books, courses, school supplies.
- Choose "Tagihan & Utilitas" for electricity, water, internet, phone bills.
- Default to "Lainnya" (UUID: %s) if nothing matches.

Merchant Extraction Rules:
- The merchant is usually at the VERY TOP.
- Clean the name: Remove addresses, phone numbers, and slogans.

Anti-Hallucination Rules:
- IGNORE "Tunai", "Cash", or "Bayar" lines when picking the "amount".
- IGNORE "Kembalian" or "Change".
- The "amount" must equal the sum of item prices if available.`, categoryLines.String(), fallbackID)

	requestBody := GeminiRequest{
		Contents: []GeminiContent{
			{
				Parts: []GeminiPart{
					{Text: prompt},
					{InlineData: &GeminiImageData{MimeType: mimeType, Data: base64Image}},
				},
			},
		},
		GenerationConfig: &GeminiGenerationConfig{
			ResponseMimeType: "application/json",
			Temperature:      0.1,
		},
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Call Gemini API
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", h.geminiModel, h.geminiAPIKey)

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
	var geminiResp GeminiResponse
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
