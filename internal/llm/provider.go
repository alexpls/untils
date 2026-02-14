package llm

import (
	"context"
	"encoding/json"
)

type MessageRole string

const (
	MessageRoleSystem    MessageRole = "system"
	MessageRoleUser      MessageRole = "user"
	MessageRoleAssistant MessageRole = "assistant"
	MessageRoleTool      MessageRole = "tool"
)

type Message struct {
	Role       MessageRole
	Content    string
	ToolCall   *ToolCall
	ToolOutput *ToolOutput
}

type ToolCall struct {
	ID        string
	Name      string
	Arguments string
}

type ToolOutput struct {
	CallID string
	Output string
}

type ToolDefinition struct {
	Name        string
	Description string
	Parameters  map[string]any
}

type CompletionRequest struct {
	Model             string
	Messages          []Message
	ResponseName      string
	ResponseSchema    map[string]any
	Tools             []ToolDefinition
	ParallelToolCalls bool
}

type TokenUsage struct {
	InputTokens  int64
	OutputTokens int64
	ExtraFields  map[string]json.RawMessage
}

type CompletionResponse struct {
	Raw       json.RawMessage
	Output    string
	ToolCalls []ToolCall
	Usage     TokenUsage
}

type Provider interface {
	Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error)
	CalculateCostUSD(model string, usage TokenUsage) (float64, error)
}
