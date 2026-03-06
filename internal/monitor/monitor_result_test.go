package monitor

import (
	"context"
	"testing"
	"time"

	"github.com/alexpls/untils/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

func TestCreateMonitorResultCorrectionRequiresLatestVisibleResult(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	deps := setupTestDeps(ctx, t)
	tx := deps.service.db.(pgx.Tx) //nolint:forcetypeassert

	older := createMonitorResultFixture(t, ctx, tx, deps.fixtures.Monitor.ID, "Older result")
	latest := createMonitorResultFixture(t, ctx, tx, deps.fixtures.Monitor.ID, "Latest result")

	err := deps.service.CreateMonitorResultCorrection(ctx, deps.fixtures.User.ID, older, CreateMonitorResultCorrectionParams{
		Correction: "Use the canonical page instead",
	})
	require.ErrorIs(t, err, ErrMonitorResultCorrectionNotAllowed)

	storedOlder := getMonitorResultByID(t, ctx, tx, older.ID)
	storedLatest := getMonitorResultByID(t, ctx, tx, latest.ID)
	require.False(t, storedOlder.Hidden)
	require.False(t, storedOlder.Correction.Valid)
	require.False(t, storedLatest.Hidden)
}

func TestCreateMonitorResultCorrectionOnReadyMonitorHidesResultAndRevalidates(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	deps := setupTestDeps(ctx, t)
	tx := deps.service.db.(pgx.Tx) //nolint:forcetypeassert

	_, err := deps.service.queries.UpdateMonitorStatus(ctx, tx, &models.UpdateMonitorStatusParams{
		ID:     deps.fixtures.Monitor.ID,
		UserID: deps.fixtures.User.ID,
		Status: models.MonitorStatusReady,
	})
	require.NoError(t, err)

	result := createMonitorResultFixture(t, ctx, tx, deps.fixtures.Monitor.ID, "Preview result")

	err = deps.service.CreateMonitorResultCorrection(ctx, deps.fixtures.User.ID, result, CreateMonitorResultCorrectionParams{
		Correction: "Treat this source as stale",
	})
	require.NoError(t, err)

	storedResult := getMonitorResultByID(t, ctx, tx, result.ID)
	require.True(t, storedResult.Hidden)
	require.Equal(t, "Treat this source as stale", storedResult.Correction.String)

	monitor, err := deps.service.queries.GetMonitor(ctx, tx, &models.GetMonitorParams{
		UserID: deps.fixtures.User.ID,
		ID:     deps.fixtures.Monitor.ID,
	})
	require.NoError(t, err)
	require.Equal(t, models.MonitorStatusValidating, monitor.Status)
	require.Equal(t, 1, countRiverJobsByKind(t, ctx, tx, "validate_draft"))
	require.Zero(t, countMonitorChecks(t, ctx, tx, deps.fixtures.Monitor.ID))
}

func TestCreateMonitorResultCorrectionOnActiveMonitorCancelsStaleChecksAndSchedulesReplacement(t *testing.T) {
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

	currentJob, err := deps.service.river.InsertTx(ctx, tx, CheckArgs{
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

	staleCheckingJob, err := deps.service.river.InsertTx(ctx, tx, CheckArgs{
		UserID:         deps.fixtures.User.ID,
		MonitorCheckID: staleCheckingCheck.ID,
	}, nil)
	require.NoError(t, err)

	result := createMonitorResultFixture(t, ctx, tx, deps.fixtures.Monitor.ID, "Visible result")

	err = deps.service.CreateMonitorResultCorrection(ctx, deps.fixtures.User.ID, result, CreateMonitorResultCorrectionParams{
		Correction: "Ignore syndicated summaries",
	})
	require.NoError(t, err)

	storedResult := getMonitorResultByID(t, ctx, tx, result.ID)
	require.True(t, storedResult.Hidden)
	require.Equal(t, "Ignore syndicated summaries", storedResult.Correction.String)

	require.Equal(t, models.RiverJobStateCancelled, riverJobState(t, ctx, tx, currentJob.Job.ID))
	require.Equal(t, models.RiverJobStateCancelled, riverJobState(t, ctx, tx, staleCheckingJob.Job.ID))
	require.Zero(t, monitorCheckCountByID(t, ctx, tx, deps.fixtures.Check.ID))
	require.Zero(t, monitorCheckCountByID(t, ctx, tx, staleCheckingCheck.ID))

	replacementCheck := latestMonitorCheck(t, ctx, tx, deps.fixtures.Monitor.ID)
	require.NotNil(t, replacementCheck)
	require.Equal(t, models.MonitorCheckStatusScheduled, replacementCheck.Status)
	require.NotEqual(t, deps.fixtures.Check.ID, replacementCheck.ID)
	require.NotEqual(t, staleCheckingCheck.ID, replacementCheck.ID)
	require.Equal(t, 1, countActiveCheckRiverJobs(t, ctx, tx, deps.fixtures.Monitor.ID))
}

func TestHideMonitorResultDoesNotScheduleReplacementCheck(t *testing.T) {
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

	result := createMonitorResultFixture(t, ctx, tx, deps.fixtures.Monitor.ID, "Timeline result")

	err = deps.service.HideMonitorResult(ctx, deps.fixtures.User.ID, result)
	require.NoError(t, err)

	storedResult := getMonitorResultByID(t, ctx, tx, result.ID)
	require.True(t, storedResult.Hidden)
	require.False(t, storedResult.Correction.Valid)
	require.Equal(t, 1, countMonitorChecks(t, ctx, tx, deps.fixtures.Monitor.ID))
}

func TestHideMonitorResultRequiresTimelineMonitorStatus(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	deps := setupTestDeps(ctx, t)
	tx := deps.service.db.(pgx.Tx) //nolint:forcetypeassert

	_, err := deps.service.queries.UpdateMonitorStatus(ctx, tx, &models.UpdateMonitorStatusParams{
		ID:     deps.fixtures.Monitor.ID,
		UserID: deps.fixtures.User.ID,
		Status: models.MonitorStatusReady,
	})
	require.NoError(t, err)

	result := createMonitorResultFixture(t, ctx, tx, deps.fixtures.Monitor.ID, "Preview result")

	err = deps.service.HideMonitorResult(ctx, deps.fixtures.User.ID, result)
	require.ErrorIs(t, err, ErrMonitorResultHideNotAllowed)

	storedResult := getMonitorResultByID(t, ctx, tx, result.ID)
	require.False(t, storedResult.Hidden)
}

func createMonitorResultFixture(t *testing.T, ctx context.Context, tx pgx.Tx, monitorID int64, headline string) *models.MonitorResult {
	t.Helper()

	citations := models.Citations{}

	result, err := models.New().CreateMonitorResult(ctx, tx, &models.CreateMonitorResultParams{
		MonitorID: monitorID,
		Headline:  headline,
		Subtitle:  "",
		Data: models.MonitorUpdateData{
			Headline: headline,
			Fields:   models.MonitorUpdateFields{},
		},
		Citations: &citations,
	})
	require.NoError(t, err)

	return result
}

func getMonitorResultByID(t *testing.T, ctx context.Context, tx pgx.Tx, resultID int64) *models.MonitorResult {
	t.Helper()

	row := tx.QueryRow(ctx, `
		select id, monitor_id, citations, created_at, correction, headline, subtitle, data, hidden
		from monitor_results
		where id = $1
	`, resultID)

	var result models.MonitorResult
	err := row.Scan(
		&result.ID,
		&result.MonitorID,
		&result.Citations,
		&result.CreatedAt,
		&result.Correction,
		&result.Headline,
		&result.Subtitle,
		&result.Data,
		&result.Hidden,
	)
	require.NoError(t, err)

	return &result
}

func countMonitorChecks(t *testing.T, ctx context.Context, tx pgx.Tx, monitorID int64) int {
	t.Helper()

	var count int
	err := tx.QueryRow(ctx, "select count(*) from monitor_checks where monitor_id = $1", monitorID).Scan(&count)
	require.NoError(t, err)
	return count
}

func countRiverJobsByKind(t *testing.T, ctx context.Context, tx pgx.Tx, kind string) int {
	t.Helper()

	var count int
	err := tx.QueryRow(ctx, "select count(*) from river_job where kind = $1 and finalized_at is null and state != 'cancelled'", kind).Scan(&count)
	require.NoError(t, err)
	return count
}

func latestMonitorCheck(t *testing.T, ctx context.Context, tx pgx.Tx, monitorID int64) *models.MonitorCheck {
	t.Helper()

	row := tx.QueryRow(ctx, `
		select id, monitor_id, status, scheduled_for, failure_reason, done_at, result
		from monitor_checks
		where monitor_id = $1
		order by scheduled_for desc, id desc
		limit 1
	`, monitorID)

	var check models.MonitorCheck
	err := row.Scan(
		&check.ID,
		&check.MonitorID,
		&check.Status,
		&check.ScheduledFor,
		&check.FailureReason,
		&check.DoneAt,
		&check.Result,
	)
	require.NoError(t, err)

	return &check
}

func countActiveCheckRiverJobs(t *testing.T, ctx context.Context, tx pgx.Tx, monitorID int64) int {
	t.Helper()

	var count int
	err := tx.QueryRow(ctx, `
		select count(*)
		from river_job rj
		join monitor_checks mc on rj.args->>'monitor_check_id' = mc.id::text
		where rj.kind = 'check'
		and rj.finalized_at is null
		and rj.state != 'cancelled'
		and mc.monitor_id = $1
	`, monitorID).Scan(&count)
	require.NoError(t, err)
	return count
}
