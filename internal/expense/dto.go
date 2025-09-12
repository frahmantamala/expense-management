package expense

import (
	"errors"
	"time"
)

type Expense struct {
	ID              int64      `json:"id" gorm:"primaryKey"`
	UserID          int64      `json:"user_id" gorm:"column:user_id;not null"`
	AmountIDR       int64      `json:"amount_idr" gorm:"column:amount_idr;not null"`
	Description     string     `json:"description" gorm:"not null"`
	Category        string     `json:"category"`
	ReceiptURL      *string    `json:"receipt_url,omitempty" gorm:"column:receipt_url"`
	ReceiptFileName *string    `json:"receipt_filename,omitempty" gorm:"column:receipt_filename"`
	ExpenseStatus   string     `json:"expense_status" gorm:"column:expense_status;default:pending_approval"`
	ExpenseDate     time.Time  `json:"expense_date" gorm:"column:expense_date;type:date"`
	SubmittedAt     time.Time  `json:"submitted_at" gorm:"column:submitted_at;default:now()"`
	ProcessedAt     *time.Time `json:"processed_at,omitempty" gorm:"column:processed_at"`
	CreatedAt       time.Time  `json:"created_at" gorm:"column:created_at;default:now()"`
	UpdatedAt       time.Time  `json:"updated_at" gorm:"column:updated_at;default:now()"`
}

func (Expense) TableName() string {
	return "expenses"
}

type CreateExpenseDTO struct {
	AmountIDR       int64     `json:"amount_idr" validate:"required,min=1"`
	Description     string    `json:"description" validate:"required,min=1,max=500"`
	Category        string    `json:"category" validate:"required"`
	ExpenseDate     time.Time `json:"expense_date" validate:"required"`
	ReceiptURL      *string   `json:"receipt_url,omitempty"`
	ReceiptFileName *string   `json:"receipt_filename,omitempty"`
}

func (dto CreateExpenseDTO) Validate() error {
	if dto.AmountIDR <= 0 {
		return errors.New("amount must be positive")
	}
	if dto.AmountIDR < 10000 {
		return errors.New("amount must be at least 10,000 IDR")
	}
	if dto.AmountIDR > 50000000 {
		return errors.New("amount must not exceed 50,000,000 IDR")
	}
	if dto.Description == "" {
		return errors.New("description is required")
	}
	if len(dto.Description) > 500 {
		return errors.New("description must be less than 500 characters")
	}
	if dto.Category == "" {
		return errors.New("category is required")
	}
	if dto.ExpenseDate.IsZero() {
		return errors.New("expense date is required")
	}

	if dto.ExpenseDate.After(time.Now()) {
		return errors.New("expense date cannot be in the future")
	}
	return nil
}

type UpdateExpenseStatusDTO struct {
	Status string `json:"status" validate:"required,oneof=approved rejected"`
	Reason string `json:"reason,omitempty"`
}

func (dto UpdateExpenseStatusDTO) Validate() error {
	if dto.Status == "" {
		return errors.New("status is required")
	}
	if dto.Status != "approved" && dto.Status != "rejected" {
		return errors.New("status must be either 'approved' or 'rejected'")
	}
	if dto.Status == "rejected" && dto.Reason == "" {
		return errors.New("reason is required when rejecting an expense")
	}
	return nil
}

type RejectExpenseDTO struct {
	Reason string `json:"reason" validate:"required"`
}

func (dto RejectExpenseDTO) Validate() error {
	if dto.Reason == "" {
		return errors.New("reason is required when rejecting an expense")
	}
	return nil
}

const (
	ExpenseStatusPendingApproval = "pending_approval"
	ExpenseStatusApproved        = "approved"
	ExpenseStatusRejected        = "rejected"
	ExpenseStatusAutoApproved    = "auto_approved"
)

const AutoApprovalThreshold = 1000000

var (
	ErrExpenseNotFound      = errors.New("expense not found")
	ErrUnauthorizedAccess   = errors.New("unauthorized access to expense")
	ErrInvalidExpenseStatus = errors.New("invalid expense status for this operation")
	ErrCannotModifyExpense  = errors.New("cannot modify expense in current status")
)
