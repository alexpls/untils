package llm

import (
	"context"
	"fmt"

	"github.com/alexpls/untils_go/internal/wideevents"
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

func (w *TriageWorkflow) Run(ctx context.Context, params *TriageParams) (*TriageWorkflowReponse, error) {
	llmEvent, _ := wideevents.GetOrCreateFromContext(ctx, newLLMEvent)
	defer llmEvent.finish()

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

		triageResp, err = triager.Run(ctx)
		if err != nil {
			return nil, err
		}
		if !triageResp.Approved {
			return &TriageWorkflowReponse{
				Triager: triageResp,
				Check:   checkResp,
			}, nil
		}

		checkParams := &CheckParams{
			Subject:      params.Subject,
			Instructions: params.Instructions,
		}

		checker := newChecker(w.service)
		checkResp, err = checker.perform(ctx, checkParams)
		if err != nil {
			lg.Error("error performing check", "error", err)
			return nil, err
		}

		if !checkResp.Success {
			feedback := "The expert couldn't answer. Try finding some different sources or picking another expert."
			if checkResp.Reason != "" {
				feedback += fmt.Sprintf(" Reason: %s", checkResp.Reason)
			}
			triager.addMessage(systemMessage(feedback))
			continue
		}

		return &TriageWorkflowReponse{
			Triager: triageResp,
			Check:   checkResp,
		}, nil
	}
}
