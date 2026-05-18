package validation

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMapValidationErrorsReturnsDomainErrors(t *testing.T) {
	errs := ValidationErrors{
		{
			Field:   "Name",
			Message: "This name is already used",
		},
	}

	require.Equal(t, errs, MapValidationErrors(errs))
	require.Equal(t, errs, MapValidationErrors(fmt.Errorf("creating token: %w", errs)))
}
