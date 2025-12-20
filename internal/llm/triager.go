package llm

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"slices"
	"time"

	"github.com/openai/openai-go/v3/responses"
)

type Triager struct {
	service *Service
}

func NewTriager(service *Service) *Triager {
	return &Triager{service: service}
}

type TriageParams struct {
	Subject string
}

//go:embed triager_prompt.md
var triagerPrompt string

type TriagerResponse struct {
	Approved         bool   `json:"approved"`
	ChosenExpert     string `json:"chosen_expert"`
	RephrasedSubject string `json:"rephrased_subject"`
	RejectedReason   string `json:"rejected_reason"`
}

func (p *Triager) Run(ctx context.Context, params *TriageParams) (*TriagerResponse, error) {
	messages := inputItems(
		systemMessage(triagerPrompt+expertsMarkdown),
		userMessage(params.Subject),
	)

	var resp *responses.Response
	var err error
	try := 0
	maxTries := 3

	for {
		if try >= maxTries {
			return nil, fmt.Errorf("max tries reached for triage prompt: %w", err)
		}
		try++

		resp, err = p.service.response(ctx, responses.ResponseNewParams{
			Model: model,
			Input: messages,
			Text:  jsonSchemaResponse("triage_prompt_response", TriagerResponse{}),
		})

		if err != nil {
			time.Sleep(time.Duration(try*500) * time.Millisecond)
			continue
		}

		sanitized := sanitizeXAIOutput(resp.OutputText())
		res := TriagerResponse{}
		if err = json.Unmarshal([]byte(sanitized), &res); err != nil {
			messages.OfInputItemList = append(messages.OfInputItemList, systemMessage(fmt.Sprintf(
				"The output was not valid JSON: %s. Ensure your response follows the correct JSON schema.",
				err.Error(),
			)))
			continue
		}

		if !slices.Contains(expertNames, res.ChosenExpert) {
			err = fmt.Errorf("invalid expert chosen: %s", res.ChosenExpert)
			messages.OfInputItemList = append(messages.OfInputItemList, systemMessage(fmt.Sprintf(
				"The chosen expert '%s' is not valid. Valid experts are: %v. Choose a valid expert.",
				res.ChosenExpert, experts,
			)))
			continue
		}

		return &res, nil
	}
}
