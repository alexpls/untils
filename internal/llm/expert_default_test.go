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

func TestExpertDefaultPerformCheck(t *testing.T) {
	oai := openai.NewClient(
		option.WithAPIKey(os.Getenv("XAI_KEY")),
		option.WithBaseURL("https://api.x.ai/v1"),
	)

	ctx := context.Background()
	svc := llm.NewService(&oai, slog.Default())
	expert := llm.NewExpertDefault(svc)
	res, err := expert.PerformCheck(ctx, &llm.CheckParams{
		Subject:        "Current power outages in Birkdale, Queensland",
		Instructions:   "",
		PreviousResult: "",
	})
	require.NoError(t, err)

	t.Logf("output: %+v", res)
}
