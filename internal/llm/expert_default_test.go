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

func TestExpertDefaultUseBrowserNavigateTool(t *testing.T) {
	oai := openai.NewClient(
		option.WithAPIKey(os.Getenv("XAI_KEY")),
		option.WithBaseURL("https://api.x.ai/v1"),
	)

	ctx := context.Background()
	svc := NewService(&oai, slog.Default())
	expert := newExpertDefault(svc)
	res, err := expert.performCheck(ctx, &CheckParams{
		Subject:      "Current power outages in QLD",
		Instructions: "You must check this URL by navigating to the page: https://www.energex.com.au/outages/outage-finder/emergency-outages-text-view/?council=Brisbane%20City&startSuburb=all&suburb=",
	})
	require.NoError(t, err)

	t.Logf("output: %+v", res)
}
