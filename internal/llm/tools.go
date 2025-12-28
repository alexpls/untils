package llm

import (
	"encoding/json"
	"fmt"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/responses"
)

// TODO: a lot of boilerplate here - consider extracting tool interface

const (
	browserNavigateToolName = "browser_navigate"
	browserClickToolName    = "browser_click"
	// intentionally different to "web_search" to avoid name collisions with OAI/x.ai
	searchToolName = "search_request"
)

type browserNavigateToolParams struct {
	URL string `json:"url"`
}

type browserClickToolParams struct {
	NodeID string `json:"node_id"`
}

type searchToolParams struct {
	Query string `json:"string"`
}

func browserTools() []responses.ToolUnionParam {
	return []responses.ToolUnionParam{
		{
			OfFunction: &responses.FunctionToolParam{
				Name:        browserNavigateToolName,
				Description: openai.String("Use a web browser to navigate to the given URL and retrieve the page contents"),
				Parameters:  jsonSchema(browserNavigateToolParams{}),
			},
		},
		{
			OfFunction: &responses.FunctionToolParam{
				Name: browserClickToolName,
				Description: openai.String("Use a web browser to click on an element on the current page, " +
					"identified by its unique ID (e.g. [learn more](click:123) - the ID is 123)"),
				Parameters: jsonSchema(browserClickToolParams{}),
			},
		},
	}
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
	case browserClickToolName:
		var params browserClickToolParams
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
