package llm

import (
	"context"

	"github.com/alexpls/untils/internal/logging"
	"github.com/alexpls/untils/internal/models"
)

type CheckWorkflow struct {
	service *Service
}

func (s *Service) NewCheckWorkflow() *CheckWorkflow {
	return &CheckWorkflow{service: s}
}

func (w *CheckWorkflow) Run(ctx context.Context, params *CheckParams) (*models.CheckResultWithSchema, error) {
	llmEvent, _ := logging.GetOrCreateFromContext(ctx, newLLMEvent)
	defer llmEvent.finish()

	checker := newChecker(w.service)

	return checker.perform(ctx, params)
}
