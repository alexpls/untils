//go:build integration

package llm

import (
	"os"
	"testing"

	"github.com/alexpls/untils_go/internal/testhelper"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/stretchr/testify/require"
)

func TestSourceFinder(t *testing.T) {
	oai := openai.NewClient(
		option.WithAPIKey(os.Getenv("XAI_KEY")),
		option.WithBaseURL("https://api.x.ai/v1"),
	)

	tl := testhelper.TestLogger(t)

	ctx, stats := withStatsContext(t.Context())
	defer stats.log(tl)

	svc := NewService(&oai, tl)
	finder := newSourceFinder(svc)

	res, err := finder.Run(ctx, &CheckParams{
		Subject: "IGN games with a rating of 9/10 or above",
	})
	require.NoError(t, err)

	tl.Info("found sources", "sources", res.Sources)
}
