package tools

import (
	"fmt"
	"strings"
)

type registry struct {
	tools []Tool
}

var Registry = registry{
	tools: []Tool{
		readInstructionTool,
		searchTool,
		browserNavigateTool,
		browserClickTool,
		browserWaitTool,
	},
}

func (r registry) Definitions() []Definition {
	defs := make([]Definition, 0, len(r.tools))
	for _, tool := range r.tools {
		defs = append(defs, tool.Definition())
	}
	return defs
}

func (r registry) Build(name string, tc *Context, args string) (*Call, error) {
	for _, tool := range r.tools {
		if tool.Definition().Name == name {
			return tool.Builder()(tc, args)
		}
	}
	return nil, fmt.Errorf("tool does not exist: %s", name)
}

func (r registry) UsageMarkdown() string {
	var sb strings.Builder

	sb.WriteString("## Available tools\n\n")
	sb.WriteString("In order to achieve this you will use the following tools:\n\n")

	for _, tool := range r.tools {
		def := tool.Definition()
		fmt.Fprintf(&sb, "- `%s` - %s\n", def.Name, def.Description)
	}

	for _, tool := range r.tools {
		def := tool.Definition()
		fmt.Fprintf(&sb, "\n## Using the `%s` tool\n\n", def.Name)
		sb.WriteString(def.UsageBody)
		sb.WriteString("\n")
	}

	return sb.String()
}
