//go:build integration

package llm

import (
	"context"
	"testing"
	"time"

	"github.com/alexpls/untils/internal/logging"
	"github.com/alexpls/untils/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckerEasySubjectWithoutSchema(t *testing.T) {
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
		Subject:        "Latest album by Tool (use wikipedia, include schema fields 'Title', 'Release date', and 'Link')",
		Schema:         models.MonitorSchemaData{}, // intentionally zero
	})
	require.NoError(t, err)
	assert.Equal(t, string(models.MonitorSchemaFieldTypeText), res.Schema.Fields.GetValue("Title"))
	assert.Equal(t, string(models.MonitorSchemaFieldTypeDate), res.Schema.Fields.GetValue("Release date"))
	assert.Equal(t, string(models.MonitorSchemaFieldTypeURL), res.Schema.Fields.GetValue("Link"))
	require.Len(t, res.Updates, 1)

	update := res.Updates[0]

	assert.Equal(t, "Fear Inoculum", update.Fields.GetValue("Title"))
	assert.Equal(t, "2019-08-30", update.Fields.GetValue("Release date"))
	assert.Contains(t, []string{
		"https://en.wikipedia.org/wiki/Fear_Inoculum",
		"https://en.wikipedia.org/wiki/Fear_Inoculum_(album)",
	}, update.Fields.GetValue("Link"))
}

func TestCheckerEasySubjectWithSchema(t *testing.T) {
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
		Schema: models.MonitorSchemaData{
			Fields: models.MonitorSchemaFields{
				{
					Type: "text",
					Name: "Album name",
				},
				{
					Type: "date",
					Name: "Release date",
				},
				{
					Type: "url",
					Name: "Link",
				},
			},
		},
	})
	require.NoError(t, err)
	require.Len(t, res.Updates, 1)

	update := res.Updates[0]

	assert.Equal(t, "Fear Inoculum", update.Fields.GetValue("Album name"))
	assert.Equal(t, "2019-08-30", update.Fields.GetValue("Release date"))
	assert.Contains(t, []string{
		"https://en.wikipedia.org/wiki/Fear_Inoculum",
		"https://en.wikipedia.org/wiki/Fear_Inoculum_(album)",
	}, update.Fields.GetValue("Link"))
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
