//go:build integration

package llm

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTriager(t *testing.T) {
	svc := newServiceForTest(t)
	ctx, _ := withStatsContext(t.Context())
	prompt := NewTriager(svc, &TriageParams{
		Subject: "Who is a good boy?",
	})
	res, err := prompt.Run(ctx)
	require.NoError(t, err)

	t.Logf("output: %+v", res)
}
