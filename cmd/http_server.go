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
	"github.com/frahmantamala/expense-management/internal/transport/rest"
	"github.com/frahmantamala/expense-management/pkg/logger"

	cors "github.com/frahmantamala/expense-management/internal/transport/middleware"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
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
	Config        *internal.Config
	DB            *gorm.DB
	Router        *chi.Mux
	HealthChecker *rest.HealthHandler
	Logger        *slog.Logger
	AuthHandler   *auth.Handler
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

	// Signal handling for graceful shutdown
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
		// close underlying sql.DB from gorm
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
	// Add middlewares: defensive recover and CORS first
	deps.Router.Use(cors.CORS)
	deps.Router.Use(middleware.RequestID)
	deps.Router.Use(middleware.Logger)
	deps.Router.Use(middleware.Recoverer)

	// Use GORM-based auth repo
	authRepo := authPostgres.NewRepository(deps.DB)

	// Token secrets come from Security config
	tokenGen := auth.NewJWTTokenGenerator(
		deps.Config.Security.SessionSecret,
		deps.Config.Security.SessionSecret,
	)

	authService := auth.NewService(authRepo, tokenGen)
	authHandler := auth.NewHandler(authService)

	// Set auth handler in dependencies
	deps.AuthHandler = authHandler

	// Register health endpoint and other routes. Pass underlying *sql.DB to the router.
	sqlDBForRoutes, _ := deps.DB.DB()
	rest.RegisterAllRoutes(deps.Router, sqlDBForRoutes, deps.AuthHandler)
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
	// get underlying *sql.DB for health checks
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

// initDB initializes the database connection
func initDB(cfg internal.DatabaseConfig) (*gorm.DB, error) {
	// open gorm db using postgres driver
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

	// verify connection; close underlying *sql.DB on failure
	if err := sqlDB.Ping(); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return gormDB, nil
}
