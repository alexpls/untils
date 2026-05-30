package llm

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/alexpls/untils/internal/browser"
	"github.com/alexpls/untils/internal/models"
	"github.com/alexpls/untils/internal/openai"
	"github.com/alexpls/untils/internal/search"
	"github.com/alexpls/untils/internal/testhelper"
	testfixtures "github.com/alexpls/untils/internal/testhelper/fixtures"
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

	opts := []openai.Option{
		openai.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
	}

	openAIBaseURL := os.Getenv("OPENAI_BASE_URL")
	if openAIBaseURL != "" {
		opts = append(opts, openai.WithBaseURL(openAIBaseURL))
	}

	oai := openai.NewClient(opts...)
	openAIModel := os.Getenv("OPENAI_MODEL")
	if openAIModel == "" {
		t.Fatal("OPENAI_MODEL is required")
	}

	ws := search.NewBraveClient(os.Getenv("BRAVE_KEY"), tl)
	browserManager := browser.NewManager(1, browser.BrowserSessionConfig{}, tl)

	svc := NewService(
		NewOpenAIProvider(oai),
		openAIModel,
		tx,
		queries,
		tl,
		ws,
		func(ctx context.Context) (browser.BrowserSession, context.CancelFunc, error) {
			return browserManager.NewSession(ctx)
		},
	)

	return &testDeps{
		service:  svc,
		tx:       tx,
		queries:  queries,
		fixtures: fixtures,
	}
}

func assertConversationDoesNotContainXAIToolCallCloseTag(
	t *testing.T,
	deps *testDeps,
	sourceType models.LLMConversationsSource,
	sourceID int64,
) {
	t.Helper()

	conversation, err := deps.queries.GetLLMConversationBySourceID(t.Context(), deps.tx, &models.GetLLMConversationBySourceIDParams{
		SourceType: sourceType,
		SourceID:   sourceID,
	})
	if err != nil {
		t.Fatalf("getting llm conversation: %v", err)
	}

	messagesJSON, err := json.Marshal(conversation.Messages)
	if err != nil {
		t.Fatalf("marshaling conversation messages: %v", err)
	}

	if strings.Contains(string(messagesJSON), "</xai:function_call>") {
		t.Fatalf("conversation contains unexpected x.ai function call close tag: %s", messagesJSON)
	}
}
