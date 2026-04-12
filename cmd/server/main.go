package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"

	"github.com/rifqimalik/cashlens-backend/internal/config"
	"github.com/rifqimalik/cashlens-backend/internal/database"
	"github.com/rifqimalik/cashlens-backend/internal/handlers"
	custommiddleware "github.com/rifqimalik/cashlens-backend/internal/middleware"
	"github.com/rifqimalik/cashlens-backend/internal/logger"
	"github.com/rifqimalik/cashlens-backend/internal/repository"
	"github.com/rifqimalik/cashlens-backend/internal/service"
	"github.com/rifqimalik/cashlens-backend/internal/telegram"
)

func main() {
	// Initialize structured logger
	log := logger.Init()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Error("Failed to load config", "error", err)
		os.Exit(1)
	}

	// Setup context
	ctx := context.Background()

	// Initialize database
	db, err := database.New(ctx, cfg.Database.URL)
	if err != nil {
		log.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	log.Info("Database connected successfully")

	// Initialize repositories
	userRepo := repository.NewUserRepository(db.Pool)
	categoryRepo := repository.NewCategoryRepository(db.Pool)
	transactionRepo := repository.NewTransactionRepository(db.Pool)
	budgetRepo := repository.NewBudgetRepository(db.Pool)
	draftRepo := repository.NewDraftRepository(db.Pool)
	chatRepo := repository.NewChatLinkRepository(db.Pool)
	refreshTokenRepo := repository.NewRefreshTokenRepository(db.Pool)

	// Initialize services
	authService := service.NewAuthService(userRepo, cfg.JWT.Secret, cfg.JWT.Expiration)
	refreshTokenService := service.NewRefreshTokenService(
		refreshTokenRepo,
		userRepo,
		cfg.JWT.Secret,
		cfg.JWT.Expiration,
		cfg.JWT.RefreshExpiration,
		cfg.JWT.MaxReuseWindow,
	)
	categoryService := service.NewCategoryService(categoryRepo)
	transactionService := service.NewTransactionService(transactionRepo)
	budgetService := service.NewBudgetService(budgetRepo)
	draftService := service.NewDraftService(draftRepo, transactionRepo)

	// Initialize handlers
	healthHandler := handlers.NewHealthHandler(db)
	authHandler := handlers.NewAuthHandler(authService, refreshTokenService)
	categoryHandler := handlers.NewCategoryHandler(categoryService)
	transactionHandler := handlers.NewTransactionHandler(transactionService)
	budgetHandler := handlers.NewBudgetHandler(budgetService)
	draftHandler := handlers.NewDraftHandler(draftService)
	receiptHandler := handlers.NewReceiptHandler(cfg.GeminiAPI.APIKey)

	// Initialize Telegram Bot
	var botService *telegram.BotService
	if cfg.Telegram.BotToken != "" {
		botService = telegram.NewBotService(
			cfg.Telegram.BotToken,
			cfg.GeminiAPI.APIKey,
			draftService,
			transactionService,
			budgetService,
			draftRepo,
			transactionRepo,
			budgetRepo,
			userRepo,
			chatRepo,
			categoryRepo,
		)
		go botService.StartPolling(context.Background())
		log.Info("Telegram bot started")
	} else {
		log.Info("Telegram bot token not configured - skipping bot initialization")
	}

	// Setup router
	r := chi.NewRouter()

	// Global middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(custommiddleware.StructuredLogger)
	r.Use(middleware.Recoverer)
	r.Use(custommiddleware.CORS)

	// Health check (public, no rate limiting)
	r.Get("/health", healthHandler.Check)

	// API routes
	r.Route("/api/v1", func(r chi.Router) {
		// Public auth routes (stricter rate limiting)
		r.Group(func(r chi.Router) {
			r.Use(httprate.LimitByIP(cfg.RateLimit.AuthRequests, cfg.RateLimit.AuthWindow))
			r.Post("/auth/register", authHandler.Register)
			r.Post("/auth/login", authHandler.Login)
			r.Post("/auth/refresh", authHandler.Refresh)
		})

		// Protected routes (standard rate limiting)
		r.Group(func(r chi.Router) {
			r.Use(custommiddleware.Auth(authService))
			r.Use(httprate.LimitByIP(cfg.RateLimit.Requests, cfg.RateLimit.Window))

			// Auth
			r.Get("/auth/me", authHandler.GetMe)
			r.Post("/auth/logout", authHandler.Logout)

			// Categories
			r.Post("/categories", categoryHandler.Create)
			r.Get("/categories", categoryHandler.List)
			r.Get("/categories/{id}", categoryHandler.Get)
			r.Put("/categories/{id}", categoryHandler.Update)
			r.Delete("/categories/{id}", categoryHandler.Delete)

			// Transactions
			r.Post("/transactions", transactionHandler.Create)
			r.Get("/transactions", transactionHandler.List)
			r.Get("/transactions/date-range", transactionHandler.ListByDateRange)
			r.Get("/transactions/{id}", transactionHandler.Get)
			r.Put("/transactions/{id}", transactionHandler.Update)
			r.Delete("/transactions/{id}", transactionHandler.Delete)

			// Budgets
			r.Post("/budgets", budgetHandler.Create)
			r.Get("/budgets", budgetHandler.List)
			r.Get("/budgets/{id}", budgetHandler.Get)
			r.Put("/budgets/{id}", budgetHandler.Update)
			r.Delete("/budgets/{id}", budgetHandler.Delete)

			// Drafts
			r.Post("/drafts", draftHandler.Create)
			r.Get("/drafts", draftHandler.List)
			r.Get("/drafts/{id}", draftHandler.Get)
			r.Post("/drafts/{id}/confirm", draftHandler.Confirm)
			r.Delete("/drafts/{id}", draftHandler.Delete)

			// Receipt Scanner
			r.Post("/receipts/scan", receiptHandler.ScanReceipt)
		})
	})

	// Setup HTTP server
	srv := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Info("Server starting",
			"port", cfg.Server.Port,
			"environment", cfg.Server.Environment,
		)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("Server failed", "error", err)
			os.Exit(1)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Server shutting down...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("Server forced to shutdown", "error", err)
		os.Exit(1)
	}

	log.Info("Server stopped gracefully")
}
