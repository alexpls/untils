package llm

import (
	"context"
	"os"
	"testing"

	"github.com/alexpls/untils/internal/browser"
	"github.com/alexpls/untils/internal/models"
	"github.com/alexpls/untils/internal/search"
	"github.com/alexpls/untils/internal/testhelper"
	testfixtures "github.com/alexpls/untils/internal/testhelper/fixtures"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

type testDeps struct {
	service  *Service
	tx       models.DBTX
	queries  *models.Queries
	fixtures testfixtures.Fixtures
}

func newTestDeps(t *testing.T) *testDeps {
	t.Helper()

	ctx := context.Background()
	tl := testhelper.TestLogger(t)
	tx := testhelper.TestTx(ctx, t)
	queries := models.New()
	fixtures := testfixtures.New(ctx, t, tx, queries)

	oai := openai.NewClient(
		option.WithAPIKey(os.Getenv("XAI_KEY")),
		option.WithBaseURL("https://api.x.ai/v1"),
	)

	ws := search.NewBraveClient(os.Getenv("BRAVE_KEY"), tl)

	svc := NewService(
		NewOpenAIProvider(&oai),
		tx,
		queries,
		tl,
		ws,
		func(ctx context.Context) (browser.BrowserCtx, context.CancelFunc) {
			return browser.NewBrowser(ctx, browser.BrowserConfig{}, tl)
		},
	)

	return &testDeps{
		service:  svc,
		tx:       tx,
		queries:  queries,
		fixtures: fixtures,
	}
}
