package llm

import (
	"context"

	"github.com/alexpls/untils_go/internal/db/sqlc"
	"github.com/alexpls/untils_go/internal/wideevents"
)

type CheckWorkflow struct {
	service *Service
	Chan    EventsChan
}

type EventsChan chan CheckEvent

type CheckEvent struct {
	Kind    sqlc.MonitorCheckEventKind
	Details sqlc.MonitorCheckEventDetails
}

func NewCheckWorkflow(service *Service, ch EventsChan) *CheckWorkflow {
	return &CheckWorkflow{service: service, Chan: ch}
}

func (w *CheckWorkflow) Run(ctx context.Context, params *CheckParams) (*sqlc.CheckResult, error) {
	llmEvent, _ := wideevents.GetOrCreateFromContext(ctx, newLLMEvent)
	defer llmEvent.finish()

	checker := newChecker(w.service, w.Chan)

	return checker.perform(ctx, params)
}
