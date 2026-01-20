package llm

import (
	"context"

	"github.com/alexpls/untils/internal/models"
	"github.com/alexpls/untils/internal/wideevents"
)

type CheckWorkflow struct {
	service *Service
	Chan    EventsChan
}

type EventsChan chan CheckEvent

type CheckEvent struct {
	Kind    models.MonitorCheckEventKind
	Details models.MonitorCheckEventDetails
}

func NewCheckWorkflow(service *Service, ch EventsChan) *CheckWorkflow {
	return &CheckWorkflow{service: service, Chan: ch}
}

func (w *CheckWorkflow) Run(ctx context.Context, params *CheckParams) (*models.CheckResult, error) {
	llmEvent, _ := wideevents.GetOrCreateFromContext(ctx, newLLMEvent)
	defer llmEvent.finish()

	checker := newChecker(w.service, w.Chan)

	return checker.perform(ctx, params)
}
