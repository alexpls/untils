package llm

import (
	"context"

	"github.com/alexpls/untils/internal/models"
	"github.com/alexpls/untils/internal/wideevents"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CheckWorkflow struct {
	service *Service
	Chan    EventsChan
	pool    *pgxpool.Pool
	queries *models.Queries
}

type EventsChan chan CheckEvent

type CheckEvent struct {
	Kind    models.MonitorCheckEventKind
	Details models.MonitorCheckEventDetails
}

func NewCheckWorkflow(service *Service, ch EventsChan, pool *pgxpool.Pool, queries *models.Queries) *CheckWorkflow {
	return &CheckWorkflow{service: service, Chan: ch, pool: pool, queries: queries}
}

func (w *CheckWorkflow) Run(ctx context.Context, params *CheckParams) (*models.CheckResult, error) {
	llmEvent, _ := wideevents.GetOrCreateFromContext(ctx, newLLMEvent)
	defer llmEvent.finish()

	checker := newChecker(w.service, w.Chan, w.pool, w.queries)

	return checker.perform(ctx, params)
}
