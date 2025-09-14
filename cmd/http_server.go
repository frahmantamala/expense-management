package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/frahmantamala/expense-management/internal"
	auth "github.com/frahmantamala/expense-management/internal/auth"
	authPostgres "github.com/frahmantamala/expense-management/internal/auth/postgres"
	"github.com/frahmantamala/expense-management/internal/category"
	categoryPostgres "github.com/frahmantamala/expense-management/internal/category/postgres"
	"github.com/frahmantamala/expense-management/internal/core/events"
	"github.com/frahmantamala/expense-management/internal/expense"
	expensePostgres "github.com/frahmantamala/expense-management/internal/expense/postgres"
	"github.com/frahmantamala/expense-management/internal/payment"
	paymentPostgres "github.com/frahmantamala/expense-management/internal/payment/postgres"
	"github.com/frahmantamala/expense-management/internal/paymentgateway"
	"github.com/frahmantamala/expense-management/internal/transport"
	"github.com/frahmantamala/expense-management/internal/transport/rest"
	"github.com/frahmantamala/expense-management/internal/user"
	userPostgres "github.com/frahmantamala/expense-management/internal/user/postgres"
	"github.com/frahmantamala/expense-management/pkg/logger"

	"github.com/go-chi/chi"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/spf13/cobra"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var httpServerCmd = &cobra.Command{
	Use:   "server",
	Short: "Start HTTP server",
	Long:  `Start the HTTP server to handle API requests`,
	Run: func(cmd *cobra.Command, args []string) {
		startHTTPServer()
	},
}

type Dependencies struct {
	Config         *internal.Config
	DB             *gorm.DB
	Router         *chi.Mux
	HealthChecker  *rest.HealthHandler
	Logger         *slog.Logger
	AuthHandler    *auth.Handler
	UserHandler    *user.Handler
	ExpenseHandler *expense.Handler
	PaymentHandler *payment.Handler
}

func startHTTPServer() {
	deps, err := initializeDependencies()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize dependencies: %v\n", err)
		os.Exit(1)
	}

	setupRoutes(deps)

	addr := fmt.Sprintf(":%d", deps.Config.Server.Port)
	slog.Info("Starting HTTP server", "address", addr)

	server := &http.Server{
		Addr:         addr,
		Handler:      deps.Router,
		ReadTimeout:  deps.Config.Server.ReadTimeout,
		WriteTimeout: deps.Config.Server.WriteTimeout,
		IdleTimeout:  deps.Config.Server.IdleTimeout,
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	serverErrChan := make(chan error, 1)
	go func() {
		serverErrChan <- server.ListenAndServe()
	}()

	select {
	case sig := <-sigChan:
		slog.Info("Received signal, shutting down...", "signal", sig)
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			slog.Error("Server shutdown error", "error", err)
		}

		if sqlDB, err := deps.DB.DB(); err == nil {
			if err := sqlDB.Close(); err != nil {
				slog.Error("Database close error", "error", err)
			}
		} else {
			slog.Error("failed to get underlying sql DB for close", "error", err)
		}
	case err := <-serverErrChan:
		if err != nil && err != http.ErrServerClosed {
			slog.Error("Server failed to start", "error", err)
			os.Exit(1)
		}
	}

	slog.Info("Server stopped")
}

func setupRoutes(deps *Dependencies) {

	authRepo := authPostgres.NewRepository(deps.DB)
	tokenGen := auth.NewJWTTokenGenerator(
		deps.Config.Security.SessionSecret,
		deps.Config.Security.SessionSecret,
		deps.Config.Security.AccessTokenDuration,
		deps.Config.Security.RefreshTokenDuration,
	)
	authService := auth.NewService(authRepo, tokenGen, deps.Config.Security.BCryptCost, deps.Logger)
	authHandler := auth.NewHandler(authService)
	deps.AuthHandler = authHandler

	userRepo := userPostgres.NewRepository(deps.DB)
	userSvc := user.NewService(userRepo)
	userHandler := user.NewHandler(userSvc)
	deps.UserHandler = userHandler

	expenseRepo := expensePostgres.NewExpenseRepository(deps.DB)

	eventBus := events.NewEventBus(deps.Logger)

	paymentRepo := paymentPostgres.NewPaymentRepository(deps.DB)

	paymentGateway := paymentgateway.NewClient(
		paymentgateway.Config{
			MockAPIURL:     deps.Config.Payment.MockAPIURL,
			APIKey:         deps.Config.Payment.APIKey,
			WebhookURL:     deps.Config.Payment.WebhookURL,
			PaymentTimeout: deps.Config.Payment.PaymentTimeout,
			MaxWorkers:     deps.Config.Payment.MaxWorkers,
			JobQueueSize:   deps.Config.Payment.JobQueueSize,
			WorkerPoolSize: deps.Config.Payment.WorkerPoolSize,
		},
		deps.Logger,
	)

	paymentService := payment.NewPaymentService(deps.Logger, paymentRepo, paymentGateway)
	paymentOrchestrator := payment.NewPaymentOrchestrator(paymentService, deps.Logger)

	permissionChecker := auth.NewPermissionChecker()

	expenseService := expense.NewService(expenseRepo, paymentOrchestrator, permissionChecker, eventBus, deps.Logger)

	paymentEventHandler := payment.NewEventHandler(paymentOrchestrator, deps.Logger)
	paymentEventHandler.RegisterEventHandlers(eventBus)

	expenseHandler := expense.NewHandler(expenseService)
	deps.ExpenseHandler = expenseHandler

	categoryRepo := categoryPostgres.NewCategoryRepository(deps.DB)
	categoryService := category.NewService(categoryRepo, deps.Logger)
	baseHandler := transport.NewBaseHandler(deps.Logger)
	categoryHandler := category.NewHandler(baseHandler, categoryService)

	paymentHandler := payment.NewHandler(expenseService, paymentService, deps.Logger)
	deps.PaymentHandler = paymentHandler

	webhookHandler := payment.NewWebhookHandler(baseHandler, paymentService, eventBus, deps.Logger)

	sqlDBForRoutes, _ := deps.DB.DB()
	rest.RegisterAllRoutes(deps.Router, sqlDBForRoutes, deps.AuthHandler, authService, deps.UserHandler, deps.ExpenseHandler, categoryHandler, deps.PaymentHandler, webhookHandler, deps.Logger)
}

func initializeDependencies() (*Dependencies, error) {
	config, err := loadConfig(".")
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	db, err := initDB(config.Database)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	router := chi.NewRouter()
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql DB from gorm: %w", err)
	}
	healthChecker := rest.NewHealthHandler(sqlDB)

	return &Dependencies{
		Config:        config,
		Logger:        logger.LoggerWrapper(),
		DB:            db,
		Router:        router,
		HealthChecker: healthChecker,
	}, nil
}

func initDB(cfg internal.DatabaseConfig) (*gorm.DB, error) {
	gormDB, err := gorm.Open(postgres.Open(cfg.Source), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to open gorm db: %w", err)
	}

	sqlDB, err := gormDB.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql DB from gorm: %w", err)
	}

	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)

	if err := sqlDB.Ping(); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return gormDB, nil
}
