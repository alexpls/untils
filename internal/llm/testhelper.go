package llm

import (
	"context"
	"os"
	"testing"

	"github.com/alexpls/untils/internal/models"
	"github.com/alexpls/untils/internal/search"
	"github.com/alexpls/untils/internal/testhelper"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

type testDeps struct {
	service *Service
	pool    models.DBTX
	queries *models.Queries
}

func newTestDeps(t *testing.T) *testDeps {
	t.Helper()

	ctx := context.Background()
	tl := testhelper.TestLogger(t)
	pool := testhelper.TestTx(ctx, t)
	queries := models.New()

	oai := openai.NewClient(
		option.WithAPIKey(os.Getenv("XAI_KEY")),
		option.WithBaseURL("https://api.x.ai/v1"),
	)

	ws := search.NewBraveClient(os.Getenv("BRAVE_KEY"), tl)

	svc := NewService(&oai, pool, queries, tl, ws)

	return &testDeps{
		service: svc,
		pool:    pool,
		queries: queries,
	}
}

func newServiceForTest(t *testing.T) *Service {
	t.Helper()
	return newTestDeps(t).service
}
