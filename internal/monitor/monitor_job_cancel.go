package monitor

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

func (s *Service) cancelMonitorJobsTx(ctx context.Context, tx pgx.Tx, monitorID int64) error {
	checkJobIDs, err := s.queries.ListCheckRiverJobIDsByMonitorID(ctx, tx, monitorID)
	if err != nil {
		return fmt.Errorf("listing check river jobs: %w", err)
	}

	validateJobIDs, err := s.queries.ListValidateDraftRiverJobIDsByMonitorID(ctx, tx, fmt.Sprintf("%d", monitorID))
	if err != nil {
		return fmt.Errorf("listing validate_draft river jobs: %w", err)
	}

	for _, jobID := range checkJobIDs {
		if _, err := s.river.JobCancelTx(ctx, tx, jobID); err != nil {
			return fmt.Errorf("cancelling check river job %d: %w", jobID, err)
		}
	}

	for _, jobID := range validateJobIDs {
		if _, err := s.river.JobCancelTx(ctx, tx, jobID); err != nil {
			return fmt.Errorf("cancelling validate_draft river job %d: %w", jobID, err)
		}
	}

	return nil
}
