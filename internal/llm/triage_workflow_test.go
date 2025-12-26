//go:build integration

package llm

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/stretchr/testify/require"
)

func TestTriageWorkflow(t *testing.T) {
	oai := openai.NewClient(
		option.WithAPIKey(os.Getenv("XAI_KEY")),
		option.WithBaseURL("https://api.x.ai/v1"),
	)

	ctx := context.Background()
	svc := NewService(&oai, slog.Default())
	triage := NewTriageWorkflow(svc)
	res, err := triage.Run(ctx, &TriageParams{
		Subject: "Who is the president of the United States?",
	})
	require.NoError(t, err)

	t.Logf("Check: %+v", res.Check)
	t.Logf("Triager: %+v", res.Triager)
}
