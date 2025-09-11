package payment

import (
	"fmt"
	"log/slog"
)

type ExpensePaymentProcessor struct {
	paymentService *PaymentService
	logger         *slog.Logger
}

func NewExpensePaymentProcessor(paymentService *PaymentService, logger *slog.Logger) *ExpensePaymentProcessor {
	return &ExpensePaymentProcessor{
		paymentService: paymentService,
		logger:         logger,
	}
}

func (p *ExpensePaymentProcessor) ProcessPayment(expenseID int64, amount int64) (paymentID, externalID string, err error) {
	paymentReq := CreatePaymentRequest(expenseID, amount)

	p.logger.Info("processing payment for expense",
		"expense_id", expenseID,
		"amount", amount,
		"external_id", paymentReq.ExternalID)

	// Call payment service
	response, err := p.paymentService.ProcessPayment(paymentReq)
	if err != nil {
		p.logger.Error("payment processing failed",
			"error", err,
			"expense_id", expenseID,
			"external_id", paymentReq.ExternalID)
		return "", paymentReq.ExternalID, fmt.Errorf("payment failed: %w", err)
	}

	// Check payment status
	if response.Data.Status != PaymentStatusSuccess {
		p.logger.Error("payment not successful",
			"status", response.Data.Status,
			"expense_id", expenseID,
			"payment_id", response.Data.ID)
		return response.Data.ID, response.Data.ExternalID, fmt.Errorf("payment failed with status: %s", response.Data.Status)
	}

	p.logger.Info("payment processed successfully",
		"expense_id", expenseID,
		"payment_id", response.Data.ID,
		"external_id", response.Data.ExternalID,
		"status", response.Data.Status)

	return response.Data.ID, response.Data.ExternalID, nil
}

func (p *ExpensePaymentProcessor) RetryPayment(externalID string, amount int64) (paymentID string, err error) {
	p.logger.Info("retrying payment", "external_id", externalID, "amount", amount)

	paymentReq := &PaymentRequest{
		Amount:     amount,
		ExternalID: externalID,
	}

	// Call payment service
	response, err := p.paymentService.RetryPayment(paymentReq)
	if err != nil {
		p.logger.Error("payment retry failed",
			"error", err,
			"external_id", externalID)
		return "", fmt.Errorf("payment retry failed: %w", err)
	}

	// Check payment status
	if response.Data.Status != PaymentStatusSuccess {
		p.logger.Error("payment retry not successful",
			"status", response.Data.Status,
			"payment_id", response.Data.ID,
			"external_id", externalID)
		return response.Data.ID, fmt.Errorf("payment retry failed with status: %s", response.Data.Status)
	}

	p.logger.Info("payment retry successful",
		"payment_id", response.Data.ID,
		"external_id", response.Data.ExternalID,
		"status", response.Data.Status)

	return response.Data.ID, nil
}
