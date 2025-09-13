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

type RepositoryAPI interface {
	Create(p *payment.Payment) error
	GetByID(id int64) (*payment.Payment, error)
	GetByExternalID(externalID string) (*payment.Payment, error)
	GetByExpenseID(expenseID int64) ([]*payment.Payment, error)
	GetLatestByExpenseID(expenseID int64) (*payment.Payment, error)
	UpdateStatus(id int64, status string, paymentMethod *string, gatewayResponse json.RawMessage, failureReason *string) error
	IncrementRetryCount(id int64) error
	GetFailedPayments(limit int) ([]*payment.Payment, error)
	GetPaymentsByStatus(status string, offset, limit int) ([]*payment.Payment, error)
	GetPaymentStats() (map[string]int64, error)
}

type PaymentService struct {
	client     *http.Client
	baseURL    string
	logger     *slog.Logger
	repository RepositoryAPI
}

func NewPaymentService(baseURL string, logger *slog.Logger, repository RepositoryAPI) *PaymentService {
	return &PaymentService{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL:    baseURL,
		logger:     logger,
		repository: repository,
	}
}

func (s *PaymentService) CreatePayment(expenseID int64, externalID string, amountIDR int64) (*payment.Payment, error) {
	paymentEntity := NewPayment(expenseID, externalID, amountIDR)

	err := s.repository.Create(paymentEntity)
	if err != nil {
		s.logger.Error("failed to create payment record", "error", err, "expense_id", expenseID)
		return nil, fmt.Errorf("failed to create payment record: %w", err)
	}

	s.logger.Info("payment record created", "payment_id", paymentEntity.ID, "expense_id", expenseID, "external_id", externalID)
	return paymentEntity, nil
}

func (s *PaymentService) ProcessPayment(req *PaymentRequest) (*PaymentResponse, error) {
	if err := req.Validate(); err != nil {
		s.logger.Error("payment request validation failed", "error", err)
		return nil, fmt.Errorf("validation error: %w", err)
	}

	paymentRecord, err := s.repository.GetByExternalID(req.ExternalID)
	if err != nil {
		s.logger.Error("payment record not found", "external_id", req.ExternalID, "error", err)
		return nil, fmt.Errorf("payment record not found: %w", err)
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		s.logger.Error("failed to marshal payment request", "error", err)
		return nil, fmt.Errorf("marshal error: %w", err)
	}

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

	resp, err := s.client.Do(httpReq)
	if err != nil {
		s.logger.Error("payment request failed", "error", err, "external_id", req.ExternalID)

		failureReason := err.Error()
		updateErr := s.repository.UpdateStatus(paymentRecord.ID, StatusFailed, nil, nil, &failureReason)
		if updateErr != nil {
			s.logger.Error("failed to update payment status after HTTP error", "error", updateErr, "payment_id", paymentRecord.ID)
		}

		return nil, fmt.Errorf("HTTP request error: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		s.logger.Error("failed to read response body", "error", err)
		return nil, fmt.Errorf("response read error: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		s.logger.Error("payment API returned error",
			"status", resp.StatusCode,
			"response", string(respBody),
			"external_id", req.ExternalID)

		failureReason := fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(respBody))
		updateErr := s.repository.UpdateStatus(paymentRecord.ID, StatusFailed, nil, respBody, &failureReason)
		if updateErr != nil {
			s.logger.Error("failed to update payment status after API error", "error", updateErr, "payment_id", paymentRecord.ID)
		}

		return nil, fmt.Errorf("payment API error: status %d, response: %s", resp.StatusCode, string(respBody))
	}

	var paymentResp PaymentResponse
	if err := json.Unmarshal(respBody, &paymentResp); err != nil {
		s.logger.Error("failed to unmarshal payment response", "error", err, "response", string(respBody))
		return nil, fmt.Errorf("response unmarshal error: %w", err)
	}

	status := MapExternalStatus(paymentResp.Data.Status)

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

func (s *PaymentService) RetryPayment(req *PaymentRequest) (*PaymentResponse, error) {
	s.logger.Info("retrying payment", "external_id", req.ExternalID, "amount", req.Amount)

	payment, err := s.repository.GetByExternalID(req.ExternalID)
	if err != nil {
		s.logger.Error("payment record not found for retry", "external_id", req.ExternalID, "error", err)
		return nil, fmt.Errorf("payment record not found: %w", err)
	}

	err = s.repository.IncrementRetryCount(payment.ID)
	if err != nil {
		s.logger.Error("failed to increment retry count", "error", err, "payment_id", payment.ID)
	}

	return s.ProcessPayment(req)
}

func (s *PaymentService) GetPaymentByExpenseID(expenseID int64) (*payment.Payment, error) {
	return s.repository.GetLatestByExpenseID(expenseID)
}
