package expense

import (
	"time"

	errors "github.com/frahmantamala/expense-management/internal"
	"github.com/frahmantamala/expense-management/internal/core/common/validation"
)

type CreateExpenseDTO struct {
	AmountIDR       int64     `json:"amount_idr" validate:"required,min=1"`
	Description     string    `json:"description" validate:"required,min=1,max=500"`
	Category        string    `json:"category" validate:"required"`
	ExpenseDate     time.Time `json:"expense_date" validate:"required"`
	ReceiptURL      *string   `json:"receipt_url,omitempty"`
	ReceiptFileName *string   `json:"receipt_filename,omitempty"`
}

func (dto CreateExpenseDTO) Validate() error {
	validator := validation.NewValidator()

	validator.Field("amount_idr", dto.AmountIDR).
		Required().
		MinInt(1, errors.ErrCodeInvalidAmount).
		MinInt(10000, errors.ErrCodeAmountTooLow).
		MaxInt(50000000, errors.ErrCodeAmountTooHigh)

	validator.Field("description", dto.Description).
		Required().
		MinLength(1).
		MaxLength(500)

	validator.Field("category", dto.Category).
		Required()

	validator.Field("expense_date", dto.ExpenseDate).
		NotFuture()

	if appErr := validator.Validate(); appErr != nil {
		return appErr
	}
	return nil
}

type UpdateExpenseStatusDTO struct {
	Status string `json:"status" validate:"required,oneof=approved rejected"`
	Reason string `json:"reason,omitempty"`
}

func (dto UpdateExpenseStatusDTO) Validate() error {
	if dto.Status == "" {
		return errors.NewValidationError("status is required", errors.ErrCodeValidationFailed)
	}
	if dto.Status != "approved" && dto.Status != "rejected" {
		return errors.NewValidationError("status must be either 'approved' or 'rejected'", errors.ErrCodeValidationFailed)
	}
	if dto.Status == "rejected" && dto.Reason == "" {
		return errors.NewValidationError("reason is required when rejecting an expense", errors.ErrCodeValidationFailed)
	}
	return nil
}

type RejectExpenseDTO struct {
	Reason string `json:"reason" validate:"required"`
}

func (dto RejectExpenseDTO) Validate() error {
	if dto.Reason == "" {
		return errors.NewValidationError("reason is required when rejecting an expense", errors.ErrCodeValidationFailed)
	}
	return nil
}

var (
	ErrExpenseNotFound      = errors.ErrExpenseNotFound
	ErrUnauthorizedAccess   = errors.ErrUnauthorizedAccess
	ErrInvalidExpenseStatus = errors.ErrInvalidExpenseStatus
	ErrCannotModifyExpense  = errors.ErrCannotModifyExpense
)
