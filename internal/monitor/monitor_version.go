package monitor

import (
	"context"
	"fmt"

	"github.com/alexpls/untils/internal/db/models"
)

type ErrVersionMismatch struct {
	mon1, mon2 *models.Monitor
}

func (e ErrVersionMismatch) Error() string {
	return fmt.Sprintf(
		"monitors version mismatch. id %d != %d or updated_at %s != %s",
		e.mon1.ID, e.mon2.ID,
		e.mon1.UpdatedAt, e.mon2.UpdatedAt)
}

func NewErrVersionMismatch(mon1, mon2 *models.Monitor) *ErrVersionMismatch {
	return &ErrVersionMismatch{mon1, mon2}
}

func (s *Service) validateMonitorsSameVersion(ctx context.Context, tx models.DBTX, mon *models.Monitor) error {
	mon2, err := s.queries.GetMonitor(ctx, tx, &models.GetMonitorParams{
		ID:     mon.ID,
		UserID: mon.UserID,
	})
	if err != nil {
		return fmt.Errorf("reloading monitor: %w", err)
	}

	if mon.ID != mon2.ID || !mon.UpdatedAt.Equal(mon2.UpdatedAt) {
		return NewErrVersionMismatch(mon, mon2)
	}
	return nil
}

func (s *Service) bumpMonitorVersion(ctx context.Context, tx models.DBTX, mon *models.Monitor) error {
	if err := s.queries.BumpMonitorVersion(ctx, tx, mon.ID); err != nil {
		return fmt.Errorf("bumping monitor version: %w", err)
	}
	return nil
}
