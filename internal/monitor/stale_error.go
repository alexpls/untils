package monitor

import (
	"errors"

	"github.com/alexpls/untils/internal/errortypes"
)

func isStaleMonitorWorkError(err error) bool {
	return errors.Is(err, &errortypes.ResourceNotFoundError{}) ||
		errors.Is(err, &errortypes.ErrSubjectMismatch{})
}
