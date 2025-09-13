package internal

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type ErrorType string

const (
	ErrorTypeValidation   ErrorType = "VALIDATION_ERROR"
	ErrorTypeNotFound     ErrorType = "NOT_FOUND"
	ErrorTypeUnauthorized ErrorType = "UNAUTHORIZED"
	ErrorTypeForbidden    ErrorType = "FORBIDDEN"
	ErrorTypeConflict     ErrorType = "CONFLICT"
	ErrorTypeInternal     ErrorType = "INTERNAL_ERROR"
	ErrorTypeExternal     ErrorType = "EXTERNAL_ERROR"
)

type ErrorCode string

const (
	ErrCodeValidationFailed   ErrorCode = "VALIDATION_FAILED"
	ErrCodeInvalidAmount      ErrorCode = "INVALID_AMOUNT"
	ErrCodeInvalidDescription ErrorCode = "INVALID_DESCRIPTION"
	ErrCodeInvalidCategory    ErrorCode = "INVALID_CATEGORY"
	ErrCodeInvalidDate        ErrorCode = "INVALID_DATE"
	ErrCodeAmountTooLow       ErrorCode = "AMOUNT_TOO_LOW"
	ErrCodeAmountTooHigh      ErrorCode = "AMOUNT_TOO_HIGH"

	ErrCodeExpenseNotFound      ErrorCode = "EXPENSE_NOT_FOUND"
	ErrCodeUnauthorizedAccess   ErrorCode = "UNAUTHORIZED_ACCESS"
	ErrCodeInvalidExpenseStatus ErrorCode = "INVALID_EXPENSE_STATUS"
	ErrCodeCannotModifyExpense  ErrorCode = "CANNOT_MODIFY_EXPENSE"

	ErrCodeInvalidCredentials ErrorCode = "INVALID_CREDENTIALS"
	ErrCodeUserInactive       ErrorCode = "USER_INACTIVE"
	ErrCodeInvalidToken       ErrorCode = "INVALID_TOKEN"
	ErrCodeTokenExpired       ErrorCode = "TOKEN_EXPIRED"

	ErrCodePaymentFailed      ErrorCode = "PAYMENT_FAILED"
	ErrCodePaymentRetryFailed ErrorCode = "PAYMENT_RETRY_FAILED"
)

type AppError struct {
	Type       ErrorType   `json:"type"`
	Code       ErrorCode   `json:"code"`
	Message    string      `json:"message"`
	Details    interface{} `json:"details,omitempty"`
	StatusCode int         `json:"-"`
	Cause      error       `json:"-"`
}

func (e *AppError) Error() string {
	if e.Details != nil {
		if validationErrors, ok := e.Details.(ValidationErrors); ok && len(validationErrors.Errors) > 0 {

			return validationErrors.Errors[0].Message
		}
	}
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

func (e *AppError) GetDetailedMessage() string {
	if e.Details != nil {
		if validationErrors, ok := e.Details.(ValidationErrors); ok {
			if len(validationErrors.Errors) == 1 {
				return validationErrors.Errors[0].Message
			} else if len(validationErrors.Errors) > 1 {
				messages := make([]string, len(validationErrors.Errors))
				for i, err := range validationErrors.Errors {
					messages[i] = err.Message
				}
				return strings.Join(messages, "; ")
			}
		}
	}
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.Cause
}

func (e *AppError) WithCause(cause error) *AppError {
	e.Cause = cause
	return e
}

func (e *AppError) WithDetails(details interface{}) *AppError {
	e.Details = details
	return e
}

type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Code    string `json:"code"`
}

type ValidationErrors struct {
	Errors []ValidationError `json:"errors"`
}

func NewValidationError(message string, code ErrorCode) *AppError {
	return &AppError{
		Type:       ErrorTypeValidation,
		Code:       code,
		Message:    message,
		StatusCode: http.StatusBadRequest,
	}
}

func NewValidationFieldError(field, message string, code ErrorCode) *AppError {
	return &AppError{
		Type:       ErrorTypeValidation,
		Code:       ErrCodeValidationFailed,
		Message:    "Validation failed",
		StatusCode: http.StatusBadRequest,
		Details: ValidationErrors{
			Errors: []ValidationError{
				{Field: field, Message: message, Code: string(code)},
			},
		},
	}
}

func NewNotFoundError(message string, code ErrorCode) *AppError {
	return &AppError{
		Type:       ErrorTypeNotFound,
		Code:       code,
		Message:    message,
		StatusCode: http.StatusNotFound,
	}
}

func NewUnauthorizedError(message string, code ErrorCode) *AppError {
	return &AppError{
		Type:       ErrorTypeUnauthorized,
		Code:       code,
		Message:    message,
		StatusCode: http.StatusUnauthorized,
	}
}

func NewForbiddenError(message string, code ErrorCode) *AppError {
	return &AppError{
		Type:       ErrorTypeForbidden,
		Code:       code,
		Message:    message,
		StatusCode: http.StatusForbidden,
	}
}

func NewInternalError(message string, cause error) *AppError {
	return &AppError{
		Type:       ErrorTypeInternal,
		Code:       "INTERNAL_ERROR",
		Message:    message,
		StatusCode: http.StatusInternalServerError,
		Cause:      cause,
	}
}

func NewConflictError(message string, code ErrorCode) *AppError {
	return &AppError{
		Type:       ErrorTypeConflict,
		Code:       code,
		Message:    message,
		StatusCode: http.StatusConflict,
	}
}

var (
	ErrExpenseNotFound      = NewNotFoundError("Expense not found", ErrCodeExpenseNotFound)
	ErrUnauthorizedAccess   = NewForbiddenError("unauthorized access to expense", ErrCodeUnauthorizedAccess)
	ErrInvalidExpenseStatus = NewValidationError("invalid expense status for this operation", ErrCodeInvalidExpenseStatus)
	ErrCannotModifyExpense  = NewValidationError("Cannot modify expense in current status", ErrCodeCannotModifyExpense)

	ErrInvalidCredentials = NewUnauthorizedError("Invalid email or password", ErrCodeInvalidCredentials)
	ErrUserInactive       = NewForbiddenError("User account is inactive", ErrCodeUserInactive)
	ErrInvalidToken       = NewUnauthorizedError("Invalid token", ErrCodeInvalidToken)
	ErrTokenExpired       = NewUnauthorizedError("Token has expired", ErrCodeTokenExpired)
)

func IsAppError(err error) (*AppError, bool) {
	if appErr, ok := err.(*AppError); ok {
		return appErr, true
	}
	return nil, false
}

type Response struct {
	Error *AppError `json:"error"`
}

func (e *AppError) ToHTTPResponse() (int, interface{}) {
	return e.StatusCode, Response{Error: e}
}

func (e *AppError) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Type    ErrorType   `json:"type"`
		Code    ErrorCode   `json:"code"`
		Message string      `json:"message"`
		Details interface{} `json:"details,omitempty"`
	}{
		Type:    e.Type,
		Code:    e.Code,
		Message: e.Message,
		Details: e.Details,
	})
}
