package llm

import (
	"context"

	"github.com/alexpls/untils/internal/db"
	"github.com/alexpls/untils/internal/logging"
	"github.com/alexpls/untils/internal/models"
)

type CheckWorkflow struct {
	service *Service
	db      db.DB
	queries *models.Queries
}

func NewCheckWorkflow(service *Service, pool db.DB, queries *models.Queries) *CheckWorkflow {
	return &CheckWorkflow{service: service, db: pool, queries: queries}
}

func (w *CheckWorkflow) Run(ctx context.Context, params *CheckParams) (*models.CheckResult, error) {
	llmEvent, _ := logging.GetOrCreateFromContext(ctx, newLLMEvent)
	defer llmEvent.finish()

	checker := newChecker(w.service, w.db, w.queries)

	return checker.perform(ctx, params)
}
