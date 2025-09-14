package payment_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"

	"github.com/frahmantamala/expense-management/internal"
	"github.com/frahmantamala/expense-management/internal/core/datamodel/payment"
	"github.com/frahmantamala/expense-management/internal/expense"
	paymentpkg "github.com/frahmantamala/expense-management/internal/payment"
)

type mockExpenseService struct {
	shouldReturnError error
	shouldCheckPerm   bool
}

func (m *mockExpenseService) RetryPayment(expenseID int64, userPermissions []string) error {
	if m.shouldReturnError != nil {
		return m.shouldReturnError
	}

	if m.shouldCheckPerm {
		hasPermission := false
		for _, perm := range userPermissions {
			if perm == "can_approve" {
				hasPermission = true
				break
			}
		}
		if !hasPermission {
			return expense.ErrUnauthorizedAccess
		}
	}

	return nil
}

type mockPaymentService struct {
	createPaymentError        error
	processPaymentError       error
	retryPaymentError         error
	getPaymentByExpenseError  error
	getPaymentByExternalError error
	updatePaymentStatusError  error
	payment                   *payment.Payment
	response                  *paymentpkg.PaymentResponse
}

func (m *mockPaymentService) CreatePayment(expenseID int64, externalID string, amountIDR int64) (*payment.Payment, error) {
	if m.createPaymentError != nil {
		return nil, m.createPaymentError
	}
	return m.payment, nil
}

func (m *mockPaymentService) ProcessPayment(req *paymentpkg.PaymentRequest) (*paymentpkg.PaymentResponse, error) {
	if m.processPaymentError != nil {
		return nil, m.processPaymentError
	}
	return m.response, nil
}

func (m *mockPaymentService) RetryPayment(req *paymentpkg.PaymentRequest) (*paymentpkg.PaymentResponse, error) {
	if m.retryPaymentError != nil {
		return nil, m.retryPaymentError
	}
	return m.response, nil
}

func (m *mockPaymentService) GetPaymentByExpenseID(expenseID int64) (*payment.Payment, error) {
	if m.getPaymentByExpenseError != nil {
		return nil, m.getPaymentByExpenseError
	}
	return m.payment, nil
}

func (m *mockPaymentService) GetPaymentByExternalID(externalID string) (*payment.Payment, error) {
	if m.getPaymentByExternalError != nil {
		return nil, m.getPaymentByExternalError
	}
	return m.payment, nil
}

func (m *mockPaymentService) UpdatePaymentStatus(paymentID int64, status string, paymentMethod *string, gatewayResponse json.RawMessage, failureReason *string) error {
	return m.updatePaymentStatusError
}

func createTestUser(id int64, permissions []string) *internal.User {
	return &internal.User{
		ID:          id,
		Email:       "test@example.com",
		Permissions: permissions,
	}
}

func createRequestWithUser(method, target string, body []byte, user *internal.User) *http.Request {
	req := httptest.NewRequest(method, target, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	ctx := internal.ContextWithUser(req.Context(), user)
	return req.WithContext(ctx)
}

var _ = ginkgo.Describe("PaymentHandler", func() {
	var (
		handler        *paymentpkg.Handler
		expenseService *mockExpenseService
		paymentService *mockPaymentService
		recorder       *httptest.ResponseRecorder
		logger         *slog.Logger
	)

	ginkgo.BeforeEach(func() {
		expenseService = &mockExpenseService{}
		paymentService = &mockPaymentService{}
		logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
		handler = paymentpkg.NewHandler(expenseService, paymentService, logger)
		recorder = httptest.NewRecorder()
	})

	ginkgo.Context("RetryPayment", func() {
		ginkgo.When("retry request is valid", func() {
			ginkgo.It("should retry payment successfully", func() {
				user := createTestUser(1, []string{"can_approve"})
				reqBody := map[string]interface{}{
					"expense_id":  "123",
					"external_id": "test-external-id",
					"amount":      100.50,
				}
				jsonBody, _ := json.Marshal(reqBody)
				req := createRequestWithUser("POST", "/api/v1/payment/retry", jsonBody, user)

				handler.RetryPayment(recorder, req)

				gomega.Expect(recorder.Code).To(gomega.Equal(http.StatusOK))
				var response map[string]interface{}
				err := json.Unmarshal(recorder.Body.Bytes(), &response)
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				gomega.Expect(response["status"]).To(gomega.Equal("payment retry initiated"))
				gomega.Expect(response["expense_id"]).To(gomega.Equal("123"))
				gomega.Expect(response["external_id"]).To(gomega.Equal("test-external-id"))
			})
		})

		ginkgo.When("request body is invalid JSON", func() {
			ginkgo.It("should return bad request", func() {
				user := createTestUser(1, []string{"can_approve"})
				req := createRequestWithUser("POST", "/api/v1/payment/retry", []byte("invalid json"), user)

				handler.RetryPayment(recorder, req)

				gomega.Expect(recorder.Code).To(gomega.Equal(http.StatusBadRequest))
			})
		})

		ginkgo.Context("when request validation fails", func() {
			ginkgo.It("should return validation error for missing expense_id", func() {
				user := createTestUser(1, []string{"can_approve"})
				reqBody := map[string]interface{}{
					"external_id": "test-external-id",
					"amount":      100.50,
				}
				jsonBody, _ := json.Marshal(reqBody)
				req := createRequestWithUser("POST", "/api/v1/payment/retry", jsonBody, user)

				handler.RetryPayment(recorder, req)

				gomega.Expect(recorder.Code).To(gomega.Equal(http.StatusBadRequest))
			})

			ginkgo.It("should return validation error for missing external_id", func() {
				user := createTestUser(1, []string{"can_approve"})
				reqBody := map[string]interface{}{
					"expense_id": "123",
					"amount":     100.50,
				}
				jsonBody, _ := json.Marshal(reqBody)
				req := createRequestWithUser("POST", "/api/v1/payment/retry", jsonBody, user)

				handler.RetryPayment(recorder, req)

				gomega.Expect(recorder.Code).To(gomega.Equal(http.StatusBadRequest))
			})
		})

		ginkgo.Context("when expense ID is invalid", func() {
			ginkgo.It("should return bad request for non-numeric expense ID", func() {
				user := createTestUser(1, []string{"can_approve"})
				reqBody := map[string]interface{}{
					"expense_id":  "invalid",
					"external_id": "test-external-id",
					"amount":      100.50,
				}
				jsonBody, _ := json.Marshal(reqBody)
				req := createRequestWithUser("POST", "/api/v1/payment/retry", jsonBody, user)

				handler.RetryPayment(recorder, req)

				gomega.Expect(recorder.Code).To(gomega.Equal(http.StatusBadRequest))
			})
		})

		ginkgo.Context("when expense service fails", func() {
			ginkgo.It("should return internal server error", func() {
				user := createTestUser(1, []string{"can_approve"})
				expenseService.shouldReturnError = errors.New("database error")
				reqBody := map[string]interface{}{
					"expense_id":  "123",
					"external_id": "test-external-id",
					"amount":      100.50,
				}
				jsonBody, _ := json.Marshal(reqBody)
				req := createRequestWithUser("POST", "/api/v1/payment/retry", jsonBody, user)

				handler.RetryPayment(recorder, req)

				gomega.Expect(recorder.Code).To(gomega.Equal(http.StatusInternalServerError))
			})
		})

		ginkgo.Context("when user lacks permission", func() {
			ginkgo.It("should return forbidden error", func() {
				user := createTestUser(1, []string{"can_read_expense"})
				expenseService.shouldCheckPerm = true
				reqBody := map[string]interface{}{
					"expense_id":  "123",
					"external_id": "test-external-id",
					"amount":      100.50,
				}
				jsonBody, _ := json.Marshal(reqBody)
				req := createRequestWithUser("POST", "/api/v1/payment/retry", jsonBody, user)

				handler.RetryPayment(recorder, req)

				gomega.Expect(recorder.Code).To(gomega.Equal(http.StatusForbidden))
			})
		})

		ginkgo.Context("when expense is not found", func() {
			ginkgo.It("should return not found error", func() {
				user := createTestUser(1, []string{"can_approve"})
				expenseService.shouldReturnError = expense.ErrExpenseNotFound
				reqBody := map[string]interface{}{
					"expense_id":  "999",
					"external_id": "test-external-id",
					"amount":      100.50,
				}
				jsonBody, _ := json.Marshal(reqBody)
				req := createRequestWithUser("POST", "/api/v1/payment/retry", jsonBody, user)

				handler.RetryPayment(recorder, req)

				gomega.Expect(recorder.Code).To(gomega.Equal(http.StatusNotFound))
			})
		})
	})

	ginkgo.Context("HTTP Method Validation", func() {
		ginkgo.Context("when using wrong HTTP method", func() {
			ginkgo.It("should handle GET requests gracefully", func() {
				user := createTestUser(1, []string{"can_approve"})
				req := createRequestWithUser("GET", "/api/v1/payment/retry", []byte{}, user)

				handler.RetryPayment(recorder, req)

				gomega.Expect(recorder.Code).To(gomega.Equal(http.StatusBadRequest))
			})
		})
	})

	ginkgo.Context("Content-Type Validation", func() {
		ginkgo.Context("when Content-Type is not application/json", func() {
			ginkgo.It("should handle different content types", func() {
				user := createTestUser(1, []string{"can_approve"})
				req := createRequestWithUser("POST", "/api/v1/payment/retry", []byte("text content"), user)
				req.Header.Set("Content-Type", "text/plain")

				handler.RetryPayment(recorder, req)

				gomega.Expect(recorder.Code).To(gomega.Equal(http.StatusBadRequest))
			})
		})
	})

	ginkgo.Context("when user is not in context", func() {
		ginkgo.It("should return unauthorized", func() {
			reqBody := map[string]interface{}{
				"expense_id":  "123",
				"external_id": "test-external-id",
				"amount":      100.50,
			}
			jsonBody, _ := json.Marshal(reqBody)
			req := httptest.NewRequest("POST", "/api/v1/payment/retry", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")

			handler.RetryPayment(recorder, req)

			gomega.Expect(recorder.Code).To(gomega.Equal(http.StatusUnauthorized))
		})
	})
})
