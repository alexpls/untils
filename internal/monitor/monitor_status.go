package monitor

import (
	"context"
	"fmt"
	"slices"

	"github.com/alexpls/untils/internal/models"
)

var validMonitorStatusTransitions = map[models.MonitorStatus][]models.MonitorStatus{
	models.MonitorStatusValidating: {
		models.MonitorStatusPreviewing,
		models.MonitorStatusRejected,
	},
	models.MonitorStatusPreviewing: {
		models.MonitorStatusValidating,
		models.MonitorStatusReady,
		models.MonitorStatusRejected,
	},
	models.MonitorStatusRejected: {
		models.MonitorStatusValidating,
	},
	models.MonitorStatusReady: {
		models.MonitorStatusValidating,
		models.MonitorStatusActive,
	},
	models.MonitorStatusActive: {
		models.MonitorStatusPaused,
	},
	models.MonitorStatusPaused: {
		models.MonitorStatusActive,
	},
}

type ErrInvalidStatusTransition struct {
	from models.MonitorStatus
	to   models.MonitorStatus
}

func (e ErrInvalidStatusTransition) Error() string {
	return fmt.Sprintf("monitor: invalid status transition from '%s' to '%s'", e.from, e.to)
}

func (s *Service) updateMonitorStatus(ctx context.Context, tx models.DBTX, mon *models.Monitor, newStatus models.MonitorStatus) (*models.Monitor, error) {
	if err := validateMonitorTransition(mon.Status, newStatus); err != nil {
		return nil, err
	}

	updatedMon, err := s.queries.UpdateMonitorStatus(ctx, tx, &models.UpdateMonitorStatusParams{
		ID:     mon.ID,
		UserID: mon.UserID,
		Status: newStatus,
	})
	if err != nil {
		return nil, fmt.Errorf("updating monitor status: %w", err)
	}

	return updatedMon, nil
}

func validateMonitorTransition(from models.MonitorStatus, to models.MonitorStatus) error {
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
