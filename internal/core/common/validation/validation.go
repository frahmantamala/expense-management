package validation

import (
	"fmt"
	"time"

	errors "github.com/frahmantamala/expense-management/internal"
)

type ValidatorFunc func(interface{}) *errors.AppError

type FieldValidator struct {
	FieldName  string
	Value      interface{}
	Validators []ValidatorFunc
}

type ValidationBuilder struct {
	fields []FieldValidator
	errors []errors.ValidationError
}

func NewValidator() *ValidationBuilder {
	return &ValidationBuilder{
		fields: make([]FieldValidator, 0),
		errors: make([]errors.ValidationError, 0),
	}
}

func (v *ValidationBuilder) Field(name string, value interface{}) *FieldValidator {
	fv := FieldValidator{
		FieldName:  name,
		Value:      value,
		Validators: make([]ValidatorFunc, 0),
	}
	v.fields = append(v.fields, fv)
	return &v.fields[len(v.fields)-1]
}

func (fv *FieldValidator) Required() *FieldValidator {
	fv.Validators = append(fv.Validators, func(value interface{}) *errors.AppError {
		switch v := value.(type) {
		case string:
			if v == "" {
				return errors.NewValidationFieldError(fv.FieldName, fmt.Sprintf("%s is required", fv.FieldName), errors.ErrCodeValidationFailed)
			}
		case int64:
			if v == 0 {

				if fv.FieldName == "amount_idr" {
					return errors.NewValidationFieldError(fv.FieldName, "amount must be positive", errors.ErrCodeValidationFailed)
				}
				return errors.NewValidationFieldError(fv.FieldName, fmt.Sprintf("%s is required", fv.FieldName), errors.ErrCodeValidationFailed)
			}
		case *string:
			if v == nil || *v == "" {
				return errors.NewValidationFieldError(fv.FieldName, fmt.Sprintf("%s is required", fv.FieldName), errors.ErrCodeValidationFailed)
			}
		}
		return nil
	})
	return fv
}

func (fv *FieldValidator) MinInt(min int64, code errors.ErrorCode) *FieldValidator {
	fv.Validators = append(fv.Validators, func(value interface{}) *errors.AppError {
		if v, ok := value.(int64); ok {
			if v < min {
				var message string

				if fv.FieldName == "amount_idr" {
					if min == 10000 {
						message = "amount must be at least 10,000 IDR"
					} else {
						message = fmt.Sprintf("amount must be at least %d", min)
					}
				} else {
					message = fmt.Sprintf("%s must be at least %d", fv.FieldName, min)
				}
				return errors.NewValidationFieldError(fv.FieldName, message, code)
			}
		}
		return nil
	})
	return fv
}

func (fv *FieldValidator) MaxInt(max int64, code errors.ErrorCode) *FieldValidator {
	fv.Validators = append(fv.Validators, func(value interface{}) *errors.AppError {
		if v, ok := value.(int64); ok {
			if v > max {
				var message string

				if fv.FieldName == "amount_idr" && max == 50000000 {
					message = "amount must not exceed 50,000,000 IDR"
				} else {
					message = fmt.Sprintf("%s must not exceed %d", fv.FieldName, max)
				}
				return errors.NewValidationFieldError(fv.FieldName, message, code)
			}
		}
		return nil
	})
	return fv
}

func (fv *FieldValidator) MinLength(min int) *FieldValidator {
	fv.Validators = append(fv.Validators, func(value interface{}) *errors.AppError {
		if v, ok := value.(string); ok {
			if len(v) < min {
				message := fmt.Sprintf("%s must be at least %d characters", fv.FieldName, min)
				return errors.NewValidationFieldError(fv.FieldName, message, errors.ErrCodeValidationFailed)
			}
		}
		return nil
	})
	return fv
}

func (fv *FieldValidator) MaxLength(max int) *FieldValidator {
	fv.Validators = append(fv.Validators, func(value interface{}) *errors.AppError {
		if v, ok := value.(string); ok {
			if len(v) > max {
				message := fmt.Sprintf("%s must not exceed %d characters", fv.FieldName, max)
				return errors.NewValidationFieldError(fv.FieldName, message, errors.ErrCodeValidationFailed)
			}
		}
		return nil
	})
	return fv
}

func (fv *FieldValidator) NotFuture() *FieldValidator {
	fv.Validators = append(fv.Validators, func(value interface{}) *errors.AppError {
		if v, ok := value.(time.Time); ok {
			if v.After(time.Now()) {
				message := fmt.Sprintf("%s cannot be in the future", fv.FieldName)
				return errors.NewValidationFieldError(fv.FieldName, message, errors.ErrCodeInvalidDate)
			}
		}
		return nil
	})
	return fv
}

func (fv *FieldValidator) Custom(validator func(interface{}) *errors.AppError) *FieldValidator {
	fv.Validators = append(fv.Validators, validator)
	return fv
}

func (v *ValidationBuilder) Validate() *errors.AppError {
	var validationErrors []errors.ValidationError

	for _, field := range v.fields {
		for _, validator := range field.Validators {
			if err := validator(field.Value); err != nil {
				if appErr, ok := errors.IsAppError(err); ok {

					if appErr.Details != nil {
						if details, ok := appErr.Details.(errors.ValidationErrors); ok {
							validationErrors = append(validationErrors, details.Errors...)
						} else {

							validationError := errors.ValidationError{
								Field:   field.FieldName,
								Message: appErr.Message,
								Code:    string(appErr.Code),
							}
							validationErrors = append(validationErrors, validationError)
						}
					} else {

						validationError := errors.ValidationError{
							Field:   field.FieldName,
							Message: appErr.Message,
							Code:    string(appErr.Code),
						}
						validationErrors = append(validationErrors, validationError)
					}
				}
			}
		}
	}

	if len(validationErrors) > 0 {
		return errors.NewValidationError("Validation failed", errors.ErrCodeValidationFailed).
			WithDetails(errors.ValidationErrors{Errors: validationErrors})
	}

	return nil
}

func ValidateExpenseAmount(amount int64) *errors.AppError {
	validator := NewValidator()
	validator.Field("amount_idr", amount).
		Required().
		MinInt(1, errors.ErrCodeInvalidAmount).
		MinInt(10000, errors.ErrCodeAmountTooLow).
		MaxInt(50000000, errors.ErrCodeAmountTooHigh)
	return validator.Validate()
}

func ValidateExpenseDescription(description string) *errors.AppError {
	validator := NewValidator()
	validator.Field("description", description).
		Required().
		MinLength(1).
		MaxLength(500)
	return validator.Validate()
}

func ValidateExpenseDate(date time.Time) *errors.AppError {
	validator := NewValidator()
	validator.Field("expense_date", date).
		NotFuture()
	return validator.Validate()
}
