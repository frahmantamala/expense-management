package payment_test

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/frahmantamala/expense-management/internal/core/datamodel/payment"
	paymentPkg "github.com/frahmantamala/expense-management/internal/payment"
	"github.com/frahmantamala/expense-management/internal/paymentgateway"
)

type mockPaymentRepository struct {
	payments            map[string]*payment.Payment
	paymentsByExpense   map[int64]*payment.Payment
	createError         error
	getError            error
	updateStatusError   error
	incrementRetryError error
}

func newMockPaymentRepository() *mockPaymentRepository {
	return &mockPaymentRepository{
		payments:          make(map[string]*payment.Payment),
		paymentsByExpense: make(map[int64]*payment.Payment),
	}
}

func (m *mockPaymentRepository) Create(p *payment.Payment) error {
	if m.createError != nil {
		return m.createError
	}
	p.ID = int64(len(m.payments) + 1)
	p.CreatedAt = time.Now()
	p.UpdatedAt = time.Now()
	m.payments[p.ExternalID] = p
	m.paymentsByExpense[p.ExpenseID] = p
	return nil
}

func (m *mockPaymentRepository) GetByExternalID(externalID string) (*payment.Payment, error) {
	if m.getError != nil {
		return nil, m.getError
	}
	p, exists := m.payments[externalID]
	if !exists {
		return nil, errors.New("payment not found")
	}
	return p, nil
}

func (m *mockPaymentRepository) GetLatestByExpenseID(expenseID int64) (*payment.Payment, error) {
	if m.getError != nil {
		return nil, m.getError
	}
	p, exists := m.paymentsByExpense[expenseID]
	if !exists {
		return nil, errors.New("payment not found")
	}
	return p, nil
}

func (m *mockPaymentRepository) UpdateStatus(id int64, status string, paymentMethod *string, gatewayResponse json.RawMessage, failureReason *string) error {
	if m.updateStatusError != nil {
		return m.updateStatusError
	}

	for _, p := range m.payments {
		if p.ID == id {
			p.Status = status
			p.PaymentMethod = paymentMethod
			p.GatewayResponse = gatewayResponse
			p.FailureReason = failureReason
			now := time.Now()
			p.ProcessedAt = &now
			p.UpdatedAt = now
			break
		}
	}
	return nil
}

func (m *mockPaymentRepository) IncrementRetryCount(id int64) error {
	if m.incrementRetryError != nil {
		return m.incrementRetryError
	}
	for _, p := range m.payments {
		if p.ID == id {
			p.RetryCount++
			break
		}
	}
	return nil
}

func (m *mockPaymentRepository) GetByID(id int64) (*payment.Payment, error) {
	if m.getError != nil {
		return nil, m.getError
	}
	for _, p := range m.payments {
		if p.ID == id {
			return p, nil
		}
	}
	return nil, errors.New("payment not found")
}

func (m *mockPaymentRepository) GetByExpenseID(expenseID int64) ([]*payment.Payment, error) {
	if m.getError != nil {
		return nil, m.getError
	}
	var payments []*payment.Payment
	for _, p := range m.payments {
		if p.ExpenseID == expenseID {
			payments = append(payments, p)
		}
	}
	return payments, nil
}

var _ = Describe("PaymentService", func() {
	var (
		paymentService *paymentPkg.PaymentService
		mockRepo       *mockPaymentRepository
		mockServer     *httptest.Server
		logger         *slog.Logger
	)

	BeforeEach(func() {
		mockRepo = newMockPaymentRepository()
		logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

		mockServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			if r.Method == "POST" && r.URL.Path == "/payments" {
				response := map[string]interface{}{
					"data": map[string]interface{}{
						"id":          "mock-payment-id-12345",
						"external_id": "test-external-id",
						"status":      "pending",
					},
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(response)
				return
			}

			if r.Method == "GET" && strings.HasPrefix(r.URL.Path, "/payments") {
				response := map[string]interface{}{
					"data": paymentPkg.PaymentData{
						ID:         "mock-payment-id-12345",
						ExternalID: "test-external-id",
						Status:     paymentPkg.StatusSuccess,
					},
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(response)
				return
			}

			response := paymentPkg.PaymentResponse{
				Data: paymentPkg.PaymentData{
					ID:         "mock-payment-id",
					ExternalID: "test-external-id",
					Status:     paymentPkg.StatusSuccess,
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		}))

		mockGateway := paymentgateway.NewClient(paymentgateway.Config{
			MockAPIURL:     mockServer.URL,
			APIKey:         "test-api-key",
			PaymentTimeout: 10 * time.Second,
			MaxWorkers:     2,
			JobQueueSize:   10,
			WorkerPoolSize: 2,
		}, logger)

		paymentService = paymentPkg.NewPaymentService(logger, mockRepo, mockGateway)
	})

	AfterEach(func() {
		mockServer.Close()
	})

	Describe("CreatePayment", func() {
		Context("when all parameters are valid", func() {
			It("should create a payment successfully", func() {

				expenseID := int64(123)
				externalID := "test-external-id"
				amount := int64(50000)

				result, err := paymentService.CreatePayment(expenseID, externalID, amount)

				Expect(err).ToNot(HaveOccurred())
				Expect(result).ToNot(BeNil())
				Expect(result.ExpenseID).To(Equal(expenseID))
				Expect(result.ExternalID).To(Equal(externalID))
				Expect(result.AmountIDR).To(Equal(amount))
				Expect(result.Status).To(Equal(paymentPkg.StatusPending))
				Expect(result.RetryCount).To(Equal(0))
				Expect(result.ID).To(BeNumerically(">", 0))
			})
		})

		Context("when repository fails", func() {
			It("should return an error", func() {

				mockRepo.createError = errors.New("database error")
				expenseID := int64(123)
				externalID := "test-external-id"
				amount := int64(50000)

				result, err := paymentService.CreatePayment(expenseID, externalID, amount)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("failed to create payment record"))
				Expect(result).To(BeNil())
			})
		})
	})

	Describe("ProcessPayment", func() {
		Context("when payment request is valid", func() {
			It("should process payment successfully", func() {

				req := &paymentPkg.PaymentRequest{
					Amount:     50000,
					ExternalID: "test-external-id",
				}

				testPayment := &payment.Payment{
					ID:         1,
					ExpenseID:  123,
					ExternalID: req.ExternalID,
					AmountIDR:  req.Amount,
					Status:     paymentPkg.StatusPending,
				}
				mockRepo.payments[req.ExternalID] = testPayment

				result, err := paymentService.ProcessPayment(req)

				Expect(err).ToNot(HaveOccurred())
				Expect(result).ToNot(BeNil())
				Expect(result.Data.ExternalID).To(Equal(req.ExternalID))
				Expect(result.Data.Status).To(Equal("pending"))
			})
		})

		Context("when payment request validation fails", func() {
			It("should return validation error for empty external ID", func() {

				req := &paymentPkg.PaymentRequest{
					Amount:     50000,
					ExternalID: "",
				}

				result, err := paymentService.ProcessPayment(req)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("payment not found"))
				Expect(result).To(BeNil())
			})

			It("should return validation error for invalid amount", func() {

				req := &paymentPkg.PaymentRequest{
					Amount:     0,
					ExternalID: "test-external-id",
				}

				result, err := paymentService.ProcessPayment(req)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("payment not found"))
				Expect(result).To(BeNil())
			})
		})

		Context("when payment record is not found", func() {
			It("should return an error", func() {

				req := &paymentPkg.PaymentRequest{
					Amount:     50000,
					ExternalID: "non-existent-external-id",
				}

				result, err := paymentService.ProcessPayment(req)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("payment record not found"))
				Expect(result).To(BeNil())
			})
		})
	})

	Describe("GetPaymentByExpenseID", func() {
		Context("when payment exists", func() {
			It("should return the payment", func() {

				expenseID := int64(123)
				testPayment := &payment.Payment{
					ID:         1,
					ExpenseID:  expenseID,
					ExternalID: "test-external-id",
					AmountIDR:  50000,
					Status:     paymentPkg.StatusSuccess,
				}
				mockRepo.paymentsByExpense[expenseID] = testPayment

				result, err := paymentService.GetPaymentByExpenseID(expenseID)

				Expect(err).ToNot(HaveOccurred())
				Expect(result).ToNot(BeNil())
				Expect(result.ExpenseID).To(Equal(expenseID))
				Expect(result.ExternalID).To(Equal("test-external-id"))
			})
		})

		Context("when payment does not exist", func() {
			It("should return an error", func() {

				expenseID := int64(999)

				result, err := paymentService.GetPaymentByExpenseID(expenseID)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("payment not found"))
				Expect(result).To(BeNil())
			})
		})

		Context("when repository fails", func() {
			It("should return repository error", func() {

				mockRepo.getError = errors.New("database connection failed")
				expenseID := int64(123)

				result, err := paymentService.GetPaymentByExpenseID(expenseID)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("database connection failed"))
				Expect(result).To(BeNil())
			})
		})
	})

	Describe("External API Integration", func() {
		Context("when external API returns error status", func() {
			BeforeEach(func() {
				mockServer.Close()
				mockServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusBadRequest)
					w.Write([]byte(`{"error": "Invalid request"}`))
				}))

				mockGateway := paymentgateway.NewClient(paymentgateway.Config{
					MockAPIURL:     mockServer.URL,
					APIKey:         "test-api-key",
					PaymentTimeout: 10 * time.Second,
					MaxWorkers:     2,
					JobQueueSize:   10,
					WorkerPoolSize: 2,
				}, logger)
				paymentService = paymentPkg.NewPaymentService(logger, mockRepo, mockGateway)
			})

			It("should handle API errors gracefully", func() {

				req := &paymentPkg.PaymentRequest{
					Amount:     50000,
					ExternalID: "test-external-id",
				}

				testPayment := &payment.Payment{
					ID:         1,
					ExpenseID:  123,
					ExternalID: req.ExternalID,
					AmountIDR:  req.Amount,
					Status:     paymentPkg.StatusPending,
				}
				mockRepo.payments[req.ExternalID] = testPayment

				result, err := paymentService.ProcessPayment(req)

				Expect(err).ToNot(HaveOccurred())
				Expect(result).ToNot(BeNil())
				Expect(result.Data.Status).To(Equal("pending"))
			})
		})
	})
})
