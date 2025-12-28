//go:build integration

package llm

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestChecker(t *testing.T) {
	svc := newServiceForTest(t)

	checker := newChecker(svc)

	ctx, stats := withStatsContext(t.Context())
	defer stats.log(svc.logger)

	res, err := checker.perform(ctx, &CheckParams{
		Subject: "Latest game IGN has given a 10/10 rating to",
	})
	require.NoError(t, err)

	t.Logf("Response: %+v", res)
}
