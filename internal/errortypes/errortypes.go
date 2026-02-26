package errortypes

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/alexpls/untils/internal/models"
)

type HTTPError interface {
	HTTPCode() int
	HTTPMessage() string
}

type ResourceNotFoundError struct {
	Resource string
	ID       any
}

func (e *ResourceNotFoundError) Error() string {
	return fmt.Sprintf("%s %v not found", e.Resource, e.ID)
}

func (e *ResourceNotFoundError) HTTPCode() int {
	return http.StatusNotFound
}

func (e *ResourceNotFoundError) HTTPMessage() string {
	return e.Error()
}

func (e *ResourceNotFoundError) Is(target error) bool {
	_, ok := target.(*ResourceNotFoundError)
	return ok
}

type InvalidMonitorStatusTransitionError struct {
	From models.MonitorStatus
	To   models.MonitorStatus
}

func (e *InvalidMonitorStatusTransitionError) Error() string {
	return fmt.Sprintf("monitor: invalid status transition from '%s' to '%s'", e.From, e.To)
}

func (e *InvalidMonitorStatusTransitionError) HTTPCode() int {
	return http.StatusInternalServerError
}

func (e *InvalidMonitorStatusTransitionError) HTTPMessage() string {
	return "internal server error"
}

func (e *InvalidMonitorStatusTransitionError) Is(target error) bool {
	_, ok := target.(*InvalidMonitorStatusTransitionError)
	return ok
}

// ErrInvalidToken represents an invalid pushover token error.
type ErrInvalidToken struct {
	Reasons []string
}

func (e *ErrInvalidToken) Error() string {
	if len(e.Reasons) == 0 {
		return "invalid token"
	}
	return fmt.Sprintf("invalid token: %s", strings.Join(e.Reasons, ", "))
}

func (e *ErrInvalidToken) HTTPCode() int {
	return http.StatusBadRequest
}

func (e *ErrInvalidToken) HTTPMessage() string {
	return e.Error()
}

func (e *ErrInvalidToken) Is(target error) bool {
	_, ok := target.(*ErrInvalidToken)
	return ok
}

// ErrSubjectMismatch represents a monitor mismatch error used for stale check/triage writes.
type ErrSubjectMismatch struct {
	MonitorID1 int64
	MonitorID2 int64
	Subject1   string
	Subject2   string
}

func NewErrSubjectMismatch(mon1, mon2 *models.Monitor) *ErrSubjectMismatch {
	return &ErrSubjectMismatch{
		MonitorID1: mon1.ID,
		MonitorID2: mon2.ID,
		Subject1:   mon1.Subject.String,
		Subject2:   mon2.Subject.String,
	}
}

func (e *ErrSubjectMismatch) Error() string {
	return fmt.Sprintf(
		"monitor subject mismatch. id %d != %d or subject %q != %q",
		e.MonitorID1, e.MonitorID2,
		e.Subject1, e.Subject2)
}

func (e *ErrSubjectMismatch) HTTPCode() int {
	return http.StatusConflict
}

func (e *ErrSubjectMismatch) HTTPMessage() string {
	return "resource has been modified, please reload and try again"
}

func (e *ErrSubjectMismatch) Is(target error) bool {
	_, ok := target.(*ErrSubjectMismatch)
	return ok
}

// ErrCheckNotScheduled represents an error when a check is not in scheduled state.
type ErrCheckNotScheduled struct{}

func (e *ErrCheckNotScheduled) Error() string {
	return "check is not in scheduled state"
}

func (e *ErrCheckNotScheduled) HTTPCode() int {
	return http.StatusConflict
}

func (e *ErrCheckNotScheduled) HTTPMessage() string {
	return "check is not in scheduled state"
}

func (e *ErrCheckNotScheduled) Is(target error) bool {
	_, ok := target.(*ErrCheckNotScheduled)
	return ok
}

// ErrMonitorPaused represents an error when a monitor is paused.
type ErrMonitorPaused struct{}

func (e *ErrMonitorPaused) Error() string {
	return "monitor is paused"
}

func (e *ErrMonitorPaused) HTTPCode() int {
	return http.StatusConflict
}

func (e *ErrMonitorPaused) HTTPMessage() string {
	return "monitor is paused"
}

func (e *ErrMonitorPaused) Is(target error) bool {
	_, ok := target.(*ErrMonitorPaused)
	return ok
}

// ErrNoPushoverUserToken represents an error when no pushover user token is found.
type ErrNoPushoverUserToken struct{}

func (e *ErrNoPushoverUserToken) Error() string {
	return "no pushover user token found"
}

func (e *ErrNoPushoverUserToken) HTTPCode() int {
	return http.StatusBadRequest
}

func (e *ErrNoPushoverUserToken) HTTPMessage() string {
	return "no pushover user token configured"
}

func (e *ErrNoPushoverUserToken) Is(target error) bool {
	_, ok := target.(*ErrNoPushoverUserToken)
	return ok
}
