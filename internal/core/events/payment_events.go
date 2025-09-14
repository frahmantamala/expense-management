package events

import (
	"time"

	"github.com/google/uuid"
)

const (
	EventTypeExpenseApproved  = "expense.approved"
	EventTypePaymentCompleted = "payment.completed"
	EventTypePaymentFailed    = "payment.failed"
)

type ExpenseApprovedEvent struct {
	BaseEvent
	ExpenseID int64  `json:"expense_id"`
	Amount    int64  `json:"amount"`
	UserID    int64  `json:"user_id"`
	Currency  string `json:"currency"`
}

func NewExpenseApprovedEvent(expenseID, amount, userID int64, currency string) *ExpenseApprovedEvent {
	return &ExpenseApprovedEvent{
		BaseEvent: BaseEvent{
			ID:        uuid.New().String(),
			Type:      EventTypeExpenseApproved,
			Timestamp: time.Now(),
			Data: map[string]interface{}{
				"expense_id": expenseID,
				"amount":     amount,
				"user_id":    userID,
				"currency":   currency,
			},
		},
		ExpenseID: expenseID,
		Amount:    amount,
		UserID:    userID,
		Currency:  currency,
	}
}

type PaymentCompletedEvent struct {
	BaseEvent
	PaymentID        string `json:"payment_id"`
	ExpenseID        int64  `json:"expense_id"`
	ExternalID       string `json:"external_id"`
	Amount           int64  `json:"amount"`
	Status           string `json:"status"`
	GatewayPaymentID string `json:"gateway_payment_id"`
}

func NewPaymentCompletedEvent(paymentID string, expenseID int64, externalID string, amount int64, status string, gatewayPaymentID string) *PaymentCompletedEvent {
	return &PaymentCompletedEvent{
		BaseEvent: BaseEvent{
			ID:        uuid.New().String(),
			Type:      EventTypePaymentCompleted,
			Timestamp: time.Now(),
			Data: map[string]interface{}{
				"payment_id":         paymentID,
				"expense_id":         expenseID,
				"external_id":        externalID,
				"amount":             amount,
				"status":             status,
				"gateway_payment_id": gatewayPaymentID,
			},
		},
		PaymentID:        paymentID,
		ExpenseID:        expenseID,
		ExternalID:       externalID,
		Amount:           amount,
		Status:           status,
		GatewayPaymentID: gatewayPaymentID,
	}
}

type PaymentFailedEvent struct {
	BaseEvent
	PaymentID     string `json:"payment_id"`
	ExpenseID     int64  `json:"expense_id"`
	ExternalID    string `json:"external_id"`
	Amount        int64  `json:"amount"`
	FailureReason string `json:"failure_reason"`
	RetryCount    int    `json:"retry_count"`
}

func NewPaymentFailedEvent(paymentID string, expenseID int64, externalID string, amount int64, failureReason string, retryCount int) *PaymentFailedEvent {
	return &PaymentFailedEvent{
		BaseEvent: BaseEvent{
			ID:        uuid.New().String(),
			Type:      EventTypePaymentFailed,
			Timestamp: time.Now(),
			Data: map[string]interface{}{
				"payment_id":     paymentID,
				"expense_id":     expenseID,
				"external_id":    externalID,
				"amount":         amount,
				"failure_reason": failureReason,
				"retry_count":    retryCount,
			},
		},
		PaymentID:     paymentID,
		ExpenseID:     expenseID,
		ExternalID:    externalID,
		Amount:        amount,
		FailureReason: failureReason,
		RetryCount:    retryCount,
	}
}
