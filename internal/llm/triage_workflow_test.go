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

func TestTriageWorkflow(t *testing.T) {
	oai := openai.NewClient(
		option.WithAPIKey(os.Getenv("XAI_KEY")),
		option.WithBaseURL("https://api.x.ai/v1"),
	)

	tl := testhelper.TestLogger(t)

	ctx := context.Background()
	svc := NewService(&oai, tl)
	triage := NewTriageWorkflow(svc)
	res, err := triage.Run(ctx, &TriageParams{
		Subject: "Latest games that IGN has given a 9/10 or higher rating",
	})
	require.NoError(t, err)

	t.Logf("Check: %+v", res.Check)
	t.Logf("Triager: %+v", res.Triager)
}
