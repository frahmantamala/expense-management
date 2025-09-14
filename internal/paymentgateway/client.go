package paymentgateway

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"

	paymentgatewaytypes "github.com/frahmantamala/expense-management/internal/core/datamodel/paymentgateway"
)

type PaymentJob struct {
	ExternalID string
	Amount     int64
	PaymentID  string
}

type Worker struct {
	ID         int
	WorkerPool chan chan PaymentJob
	JobChannel chan PaymentJob
	Logger     *slog.Logger
}

func NewWorker(id int, workerPool chan chan PaymentJob, logger *slog.Logger) *Worker {
	return &Worker{
		ID:         id,
		WorkerPool: workerPool,
		JobChannel: make(chan PaymentJob),
		Logger:     logger,
	}
}

func (w *Worker) Start(ctx context.Context, wg *sync.WaitGroup, processFunc func(PaymentJob)) {
	wg.Add(1)
	go func() {
		defer wg.Done()

		for {

			w.WorkerPool <- w.JobChannel

			select {
			case job := <-w.JobChannel:
				w.Logger.Debug("worker processing job", "worker_id", w.ID, "external_id", job.ExternalID)
				processFunc(job)
			case <-ctx.Done():
				w.Logger.Debug("worker shutting down", "worker_id", w.ID)
				return
			}
		}
	}()
}

type Client struct {
	mockAPIURL     string
	apiKey         string
	webhookURL     string
	paymentTimeout time.Duration
	logger         *slog.Logger

	jobQueue   chan PaymentJob
	workerPool chan chan PaymentJob
	maxWorkers int
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	once       sync.Once
}

type Config struct {
	MockAPIURL     string
	APIKey         string
	WebhookURL     string
	PaymentTimeout time.Duration
	MaxWorkers     int
	JobQueueSize   int
	WorkerPoolSize int
}

func NewClient(config Config, logger *slog.Logger) *Client {
	ctx, cancel := context.WithCancel(context.Background())

	maxWorkers := config.MaxWorkers
	if maxWorkers <= 0 {
		maxWorkers = 10
	}

	jobQueueSize := config.JobQueueSize
	if jobQueueSize <= 0 {
		jobQueueSize = 100
	}

	workerPoolSize := config.WorkerPoolSize
	if workerPoolSize <= 0 {
		workerPoolSize = maxWorkers
	}

	client := &Client{
		mockAPIURL:     config.MockAPIURL,
		apiKey:         config.APIKey,
		webhookURL:     config.WebhookURL,
		paymentTimeout: config.PaymentTimeout,
		logger:         logger,

		maxWorkers: maxWorkers,
		jobQueue:   make(chan PaymentJob, jobQueueSize),
		workerPool: make(chan chan PaymentJob, workerPoolSize),
		ctx:        ctx,
		cancel:     cancel,
	}

	client.startWorkerPool()

	return client
}

func (c *Client) startWorkerPool() {
	c.once.Do(func() {

		for i := 0; i < c.maxWorkers; i++ {
			worker := NewWorker(i, c.workerPool, c.logger)
			worker.Start(c.ctx, &c.wg, c.processPaymentJob)
		}

		go c.dispatch()

		c.logger.Info("payment gateway worker pool started",
			"max_workers", c.maxWorkers,
			"queue_size", cap(c.jobQueue))
	})
}

func (c *Client) dispatch() {
	defer c.wg.Done()
	c.wg.Add(1)

	for {
		select {
		case job := <-c.jobQueue:

			select {
			case jobChannel := <-c.workerPool:

				select {
				case jobChannel <- job:

				case <-c.ctx.Done():
					c.logger.Info("dispatcher shutting down")
					return
				}
			case <-c.ctx.Done():
				c.logger.Info("dispatcher shutting down")
				return
			}
		case <-c.ctx.Done():
			c.logger.Info("dispatcher shutting down")
			return
		}
	}
}

func (c *Client) Shutdown() {
	c.logger.Info("shutting down payment gateway client")
	c.cancel()
	c.wg.Wait()
	c.logger.Info("payment gateway client shutdown complete")
}

func (c *Client) ProcessPayment(req *paymentgatewaytypes.PaymentRequest) (*paymentgatewaytypes.PaymentResponse, error) {
	if err := req.Validate(); err != nil {
		c.logger.Error("payment request validation failed", "error", err)
		return nil, fmt.Errorf("validation error: %w", err)
	}

	c.logger.Info("postman: initiating async payment processing",
		"external_id", req.ExternalID,
		"amount", req.Amount,
		"api_url", c.mockAPIURL)

	paymentID, err := c.initiatePaymentWithPostman(req)
	if err != nil {
		c.logger.Warn("payment initiation failed, will handle in background worker",
			"external_id", req.ExternalID,
			"error", err)

		paymentID = fmt.Sprintf("postman_%s", req.ExternalID)
	}

	resp := &paymentgatewaytypes.PaymentResponse{
		Data: paymentgatewaytypes.PaymentData{
			ID:         paymentID,
			ExternalID: req.ExternalID,
			Status:     paymentgatewaytypes.PaymentStatusPending,
		},
	}

	job := PaymentJob{
		ExternalID: req.ExternalID,
		Amount:     req.Amount,
		PaymentID:  paymentID,
	}

	select {
	case c.jobQueue <- job:
		c.logger.Info("postman: payment job queued for processing",
			"external_id", req.ExternalID,
			"payment_id", resp.Data.ID,
			"queue_length", len(c.jobQueue))
	default:
		c.logger.Warn("postman: job queue full, rejecting payment",
			"external_id", req.ExternalID,
			"queue_capacity", cap(c.jobQueue))
		return nil, fmt.Errorf("payment queue full, please try again later")
	}

	return resp, nil
}

func (c *Client) initiatePaymentWithPostman(req *paymentgatewaytypes.PaymentRequest) (string, error) {

	payload := map[string]interface{}{
		"external_id":  req.ExternalID,
		"amount":       req.Amount,
		"currency":     "IDR",
		"description":  "Payment processing",
		"callback_url": c.webhookURL,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal payment request: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), c.paymentTimeout)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.mockAPIURL+"/payments", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create HTTP request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: c.paymentTimeout}
	resp, err := client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("Postman API returned status %d", resp.StatusCode)
	}

	var apiResponse struct {
		Data struct {
			ID         string `json:"id"`
			ExternalID string `json:"external_id"`
			Status     string `json:"status"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	c.logger.Info("payment initiated with Postman API",
		"payment_id", apiResponse.Data.ID,
		"external_id", apiResponse.Data.ExternalID,
		"status", apiResponse.Data.Status)

	return apiResponse.Data.ID, nil
}

func (c *Client) processPaymentJob(job PaymentJob) {
	c.logger.Info("processing payment job", "external_id", job.ExternalID)

	isFallback := strings.HasPrefix(job.PaymentID, "postman_")

	var status paymentgatewaytypes.PaymentStatus
	var failureReason string

	if isFallback {

		c.logger.Info("retrying payment initiation", "external_id", job.ExternalID)

		req := &paymentgatewaytypes.PaymentRequest{
			Amount:     job.Amount,
			ExternalID: job.ExternalID,
		}

		realPaymentID, err := c.initiatePaymentWithPostman(req)
		if err != nil {

			status = paymentgatewaytypes.PaymentStatusFailed
			failureReason = fmt.Sprintf("Payment initiation failed: %v", err)
			c.logger.Error("payment initiation retry failed",
				"external_id", job.ExternalID,
				"error", err)
		} else {

			job.PaymentID = realPaymentID
			c.logger.Info("payment initiation retry successful",
				"external_id", job.ExternalID,
				"payment_id", realPaymentID)
		}
	}

	if status == "" {

		delay := time.Duration(1+rand.Intn(4)) * time.Second

		select {
		case <-time.After(delay):

		case <-c.ctx.Done():
			c.logger.Info("payment job cancelled", "external_id", job.ExternalID)
			return
		}

		if rand.Float32() < 0.9 {
			status = paymentgatewaytypes.PaymentStatusSuccess
			c.logger.Info("postman simulation: payment successful",
				"external_id", job.ExternalID,
				"delay_seconds", delay.Seconds())
		} else {
			status = paymentgatewaytypes.PaymentStatusFailed
			failureReason = "Insufficient funds"
			c.logger.Info("postman simulation: payment failed",
				"external_id", job.ExternalID,
				"reason", failureReason,
				"delay_seconds", delay.Seconds())
		}
	}

	c.sendCallbackToWebhook(job.ExternalID, status, job.Amount, job.PaymentID, failureReason)
}

func (c *Client) GetPaymentStatus(externalID string) (*paymentgatewaytypes.PaymentResponse, error) {
	c.logger.Info("postman: getting payment status", "external_id", externalID)

	ctx, cancel := context.WithTimeout(context.Background(), c.paymentTimeout)
	defer cancel()

	url := fmt.Sprintf("%s/payments?external_id=%s", c.mockAPIURL, externalID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get payment status: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var apiResponse struct {
		Data paymentgatewaytypes.PaymentData `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &paymentgatewaytypes.PaymentResponse{
		Data: apiResponse.Data,
	}, nil
}

func (c *Client) sendCallbackToWebhook(externalID string, status paymentgatewaytypes.PaymentStatus, amount int64, paymentID string, failureReason string) {

	select {
	case <-c.ctx.Done():
		c.logger.Info("webhook callback cancelled", "external_id", externalID)
		return
	default:

	}

	callbackPayload := map[string]interface{}{
		"external_id":        externalID,
		"status":             string(status),
		"gateway_payment_id": paymentID,
		"amount":             amount,
	}

	if failureReason != "" {
		callbackPayload["failure_reason"] = failureReason
	}

	jsonData, err := json.Marshal(callbackPayload)
	if err != nil {
		c.logger.Error("postman simulation: failed to marshal callback", "error", err)
		return
	}

	c.logger.Info("postman simulation: sending webhook callback",
		"external_id", externalID,
		"status", status,
		"webhook_url", c.webhookURL)

	ctx, cancel := context.WithTimeout(c.ctx, 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", c.webhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		c.logger.Error("postman simulation: failed to create webhook request",
			"error", err,
			"external_id", externalID)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		c.logger.Error("postman simulation: webhook callback failed",
			"error", err,
			"external_id", externalID)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		c.logger.Info("postman simulation: webhook callback successful",
			"external_id", externalID,
			"status_code", resp.StatusCode)
	} else {
		c.logger.Warn("postman simulation: webhook callback error",
			"external_id", externalID,
			"status_code", resp.StatusCode)
	}
}
