package tinytemplate

import (
	"fmt"
	"strings"
)

type Template struct {
	parts      []part
	references []string
}

type part struct {
	text  string
	field string
}

func Parse(input string) (Template, error) {
	t := Template{}
	i := 0

	for i < len(input) {
		open := strings.Index(input[i:], "{{")
		close := strings.Index(input[i:], "}}")
		if close >= 0 && (open < 0 || close < open) {
			return Template{}, fmt.Errorf("unexpected closing delimiter at position %d", i+close)
		}

		if open < 0 {
			if i < len(input) {
				t.parts = append(t.parts, part{text: input[i:]})
			}
			break
		}

		open += i
		if open > i {
			t.parts = append(t.parts, part{text: input[i:open]})
		}

		fieldStart := open + 2
		fieldEnd := strings.Index(input[fieldStart:], "}}")
		if fieldEnd < 0 {
			return Template{}, fmt.Errorf("unclosed placeholder at position %d", open)
		}

		fieldEnd += fieldStart
		fieldName := strings.TrimSpace(input[fieldStart:fieldEnd])
		if fieldName == "" {
			return Template{}, fmt.Errorf("empty placeholder at position %d", open)
		}

		if strings.Contains(fieldName, "{{") || strings.Contains(fieldName, "}}") {
			return Template{}, fmt.Errorf("invalid placeholder %q at position %d", fieldName, open)
		}

		t.references = append(t.references, fieldName)
		t.parts = append(t.parts, part{field: fieldName})
		i = fieldEnd + 2
	}

	return t, nil
}

func (t Template) References() []string {
	refs := make([]string, len(t.references))
	copy(refs, t.references)
	return refs
}

func (t Template) Render(values map[string]string) (string, error) {
	return t.RenderFunc(func(name string) (string, bool) {
		v, ok := values[name]
		return v, ok
	})
}

func (t Template) RenderFunc(resolve func(name string) (string, bool)) (string, error) {
	var b strings.Builder

	for _, p := range t.parts {
		if p.field == "" {
			b.WriteString(p.text)
			continue
		}

		value, ok := resolve(p.field)
		if !ok {
			return "", fmt.Errorf("missing value for field %q", p.field)
		}
		b.WriteString(value)
	}

	return b.String(), nil
}
