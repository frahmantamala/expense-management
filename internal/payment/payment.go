package payment

import (
	"time"

	"github.com/frahmantamala/expense-management/internal/core/datamodel/payment"
)

// PaymentView represents API-ready view model for payment
type PaymentView struct {
	ID            int64      `json:"id"`
	ExpenseID     int64      `json:"expense_id"`
	ExternalID    string     `json:"external_id"`
	AmountIDR     int64      `json:"amount_idr"`
	Status        string     `json:"status"`
	PaymentMethod *string    `json:"payment_method,omitempty"`
	FailureReason *string    `json:"failure_reason,omitempty"`
	RetryCount    int        `json:"retry_count"`
	ProcessedAt   *time.Time `json:"processed_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// PaymentSummaryView represents simplified view for listing
type PaymentSummaryView struct {
	ID         int64     `json:"id"`
	ExternalID string    `json:"external_id"`
	AmountIDR  int64     `json:"amount_idr"`
	Status     string    `json:"status"`
	RetryCount int       `json:"retry_count"`
	CreatedAt  time.Time `json:"created_at"`
}

// ToView converts payment domain model to API view model
func ToView(p *payment.Payment) *PaymentView {
	return &PaymentView{
		ID:            p.ID,
		ExpenseID:     p.ExpenseID,
		ExternalID:    p.ExternalID,
		AmountIDR:     p.AmountIDR,
		Status:        p.Status,
		PaymentMethod: p.PaymentMethod,
		FailureReason: p.FailureReason,
		RetryCount:    p.RetryCount,
		ProcessedAt:   p.ProcessedAt,
		CreatedAt:     p.CreatedAt,
		UpdatedAt:     p.UpdatedAt,
	}
}

// ToSummaryView converts to summary view
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
