package handlers

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type ReceiptHandler struct {
	geminiAPIKey string
}

func NewReceiptHandler(geminiAPIKey string) *ReceiptHandler {
	return &ReceiptHandler{geminiAPIKey: geminiAPIKey}
}

func (h *ReceiptHandler) ScanReceipt(w http.ResponseWriter, r *http.Request) {
	if h.geminiAPIKey == "" {
		http.Error(w, "Gemini API key not configured", http.StatusNotImplemented)
		return
	}

	// Parse multipart form (max 10MB)
	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("image")
	if err != nil {
		http.Error(w, "Image file is required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Validate file type
	if header.Header.Get("Content-Type") != "image/jpeg" && header.Header.Get("Content-Type") != "image/png" {
		http.Error(w, "Only JPEG and PNG images are supported", http.StatusBadRequest)
		return
	}

	// Read file
	imageData, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "Failed to read image", http.StatusInternalServerError)
		return
	}

	// Call Gemini API
	result, err := h.callGeminiVision(imageData)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to scan receipt: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"data": result,
	})
}

// Gemini API response structures
type GeminiRequest struct {
	Contents         []GeminiContent        `json:"contents"`
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

func (h *ReceiptHandler) callGeminiVision(imageData []byte) (map[string]any, error) {
	// Encode image to base64
	base64Image := base64.StdEncoding.EncodeToString(imageData)
	
	// Determine MIME type from magic bytes
	mimeType := "image/jpeg" // default
	if bytes.HasPrefix(imageData, []byte{0x89, 0x50, 0x4E, 0x47}) {
		mimeType = "image/png"
	}

	// Build request
	prompt := `You are a high-precision receipt parsing engine. Analyze this image and return ONLY a JSON object.

Structure:
{
  "amount": <number: actual total paid for items>,
  "currency": "IDR",
  "date": "YYYY-MM-DD",
  "merchant": "<string: brand name from top of receipt>",
  "category": "<string: internal_category_id>",
  "items": [{"name": string, "price": number}],
  "confidence": <number: 0-100>
}

Merchant Extraction Rules:
- The merchant is usually at the VERY TOP.
- Stylized fonts can be tricky. Look at the item list to confirm if the merchant name appears there too.
- Clean the name: Remove addresses, phone numbers, and slogans.

Category Selection Logic:
- cat_food: If you see items like "Mie", "Bakso", "Ayam", "Cendol", "Nasi", "Drink", "Food", or any restaurant names.
- cat_shopping: For clothes, electronics, or general department stores.
- cat_transport: For fuel, parking, or rideshare.
- Default to "cat_other_expense" only if absolutely zero context is found.

Anti-Hallucination Rules:
- IGNORE "Tunai", "Cash", or "Bayar" lines when picking the "amount".
- IGNORE "Kembalian" or "Change".
- The "amount" must equal the sum of item prices if available.`

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
