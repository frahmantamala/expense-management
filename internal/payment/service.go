package payment

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// PaymentService handles payment operations
type PaymentService struct {
	client  *http.Client
	baseURL string
	logger  *slog.Logger
}

// NewPaymentService creates a new payment service
func NewPaymentService(baseURL string, logger *slog.Logger) *PaymentService {
	return &PaymentService{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: baseURL,
		logger:  logger,
	}
}

// ProcessPayment sends a payment request to the external API
func (s *PaymentService) ProcessPayment(req *PaymentRequest) (*PaymentResponse, error) {
	if err := req.Validate(); err != nil {
		s.logger.Error("payment request validation failed", "error", err)
		return nil, fmt.Errorf("validation error: %w", err)
	}

	// Testing: Simulate payment failure for specific amounts or external IDs
	if s.shouldSimulateFailure(req) {
		s.logger.Info("simulating payment failure for testing",
			"external_id", req.ExternalID,
			"amount", req.Amount)

		return &PaymentResponse{
			Data: PaymentData{
				ID:         fmt.Sprintf("failed-%s", req.ExternalID),
				ExternalID: req.ExternalID,
				Status:     PaymentStatusFailed,
			},
		}, nil
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
		return nil, fmt.Errorf("payment API error: status %d, response: %s", resp.StatusCode, string(respBody))
	}

	// Parse response
	var paymentResp PaymentResponse
	if err := json.Unmarshal(respBody, &paymentResp); err != nil {
		s.logger.Error("failed to unmarshal payment response", "error", err, "response", string(respBody))
		return nil, fmt.Errorf("response unmarshal error: %w", err)
	}

	s.logger.Info("payment processed successfully",
		"payment_id", paymentResp.Data.ID,
		"external_id", paymentResp.Data.ExternalID,
		"status", paymentResp.Data.Status)

	return &paymentResp, nil
}

// simulate fail case
func (s *PaymentService) shouldSimulateFailure(req *PaymentRequest) bool {
	failureAmounts := []int64{
		9999999,
		8888888,
	}

	for _, failAmount := range failureAmounts {
		if req.Amount == failAmount || req.Amount%failAmount == 0 {
			return true
		}
	}

	if strings.Contains(strings.ToLower(req.ExternalID), "fail") {
		return true
	}

	return false
}

func (s *PaymentService) RetryPayment(req *PaymentRequest) (*PaymentResponse, error) {
	s.logger.Info("retrying payment", "external_id", req.ExternalID, "amount", req.Amount)
	return s.ProcessPayment(req)
}
