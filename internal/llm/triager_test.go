//go:build integration

package llm

import (
	"log/slog"
	"os"
	"testing"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/stretchr/testify/require"
)

func TestTriager(t *testing.T) {
	oai := openai.NewClient(
		option.WithAPIKey(os.Getenv("XAI_KEY")),
		option.WithBaseURL("https://api.x.ai/v1"),
	)

	ctx, _ := withStatsContext(t.Context())
	svc := NewService(&oai, slog.Default())
	prompt := NewTriager(svc, &TriageParams{
		Subject: "Who is a good boy?",
	})
	res, err := prompt.Run(ctx)
	require.NoError(t, err)

	t.Logf("output: %+v", res)
}
