package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/rifqimalik/cashlens-backend/internal/service"
)

type contextKey string

const UserIDKey contextKey = "user_id"

func Auth(authService service.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			writeUnauthorized := func(msg string) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{"error": msg})
			}

			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				writeUnauthorized("missing authorization header")
				return
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				writeUnauthorized("invalid authorization header format")
				return
			}

			token := parts[1]
			userID, err := authService.ValidateToken(token)
			if err != nil {
				writeUnauthorized("invalid or expired token")
				return
			}

			ctx := context.WithValue(r.Context(), UserIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
