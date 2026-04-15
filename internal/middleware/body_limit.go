package middleware

import (
	"net/http"

	apperrors "github.com/rifqimalik/cashlens-backend/internal/errors"
)

// MaxBodyLimit sets maximum request body size (default 1MB for JSON, handled separately for multipart)
func MaxBodyLimit(bytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.ContentLength > bytes {
				apperrors.WriteJSONError(w, "Request body too large", http.StatusRequestEntityTooLarge)
				return
			}

			r.Body = http.MaxBytesReader(w, r.Body, bytes)
			next.ServeHTTP(w, r)
		})
	}
}
