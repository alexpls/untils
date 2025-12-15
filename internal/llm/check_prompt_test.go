//go:build integration

package llm_test

import (
	"log/slog"
	"os"
	"testing"

	"github.com/alexpls/untils_go/internal/llm"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/stretchr/testify/require"
)

func TestCheckPrompt(t *testing.T) {
	oai := openai.NewClient(
		option.WithAPIKey(os.Getenv("XAI_KEY")),
		option.WithBaseURL("https://api.x.ai/v1"),
	)
	s := llm.NewService(&oai, slog.Default())
	res, err := s.CheckPrompt(t.Context(), llm.CheckPromptParams{
		Subject: "Latest documentary by Adam Curtis",
	})
	require.NoError(t, err)

	t.Logf("output: %+v", res)
}
