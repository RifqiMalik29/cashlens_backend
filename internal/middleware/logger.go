package middleware

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/rifqimalik/cashlens-backend/internal/logger"
)

// StructuredLogger is a middleware that adds structured logging with request ID correlation
func StructuredLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get request ID (set by chi's middleware.RequestID)
		requestID := middleware.GetReqID(r.Context())
		if requestID == "" {
			requestID = "unknown"
		}

		// Create a logger with request ID
		defaultLogger := logger.GetDefault()
		reqLogger := defaultLogger.WithRequestID(requestID).With(
			"method", r.Method,
			"path", r.URL.Path,
			"remote_addr", r.RemoteAddr,
		)

		// Add logger to context
		ctx := logger.WithContext(r.Context(), reqLogger)
		r = r.WithContext(ctx)

		// Log the request
		reqLogger.Info("request started",
			"user_agent", r.UserAgent(),
		)

		next.ServeHTTP(w, r)
	})
}

// GetLogger retrieves the logger from context or returns the default logger
func GetLogger(ctx context.Context) *logger.Logger {
	return logger.FromContext(ctx, logger.GetDefault())
}
