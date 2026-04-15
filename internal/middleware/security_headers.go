package middleware

import (
	"net/http"
)

// SecurityHeadersConfig holds configuration for security headers
type SecurityHeadersConfig struct {
	Environment string // "development" or "production"
}

// SecurityHeaders adds security headers to all responses
func SecurityHeaders(cfg SecurityHeadersConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Prevent MIME type sniffing
			w.Header().Set("X-Content-Type-Options", "nosniff")

			// Prevent clickjacking
			w.Header().Set("X-Frame-Options", "DENY")

			// Enable XSS protection
			w.Header().Set("X-XSS-Protection", "1; mode=block")

			// Enforce HTTPS only in production
			if cfg.Environment == "production" {
				w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			}

			// Referrer Policy
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

			// Content Security Policy (basic)
			w.Header().Set("Content-Security-Policy", "default-src 'none'")

			next.ServeHTTP(w, r)
		})
	}
}
