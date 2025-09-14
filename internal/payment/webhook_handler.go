package payment

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/frahmantamala/expense-management/internal/core/events"
	"github.com/frahmantamala/expense-management/internal/transport"
)

type WebhookHandler struct {
	*transport.BaseHandler
	paymentService ServiceAPI
	eventBus       *events.EventBus
	logger         *slog.Logger
}

func NewWebhookHandler(baseHandler *transport.BaseHandler, paymentService ServiceAPI, eventBus *events.EventBus, logger *slog.Logger) *WebhookHandler {
	return &WebhookHandler{
		BaseHandler:    baseHandler,
		paymentService: paymentService,
		eventBus:       eventBus,
		logger:         logger,
	}
}

type PaymentCallbackRequest struct {
	ExternalID       string `json:"external_id"`
	Status           string `json:"status"`
	GatewayPaymentID string `json:"gateway_payment_id"`
	Amount           int64  `json:"amount"`
	FailureReason    string `json:"failure_reason,omitempty"`
}

type PaymentCallbackResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

func (h *WebhookHandler) HandlePaymentCallback(w http.ResponseWriter, r *http.Request) {
	var req PaymentCallbackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("invalid payment callback request", "error", err)
		h.WriteErrorResponse(w, http.StatusBadRequest, "invalid request body")
		return
	}

	h.logger.Info("received payment callback",
		"external_id", req.ExternalID,
		"status", req.Status,
		"gateway_payment_id", req.GatewayPaymentID,
		"amount", req.Amount)

	if req.ExternalID == "" {
		h.logger.Error("payment callback missing external_id")
		h.WriteErrorResponse(w, http.StatusBadRequest, "external_id is required")
		return
	}

	if req.Status == "" {
		h.logger.Error("payment callback missing status", "external_id", req.ExternalID)
		h.WriteErrorResponse(w, http.StatusBadRequest, "status is required")
		return
	}

	err := h.processPaymentCallback(&req)
	if err != nil {
		h.logger.Error("failed to process payment callback",
			"error", err,
			"external_id", req.ExternalID,
			"status", req.Status)
		h.WriteErrorResponse(w, http.StatusInternalServerError, "failed to process payment callback")
		return
	}

	response := PaymentCallbackResponse{
		Status:  "success",
		Message: "callback processed successfully",
	}

	h.logger.Info("payment callback processed successfully",
		"external_id", req.ExternalID,
		"status", req.Status)

	h.WriteJSON(w, http.StatusOK, response)
}

func (h *WebhookHandler) processPaymentCallback(req *PaymentCallbackRequest) error {

	payment, err := h.paymentService.GetPaymentByExternalID(req.ExternalID)
	if err != nil {
		return fmt.Errorf("payment not found for external_id %s: %w", req.ExternalID, err)
	}

	h.logger.Info("processing payment callback for payment record",
		"payment_id", payment.ID,
		"expense_id", payment.ExpenseID,
		"external_id", req.ExternalID,
		"current_status", payment.Status,
		"new_status", req.Status)

	internalStatus := MapExternalStatus(req.Status)

	callbackData := map[string]interface{}{
		"gateway_payment_id": req.GatewayPaymentID,
		"gateway_status":     req.Status,
		"amount":             req.Amount,
		"callback_time":      time.Now().UTC(),
	}

	if req.FailureReason != "" {
		callbackData["failure_reason"] = req.FailureReason
	}

	callbackJSON, _ := json.Marshal(callbackData)

	var failureReason *string
	if req.FailureReason != "" {
		failureReason = &req.FailureReason
	}

	err = h.paymentService.UpdatePaymentStatus(payment.ID, internalStatus, nil, callbackJSON, failureReason)
	if err != nil {
		return fmt.Errorf("failed to update payment status: %w", err)
	}

	if internalStatus == StatusSuccess {
		event := events.NewPaymentCompletedEvent(
			fmt.Sprintf("%d", payment.ID),
			payment.ExpenseID,
			req.ExternalID,
			req.Amount,
			internalStatus,
			req.GatewayPaymentID,
		)
		h.eventBus.Publish(context.Background(), event)
		h.logger.Info("published payment completed event", "event_id", event.EventID())
	} else if internalStatus == StatusFailed {
		event := events.NewPaymentFailedEvent(
			fmt.Sprintf("%d", payment.ID),
			payment.ExpenseID,
			req.ExternalID,
			req.Amount,
			req.FailureReason,
			payment.RetryCount,
		)
		h.eventBus.Publish(context.Background(), event)
		h.logger.Info("published payment failed event", "event_id", event.EventID())
	}

	h.logger.Info("payment status updated successfully",
		"payment_id", payment.ID,
		"external_id", req.ExternalID,
		"old_status", payment.Status,
		"new_status", internalStatus)

	return nil
}

func (h *WebhookHandler) WriteErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	response := map[string]string{
		"error": message,
	}
	h.WriteJSON(w, statusCode, response)
}
