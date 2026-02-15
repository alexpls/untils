package tinytemplate

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseAndRender(t *testing.T) {
	tt, err := Parse("Release: {{ Release date }} ({{Score}})")
	require.NoError(t, err)

	require.Equal(t, []string{"Release date", "Score"}, tt.References())

	out, err := tt.Render(map[string]string{
		"Release date": "2026-02-10",
		"Score":        "9/10",
	})
	require.NoError(t, err)
	require.Equal(t, "Release: 2026-02-10 (9/10)", out)
}

func TestRenderMissingValue(t *testing.T) {
	tt, err := Parse("{{Release date}}")
	require.NoError(t, err)

	_, err = tt.Render(map[string]string{})
	require.ErrorContains(t, err, `missing value for field "Release date"`)
}

func TestRenderValueWithTemplateDelimiters(t *testing.T) {
	tt, err := Parse("Title: {{Title}}")
	require.NoError(t, err)

	out, err := tt.Render(map[string]string{
		"Title": "Thing {{not a field}} ok }} and {{",
	})
	require.NoError(t, err)
	require.Equal(t, "Title: Thing {{not a field}} ok }} and {{", out)
}

func TestParseErrors(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		errContains string
	}{
		{
			name:        "unclosed placeholder",
			input:       "Release: {{Release date",
			errContains: "unclosed placeholder",
		},
		{
			name:        "unexpected closing delimiter",
			input:       "Release date}}",
			errContains: "unexpected closing delimiter",
		},
		{
			name:        "empty placeholder",
			input:       "{{   }}",
			errContains: "empty placeholder",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse(tt.input)
			require.ErrorContains(t, err, tt.errContains)
		})
	}
}
