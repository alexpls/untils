package llm

import "context"

type CheckWorkflow struct {
	service *Service
}

func NewCheckWorkflow(service *Service) *CheckWorkflow {
	return &CheckWorkflow{service: service}
}

type CheckWorkflowParams struct {
	*CheckParams
	ExpertName string
}

func (w *CheckWorkflow) Run(parentCtx context.Context, params *CheckWorkflowParams) (*CheckResponse, error) {
	expert := newExpert(params.ExpertName, w.service)
	ctx, stats := withStatsContext(parentCtx)
	defer stats.log(w.service.logger)
	return expert.performCheck(ctx, params.CheckParams)
}
