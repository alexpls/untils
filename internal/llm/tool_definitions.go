package llm

import llmtools "github.com/alexpls/untils/internal/llm/tools"

func toProviderTools(defs []llmtools.Definition) []ToolDefinition {
	tools := make([]ToolDefinition, 0, len(defs))
	for _, def := range defs {
		tools = append(tools, ToolDefinition{
			Name:        def.Name,
			Description: def.Description,
			Parameters:  jsonSchema(def.Params),
		})
	}
	return tools
}
