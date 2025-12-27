//go:build integration

package llm

import (
	"context"
	"os"
	"testing"

	"github.com/alexpls/untils_go/internal/testhelper"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/stretchr/testify/require"
)

func TestCheckWorkflow(t *testing.T) {
	oai := openai.NewClient(
		option.WithAPIKey(os.Getenv("XAI_KEY")),
		option.WithBaseURL("https://api.x.ai/v1"),
	)

	tl := testhelper.TestLogger(t)

	ctx := context.Background()
	svc := NewService(&oai, tl)
	checker := NewCheckWorkflow(svc)
	res, err := checker.Run(ctx, &CheckWorkflowParams{
		ExpertName: "default",
		CheckParams: &CheckParams{
			Subject: "Latest power outages in Brisbane, QLD",
		},
	})
	require.NoError(t, err)

	t.Logf("Response: %+v", res)
}
