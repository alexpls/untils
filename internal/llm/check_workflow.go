package llm

import (
	"context"

	"github.com/alexpls/untils_go/internal/wideevents"
)

type CheckWorkflow struct {
	service *Service
}

func NewCheckWorkflow(service *Service) *CheckWorkflow {
	return &CheckWorkflow{service: service}
}

func (w *CheckWorkflow) Run(ctx context.Context, params *CheckParams) (*CheckResult, error) {
	llmEvent, _ := wideevents.GetOrCreateFromContext(ctx, newLLMEvent)
	defer llmEvent.finish()

	checker := newChecker(w.service)
	return checker.perform(ctx, params)
}
