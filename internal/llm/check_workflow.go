package llm

import (
	"context"

	"github.com/alexpls/untils/internal/logging"
	"github.com/alexpls/untils/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CheckWorkflow struct {
	service *Service
	pool    *pgxpool.Pool
	queries *models.Queries
}

func NewCheckWorkflow(service *Service, pool *pgxpool.Pool, queries *models.Queries) *CheckWorkflow {
	return &CheckWorkflow{service: service, pool: pool, queries: queries}
}

func (w *CheckWorkflow) Run(ctx context.Context, params *CheckParams) (*models.CheckResult, error) {
	llmEvent, _ := logging.GetOrCreateFromContext(ctx, newLLMEvent)
	defer llmEvent.finish()

	checker := newChecker(w.service, w.pool, w.queries)

	return checker.perform(ctx, params)
}
