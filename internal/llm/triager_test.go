//go:build integration

package llm

import (
	"testing"

	"github.com/alexpls/untils/internal/logging"
	"github.com/stretchr/testify/require"
)

func TestTriager(t *testing.T) {
	t.Parallel()

	deps := newTestDeps(t)

	events := make(logging.Events)
	ctx := logging.ContextWithEvents(t.Context(), events)

	prompt := NewTriager(deps.service, &CheckParams{
		Subject: "Who is a good boy?",
	})
	res, err := prompt.Run(ctx)
	require.NoError(t, err)

	t.Logf("output: %+v", res)
}
