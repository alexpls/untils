//go:build integration

package llm_test

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/alexpls/untils_go/internal/llm"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/stretchr/testify/require"
)

func TestTriager(t *testing.T) {
	oai := openai.NewClient(
		option.WithAPIKey(os.Getenv("XAI_KEY")),
		option.WithBaseURL("https://api.x.ai/v1"),
	)

	ctx := context.Background()
	svc := llm.NewService(&oai, slog.Default())
	prompt := llm.NewTriager(svc)
	res, err := prompt.Run(ctx, &llm.TriageParams{
		Subject: "Who is a good boy?",
	})
	require.NoError(t, err)

	t.Logf("output: %+v", res)
}
