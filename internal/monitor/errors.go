package monitor

import (
	"errors"
)

var (
	ErrNotifierNotConfigured             = errors.New("notifier is not configured")
	ErrMonitorResultCorrectionNotAllowed = errors.New("correction is only allowed on the latest visible result")
	ErrMonitorResultHideNotAllowed       = errors.New("result can only be hidden from the activity timeline")
)
