package payment

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/frahmantamala/expense-management/internal/core/events"
)

type EventHandler struct {
	orchestrator *PaymentOrchestrator
	logger       *slog.Logger
}

func NewEventHandler(orchestrator *PaymentOrchestrator, logger *slog.Logger) *EventHandler {
	return &EventHandler{
		orchestrator: orchestrator,
		logger:       logger,
	}
}

func (h *EventHandler) HandleExpenseApproved(ctx context.Context, event events.Event) error {
	expenseEvent, ok := event.(*events.ExpenseApprovedEvent)
	if !ok {
		h.logger.Error("invalid event type for expense approved handler", "event_type", event.EventType())
		return fmt.Errorf("expected ExpenseApprovedEvent, got %T", event)
	}

	h.logger.Info("handling expense approved event for payment processing",
		"expense_id", expenseEvent.ExpenseID,
		"amount", expenseEvent.Amount,
		"user_id", expenseEvent.UserID,
		"event_id", expenseEvent.EventID())

	externalID, err := h.orchestrator.ProcessPayment(expenseEvent.ExpenseID, expenseEvent.Amount)
	if err != nil {
		h.logger.Error("failed to process payment for approved expense",
			"error", err,
			"expense_id", expenseEvent.ExpenseID,
			"amount", expenseEvent.Amount,
			"event_id", expenseEvent.EventID())
		return fmt.Errorf("payment processing failed for expense %d: %w", expenseEvent.ExpenseID, err)
	}

	h.logger.Info("payment processing initiated successfully",
		"expense_id", expenseEvent.ExpenseID,
		"external_id", externalID,
		"amount", expenseEvent.Amount,
		"event_id", expenseEvent.EventID())

	return nil
}

func (h *EventHandler) RegisterEventHandlers(eventBus *events.EventBus) {
	eventBus.Subscribe(events.EventTypeExpenseApproved, h.HandleExpenseApproved)

	h.logger.Info("payment event handlers registered",
		"handlers", []string{events.EventTypeExpenseApproved})
}
