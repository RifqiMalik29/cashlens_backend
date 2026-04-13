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
	"github.com/rifqimalik/cashlens-backend/internal/pkg/xendit"
)

// getEnv returns environment variable or default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

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

	// Run database migrations
	if err := database.Migrate(cfg.Database.URL, "migrations"); err != nil {
		log.Error("Failed to run migrations", "error", err)
		os.Exit(1)
	}
	log.Info("Database migrations applied")

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
	quotaRepo := repository.NewQuotaRepository(db.Pool)
	subEventRepo := repository.NewSubscriptionEventRepository(db.Pool)
	pendingInvoiceRepo := repository.NewPendingInvoiceRepository(db.Pool)
	winBackRepo := repository.NewWinBackRepository(db.Pool)

	// Initialize Xendit client
	xenditClient := xendit.NewXenditClient(cfg.Payment.XenditSecretKey)

	// Initialize services
	categorySeedingService := service.NewCategorySeedingService(categoryRepo)
	quotaService := service.NewQuotaService(quotaRepo, userRepo)
	subscriptionService := service.NewSubscriptionService(
		userRepo,
		subEventRepo,
		pendingInvoiceRepo,
		xenditClient,
		cfg.Payment.XenditWebhookToken,
	)
	winBackService := service.NewWinBackService(winBackRepo, chatRepo, cfg.Telegram.BotToken)
	
	authService := service.NewAuthService(userRepo, categorySeedingService, chatRepo, cfg.JWT.Secret, cfg.JWT.Expiration)
	refreshTokenService := service.NewRefreshTokenService(
		refreshTokenRepo,
		userRepo,
		cfg.JWT.Secret,
		cfg.JWT.Expiration,
		cfg.JWT.RefreshExpiration,
		cfg.JWT.MaxReuseWindow,
	)
	categoryService := service.NewCategoryService(categoryRepo)
	transactionService := service.NewTransactionService(transactionRepo, quotaService)
	budgetService := service.NewBudgetService(budgetRepo)
	draftService := service.NewDraftService(draftRepo, transactionRepo)

	// Initialize handlers
	healthHandler := handlers.NewHealthHandler(db)
	authHandler := handlers.NewAuthHandler(authService, refreshTokenService)
	categoryHandler := handlers.NewCategoryHandler(categoryService)
	transactionHandler := handlers.NewTransactionHandler(transactionService)
	budgetHandler := handlers.NewBudgetHandler(budgetService)
	draftHandler := handlers.NewDraftHandler(draftService)
	receiptHandler := handlers.NewReceiptHandler(cfg.GeminiAPI.APIKey, cfg.GeminiAPI.ScanningModel, quotaService, categoryRepo)
	subscriptionHandler := handlers.NewSubscriptionHandler(
		quotaService,
		userRepo,
		subscriptionService,
		xenditClient,
		cfg.Payment.XenditWebhookToken,
		getEnv("XENDIT_SUCCESS_URL", "cashlens://payment/success"),
		getEnv("XENDIT_FAILURE_URL", "cashlens://payment/failed"),
	)

	// Initialize Telegram Bot
	var botService *telegram.BotService
	if cfg.Telegram.BotToken != "" {
		botService = telegram.NewBotService(
			cfg.Telegram.BotToken,
			cfg.GeminiAPI.APIKey,
			cfg.GeminiAPI.TelegramModel,
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

	// Start win-back campaign scheduler (runs daily at 9 AM)
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()

		// Run once on startup after 1 hour delay
		time.Sleep(1 * time.Hour)

		for range ticker.C {
			count, err := winBackService.RunWinBackCampaign(context.Background())
			if err != nil {
				log.Error("Win-back campaign failed", "error", err)
			} else if count > 0 {
				log.Info("Win-back campaign completed", "users_sent", count)
			}
		}
	}()
	log.Info("Win-back campaign scheduler started")

	// Start expired invoice cleanup (runs daily)
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			count, err := pendingInvoiceRepo.ExpireStale(context.Background())
			if err != nil {
				log.Error("Expired invoice cleanup failed", "error", err)
			} else if count > 0 {
				log.Info("Expired invoices cleaned up", "count", count)
			}
		}
	}()

	// Setup router
	r := chi.NewRouter()

	// Global middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(custommiddleware.StructuredLogger)
	r.Use(middleware.Recoverer)
	r.Use(custommiddleware.CORS(custommiddleware.CORSConfig{
		AllowedOrigins: []string{"*"}, // Add your production domains here
		Environment:    cfg.Server.Environment,
	}))
	r.Use(custommiddleware.SecurityHeaders(custommiddleware.SecurityHeadersConfig{
		Environment: cfg.Server.Environment,
	}))

	// Warn if production CORS is not configured
	if cfg.Server.Environment == "production" && len([]string{}) == 0 {
		log.Warn("WARNING: Production CORS allowed origins is empty — all browser requests will be 403'd. Update AllowedOrigins in cmd/server/main.go")
	}

	// Health check (public, no rate limiting)
	r.Get("/health", healthHandler.Check)

	// API routes
	r.Route("/api/v1", func(r chi.Router) {
		// Public auth routes (stricter rate limiting)
		r.Group(func(r chi.Router) {
			r.Use(custommiddleware.MaxBodyLimit(1 << 20)) // 1MB limit
			r.Use(httprate.LimitByIP(cfg.RateLimit.AuthRequests, cfg.RateLimit.AuthWindow))
			r.Post("/auth/register", authHandler.Register)
			r.Post("/auth/login", authHandler.Login)
			r.Post("/auth/refresh", authHandler.Refresh)
		})

		// Protected routes (standard rate limiting)
		r.Group(func(r chi.Router) {
			r.Use(custommiddleware.Auth(authService))
			r.Use(custommiddleware.SubscriptionExpiryCheck(userRepo, subEventRepo))
			r.Use(httprate.LimitByIP(cfg.RateLimit.Requests, cfg.RateLimit.Window))
			r.Use(custommiddleware.MaxBodyLimit(1 << 20)) // 1MB limit for JSON requests

			// Auth
			r.Get("/auth/me", authHandler.GetMe)
			r.Get("/auth/telegram/status", authHandler.GetTelegramStatus)
			r.Delete("/auth/telegram/status", authHandler.UnlinkTelegram)
			r.Post("/auth/logout", authHandler.Logout)

			// Subscription
			r.Get("/subscription", subscriptionHandler.GetSubscriptionStatus)
			r.Post("/subscription/verify", subscriptionHandler.VerifyPayment)
			r.Post("/payments/create-invoice", subscriptionHandler.CreateInvoice)

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
		})

		// Receipt Scanner (separate group with higher body limit for image uploads)
		r.Group(func(r chi.Router) {
			r.Use(custommiddleware.Auth(authService))
			r.Use(custommiddleware.MaxBodyLimit(10 << 20)) // 10MB for image uploads
			r.Post("/receipts/scan", receiptHandler.ScanReceipt)
		})

		// Webhook routes (rate-limited; full signature verification required before enabling)
		r.Group(func(r chi.Router) {
			r.Use(httprate.LimitByIP(cfg.RateLimit.AuthRequests, cfg.RateLimit.AuthWindow))
			r.Post("/webhooks/payment", subscriptionHandler.PaymentWebhook)
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
