package llm

import (
	"context"
	"fmt"
)

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

func (w *CheckWorkflow) Run(parentCtx context.Context, params *CheckWorkflowParams) (*CheckResult, error) {
	ctx, stats := withStatsContext(parentCtx)
	defer stats.log(w.service.logger)

	sourceFinder := newSourceFinder(w.service)
	sourcesResp, err := sourceFinder.Run(ctx, params.CheckParams)
	if err != nil {
		return nil, fmt.Errorf("finding sources: %w", err)
	}

	params.Sources = sourcesResp.Sources

	expert := newExpert(params.ExpertName, w.service)
	return expert.performCheck(ctx, params.CheckParams)
}
