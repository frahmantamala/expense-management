package payment

import (
	errors "github.com/frahmantamala/expense-management/internal"
	"github.com/frahmantamala/expense-management/internal/core/common/validation"
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

// PaymentRetryRequest represents retry request
type PaymentRetryRequest struct {
	ExternalID string `json:"external_id" validate:"required"`
	ExpenseID  string `json:"expense_id" validate:"required"`
}

// Validate validates the PaymentRetryRequest
func (r *PaymentRetryRequest) Validate() error {
	validator := validation.NewValidator()

	validator.Field("external_id", r.ExternalID).Required()
	validator.Field("expense_id", r.ExpenseID).Required()

	if appErr := validator.Validate(); appErr != nil {
		return appErr
	}
	return nil
}

func (p *PaymentRequest) Validate() error {
	validator := validation.NewValidator()

	validator.Field("amount", p.Amount).Required().MinInt(10001, errors.ErrCodeInvalidAmount)
	validator.Field("external_id", p.ExternalID).Required()

	if appErr := validator.Validate(); appErr != nil {
		return appErr
	}
	return nil
}
