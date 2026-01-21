package llm

import (
	"os"
	"testing"

	"github.com/alexpls/untils/internal/models"
	"github.com/alexpls/untils/internal/search"
	"github.com/alexpls/untils/internal/testhelper"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

type testDeps struct {
	service *Service
	pool    *pgxpool.Pool
	queries *models.Queries
}

func newTestDeps(t *testing.T) *testDeps {
	t.Helper()

	tl := testhelper.TestLogger(t)
	pool := testhelper.TestDB(t)
	queries := models.New()

	oai := openai.NewClient(
		option.WithAPIKey(os.Getenv("XAI_KEY")),
		option.WithBaseURL("https://api.x.ai/v1"),
	)

	ws := search.NewBraveClient(os.Getenv("BRAVE_KEY"), tl)

	svc := NewService(&oai, tl, ws)

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
