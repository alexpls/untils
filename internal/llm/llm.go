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

type TriageWorkflowRunner interface {
	Run(context.Context, *CheckParams) (*TriagerResponse, error)
}

type CheckWorkflowRunner interface {
	Run(context.Context, *CheckParams) (*models.CheckResultWithSchema, error)
}

type Service struct {
	provider          Provider
	model             string
	db                db.DB
	queries           *models.Queries
	logger            *slog.Logger
	webSearcher       search.WebSearcher
	newBrowserSession func(ctx context.Context) (browser.BrowserSession, context.CancelFunc, error)
}

func NewService(
	provider Provider,
	model string,
	db db.DB,
	queries *models.Queries,
	logger *slog.Logger,
	webSearcher search.WebSearcher,
	newBrowserSession func(ctx context.Context) (browser.BrowserSession, context.CancelFunc, error),
) *Service {
	return &Service{
		provider:          provider,
		model:             model,
		db:                db,
		queries:           queries,
		logger:            logger,
		webSearcher:       webSearcher,
		newBrowserSession: newBrowserSession,
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
