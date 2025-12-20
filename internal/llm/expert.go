package llm

import (
	"context"
	"errors"
	"maps"
	"slices"
)

var experts = map[string]expertDefinition{
	"default": {
		description: "A generic expert capable of handling a wide range of subjects. " +
			"Fallback option when no specialized expert is suitable.",
		builder: NewExpertDefault,
	},
}
var expertNames = slices.Collect(maps.Keys(experts))

type ErrUnsupportedExpert struct {
	Name string
}

func (e ErrUnsupportedExpert) Error() string {
	return "unsupported expert: " + e.Name
}

var ErrUnkonwnExpert = errors.New("unknown expert")

func BuildExpert(name string, service *Service) Expert {
	ex, ok := experts[name]
	if !ok {
		panic("invalid expert: " + name)
	}
	return ex.builder(service)
}

type Expert interface {
	PerformCheck(ctx context.Context, params *CheckParams) (*CheckResponse, error)
}

type expertDefinition struct {
	description string
	builder     func(service *Service) Expert
}

var expertsMarkdown string

func init() {
	expertsMarkdown = "## Available experts\n\n"
	for name, definition := range experts {
		expertsMarkdown += "- " + name + ": " + definition.description + "\n"
	}
}
