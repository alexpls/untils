package llm

import "context"

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
	triager := NewTriager(w.service)
	triageResp, err := triager.Run(ctx, params)
	if err != nil {
		return nil, err
	}

	expert := NewExpert(triageResp.ChosenExpert, w.service)
	checkResp, err := expert.PerformCheck(ctx, &CheckParams{
		Subject:        params.Subject,
		Instructions:   "", // TODO include from monitor
		PreviousResult: "",
	})
	if err != nil {
		return nil, err
	}

	return &TriageWorkflowReponse{
		Triager: triageResp,
		Check:   checkResp,
	}, nil
}
