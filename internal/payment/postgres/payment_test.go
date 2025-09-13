package postgres

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/frahmantamala/expense-management/internal/core/datamodel/payment"
	paymentpkg "github.com/frahmantamala/expense-management/internal/payment"
)

func TestPaymentRepository(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "Payment Repository Suite")
}

// PaymentSQLite is a test-specific version with text instead of jsonb for SQLite compatibility
type PaymentSQLite struct {
	ID              int64      `json:"id" gorm:"primaryKey"`
	ExpenseID       int64      `json:"expense_id" gorm:"column:expense_id;not null"`
	ExternalID      string     `json:"external_id" gorm:"column:external_id;not null;uniqueIndex"`
	AmountIDR       int64      `json:"amount_idr" gorm:"column:amount_idr;not null"`
	Status          string     `json:"status" gorm:"column:status;default:pending"`
	PaymentMethod   *string    `json:"payment_method,omitempty" gorm:"column:payment_method"`
	GatewayResponse string     `json:"gateway_response,omitempty" gorm:"column:gateway_response;type:text"` // Use text for SQLite
	FailureReason   *string    `json:"failure_reason,omitempty" gorm:"column:failure_reason"`
	RetryCount      int        `json:"retry_count" gorm:"column:retry_count;default:0"`
	ProcessedAt     *time.Time `json:"processed_at,omitempty" gorm:"column:processed_at"`
	CreatedAt       time.Time  `json:"created_at" gorm:"column:created_at"`
	UpdatedAt       time.Time  `json:"updated_at" gorm:"column:updated_at"`
}

func (PaymentSQLite) TableName() string {
	return "payments"
}

// BeforeCreate sets timestamps before creating
func (p *PaymentSQLite) BeforeCreate(tx *gorm.DB) error {
	now := time.Now().UTC()
	p.CreatedAt = now
	p.UpdatedAt = now
	return nil
}

// BeforeUpdate sets updated timestamp before updating
func (p *PaymentSQLite) BeforeUpdate(tx *gorm.DB) error {
	p.UpdatedAt = time.Now().UTC()
	return nil
}

var _ = ginkgo.Describe("PaymentRepository", func() {
	var (
		db   *gorm.DB
		repo paymentpkg.RepositoryAPI
	)

	ginkgo.BeforeEach(func() {
		// Use in-memory SQLite for testing
		var err error
		db, err = gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
			NowFunc: func() time.Time {
				return time.Now().UTC()
			},
		})
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		// Auto-migrate using the SQLite-compatible struct
		err = db.AutoMigrate(&PaymentSQLite{})
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		repo = NewPaymentRepository(db)
	})

	ginkgo.Describe("Create", func() {
		ginkgo.Context("when creating a payment successfully", func() {
			ginkgo.It("should insert payment and set ID", func() {
				// Given
				testPayment := &payment.Payment{
					ExpenseID:  123,
					ExternalID: "ext-123",
					AmountIDR:  50000,
					Status:     paymentpkg.StatusPending,
					RetryCount: 0,
				}

				// When
				err := repo.Create(testPayment)

				// Then
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				gomega.Expect(testPayment.ID).To(gomega.BeNumerically(">", 0))
				// Note: In test environment, timestamps might not be auto-populated by GORM
			})
		})

		ginkgo.Context("when creating payment with duplicate external ID", func() {
			ginkgo.It("should return error", func() {
				// Given
				firstPayment := &payment.Payment{
					ExpenseID:  123,
					ExternalID: "ext-123",
					AmountIDR:  50000,
					Status:     paymentpkg.StatusPending,
				}

				secondPayment := &payment.Payment{
					ExpenseID:  456,
					ExternalID: "ext-123", // Same external ID
					AmountIDR:  75000,
					Status:     paymentpkg.StatusPending,
				}

				// When
				err1 := repo.Create(firstPayment)
				err2 := repo.Create(secondPayment)

				// Then
				gomega.Expect(err1).ToNot(gomega.HaveOccurred())
				gomega.Expect(err2).To(gomega.HaveOccurred()) // Should fail due to unique constraint
			})
		})
	})

	ginkgo.Describe("GetByExternalID", func() {
		ginkgo.BeforeEach(func() {
			// Create test payment
			testPayment := &payment.Payment{
				ExpenseID:     123,
				ExternalID:    "ext-123",
				AmountIDR:     50000,
				Status:        paymentpkg.StatusSuccess,
				PaymentMethod: func() *string { s := "bank_transfer"; return &s }(),
				RetryCount:    0,
			}
			err := repo.Create(testPayment)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
		})

		ginkgo.Context("when payment exists", func() {
			ginkgo.It("should return the payment", func() {
				// When
				result, err := repo.GetByExternalID("ext-123")

				// Then
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				gomega.Expect(result).ToNot(gomega.BeNil())
				gomega.Expect(result.ExpenseID).To(gomega.Equal(int64(123)))
				gomega.Expect(result.ExternalID).To(gomega.Equal("ext-123"))
				gomega.Expect(result.AmountIDR).To(gomega.Equal(int64(50000)))
				gomega.Expect(result.Status).To(gomega.Equal(paymentpkg.StatusSuccess))
				gomega.Expect(*result.PaymentMethod).To(gomega.Equal("bank_transfer"))
			})
		})

		ginkgo.Context("when payment does not exist", func() {
			ginkgo.It("should return error", func() {
				// When
				result, err := repo.GetByExternalID("non-existent")

				// Then
				gomega.Expect(err).To(gomega.HaveOccurred())
				gomega.Expect(result).To(gomega.BeNil())
			})
		})
	})

	ginkgo.Describe("GetLatestByExpenseID", func() {
		ginkgo.BeforeEach(func() {
			// Create multiple payments for same expense
			payments := []*payment.Payment{
				{
					ExpenseID:  123,
					ExternalID: "ext-123-1",
					AmountIDR:  50000,
					Status:     paymentpkg.StatusFailed,
					CreatedAt:  time.Now().Add(-2 * time.Hour),
				},
				{
					ExpenseID:  123,
					ExternalID: "ext-123-2",
					AmountIDR:  50000,
					Status:     paymentpkg.StatusSuccess,
					CreatedAt:  time.Now().Add(-1 * time.Hour),
				},
			}

			for _, p := range payments {
				err := repo.Create(p)
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
			}
		})

		ginkgo.Context("when payments exist for expense", func() {
			ginkgo.It("should return the latest payment", func() {
				// When
				result, err := repo.GetLatestByExpenseID(123)

				// Then
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				gomega.Expect(result).ToNot(gomega.BeNil())
				gomega.Expect(result.ExternalID).To(gomega.Equal("ext-123-2"))
				gomega.Expect(result.Status).To(gomega.Equal(paymentpkg.StatusSuccess))
			})
		})

		ginkgo.Context("when no payments exist for expense", func() {
			ginkgo.It("should return error", func() {
				// When
				result, err := repo.GetLatestByExpenseID(999)

				// Then
				gomega.Expect(err).To(gomega.HaveOccurred())
				gomega.Expect(result).To(gomega.BeNil())
			})
		})
	})

	ginkgo.Describe("UpdateStatus", func() {
		var testPayment *payment.Payment

		ginkgo.BeforeEach(func() {
			testPayment = &payment.Payment{
				ExpenseID:  123,
				ExternalID: "ext-123",
				AmountIDR:  50000,
				Status:     paymentpkg.StatusPending,
				RetryCount: 0,
			}
			err := repo.Create(testPayment)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
		})

		ginkgo.Context("when updating status successfully", func() {
			ginkgo.It("should update payment status with all fields", func() {
				// Given
				paymentMethod := "bank_transfer"
				gatewayResponse := json.RawMessage(`{"transaction_id": "tx123"}`)
				failureReason := "Network timeout"

				// When
				err := repo.UpdateStatus(testPayment.ID, paymentpkg.StatusSuccess, &paymentMethod, gatewayResponse, &failureReason)

				// Then
				gomega.Expect(err).ToNot(gomega.HaveOccurred())

				// Verify the update
				updated, err := repo.GetByID(testPayment.ID)
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				gomega.Expect(updated.Status).To(gomega.Equal(paymentpkg.StatusSuccess))
				gomega.Expect(*updated.PaymentMethod).To(gomega.Equal("bank_transfer"))
				gomega.Expect(*updated.FailureReason).To(gomega.Equal("Network timeout"))
				gomega.Expect(updated.ProcessedAt).ToNot(gomega.BeNil())
			})

			ginkgo.It("should update status with nil optional fields", func() {
				// When
				err := repo.UpdateStatus(testPayment.ID, paymentpkg.StatusFailed, nil, nil, nil)

				// Then
				gomega.Expect(err).ToNot(gomega.HaveOccurred())

				// Verify the update
				updated, err := repo.GetByID(testPayment.ID)
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				gomega.Expect(updated.Status).To(gomega.Equal(paymentpkg.StatusFailed))
			})
		})

		ginkgo.Context("when payment not found", func() {
			ginkgo.It("should succeed but not affect any rows", func() {
				// When
				err := repo.UpdateStatus(999, paymentpkg.StatusSuccess, nil, nil, nil)

				// Then
				gomega.Expect(err).ToNot(gomega.HaveOccurred()) // GORM doesn't return error for 0 affected rows
			})
		})
	})

	ginkgo.Describe("IncrementRetryCount", func() {
		var testPayment *payment.Payment

		ginkgo.BeforeEach(func() {
			testPayment = &payment.Payment{
				ExpenseID:  123,
				ExternalID: "ext-123",
				AmountIDR:  50000,
				Status:     paymentpkg.StatusFailed,
				RetryCount: 2,
			}
			err := repo.Create(testPayment)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
		})

		ginkgo.Context("when incrementing retry count successfully", func() {
			ginkgo.It("should increment retry count", func() {
				// When
				err := repo.IncrementRetryCount(testPayment.ID)

				// Then
				gomega.Expect(err).ToNot(gomega.HaveOccurred())

				// Verify the increment
				updated, err := repo.GetByID(testPayment.ID)
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				gomega.Expect(updated.RetryCount).To(gomega.Equal(3))
			})
		})

		ginkgo.Context("when payment not found", func() {
			ginkgo.It("should succeed but not affect any rows", func() {
				// When
				err := repo.IncrementRetryCount(999)

				// Then
				gomega.Expect(err).ToNot(gomega.HaveOccurred()) // GORM doesn't return error for 0 affected rows
			})
		})
	})

	ginkgo.Describe("GetByExpenseID", func() {
		ginkgo.BeforeEach(func() {
			// Create multiple payments for same expense
			payments := []*payment.Payment{
				{
					ExpenseID:  123,
					ExternalID: "ext-123-1",
					AmountIDR:  50000,
					Status:     paymentpkg.StatusFailed,
					CreatedAt:  time.Now().Add(-2 * time.Hour),
				},
				{
					ExpenseID:  123,
					ExternalID: "ext-123-2",
					AmountIDR:  50000,
					Status:     paymentpkg.StatusSuccess,
					CreatedAt:  time.Now().Add(-1 * time.Hour),
				},
				{
					ExpenseID:  456, // Different expense
					ExternalID: "ext-456",
					AmountIDR:  75000,
					Status:     paymentpkg.StatusPending,
				},
			}

			for _, p := range payments {
				err := repo.Create(p)
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
			}
		})

		ginkgo.Context("when payments exist for expense", func() {
			ginkgo.It("should return all payments ordered by created_at DESC", func() {
				// When
				results, err := repo.GetByExpenseID(123)

				// Then
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				gomega.Expect(results).To(gomega.HaveLen(2))
				gomega.Expect(results[0].ExternalID).To(gomega.Equal("ext-123-2")) // Most recent first
				gomega.Expect(results[1].ExternalID).To(gomega.Equal("ext-123-1"))
			})
		})

		ginkgo.Context("when no payments exist for expense", func() {
			ginkgo.It("should return empty slice", func() {
				// When
				results, err := repo.GetByExpenseID(999)

				// Then
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				gomega.Expect(results).To(gomega.BeEmpty())
			})
		})
	})
})
