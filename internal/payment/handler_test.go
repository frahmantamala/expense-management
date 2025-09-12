package payment_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"

	"github.com/frahmantamala/expense-management/internal/auth"
	"github.com/frahmantamala/expense-management/internal/expense"
	"github.com/frahmantamala/expense-management/internal/payment"
	"github.com/frahmantamala/expense-management/internal/transport"
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

func createTestUser(id int64, permissions []string) *auth.User {
	return &auth.User{
		ID:          id,
		Email:       fmt.Sprintf("user%d@example.com", id),
		Permissions: permissions,
	}
}

func createRequestWithUser(method, target string, body []byte, user *auth.User) *http.Request {
	req := httptest.NewRequest(method, target, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	// Add user to context
	ctx := context.WithValue(req.Context(), auth.ContextUserKey, user)
	return req.WithContext(ctx)
}

var _ = ginkgo.Describe("PaymentHandler", func() {
	var (
		handler        *payment.Handler
		expenseService *mockExpenseService
		recorder       *httptest.ResponseRecorder
		logger         *slog.Logger
	)

	ginkgo.BeforeEach(func() {
		expenseService = &mockExpenseService{}
		logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
		handler = payment.NewHandler(expenseService, logger)
		handler.BaseHandler = *transport.NewBaseHandler(logger)
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
				user := createTestUser(1, []string{"can_read_expense"}) // No approval permission
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
