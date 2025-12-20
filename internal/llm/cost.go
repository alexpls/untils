package llm

import (
	"encoding/json"
	"fmt"

	"github.com/openai/openai-go/v3/responses"
)

func calculateCost(model string, response *responses.Response) (float64, error) {
	cost := 0.0

	switch model {
	case "grok-4-1-fast-reasoning":
	case "grok-4-1-fast-non-reasoning":
		// https://docs.x.ai/docs/models
		per1MInputToken, per1MOutputToken := 0.2, 0.5
		per1KSearchCalls, per1KXSearchCalls := 5.0, 5.0

		cost += float64(response.Usage.InputTokens) / 1_000_000 * per1MInputToken
		cost += float64(response.Usage.OutputTokens) / 1_000_000 * per1MOutputToken
		if raw, ok := response.Usage.JSON.ExtraFields["server_side_tool_usage_details"]; ok {
			toolUsage := struct {
				WebSearchCalls int `json:"web_search_calls"`
				XSearchCalls   int `json:"x_search_calls"`
			}{}
			if err := json.Unmarshal([]byte(raw.Raw()), &toolUsage); err != nil {
				return 0.0, fmt.Errorf("unmarshaling tool usage: %w", err)
			}
			cost += float64(toolUsage.WebSearchCalls) / 1_000 * per1KSearchCalls
			cost += float64(toolUsage.XSearchCalls) / 1_000 * per1KXSearchCalls
		}
	default:
		return 0.0, fmt.Errorf("unsupported model: %s", model)
	}

	return cost, nil
}
