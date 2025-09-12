package payment

import (
	"fmt"
	"log/slog"

	"github.com/frahmantamala/expense-management/internal/core/datamodel/payment"
)

// ExpensePaymentProcessor handles communication between expense and payment
type ExpensePaymentProcessor struct {
	paymentService PaymentServiceInterface
	logger         *slog.Logger
}

type PaymentServiceInterface interface {
	CreatePayment(expenseID int64, externalID string, amountIDR int64) (*payment.Payment, error)
	ProcessPayment(req *PaymentRequest) (*PaymentResponse, error)
	RetryPayment(req *PaymentRequest) (*PaymentResponse, error)
	GetPaymentByExpenseID(expenseID int64) (*payment.Payment, error)
}

func NewExpensePaymentProcessor(paymentService PaymentServiceInterface, logger *slog.Logger) *ExpensePaymentProcessor {
	return &ExpensePaymentProcessor{
		paymentService: paymentService,
		logger:         logger,
	}
}

func (p *ExpensePaymentProcessor) ProcessPayment(expenseID int64, amount int64) (externalID string, err error) {
	externalID = fmt.Sprintf("exp-%d-%d", expenseID, amount)

	p.logger.Info("initiating payment processing",
		"expense_id", expenseID,
		"amount", amount,
		"external_id", externalID)

	// Create payment record
	payment, err := p.paymentService.CreatePayment(expenseID, externalID, amount)
	if err != nil {
		p.logger.Error("failed to create payment record",
			"error", err,
			"expense_id", expenseID)
		return "", fmt.Errorf("failed to create payment record: %w", err)
	}

	// Process payment via external gateway
	paymentReq := &PaymentRequest{
		Amount:     amount,
		ExternalID: externalID,
	}

	response, err := p.paymentService.ProcessPayment(paymentReq)
	if err != nil {
		p.logger.Error("payment processing failed",
			"error", err,
			"expense_id", expenseID,
			"external_id", externalID,
			"payment_id", payment.ID)
		return externalID, fmt.Errorf("payment processing failed: %w", err)
	}

	// Log result
	if response.Data.Status == PaymentStatusSuccess {
		p.logger.Info("payment processed successfully",
			"expense_id", expenseID,
			"external_id", externalID,
			"gateway_payment_id", response.Data.ID)
	} else {
		p.logger.Warn("payment processing completed with non-success status",
			"expense_id", expenseID,
			"external_id", externalID,
			"status", response.Data.Status)
	}

	return externalID, nil
}

func (p *ExpensePaymentProcessor) RetryPayment(expenseID int64, externalID string) error {
	p.logger.Info("retrying payment",
		"expense_id", expenseID,
		"external_id", externalID)

	// Get payment record to validate
	payment, err := p.paymentService.GetPaymentByExpenseID(expenseID)
	if err != nil {
		p.logger.Error("payment record not found for retry",
			"error", err,
			"expense_id", expenseID)
		return fmt.Errorf("payment record not found: %w", err)
	}

	// Validate payment can be retried
	if !payment.CanRetry() {
		p.logger.Warn("payment cannot be retried",
			"expense_id", expenseID,
			"payment_status", payment.Status,
			"retry_count", payment.RetryCount)
		return fmt.Errorf("payment cannot be retried (status: %s, retries: %d)", payment.Status, payment.RetryCount)
	}

	// Process retry
	paymentReq := &PaymentRequest{
		Amount:     payment.AmountIDR,
		ExternalID: externalID,
	}

	response, err := p.paymentService.RetryPayment(paymentReq)
	if err != nil {
		p.logger.Error("payment retry failed",
			"error", err,
			"expense_id", expenseID,
			"external_id", externalID)
		return fmt.Errorf("payment retry failed: %w", err)
	}

	p.logger.Info("payment retry completed",
		"expense_id", expenseID,
		"external_id", externalID,
		"status", response.Data.Status)

	return nil
}

// GetPaymentStatus gets the current payment status for an expense
func (p *ExpensePaymentProcessor) GetPaymentStatus(expenseID int64) (interface{}, error) {
	paymentRecord, err := p.paymentService.GetPaymentByExpenseID(expenseID)
	if err != nil {
		p.logger.Error("failed to get payment for expense",
			"error", err,
			"expense_id", expenseID)
		return nil, fmt.Errorf("failed to get payment status: %w", err)
	}

	return ToView(paymentRecord), nil
}
