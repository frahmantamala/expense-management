package payment

import (
	"errors"
	"fmt"
)

// PaymentRequest represents the request payload for payment API
type PaymentRequest struct {
	Amount     int64  `json:"amount"`
	ExternalID string `json:"external_id"`
}

// PaymentResponse represents the response from payment API
type PaymentResponse struct {
	Data PaymentData `json:"data"`
}

// PaymentData represents the payment data in response
type PaymentData struct {
	ID         string `json:"id"`
	ExternalID string `json:"external_id"`
	Status     string `json:"status"`
}

// Payment status constants
const (
	PaymentStatusPending = "pending"
	PaymentStatusSuccess = "success"
	PaymentStatusFailed  = "failed"
)

// Validation
func (p *PaymentRequest) Validate() error {
	if p.Amount <= 0 {
		return errors.New("amount must be greater than 0")
	}
	if p.ExternalID == "" {
		return errors.New("external_id is required")
	}
	return nil
}

// PaymentRetryRequest represents retry request
type PaymentRetryRequest struct {
	ExternalID string `json:"external_id" validate:"required"`
	ExpenseID  string `json:"expense_id" validate:"required"`
}

// Validate validates the PaymentRetryRequest
func (r *PaymentRetryRequest) Validate() error {
	if r.ExternalID == "" {
		return errors.New("external_id is required")
	}
	if r.ExpenseID == "" {
		return errors.New("expense_id is required")
	}
	return nil
}

// CreatePaymentRequest creates a payment request for an expense
func CreatePaymentRequest(expenseID int64, amount int64) *PaymentRequest {
	return &PaymentRequest{
		Amount:     amount,
		ExternalID: fmt.Sprintf("expense-%d-%d", expenseID, amount), // Simple external ID format
	}
}

// CreateFailureTestPaymentRequest creates a payment request that will simulate failure
func CreateFailureTestPaymentRequest(expenseID int64, amount int64) *PaymentRequest {
	return &PaymentRequest{
		Amount:     amount,
		ExternalID: fmt.Sprintf("expense-%d-%d-fail", expenseID, amount), // Contains "fail" to trigger simulation
	}
}
