package llm

import (
	"encoding/json"
	"fmt"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/responses"
)

const (
	browserNavigateToolName = "browser_navigate"
	// intentionally different to "web_search" to avoid name collisions with OAI/x.ai
	searchToolName = "search_request"
)

type browserNavigateToolParams struct {
	URL string `json:"url"`
}

type searchToolParams struct {
	Query string `json:"string"`
}

func browserTools() []responses.ToolUnionParam {
	return []responses.ToolUnionParam{{
		OfFunction: &responses.FunctionToolParam{
			Name:        browserNavigateToolName,
			Description: openai.String("Use a web browser to navigate to the given URL and retrieve the page contents"),
			Parameters:  jsonSchema(browserNavigateToolParams{}),
		},
	}}
}

func searchTools() []responses.ToolUnionParam {
	return []responses.ToolUnionParam{{
		OfFunction: &responses.FunctionToolParam{
			Name:        searchToolName,
			Description: openai.String("Use a web search engine to search for the given query and retrieve relevant results"),
			Parameters:  jsonSchema(searchToolParams{}),
		},
	}}
}

func toolCallParams(name, args string) (any, error) {
	switch name {
	case browserNavigateToolName:
		var params browserNavigateToolParams
		if err := json.Unmarshal([]byte(args), &params); err != nil {
			return nil, fmt.Errorf("unmarshaling tool call params: %w", err)
		}
		return params, nil
	case searchToolName:
		var params searchToolParams
		if err := json.Unmarshal([]byte(args), &params); err != nil {
			return nil, fmt.Errorf("unmarshaling tool call params: %w", err)
		}
		return params, nil
	default:
		return nil, fmt.Errorf("tool does not exist: %s", name)
	}
}
