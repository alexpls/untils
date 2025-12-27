package llm

import (
	"context"
	"fmt"
)

type TriageWorkflow struct {
	service *Service
}

func NewTriageWorkflow(service *Service) *TriageWorkflow {
	return &TriageWorkflow{service: service}
}

type TriageWorkflowReponse struct {
	Triager *TriagerResponse
	Check   *CheckResult
}

func (w *TriageWorkflow) Run(parentCtx context.Context, params *TriageParams) (*TriageWorkflowReponse, error) {
	ctx, stats := withStatsContext(parentCtx)
	defer stats.log(w.service.logger)

	lg := w.service.logger.With("workflow", "triage")

	maxTurns := 3
	turn := 0

	var err error
	var triageResp *TriagerResponse
	var checkResp *CheckResult
	triager := NewTriager(w.service, params)

	for {
		if turn >= maxTurns {
			return nil, fmt.Errorf("max turns reached in triage workflow: %w", err)
		}
		turn++

		lg.Info("running triager")

		triageResp, err = triager.Run(ctx)
		if err != nil {
			return nil, err
		}
		if !triageResp.Approved {
			lg.Warn("triage not approved", "reason", triageResp.RejectedReason)
			return &TriageWorkflowReponse{
				Triager: triageResp,
				Check:   checkResp,
			}, nil
		}

		lg.Info("triager approved")

		subject := triageResp.RephrasedSubject
		if subject == "" {
			subject = params.Subject
		}

		checkParams := &CheckParams{
			Subject:      subject,
			Instructions: params.Instructions,
		}

		lg.Info("finding sources")
		sourceFinder := newSourceFinder(w.service)
		sourcesResp, err := sourceFinder.Run(ctx, checkParams)
		if err != nil {
			lg.Error("error finding sources", "error", err)
			return nil, err
		}

		lg.Info("sources found", "sources", sourcesResp)

		checkParams.Sources = sourcesResp.Sources

		lg.Info("checking")
		expert := newExpert(triageResp.ChosenExpert, w.service)
		checkResp, err = expert.performCheck(ctx, checkParams)
		if err != nil {
			lg.Error("error performing check", "error", err)
			return nil, err
		}

		if !checkResp.Success {
			lg.Warn("check unsuccessful, sending back")

			feedback := "The expert couldn't answer. Try finding some different sources or picking another expert."
			if checkResp.Reason != "" {
				feedback += fmt.Sprintf(" Reason: %s", checkResp.Reason)
			}
			triager.addMessage(systemMessage(feedback))
			continue
		}

		lg.Info("check successful")

		return &TriageWorkflowReponse{
			Triager: triageResp,
			Check:   checkResp,
		}, nil
	}
}
