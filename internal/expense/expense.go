package expense

import (
	"time"

	expenseDatamodel "github.com/frahmantamala/expense-management/internal/core/datamodel/expense"
)

type Expense struct {
	ID              int64      `json:"id"`
	UserID          int64      `json:"user_id"`
	AmountIDR       int64      `json:"amount_idr"`
	Description     string     `json:"description"`
	Category        string     `json:"category"`
	ReceiptURL      *string    `json:"receipt_url,omitempty"`
	ReceiptFileName *string    `json:"receipt_filename,omitempty"`
	ExpenseStatus   string     `json:"expense_status"`
	ExpenseDate     time.Time  `json:"expense_date"`
	SubmittedAt     time.Time  `json:"submitted_at"`
	ProcessedAt     *time.Time `json:"processed_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

const (
	ExpenseStatusPendingApproval = "pending_approval"
	ExpenseStatusApproved        = "approved"
	ExpenseStatusRejected        = "rejected"
	AutoApprovalThreshold        = 100000
)

func (e *Expense) CanBeApproved() bool {
	return e.ExpenseStatus == ExpenseStatusPendingApproval
}

func (e *Expense) CanBeRejected() bool {
	return e.ExpenseStatus == ExpenseStatusPendingApproval
}

func (e *Expense) ShouldBeAutoApproved() bool {
	return e.AmountIDR < AutoApprovalThreshold
}

func (e *Expense) Approve() {
	e.ExpenseStatus = ExpenseStatusApproved
	now := time.Now()
	e.ProcessedAt = &now
	e.UpdatedAt = now
}

func (e *Expense) Reject() {
	e.ExpenseStatus = ExpenseStatusRejected
	now := time.Now()
	e.ProcessedAt = &now
	e.UpdatedAt = now
}

func (e *Expense) NeedsPaymentProcessing() bool {
	return e.ExpenseStatus == ExpenseStatusApproved
}

func NewExpense(userID int64, dto CreateExpenseDTO) *Expense {
	now := time.Now()

	expense := &Expense{
		UserID:          userID,
		AmountIDR:       dto.AmountIDR,
		Description:     dto.Description,
		Category:        dto.Category,
		ReceiptURL:      dto.ReceiptURL,
		ReceiptFileName: dto.ReceiptFileName,
		ExpenseStatus:   ExpenseStatusPendingApproval,
		ExpenseDate:     dto.ExpenseDate,
		SubmittedAt:     now,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if expense.ShouldBeAutoApproved() {
		expense.Approve()
	}

	return expense
}

func ToDataModel(e *Expense) *expenseDatamodel.Expense {
	return &expenseDatamodel.Expense{
		ID:              e.ID,
		UserID:          e.UserID,
		AmountIDR:       e.AmountIDR,
		Description:     e.Description,
		Category:        e.Category,
		ReceiptURL:      e.ReceiptURL,
		ReceiptFileName: e.ReceiptFileName,
		ExpenseStatus:   e.ExpenseStatus,
		ExpenseDate:     e.ExpenseDate,
		SubmittedAt:     e.SubmittedAt,
		ProcessedAt:     e.ProcessedAt,
		CreatedAt:       e.CreatedAt,
		UpdatedAt:       e.UpdatedAt,
	}
}

func FromDataModel(e *expenseDatamodel.Expense) *Expense {
	return &Expense{
		ID:              e.ID,
		UserID:          e.UserID,
		AmountIDR:       e.AmountIDR,
		Description:     e.Description,
		Category:        e.Category,
		ReceiptURL:      e.ReceiptURL,
		ReceiptFileName: e.ReceiptFileName,
		ExpenseStatus:   e.ExpenseStatus,
		ExpenseDate:     e.ExpenseDate,
		SubmittedAt:     e.SubmittedAt,
		ProcessedAt:     e.ProcessedAt,
		CreatedAt:       e.CreatedAt,
		UpdatedAt:       e.UpdatedAt,
	}
}

func FromDataModelSlice(expenses []*expenseDatamodel.Expense) []*Expense {
	result := make([]*Expense, len(expenses))
	for i, e := range expenses {
		result[i] = FromDataModel(e)
	}
	return result
}
