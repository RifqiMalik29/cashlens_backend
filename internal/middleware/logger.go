package middleware

import (
	"context"
	"net/http"
	"time"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/rifqimalik/cashlens-backend/internal/logger"
)

// responseWriter wraps http.ResponseWriter to capture status code and bytes written
type responseWriter struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.bytes += n
	return n, err
}

// StructuredLogger is a middleware that logs request start and completion with status/latency
func StructuredLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := chimiddleware.GetReqID(r.Context())
		if requestID == "" {
			requestID = "unknown"
		}

		reqLogger := logger.GetDefault().WithRequestID(requestID).With(
			"method", r.Method,
			"path", r.URL.Path,
			"remote_addr", r.RemoteAddr,
		)

		ctx := logger.WithContext(r.Context(), reqLogger)
		r = r.WithContext(ctx)

		rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}
		start := time.Now()

		next.ServeHTTP(rw, r)

		latency := time.Since(start)
		logFn := reqLogger.Info
		if rw.status >= 500 {
			logFn = reqLogger.Error
		} else if rw.status >= 400 {
			logFn = reqLogger.Warn
		}

		logFn("request completed",
			"status", rw.status,
			"latency_ms", latency.Milliseconds(),
			"bytes", rw.bytes,
		)
	})
}

// GetLogger retrieves the logger from context or returns the default logger
func GetLogger(ctx context.Context) *logger.Logger {
	return logger.FromContext(ctx, logger.GetDefault())
}
