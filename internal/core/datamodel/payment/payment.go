package payment

import (
	"encoding/json"
	"time"
)

// Payment represents the core payment domain entity
type Payment struct {
	ID              int64           `json:"id" gorm:"primaryKey"`
	ExpenseID       int64           `json:"expense_id" gorm:"column:expense_id;not null"`
	ExternalID      string          `json:"external_id" gorm:"column:external_id;not null;uniqueIndex"`
	AmountIDR       int64           `json:"amount_idr" gorm:"column:amount_idr;not null"`
	Status          string          `json:"status" gorm:"column:status;default:pending"`
	PaymentMethod   *string         `json:"payment_method,omitempty" gorm:"column:payment_method"`
	GatewayResponse json.RawMessage `json:"gateway_response,omitempty" gorm:"column:gateway_response;type:jsonb"`
	FailureReason   *string         `json:"failure_reason,omitempty" gorm:"column:failure_reason"`
	RetryCount      int             `json:"retry_count" gorm:"column:retry_count;default:0"`
	ProcessedAt     *time.Time      `json:"processed_at,omitempty" gorm:"column:processed_at"`
	CreatedAt       time.Time       `json:"created_at" gorm:"column:created_at;default:now()"`
	UpdatedAt       time.Time       `json:"updated_at" gorm:"column:updated_at;default:now()"`
}

// TableName returns the table name for GORM
func (Payment) TableName() string {
	return "payments"
}

// Payment status constants
const (
	StatusPending = "pending"
	StatusSuccess = "success"
	StatusFailed  = "failed"
)

// NewPayment creates a new payment domain model
func NewPayment(expenseID int64, externalID string, amountIDR int64) *Payment {
	return &Payment{
		ExpenseID:  expenseID,
		ExternalID: externalID,
		AmountIDR:  amountIDR,
		Status:     StatusPending,
		RetryCount: 0,
	}
}

// MarkAsSuccess marks payment as successful
func (p *Payment) MarkAsSuccess(paymentMethod *string, gatewayResponse json.RawMessage) {
	p.Status = StatusSuccess
	p.PaymentMethod = paymentMethod
	p.GatewayResponse = gatewayResponse
	now := time.Now()
	p.ProcessedAt = &now
}

// MarkAsFailed marks payment as failed
func (p *Payment) MarkAsFailed(failureReason string, gatewayResponse json.RawMessage) {
	p.Status = StatusFailed
	p.FailureReason = &failureReason
	p.GatewayResponse = gatewayResponse
	now := time.Now()
	p.ProcessedAt = &now
}

// IncrementRetryCount increments the retry count
func (p *Payment) IncrementRetryCount() {
	p.RetryCount++
}

// CanRetry checks if payment can be retried
func (p *Payment) CanRetry() bool {
	return p.Status == StatusFailed && p.RetryCount < 3
}

// IsCompleted checks if payment is in a final state
func (p *Payment) IsCompleted() bool {
	return p.Status == StatusSuccess || p.Status == StatusFailed
}

// IsPending checks if payment is still pending
func (p *Payment) IsPending() bool {
	return p.Status == StatusPending
}
