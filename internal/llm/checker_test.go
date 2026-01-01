//go:build integration

package llm

import (
	"log/slog"
	"testing"

	"github.com/alexpls/untils_go/internal/wideevents"
	"github.com/stretchr/testify/require"
)

func TestChecker(t *testing.T) {
	svc := newServiceForTest(t)

	ch := make(EventsChan)
	defer close(ch)

	checker := newChecker(svc, ch)

	go func() {
		for ev := range ch {
			t.Logf("Check event: kind=%s details=%+v", ev.Kind, ev.Details)
		}
	}()

	events := make(wideevents.Events)
	ctx := wideevents.ContextWithEvents(t.Context(), events)
	llmEvent := wideevents.GetOrCreate(events, newLLMEvent)

	defer func() {
		llmEvent.finish()
		svc.logger.LogAttrs(ctx, slog.LevelInfo, "llm workflow complete", events.SlogAttrs()...)
	}()

	res, err := checker.perform(ctx, &CheckParams{
		Subject: "Latest game IGN has given a 10/10 rating to",
	})
	require.NoError(t, err)

	t.Logf("Response: %+v", res)
}
