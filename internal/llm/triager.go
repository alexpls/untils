package llm

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/alexpls/untils/internal/models"
)

type Triager struct {
	service      *Service
	conversation *dbConversation
	params       *CheckParams
}

func NewTriager(service *Service, params *CheckParams) *Triager {
	return &Triager{
		service:      service,
		conversation: newDBConversation(service),
		params:       params,
	}
}

//go:embed triager_prompt.md
var triagerPrompt string

type TriagerResponse struct {
	Approved       bool   `json:"approved"`
	RejectedReason string `json:"rejected_reason"`
}

func (p *Triager) Run(ctx context.Context) (*TriagerResponse, error) {
	if err := p.conversation.start(ctx, p.params.UserID, p.params.MonitorID, models.LlmConversationsSourceTriage); err != nil {
		return nil, err
	}
	if err := p.conversation.addSystem(ctx, triagerPrompt); err != nil {
		return nil, fmt.Errorf("failed to log system message: %w", err)
	}
	if err := p.conversation.addUser(ctx, p.params.UserMessageString()); err != nil {
		return nil, fmt.Errorf("failed to log user message: %w", err)
	}

	res, err := runAgent[TriagerResponse](ctx, p.service, agentRunOptions[TriagerResponse]{
		model:          modelNonReasoning,
		responseName:   "TriagerResponse",
		responseSchema: jsonSchema(TriagerResponse{}),
		maxTurns:       3,
		conversation:   p.conversation,
	})
	if err != nil {
		return nil, fmt.Errorf("max tries reached for triage prompt: %w", err)
	}

	return res, nil
}
