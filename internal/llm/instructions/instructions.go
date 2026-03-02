package instructions

import (
	"fmt"
	"strings"
)

type instruction interface {
	Name() string
	Description() string
	Body() string
}

type registry struct {
	instructions []instruction
}

var Registry = registry{
	instructions: []instruction{
		episodicContent{},
	},
}

func (r registry) Index() string {
	var sb strings.Builder

	for _, inst := range r.instructions {
		fmt.Fprintf(&sb, "- %s: %s\n", inst.Name(), inst.Description())
	}

	return sb.String()
}

func (r registry) Body(name string) (string, error) {
	for _, inst := range r.instructions {
		if inst.Name() == name {
			return inst.Body(), nil
		}
	}
	return "", fmt.Errorf("instruction '%s' not found", name)
}
