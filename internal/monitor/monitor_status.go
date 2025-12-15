package monitor

import (
	"context"
	"fmt"
	"slices"

	"github.com/alexpls/untils_go/internal/db/sqlc"
)

var validMonitorStatusTransitions = map[sqlc.MonitorStatus][]sqlc.MonitorStatus{
	sqlc.MonitorStatusValidating: {
		sqlc.MonitorStatusPreviewing,
		sqlc.MonitorStatusRejected,
	},
	sqlc.MonitorStatusPreviewing: {
		sqlc.MonitorStatusReady,
		sqlc.MonitorStatusRejected,
	},
	sqlc.MonitorStatusRejected: {
		sqlc.MonitorStatusValidating,
	},
	sqlc.MonitorStatusReady: {
		sqlc.MonitorStatusValidating,
		sqlc.MonitorStatusActive,
	},
	sqlc.MonitorStatusActive: {},
}

type ErrInvalidStatusTransition struct {
	from sqlc.MonitorStatus
	to   sqlc.MonitorStatus
}

func (e ErrInvalidStatusTransition) Error() string {
	return fmt.Sprintf("monitor: invalid status transition from '%s' to '%s'", e.from, e.to)
}

func (s *Service) updateMonitorStatus(ctx context.Context, tx sqlc.DBTX, mon *sqlc.Monitor, newStatus sqlc.MonitorStatus) (*sqlc.Monitor, error) {
	if err := validateMonitorTransition(mon.Status, newStatus); err != nil {
		return nil, err
	}

	updatedMon, err := s.queries.UpdateMonitorStatus(ctx, tx, &sqlc.UpdateMonitorStatusParams{
		ID:     mon.ID,
		UserID: mon.UserID,
		Status: newStatus,
	})
	if err != nil {
		return nil, fmt.Errorf("updating monitor status: %w", err)
	}

	return updatedMon, nil
}

func validateMonitorTransition(from sqlc.MonitorStatus, to sqlc.MonitorStatus) error {
	if from == to {
		return nil
	}

	validTos, ok := validMonitorStatusTransitions[from]
	if !ok {
		return fmt.Errorf("monitor: unconfigured 'from' status: %s", from)
	}
	if !slices.Contains(validTos, to) {
		return &ErrInvalidStatusTransition{from: from, to: to}
	}
	return nil
}
