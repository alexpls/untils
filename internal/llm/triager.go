package llm

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"time"

	"github.com/alexpls/untils/internal/db/models"
	"github.com/openai/openai-go/v3/responses"
)

type Triager struct {
	service  *Service
	messages responses.ResponseNewParamsInputUnion
}

func NewTriager(service *Service, params *TriageParams) *Triager {
	msg := "Subject: " + params.Subject

	if params.PreviousResult != nil {
		d, err := json.Marshal(params.PreviousResult)
		if err != nil {
			service.logger.Error("failed to marshal previous result", "error", err)
		} else {
			msg += "\n\nPrevious result:\n" + string(d)
		}
	}

	if params.UserFeedback != "" {
		msg += "\n\nUser feedback on the previous result:\n" + params.UserFeedback
	}

	messages := inputItems(
		systemMessage(triagerPrompt),
		userMessage(msg),
	)

	return &Triager{service: service, messages: messages}
}

type TriageParams struct {
	Subject        string
	PreviousResult *models.CheckResult
	UserFeedback   string
}

//go:embed triager_prompt.md
var triagerPrompt string

type TriagerResponse struct {
	Approved       bool   `json:"approved"`
	RejectedReason string `json:"rejected_reason"`
}

func (p *Triager) Run(ctx context.Context) (*TriagerResponse, error) {
	var resp *responseResult
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
			Input: p.messages,
			Text:  jsonSchemaResponse(TriagerResponse{}),
		})

		if err != nil {
			time.Sleep(time.Duration(try*500) * time.Millisecond)
			continue
		}

		sanitized := sanitizeXAIOutput(resp.OutputText())
		res := TriagerResponse{}
		if err = json.Unmarshal([]byte(sanitized), &res); err != nil {
			p.addMessage(systemMessage(fmt.Sprintf(
				"The output was not valid JSON: %s. Ensure your response follows the correct JSON schema.",
				err.Error(),
			)))
			continue
		}

		return &res, nil
	}
}

func (p *Triager) addMessage(message responses.ResponseInputItemUnionParam) {
	p.messages.OfInputItemList = append(p.messages.OfInputItemList, message)
}
