//go:build integration

package llm

import (
	"testing"

	"github.com/alexpls/untils/internal/wideevents"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckerEasySubject(t *testing.T) {
	svc := newServiceForTest(t)

	ch := make(EventsChan)
	defer close(ch)

	checker := newChecker(svc, ch)

	go func() {
		for range ch {
			// draining the channel
		}
	}()

	events := make(wideevents.Events)
	ctx := wideevents.ContextWithEvents(t.Context(), events)
	llmEvent := wideevents.GetOrCreate(events, newLLMEvent)
	defer llmEvent.finish()

	res, err := checker.perform(ctx, &CheckParams{
		Subject:      "Latest album by Tool",
		Instructions: "Use https://en.wikipedia.org/wiki/Tool_discography",
	})
	require.NoError(t, err)

	assert.Contains(t, res.ResultPlaintext, "Fear Inoculum")
	assert.Len(t, res.Citations, 1)
	assert.Contains(t, res.Citations[0].URL, "wikipedia.org")
	assert.Contains(t, res.Citations[0].FaviconURL, "wikipedia.org/static/favicon/wikipedia.ico")
}
