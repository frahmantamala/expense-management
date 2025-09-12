package payment

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/frahmantamala/expense-management/internal/core/datamodel/payment"
)

// PaymentRepository interface for payment database operations
type PaymentRepository interface {
	Create(p *payment.Payment) error
	GetByExternalID(externalID string) (*payment.Payment, error)
	GetLatestByExpenseID(expenseID int64) (*payment.Payment, error)
	UpdateStatus(id int64, status string, paymentMethod *string, gatewayResponse json.RawMessage, failureReason *string) error
	IncrementRetryCount(id int64) error
}

// PaymentService handles payment operations with external API and database
type PaymentService struct {
	client     *http.Client
	baseURL    string
	logger     *slog.Logger
	repository PaymentRepository
}

// NewPaymentService creates a new payment service
func NewPaymentService(baseURL string, logger *slog.Logger, repository PaymentRepository) *PaymentService {
	return &PaymentService{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL:    baseURL,
		logger:     logger,
		repository: repository,
	}
}

// CreatePayment creates a new payment record in database
func (s *PaymentService) CreatePayment(expenseID int64, externalID string, amountIDR int64) (*payment.Payment, error) {
	paymentEntity := &payment.Payment{
		ExpenseID:  expenseID,
		ExternalID: externalID,
		AmountIDR:  amountIDR,
		Status:     payment.StatusPending,
		RetryCount: 0,
	}

	err := s.repository.Create(paymentEntity)
	if err != nil {
		s.logger.Error("failed to create payment record", "error", err, "expense_id", expenseID)
		return nil, fmt.Errorf("failed to create payment record: %w", err)
	}

	s.logger.Info("payment record created", "payment_id", paymentEntity.ID, "expense_id", expenseID, "external_id", externalID)
	return paymentEntity, nil
}

// ProcessPayment sends a payment request to the external API and updates database
func (s *PaymentService) ProcessPayment(req *PaymentRequest) (*PaymentResponse, error) {
	if err := req.Validate(); err != nil {
		s.logger.Error("payment request validation failed", "error", err)
		return nil, fmt.Errorf("validation error: %w", err)
	}

	// Get payment record from database
	paymentRecord, err := s.repository.GetByExternalID(req.ExternalID)
	if err != nil {
		s.logger.Error("payment record not found", "external_id", req.ExternalID, "error", err)
		return nil, fmt.Errorf("payment record not found: %w", err)
	}

	// Prepare request body
	reqBody, err := json.Marshal(req)
	if err != nil {
		s.logger.Error("failed to marshal payment request", "error", err)
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/v1/payments", s.baseURL)
	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		s.logger.Error("failed to create HTTP request", "error", err)
		return nil, fmt.Errorf("request creation error: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	s.logger.Info("sending payment request",
		"url", url,
		"external_id", req.ExternalID,
		"amount", req.Amount)

	// Send request
	resp, err := s.client.Do(httpReq)
	if err != nil {
		s.logger.Error("payment request failed", "error", err, "external_id", req.ExternalID)

		// Update payment status to failed
		failureReason := err.Error()
		updateErr := s.repository.UpdateStatus(paymentRecord.ID, payment.StatusFailed, nil, nil, &failureReason)
		if updateErr != nil {
			s.logger.Error("failed to update payment status after HTTP error", "error", updateErr, "payment_id", paymentRecord.ID)
		}

		return nil, fmt.Errorf("HTTP request error: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		s.logger.Error("failed to read response body", "error", err)
		return nil, fmt.Errorf("response read error: %w", err)
	}

	// Check HTTP status
	if resp.StatusCode != http.StatusOK {
		s.logger.Error("payment API returned error",
			"status", resp.StatusCode,
			"response", string(respBody),
			"external_id", req.ExternalID)

		// Update payment status to failed
		failureReason := fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(respBody))
		updateErr := s.repository.UpdateStatus(paymentRecord.ID, payment.StatusFailed, nil, respBody, &failureReason)
		if updateErr != nil {
			s.logger.Error("failed to update payment status after API error", "error", updateErr, "payment_id", paymentRecord.ID)
		}

		return nil, fmt.Errorf("payment API error: status %d, response: %s", resp.StatusCode, string(respBody))
	}

	// Parse response
	var paymentResp PaymentResponse
	if err := json.Unmarshal(respBody, &paymentResp); err != nil {
		s.logger.Error("failed to unmarshal payment response", "error", err, "response", string(respBody))
		return nil, fmt.Errorf("response unmarshal error: %w", err)
	}

	// Update payment status based on response
	var status string
	switch paymentResp.Data.Status {
	case PaymentStatusSuccess:
		status = payment.StatusSuccess
	case PaymentStatusFailed:
		status = payment.StatusFailed
	default:
		status = payment.StatusPending
	}

	err = s.repository.UpdateStatus(paymentRecord.ID, status, nil, respBody, nil)
	if err != nil {
		s.logger.Error("failed to update payment status", "error", err, "payment_id", paymentRecord.ID)
	}

	s.logger.Info("payment processed successfully",
		"payment_id", paymentResp.Data.ID,
		"external_id", paymentResp.Data.ExternalID,
		"status", paymentResp.Data.Status)

	return &paymentResp, nil
}

// RetryPayment retries a failed payment
func (s *PaymentService) RetryPayment(req *PaymentRequest) (*PaymentResponse, error) {
	s.logger.Info("retrying payment", "external_id", req.ExternalID, "amount", req.Amount)

	// Get payment record and increment retry count
	payment, err := s.repository.GetByExternalID(req.ExternalID)
	if err != nil {
		s.logger.Error("payment record not found for retry", "external_id", req.ExternalID, "error", err)
		return nil, fmt.Errorf("payment record not found: %w", err)
	}

	// Increment retry count
	err = s.repository.IncrementRetryCount(payment.ID)
	if err != nil {
		s.logger.Error("failed to increment retry count", "error", err, "payment_id", payment.ID)
	}

	// Process payment (this will handle status updates)
	return s.ProcessPayment(req)
}

// GetPaymentByExpenseID gets the latest payment for an expense
func (s *PaymentService) GetPaymentByExpenseID(expenseID int64) (*payment.Payment, error) {
	return s.repository.GetLatestByExpenseID(expenseID)
}
