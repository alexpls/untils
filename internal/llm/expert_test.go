package llm

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExpertsMarkdown(t *testing.T) {
	require.Equal(t, expertsMarkdown, `## Available experts

- default: A generic expert capable of handling a wide range of subjects. Fallback option when no specialized expert is suitable.
`)
}
