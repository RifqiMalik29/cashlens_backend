package main

import (
	"context"
	"log"
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
	"github.com/rifqimalik/cashlens-backend/internal/repository"
	"github.com/rifqimalik/cashlens-backend/internal/service"
	"github.com/rifqimalik/cashlens-backend/internal/telegram"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Setup context
	ctx := context.Background()

	// Initialize database
	db, err := database.New(ctx, cfg.Database.URL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()
	log.Println("Database connected successfully")

	// Initialize repositories
	userRepo := repository.NewUserRepository(db.Pool)
	categoryRepo := repository.NewCategoryRepository(db.Pool)
	transactionRepo := repository.NewTransactionRepository(db.Pool)
	budgetRepo := repository.NewBudgetRepository(db.Pool)
	draftRepo := repository.NewDraftRepository(db.Pool)
	chatRepo := repository.NewChatLinkRepository(db.Pool)

	// Initialize services
	authService := service.NewAuthService(userRepo, cfg.JWT.Secret, cfg.JWT.Expiration)
	categoryService := service.NewCategoryService(categoryRepo)
	transactionService := service.NewTransactionService(transactionRepo)
	budgetService := service.NewBudgetService(budgetRepo)
	draftService := service.NewDraftService(draftRepo, transactionRepo)

	// Initialize handlers
	healthHandler := handlers.NewHealthHandler(db)
	authHandler := handlers.NewAuthHandler(authService)
	categoryHandler := handlers.NewCategoryHandler(categoryService)
	transactionHandler := handlers.NewTransactionHandler(transactionService)
	budgetHandler := handlers.NewBudgetHandler(budgetService)
	draftHandler := handlers.NewDraftHandler(draftService)
	receiptHandler := handlers.NewReceiptHandler(cfg.GeminiAPI.APIKey)

	// Initialize Telegram Bot
	var botService *telegram.BotService
	if cfg.Telegram.BotToken != "" {
		botService = telegram.NewBotService(cfg.Telegram.BotToken, draftService, userRepo, chatRepo)
		go botService.StartPolling(context.Background())
		log.Println("Telegram bot started")
	} else {
		log.Println("Telegram bot token not configured - skipping bot initialization")
	}

	// Setup router
	r := chi.NewRouter()

	// Global middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(custommiddleware.CORS)
	r.Use(httprate.LimitByIP(cfg.RateLimit.Requests, cfg.RateLimit.Window))

	// Health check (public)
	r.Get("/health", healthHandler.Check)

	// API routes
	r.Route("/api/v1", func(r chi.Router) {
		// Public auth routes
		r.Post("/auth/register", authHandler.Register)
		r.Post("/auth/login", authHandler.Login)

		// Protected routes
		r.Group(func(r chi.Router) {
			r.Use(custommiddleware.Auth(authService))

			// Auth
			r.Get("/auth/me", authHandler.GetMe)

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
		log.Printf("Server starting on port %s (environment: %s)", cfg.Server.Port, cfg.Server.Environment)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Server shutting down...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server stopped gracefully")
}
