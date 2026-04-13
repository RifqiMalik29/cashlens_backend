package config

import (
	"fmt"
	"os"
	"strconv"
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
}

type ServerConfig struct {
	Port        string
	Environment string
}

type DatabaseConfig struct {
	URL string
}

type JWTConfig struct {
	Secret             string
	Expiration         time.Duration
	RefreshExpiration  time.Duration
	MaxReuseWindow     time.Duration // For rotation: allow reusing token briefly
}

type RateLimitConfig struct {
	Requests int
	Window   time.Duration
	// Auth-specific limits (typically stricter)
	AuthRequests int
	AuthWindow   time.Duration
}

type Gemini struct {
	APIKey string
}

type Telegram struct {
	BotToken string
}

type Payment struct {
	XenditWebhookToken string
	XenditSecretKey    string
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
			Secret:             getEnv("JWT_SECRET", ""),
			Expiration:         parseDuration(getEnv("JWT_EXPIRATION", "24h"), 24*time.Hour),
			RefreshExpiration:  parseDuration(getEnv("JWT_REFRESH_EXPIRATION", "168h"), 168*time.Hour), // 7 days
			MaxReuseWindow:     parseDuration(getEnv("JWT_MAX_REUSE_WINDOW", "5m"), 5*time.Minute),     // 5 minutes
		},
		RateLimit: RateLimitConfig{
			Requests:     parseInt(getEnv("RATE_LIMIT_REQUESTS", "100"), 100),
			Window:       parseDuration(getEnv("RATE_LIMIT_WINDOW", "1m"), time.Minute),
			AuthRequests: parseInt(getEnv("RATE_LIMIT_AUTH_REQUESTS", "20"), 20),
			AuthWindow:   parseDuration(getEnv("RATE_LIMIT_AUTH_WINDOW", "5m"), 5*time.Minute),
		},
		GeminiAPI: Gemini{
			APIKey: os.Getenv("GEMINI_API_KEY"),
		},
		Telegram: Telegram{
			BotToken: os.Getenv("TELEGRAM_BOT_TOKEN"),
		},
		Payment: Payment{
			XenditWebhookToken: os.Getenv("XENDIT_WEBHOOK_TOKEN"),
			XenditSecretKey:    os.Getenv("XENDIT_SECRET_KEY"),
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
