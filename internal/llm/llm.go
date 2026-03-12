package llm

import (
	"context"
	"log/slog"

	"github.com/alexpls/untils/internal/browser"
	"github.com/alexpls/untils/internal/db"
	"github.com/alexpls/untils/internal/logging"
	"github.com/alexpls/untils/internal/models"
	"github.com/alexpls/untils/internal/search"
)

var modelNonReasoning = "grok-4-1-fast-non-reasoning"
var modelReasoning = "grok-4-1-fast-reasoning"

type Service struct {
	provider    Provider
	db          db.DB
	queries     *models.Queries
	logger      *slog.Logger
	webSearcher search.WebSearcher
	newBrowserCtx func(ctx context.Context) (browser.BrowserCtx, context.CancelFunc)
}

func NewService(
	provider Provider,
	db db.DB,
	queries *models.Queries,
	logger *slog.Logger,
	webSearcher search.WebSearcher,
	newBrowserCtx func(ctx context.Context) (browser.BrowserCtx, context.CancelFunc),
) *Service {
	return &Service{
		provider:    provider,
		db:          db,
		queries:     queries,
		logger:      logger,
		webSearcher: webSearcher,
		newBrowserCtx: newBrowserCtx,
	}
}

func (s *Service) response(ctx context.Context, params CompletionRequest) (*CompletionResponse, error) {
	llmEvent, _ := logging.GetOrCreateFromContext(ctx, newLLMEvent)
	turn := llmEvent.newTurn()

	defer turn.finish()

	resp, err := s.provider.Complete(ctx, params)

	if err != nil {
		turn.addError(err)
		return nil, err
	}

	cost, err := s.provider.CalculateCostUSD(params.Model, resp.Usage)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to calculate cost", "error", err)
	} else {
		turn.addCost(cost)
	}

	for _, item := range resp.ToolCalls {
		turn.incrToolCall(item.Name)
	}

	return resp, nil
}
