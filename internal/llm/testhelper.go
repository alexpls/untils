package llm

import (
	"os"
	"testing"

	"github.com/alexpls/untils/internal/search"
	"github.com/alexpls/untils/internal/testhelper"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

func newServiceForTest(t *testing.T) *Service {
	t.Helper()

	tl := testhelper.TestLogger(t)

	oai := openai.NewClient(
		option.WithAPIKey(os.Getenv("XAI_KEY")),
		option.WithBaseURL("https://api.x.ai/v1"),
	)

	ws := search.NewBraveClient(os.Getenv("BRAVE_KEY"), tl)

	svc := NewService(&oai, tl, ws)

	return svc
}
