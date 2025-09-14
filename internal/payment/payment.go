package payment

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/frahmantamala/expense-management/internal/core/datamodel/payment"
)

// Payment errors
var (
	ErrExternalIDAlreadyExists = errors.New("external_id already exists")
	ErrPaymentNotFound         = errors.New("payment not found")
	ErrInvalidPaymentStatus    = errors.New("invalid payment status")
)

type ServiceAPI interface {
	CreatePayment(expenseID int64, externalID string, amountIDR int64) (*payment.Payment, error)
	ProcessPayment(req *PaymentRequest) (*PaymentResponse, error)
	RetryPayment(req *PaymentRequest) (*PaymentResponse, error)
	GetPaymentByExpenseID(expenseID int64) (*payment.Payment, error)
	GetPaymentByExternalID(externalID string) (*payment.Payment, error)
	UpdatePaymentStatus(paymentID int64, status string, paymentMethod *string, gatewayResponse json.RawMessage, failureReason *string) error
}

type PaymentView struct {
	ID              int64           `json:"id"`
	ExpenseID       int64           `json:"expense_id"`
	ExternalID      string          `json:"external_id"`
	AmountIDR       int64           `json:"amount_idr"`
	Status          string          `json:"status"`
	PaymentMethod   *string         `json:"payment_method,omitempty"`
	GatewayResponse json.RawMessage `json:"gateway_response,omitempty"`
	FailureReason   *string         `json:"failure_reason,omitempty"`
	RetryCount      int             `json:"retry_count"`
	ProcessedAt     *time.Time      `json:"processed_at,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

type PaymentSummaryView struct {
	ID         int64     `json:"id"`
	ExternalID string    `json:"external_id"`
	AmountIDR  int64     `json:"amount_idr"`
	Status     string    `json:"status"`
	RetryCount int       `json:"retry_count"`
	CreatedAt  time.Time `json:"created_at"`
}

const (
	StatusPending = "pending"
	StatusSuccess = "success"
	StatusFailed  = "failed"
)

func NewPayment(expenseID int64, externalID string, amountIDR int64) *payment.Payment {
	now := time.Now()
	return &payment.Payment{
		ExpenseID:  expenseID,
		ExternalID: externalID,
		AmountIDR:  amountIDR,
		Status:     StatusPending,
		RetryCount: 0,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

func MarkAsSuccess(p *payment.Payment, paymentMethod *string, gatewayResponse json.RawMessage) {
	p.Status = StatusSuccess
	p.PaymentMethod = paymentMethod
	p.GatewayResponse = gatewayResponse
	now := time.Now()
	p.ProcessedAt = &now
	p.UpdatedAt = now
}

func MarkAsFailed(p *payment.Payment, failureReason string, gatewayResponse json.RawMessage) {
	p.Status = StatusFailed
	p.FailureReason = &failureReason
	p.GatewayResponse = gatewayResponse
	now := time.Now()
	p.ProcessedAt = &now
	p.UpdatedAt = now
}

func IncrementRetryCount(p *payment.Payment) {
	p.RetryCount++
	p.UpdatedAt = time.Now()
}

func CanRetry(p *payment.Payment) bool {
	return p.Status == StatusFailed && p.RetryCount < 3
}

func IsCompleted(p *payment.Payment) bool {
	return p.Status == StatusSuccess || p.Status == StatusFailed
}

func IsPending(p *payment.Payment) bool {
	return p.Status == StatusPending
}

func MapExternalStatus(externalStatus string) string {
	switch strings.ToLower(externalStatus) {
	case "success", "completed", "paid":
		return StatusSuccess
	case "failed", "cancelled", "declined":
		return StatusFailed
	default:
		return StatusPending
	}
}

func ToView(p *payment.Payment) *PaymentView {
	return &PaymentView{
		ID:              p.ID,
		ExpenseID:       p.ExpenseID,
		ExternalID:      p.ExternalID,
		AmountIDR:       p.AmountIDR,
		Status:          p.Status,
		PaymentMethod:   p.PaymentMethod,
		GatewayResponse: p.GatewayResponse,
		FailureReason:   p.FailureReason,
		RetryCount:      p.RetryCount,
		ProcessedAt:     p.ProcessedAt,
		CreatedAt:       p.CreatedAt,
		UpdatedAt:       p.UpdatedAt,
	}
}

func ToSummaryView(p *payment.Payment) *PaymentSummaryView {
	return &PaymentSummaryView{
		ID:         p.ID,
		ExternalID: p.ExternalID,
		AmountIDR:  p.AmountIDR,
		Status:     p.Status,
		RetryCount: p.RetryCount,
		CreatedAt:  p.CreatedAt,
	}
}
