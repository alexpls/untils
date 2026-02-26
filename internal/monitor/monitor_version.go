package monitor

import (
	"context"
	"fmt"

	"github.com/alexpls/untils/internal/errortypes"
	"github.com/alexpls/untils/internal/models"
)

func (s *Service) validateMonitorSubjectUnchanged(ctx context.Context, tx models.DBTX, mon *models.Monitor) error {
	mon2, err := s.queries.GetMonitor(ctx, tx, &models.GetMonitorParams{
		ID:     mon.ID,
		UserID: mon.UserID,
	})
	if err != nil {
		return fmt.Errorf("reloading monitor: %w", err)
	}

	if mon.ID != mon2.ID || mon.Subject.String != mon2.Subject.String {
		return errortypes.NewErrSubjectMismatch(mon, mon2)
	}
	return nil
}

func (s *Service) bumpMonitorVersion(ctx context.Context, tx models.DBTX, mon *models.Monitor) error {
	if err := s.queries.BumpMonitorVersion(ctx, tx, mon.ID); err != nil {
		return fmt.Errorf("bumping monitor version: %w", err)
	}
	return nil
}
