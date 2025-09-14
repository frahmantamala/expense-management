package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/frahmantamala/expense-management/internal/core/events"
	"github.com/frahmantamala/expense-management/internal/paymentgateway"
	"github.com/frahmantamala/expense-management/pkg/logger"
	"github.com/spf13/cobra"
)

var workerCmd = &cobra.Command{
	Use:   "worker",
	Short: "Start worker pools for various services",
	Long:  `Start and manage worker pools for different services like payment processing, notifications, file processing, etc.`,
}

// Payment worker command
var paymentWorkerCmd = &cobra.Command{
	Use:   "payment",
	Short: "Start payment gateway worker pool",
	Long:  `Start the payment gateway worker pool for processing payment jobs`,
	Run: func(cmd *cobra.Command, args []string) {
		startPaymentWorker()
	},
}

// Event Bus worker command
var eventWorkerCmd = &cobra.Command{
	Use:   "events",
	Short: "Start event bus worker",
	Long:  `Start the event bus `,
	Run: func(cmd *cobra.Command, args []string) {
		startEventWorker()
	},
}

var (
	maxWorkers     int
	jobQueueSize   int
	workerPoolSize int
	apiURL         string
	apiKey         string
	webhookURL     string
)

func startPaymentWorker() {
	config, err := loadConfig(".")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	logger := logger.LoggerWrapper()

	// Use command line flags if provided, otherwise use config values
	paymentConfig := paymentgateway.Config{
		MockAPIURL:     getStringFlag(apiURL, config.Payment.MockAPIURL),
		APIKey:         getStringFlag(apiKey, config.Payment.APIKey),
		WebhookURL:     getStringFlag(webhookURL, config.Payment.WebhookURL),
		PaymentTimeout: config.Payment.PaymentTimeout,
		MaxWorkers:     getIntFlag(maxWorkers, config.Payment.MaxWorkers),
		JobQueueSize:   getIntFlag(jobQueueSize, config.Payment.JobQueueSize),
		WorkerPoolSize: getIntFlag(workerPoolSize, config.Payment.WorkerPoolSize),
	}

	logger.Info("starting worker payment",
		"max_workers", paymentConfig.MaxWorkers,
		"job_queue_size", paymentConfig.JobQueueSize,
		"worker_pool_size", paymentConfig.WorkerPoolSize,
		"api_url", paymentConfig.MockAPIURL,
		"webhook_url", paymentConfig.WebhookURL)

	// create payment gateway client
	client := paymentgateway.NewClient(paymentConfig, logger)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	logger.Info("payment worker is running. Press Ctrl+C to stop.")

	// wait for shutdown signal
	sig := <-sigChan
	logger.Info("received signal, shutting down payment worker", "signal", sig)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	shutdownDone := make(chan struct{})
	go func() {
		client.Shutdown()
		close(shutdownDone)
	}()

	select {
	case <-shutdownDone:
		logger.Info("payment worker pool shutdown complete")
	case <-ctx.Done():
		logger.Warn("shutdown timeout reached, forcing exit")
	}
}

func startEventWorker() {
	_, err := loadConfig(".")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	logger := logger.LoggerWrapper()

	eventBus := events.NewEventBus(logger)

	eventBus.Subscribe("test.event", func(ctx context.Context, event events.Event) error {
		logger.Info("received test event",
			"event_id", event.EventID(),
			"event_type", event.EventType(),
			"payload", event.Payload())
		return nil
	})

	logger.Info("event bus worker started. Waiting for events...")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	logger.Info("event bus is running. Press Ctrl+C to stop.")

	sig := <-sigChan
	logger.Info("received signal, shutting down event bus", "signal", sig)
	logger.Info("event bus shutdown complete")
}

func getStringFlag(flagValue, configValue string) string {
	if flagValue != "" {
		return flagValue
	}
	return configValue
}

func getIntFlag(flagValue, configValue int) int {
	if flagValue > 0 {
		return flagValue
	}
	return configValue
}

func init() {
	paymentWorkerCmd.Flags().IntVar(&maxWorkers, "max-workers", 0, "Maximum number of workers (overrides config)")
	paymentWorkerCmd.Flags().IntVar(&jobQueueSize, "job-queue-size", 0, "Job queue buffer size (overrides config)")
	paymentWorkerCmd.Flags().IntVar(&workerPoolSize, "worker-pool-size", 0, "Worker pool channel size (overrides config)")
	paymentWorkerCmd.Flags().StringVar(&apiURL, "api-url", "", "Payment gateway API URL (overrides config)")
	paymentWorkerCmd.Flags().StringVar(&apiKey, "api-key", "", "Payment gateway API key (overrides config)")
	paymentWorkerCmd.Flags().StringVar(&webhookURL, "webhook-url", "", "Webhook callback URL (overrides config)")

	workerCmd.AddCommand(paymentWorkerCmd)
	workerCmd.AddCommand(eventWorkerCmd)

	rootCmd.AddCommand(workerCmd)
}
