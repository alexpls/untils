//go:build integration

package llm

import (
	"testing"
	"time"

	"github.com/alexpls/untils/internal/models"
	"github.com/alexpls/untils/internal/wideevents"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckerEasySubject(t *testing.T) {
	deps := newTestDeps(t)
	ctx := t.Context()

	fixtures := testFixtures(t, deps)
	user := fixtures.user
	check := fixtures.check

	ch := make(EventsChan)
	defer close(ch)

	checker := newChecker(deps.service, ch, deps.pool, deps.queries)

	go func() {
		for range ch {
			// draining the channel
		}
	}()

	events := make(wideevents.Events)
	ctx = wideevents.ContextWithEvents(ctx, events)
	llmEvent := wideevents.GetOrCreate(events, newLLMEvent)
	defer llmEvent.finish()

	res, err := checker.perform(ctx, &CheckParams{
		UserID:         user.ID,
		MonitorCheckID: check.ID,
		Subject:        "Latest album by Tool",
	})
	require.NoError(t, err)

	assert.Contains(t, res.ResultPlaintext, "Fear Inoculum")
	assert.Len(t, res.Citations, 1)
	assert.Contains(t, res.Citations[0].URL, "wikipedia.org")
	assert.Contains(t, res.Citations[0].FaviconURL, "wikipedia.org/static/favicon/wikipedia.ico")
}

type fixtures struct {
	user  *models.User
	check *models.MonitorCheck
}

func testFixtures(t *testing.T, deps *testDeps) fixtures {
	t.Helper()
	ctx := t.Context()

	user, err := deps.queries.CreateUser(ctx, deps.pool, &models.CreateUserParams{
		Email:        "test@example.com",
		PasswordHash: "hash",
		Timezone:     "UTC",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	})
	require.NoError(t, err)

	monitor, err := deps.queries.CreateMonitor(ctx, deps.pool, &models.CreateMonitorParams{
		UserID:  user.ID,
		Subject: pgtype.Text{String: "Latest album by Tool", Valid: true},
	})
	require.NoError(t, err)

	check, err := deps.queries.CreateMonitorCheck(ctx, deps.pool, &models.CreateMonitorCheckParams{
		MonitorID:    monitor.ID,
		Status:       models.MonitorCheckStatusScheduled,
		ScheduledFor: time.Now(),
	})
	require.NoError(t, err)

	return fixtures{
		user:  user,
		check: check,
	}
}
