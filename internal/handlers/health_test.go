package handlers

import (
	"testing"
)

func TestHealthCheck(t *testing.T) {
	// TODO: Implement test with mock database
	// req := httptest.NewRequest(http.MethodGet, "/health", nil)
	// w := httptest.NewRecorder()

	// handler.Check(w, req)

	// res := w.Result()
	// if res.StatusCode != http.StatusOK {
	//     t.Errorf("expected status OK, got %v", res.Status)
	// }

	t.Skip("Test requires database mock - to be implemented")
}
