package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	mdhtml "github.com/yuin/goldmark/renderer/html"
)

func TestParseDocFileExtractsH2Headings(t *testing.T) {
	t.Parallel()

	renderer := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithParserOptions(parser.WithAutoHeadingID()),
		goldmark.WithRendererOptions(mdhtml.WithUnsafe()),
	)

	page, err := parseDocFile("../../docs/public/003-self-hosting/001-quickstart.md", renderer)
	if err != nil {
		t.Fatalf("parseDocFile() error = %v", err)
	}

	if len(page.Headings) == 0 {
		t.Fatal("page.Headings is empty")
	}
	require.NotContains(t, page.ContentHTML, "<h1")
	require.Contains(t, page.ContentHTML, "<span")
	require.NotContains(t, page.ContentHTML, "<pre><code class=\"language-sh\">")

	firstHeading := page.Headings[0]
	if firstHeading.Title != "Prerequisites" {
		t.Fatalf("first heading title = %q, want %q", firstHeading.Title, "Prerequisites")
	}
	if firstHeading.ID != "prerequisites" {
		t.Fatalf("first heading id = %q, want %q", firstHeading.ID, "prerequisites")
	}
}

func TestHighlightCodeBlocksLeavesUnknownLanguagesUnchanged(t *testing.T) {
	t.Parallel()

	content := `<p>Example</p><pre><code class="language-unknownlang">hello</code></pre>`

	highlighted, err := highlightCodeBlocks(content)
	require.NoError(t, err)
	require.Equal(t, content, highlighted)
}

func TestHighlightCodeBlocksHighlightsKnownLanguages(t *testing.T) {
	t.Parallel()

	content := `<pre><code class="language-sh">echo &quot;hi&quot;</code></pre>`

	highlighted, err := highlightCodeBlocks(content)
	require.NoError(t, err)
	require.Contains(t, highlighted, "<span")
	require.Contains(t, highlighted, "echo")
	require.True(t, strings.HasPrefix(highlighted, "<pre"))
}
