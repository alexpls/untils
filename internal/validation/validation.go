package validation

import (
	"errors"
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

type ValidationError struct {
	Field   string
	Message string
}

type ValidationErrors []ValidationError

func (errs ValidationErrors) Error() string {
	if len(errs) == 0 {
		return "validation failed"
	}

	parts := make([]string, 0, len(errs))
	for _, err := range errs {
		parts = append(parts, fmt.Sprintf("%s: %s", err.Field, err.Message))
	}
	return "validation failed: " + strings.Join(parts, ", ")
}

type HasValidationErrors interface {
	GetValidationErrors() ValidationErrors
}

// MapValidationErrors maps the error passed in to domain ValidationErrors that
// are ready for display to users.
//
// [nil] is returned if the error does not contain domain or validator
// validation errors.
func MapValidationErrors(err error) ValidationErrors {
	domainErrs, ok := errors.AsType[ValidationErrors](err)
	if ok {
		return domainErrs
	}

	validationErrs, ok := err.(validator.ValidationErrors)
	if !ok {
		return nil
	}

	errors := make([]ValidationError, 0, len(validationErrs))
	for _, fieldErr := range validationErrs {
		errors = append(errors, ValidationError{
			Field:   fieldErr.Field(),
			Message: formatFieldError(fieldErr),
		})
	}

	return errors
}

func formatFieldError(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "This field is required"
	case "min":
		return fmt.Sprintf("This must be at least %s characters", fe.Param())
	case "max":
		return fmt.Sprintf("This must be at most %s characters", fe.Param())
	case "url":
		return "This must be a valid URL"
	default:
		return "This field is invalid"
	}
}
