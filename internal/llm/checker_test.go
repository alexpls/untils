//go:build integration

package llm

import (
	"context"
	"testing"
	"time"

	"github.com/alexpls/untils/internal/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckerEasySubject(t *testing.T) {
	t.Parallel()

	deps := newTestDeps(t)
	ctx := t.Context()

	checker := newChecker(deps.service)

	events := make(logging.Events)
	ctx = logging.ContextWithEvents(ctx, events)
	llmEvent := logging.GetOrCreate(events, newLLMEvent)
	defer llmEvent.finish()

	res, err := checker.perform(ctx, &CheckParams{
		UserID:         deps.fixtures.User.ID,
		MonitorCheckID: deps.fixtures.Check.ID,
		Subject:        "Latest album by Tool (use wikipedia)",
	})
	require.NoError(t, err)

	assert.Contains(t, res.ResultPlaintext, "Fear Inoculum")
	assert.Contains(t, res.Citations[0].URL, "wikipedia.org")
	assert.Contains(t, res.Citations[0].FaviconURL, "wikipedia.org/static/favicon/wikipedia.ico")
}

func TestCheckerContextCancellation(t *testing.T) {
	t.Parallel()

	deps := newTestDeps(t)
	checker := newChecker(deps.service)

	ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancel()

	res, err := checker.perform(ctx, &CheckParams{
		UserID:         deps.fixtures.User.ID,
		MonitorCheckID: deps.fixtures.Check.ID,
		Subject:        "Latest album by Tool",
	})

	assert.Nil(t, res)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}
