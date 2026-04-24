package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Server     ServerConfig
	Database   DatabaseConfig
	JWT        JWTConfig
	RateLimit  RateLimitConfig
	GeminiAPI  Gemini
	Telegram   Telegram
	Payment    Payment
	Google     GoogleConfig
	Mail       MailConfig
	Monitoring Monitoring
	CORS       CORSConfig
}

type CORSConfig struct {
	AllowedOrigins []string
}

type Monitoring struct {
	SentryDSN string
}

type MailConfig struct {
	Host           string
	Port           int
	User           string
	Password       string
	From           string
	BaseURL        string
	MobileDeepLink string
}

type ServerConfig struct {
	Port        string
	Environment string
}

type DatabaseConfig struct {
	URL string
}

type JWTConfig struct {
	Secret            string
	Expiration        time.Duration
	RefreshExpiration time.Duration
	MaxReuseWindow    time.Duration // For rotation: allow reusing token briefly
}

type RateLimitConfig struct {
	Requests int
	Window   time.Duration
	// Auth-specific limits (typically stricter)
	AuthRequests int
	AuthWindow   time.Duration
	// AI Scanner limits (expensive)
	ScannerRequests int
	ScannerWindow   time.Duration
}

type Gemini struct {
	APIKey                 string
	ScanningModel          string
	TelegramModel          string
	TelegramFallbackModels []string
}

type Telegram struct {
	BotToken      string
	WebhookSecret string
	Mode          string
}

type Payment struct {
	RevenueCatAPIKey        string
	RevenueCatWebhookSecret string
}

type GoogleConfig struct {
	ClientID string
}

func Load() (*Config, error) {
	// Load .env file (ignore error if not found)
	_ = godotenv.Load()

	cfg := &Config{
		Server: ServerConfig{
			Port:        getEnv("PORT", "8080"),
			Environment: getEnv("ENVIRONMENT", "development"),
		},
		Database: DatabaseConfig{
			URL: getEnv("DATABASE_URL", ""),
		},
		JWT: JWTConfig{
			Secret:            getEnv("JWT_SECRET", ""),
			Expiration:        parseDuration(getEnv("JWT_EXPIRATION", "24h"), 24*time.Hour),
			RefreshExpiration: parseDuration(getEnv("JWT_REFRESH_EXPIRATION", "168h"), 168*time.Hour), // 7 days
			MaxReuseWindow:    parseDuration(getEnv("JWT_MAX_REUSE_WINDOW", "5m"), 5*time.Minute),     // 5 minutes
		},
		RateLimit: RateLimitConfig{
			Requests:        parseInt(getEnv("RATE_LIMIT_REQUESTS", "100"), 100),
			Window:          parseDuration(getEnv("RATE_LIMIT_WINDOW", "1m"), time.Minute),
			AuthRequests:    parseInt(getEnv("RATE_LIMIT_AUTH_REQUESTS", "20"), 20),
			AuthWindow:      parseDuration(getEnv("RATE_LIMIT_AUTH_WINDOW", "5m"), 5*time.Minute),
			ScannerRequests: parseInt(getEnv("RATE_LIMIT_SCANNER_REQUESTS", "5"), 5),
			ScannerWindow:   parseDuration(getEnv("RATE_LIMIT_SCANNER_WINDOW", "1m"), time.Minute),
		},
		GeminiAPI: Gemini{
			APIKey:                 os.Getenv("GEMINI_API_KEY"),
			ScanningModel:          getEnv("SCANNING_AI", "gemini-2.5-flash"),
			TelegramModel:          getEnv("TELEGRAM_AI", "gemini-2.5-flash"),
			TelegramFallbackModels: []string{},
		},
		Telegram: Telegram{
			BotToken:      os.Getenv("TELEGRAM_BOT_TOKEN"),
			WebhookSecret: os.Getenv("TELEGRAM_WEBHOOK_SECRET"),
			Mode:          getEnv("TELEGRAM_MODE", "webhook"),
		},
		Payment: Payment{
			RevenueCatAPIKey:        os.Getenv("REVENUECAT_API_KEY"),
			RevenueCatWebhookSecret: os.Getenv("REVENUECAT_WEBHOOK_SECRET"),
		},
		Google: GoogleConfig{
			ClientID: os.Getenv("GOOGLE_CLIENT_ID"),
		},
		Mail: MailConfig{
			Host:           getEnv("SMTP_HOST", "localhost"),
			Port:           parseInt(getEnv("SMTP_PORT", "587"), 587),
			User:           os.Getenv("SMTP_USER"),
			Password:       os.Getenv("SMTP_PASSWORD"),
			From:           getEnv("SMTP_FROM", "noreply@cashlens.com"),
			BaseURL:        getEnv("BASE_URL", "http://localhost:8080"),
			MobileDeepLink: getEnv("MOBILE_DEEPLINK", "cashlens://auth/confirm"),
		},
		Monitoring: Monitoring{
			SentryDSN: os.Getenv("SENTRY_DSN"),
		},
		CORS: CORSConfig{
			AllowedOrigins: strings.Split(getEnv("ALLOWED_ORIGINS", "*"), ","),
		},
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) Validate() error {
	if c.Database.URL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	if c.JWT.Secret == "" {
		return fmt.Errorf("JWT_SECRET is required")
	}
	if c.Server.Environment == "production" && c.Google.ClientID == "" {
		return fmt.Errorf("GOOGLE_CLIENT_ID is required in production")
	}
	if c.Server.Environment == "production" && c.Payment.RevenueCatWebhookSecret == "" {
		return fmt.Errorf("REVENUECAT_WEBHOOK_SECRET is required in production")
	}
	if c.Server.Environment == "production" && c.Payment.RevenueCatAPIKey == "" {
		return fmt.Errorf("REVENUECAT_API_KEY is required in production")
	}
	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func parseInt(s string, defaultValue int) int {
	if v, err := strconv.Atoi(s); err == nil {
		return v
	}
	return defaultValue
}

func parseDuration(s string, defaultValue time.Duration) time.Duration {
	if d, err := time.ParseDuration(s); err == nil {
		return d
	}
	return defaultValue
}
