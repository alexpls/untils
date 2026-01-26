package llm

import (
	"context"

	"github.com/alexpls/untils/internal/logging"
)

type TriageWorkflow struct {
	service *Service
}

func (s *Service) NewTriageWorkflow() *TriageWorkflow {
	return &TriageWorkflow{service: s}
}

func (w *TriageWorkflow) Run(ctx context.Context, params *CheckParams) (*TriagerResponse, error) {
	llmEvent, _ := logging.GetOrCreateFromContext(ctx, newLLMEvent)
	defer llmEvent.finish()

	triager := NewTriager(w.service, params)
	return triager.Run(ctx)
}
