package monitor

import (
	"context"
	"testing"
	"time"

	"github.com/alexpls/untils/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

func TestUpdateMonitorDraftCancelsStaleJobsAndChecks(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	deps := setupTestDeps(ctx, t)
	tx := deps.service.db.(pgx.Tx) //nolint:forcetypeassert

	checkJob, err := deps.service.river.InsertTx(ctx, tx, CheckArgs{
		UserID:         deps.fixtures.User.ID,
		MonitorCheckID: deps.fixtures.Check.ID,
	}, nil)
	require.NoError(t, err)

	staleCheckingCheck, err := deps.service.queries.CreateMonitorCheck(ctx, tx, &models.CreateMonitorCheckParams{
		MonitorID:    deps.fixtures.Monitor.ID,
		Status:       models.MonitorCheckStatusChecking,
		ScheduledFor: time.Now(),
	})
	require.NoError(t, err)

	secondCheckJob, err := deps.service.river.InsertTx(ctx, tx, CheckArgs{
		UserID:         deps.fixtures.User.ID,
		MonitorCheckID: staleCheckingCheck.ID,
	}, nil)
	require.NoError(t, err)

	validateJob, err := deps.service.river.InsertTx(ctx, tx, ValidateMonitorArgs{
		UserID:    deps.fixtures.User.ID,
		MonitorID: deps.fixtures.Monitor.ID,
	}, nil)
	require.NoError(t, err)

	_, err = deps.service.UpdateMonitorDraft(ctx, deps.fixtures.User.ID, deps.fixtures.Monitor.ID, UpdateMonitorDraftParams{
		MonitorCommonParams: MonitorCommonParams{
			Subject: "Updated monitor subject that should trigger cancellation",
		},
	})
	require.NoError(t, err)

	require.Equal(t, models.RiverJobStateCancelled, riverJobState(t, ctx, tx, checkJob.Job.ID))
	require.Equal(t, models.RiverJobStateCancelled, riverJobState(t, ctx, tx, secondCheckJob.Job.ID))
	require.Equal(t, models.RiverJobStateCancelled, riverJobState(t, ctx, tx, validateJob.Job.ID))
	require.Zero(t, monitorCheckCountByID(t, ctx, tx, deps.fixtures.Check.ID))
	require.Zero(t, monitorCheckCountByID(t, ctx, tx, staleCheckingCheck.ID))
}

func TestDeleteMonitorCancelsRelatedJobs(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	deps := setupTestDeps(ctx, t)
	tx := deps.service.db.(pgx.Tx) //nolint:forcetypeassert

	checkJob, err := deps.service.river.InsertTx(ctx, tx, CheckArgs{
		UserID:         deps.fixtures.User.ID,
		MonitorCheckID: deps.fixtures.Check.ID,
	}, nil)
	require.NoError(t, err)

	validateJob, err := deps.service.river.InsertTx(ctx, tx, ValidateMonitorArgs{
		UserID:    deps.fixtures.User.ID,
		MonitorID: deps.fixtures.Monitor.ID,
	}, nil)
	require.NoError(t, err)

	err = deps.service.DeleteMonitor(ctx, deps.fixtures.User.ID, deps.fixtures.Monitor.ID)
	require.NoError(t, err)

	require.Equal(t, models.RiverJobStateCancelled, riverJobState(t, ctx, tx, checkJob.Job.ID))
	require.Equal(t, models.RiverJobStateCancelled, riverJobState(t, ctx, tx, validateJob.Job.ID))
}

func TestSetMonitorPausedCancelsMonitorJobs(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	deps := setupTestDeps(ctx, t)
	tx := deps.service.db.(pgx.Tx) //nolint:forcetypeassert

	_, err := deps.service.queries.UpdateMonitorStatus(ctx, tx, &models.UpdateMonitorStatusParams{
		ID:     deps.fixtures.Monitor.ID,
		UserID: deps.fixtures.User.ID,
		Status: models.MonitorStatusActive,
	})
	require.NoError(t, err)

	checkJob, err := deps.service.river.InsertTx(ctx, tx, CheckArgs{
		UserID:         deps.fixtures.User.ID,
		MonitorCheckID: deps.fixtures.Check.ID,
	}, nil)
	require.NoError(t, err)

	validateJob, err := deps.service.river.InsertTx(ctx, tx, ValidateMonitorArgs{
		UserID:    deps.fixtures.User.ID,
		MonitorID: deps.fixtures.Monitor.ID,
	}, nil)
	require.NoError(t, err)

	updatedMonitor, err := deps.service.SetMonitorPaused(ctx, deps.fixtures.User, deps.fixtures.Monitor.ID, true)
	require.NoError(t, err)
	require.Equal(t, models.MonitorStatusPaused, updatedMonitor.Status)
	require.Equal(t, models.RiverJobStateCancelled, riverJobState(t, ctx, tx, checkJob.Job.ID))
	require.Equal(t, models.RiverJobStateCancelled, riverJobState(t, ctx, tx, validateJob.Job.ID))
}

func riverJobState(t *testing.T, ctx context.Context, tx pgx.Tx, jobID int64) models.RiverJobState {
	t.Helper()

	var state models.RiverJobState
	err := tx.QueryRow(ctx, "select state from river_job where id = $1", jobID).Scan(&state)
	require.NoError(t, err)
	return state
}

func monitorCheckCountByID(t *testing.T, ctx context.Context, tx pgx.Tx, checkID int64) int {
	t.Helper()

	var count int
	err := tx.QueryRow(ctx, "select count(*) from monitor_checks where id = $1", checkID).Scan(&count)
	require.NoError(t, err)
	return count
}
