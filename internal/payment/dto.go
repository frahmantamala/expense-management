package payment

import (
	errors "github.com/frahmantamala/expense-management/internal"
	"github.com/frahmantamala/expense-management/internal/core/common/validation"
)

type PaymentRequest struct {
	Amount     int64  `json:"amount"`
	ExternalID string `json:"external_id"`
}

type PaymentResponse struct {
	Data PaymentData `json:"data"`
}

type PaymentData struct {
	ID         string `json:"id"`
	ExternalID string `json:"external_id"`
	Status     string `json:"status"`
}

const (
	PaymentStatusPending = "pending"
	PaymentStatusSuccess = "success"
	PaymentStatusFailed  = "failed"
)

type PaymentRetryRequest struct {
	ExternalID string `json:"external_id" validate:"required"`
	ExpenseID  string `json:"expense_id" validate:"required"`
}

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
