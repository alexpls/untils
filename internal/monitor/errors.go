package monitor

import (
	"errors"
)

var (
	ErrMonitorNotFound       = errors.New("monitor not found")
	ErrMonitorCheckNotFound  = errors.New("monitor check not found")
	ErrNotifierNotConfigured = errors.New("notifier is not configured")
)
