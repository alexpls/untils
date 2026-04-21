package validation

import (
	"fmt"

	"github.com/go-playground/validator/v10"
)

type ValidationError struct {
	Field   string
	Message string
}

type ValidationErrors []ValidationError

type HasValidationErrors interface {
	GetValidationErrors() ValidationErrors
}

// MapValidationErrors maps the error passed in to domain ValidationErrors
// that are ready for display to users.
//
// [nil] is returned if the error is not a [validator.ValidationErrors].
func MapValidationErrors(err error) ValidationErrors {
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
