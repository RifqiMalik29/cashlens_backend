package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
)

// contextKey is a custom type for context keys to avoid collisions
type contextKey struct {
	name string
}

// RequestIDKey is the context key for request ID
var RequestIDKey = contextKey{"request_id"}

// Logger wraps slog.Logger and provides additional helper methods
type Logger struct {
	*slog.Logger
}

// New creates a new structured logger with the configured handler
func New(level slog.Level, format string) *Logger {
	var handler slog.Handler

	opts := &slog.HandlerOptions{
		Level: level,
		// Add source file and line number in development
		AddSource: os.Getenv("ENVIRONMENT") == "development",
	}

	switch format {
	case "json":
		handler = slog.NewJSONHandler(os.Stdout, opts)
	default:
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	return &Logger{
		Logger: slog.New(handler),
	}
}

// NewWithWriter creates a logger that writes to the provided writer
func NewWithWriter(w io.Writer, level slog.Level, format string) *Logger {
	var handler slog.Handler

	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: os.Getenv("ENVIRONMENT") == "development",
	}

	switch format {
	case "json":
		handler = slog.NewJSONHandler(w, opts)
	default:
		handler = slog.NewTextHandler(w, opts)
	}

	return &Logger{
		Logger: slog.New(handler),
	}
}

// WithRequestID adds a request ID to the logger for correlation
func (l *Logger) WithRequestID(requestID string) *Logger {
	return &Logger{
		Logger: l.Logger.With(slog.String("request_id", requestID)),
	}
}

// With adds key-value pairs to the logger
func (l *Logger) With(args ...any) *Logger {
	return &Logger{
		Logger: l.Logger.With(args...),
	}
}

// FromContext retrieves a logger from the context, or returns the default logger
func FromContext(ctx context.Context, defaultLogger *Logger) *Logger {
	if logger, ok := ctx.Value(loggerKey{}).(*Logger); ok {
		return logger
	}
	return defaultLogger
}

// WithContext adds a logger to the context
func WithContext(ctx context.Context, logger *Logger) context.Context {
	return context.WithValue(ctx, loggerKey{}, logger)
}

type loggerKey struct{}

// Default logger instance
var defaultLogger *Logger

// Init initializes the default logger based on environment variables
func Init() *Logger {
	level := slog.LevelInfo
	if os.Getenv("ENVIRONMENT") == "development" {
		level = slog.LevelDebug
	}

	format := os.Getenv("LOG_FORMAT")
	if format == "" {
		format = "text"
	}

	defaultLogger = New(level, format)
	slog.SetDefault(defaultLogger.Logger)

	return defaultLogger
}

// GetDefault returns the default logger
func GetDefault() *Logger {
	if defaultLogger == nil {
		return Init()
	}
	return defaultLogger
}
