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
	Check   *CheckResponse
}

func (w *TriageWorkflow) Run(ctx context.Context, params *TriageParams) (*TriageWorkflowReponse, error) {
	maxTurns := 3
	turn := 0

	var err error
	var triageResp *TriagerResponse
	var checkResp *CheckResponse
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

		subject := triageResp.RephrasedSubject
		if subject == "" {
			subject = params.Subject
		}

		expert := NewExpert(triageResp.ChosenExpert, w.service)
		checkResp, err = expert.PerformCheck(ctx, &CheckParams{
			Subject:        subject,
			Instructions:   params.Instructions,
			PreviousResult: "", // first check, so no prev result
		})
		if err != nil {
			return nil, err
		}

		if !checkResp.Answered {
			feedback := "The expert couldn't answer. Try picking another expert or rephrasing the subject."
			if checkResp.RejectionReason != "" {
				feedback += fmt.Sprintf(" Rejection reason: %s", checkResp.RejectionReason)
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
