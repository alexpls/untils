package llm

import "context"

type CheckWorkflow struct {
	service *Service
}

func NewCheckWorkflow(service *Service) *CheckWorkflow {
	return &CheckWorkflow{service: service}
}

func (w *CheckWorkflow) Run(parentCtx context.Context, params *CheckParams) (*CheckResult, error) {
	ctx, stats := withStatsContext(parentCtx)
	defer stats.log(w.service.logger)

	checker := newChecker(w.service)
	return checker.perform(ctx, params)
}
