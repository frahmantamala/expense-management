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
	"github.com/frahmantamala/expense-management/internal/transport/rest"
	"github.com/frahmantamala/expense-management/pkg/logger"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/spf13/cobra"
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
	DB            *sqlx.DB
	Router        *chi.Mux
	HealthChecker *rest.HealthHandler
	Logger        *slog.Logger
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
		if err := deps.DB.Close(); err != nil {
			slog.Error("Database close error", "error", err)
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
	// Add middlewares
	deps.Router.Use(middleware.RequestID)
	deps.Router.Use(middleware.Logger)
	deps.Router.Use(middleware.Recoverer)
	// Register health endpoint and other routes
	rest.RegisterAllRoutes(deps.Router, deps.DB.DB)
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
	healthChecker := rest.NewHealthHandler(db.DB)

	return &Dependencies{
		Config:        config,
		Logger:        logger.L(),
		DB:            db,
		Router:        router,
		HealthChecker: healthChecker,
	}, nil
}

// initDB initializes the database connection
func initDB(cfg internal.DatabaseConfig) (*sqlx.DB, error) {
	const driver = "pgx"

	// register traced pgx stdlib driver
	dbConn, err := sqlx.Connect(driver, cfg.Source)
	if err != nil {
		return nil, fmt.Errorf("failed to open traced db connection: %w", err)
	}

	dbConn.SetMaxIdleConns(cfg.MaxIdleConns)
	dbConn.SetMaxOpenConns(cfg.MaxOpenConns)

	// verify connection; close underlying *sql.DB on failure
	if err := dbConn.Ping(); err != nil {
		_ = dbConn.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return sqlx.NewDb(dbConn.DB, driver), dbConn.Ping()
}
