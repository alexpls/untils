package codehighlight

import (
	"bytes"
	"context"
	stdhtml "html"
	"io"
	"regexp"
	"strings"

	"github.com/a-h/templ"
	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
)

var codeBlockPattern = regexp.MustCompile(`(?s)<pre><code(?: class="language-([^"]+)")?>(.*?)</code></pre>`)

const docsCodeStyle = "github-dark"

func Block(language, source string) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		return renderBlock(ctx, w, language, source)
	})
}

func HTML(content string) (string, error) {
	var highlightErr error

	highlighted := codeBlockPattern.ReplaceAllStringFunc(content, func(block string) string {
		if highlightErr != nil {
			return block
		}

		matches := codeBlockPattern.FindStringSubmatch(block)
		if len(matches) != 3 {
			return block
		}

		var rendered bytes.Buffer
		if err := renderBlock(context.Background(), &rendered, matches[1], stdhtml.UnescapeString(matches[2])); err != nil {
			highlightErr = err
			return block
		}
		return rendered.String()
	})

	if highlightErr != nil {
		return "", highlightErr
	}

	return highlighted, nil
}

func renderBlock(ctx context.Context, w io.Writer, language, source string) error {
	language = strings.TrimSpace(language)
	codeClass := ""
	if language != "" {
		codeClass = "language-" + language
	}
	contentHTML := stdhtml.EscapeString(source)

	if lexer := lexers.Get(language); lexer != nil {
		iterator, err := chroma.Coalesce(lexer).Tokenise(nil, source)
		if err != nil {
			return err
		}

		var highlighted bytes.Buffer
		formatter := html.New(
			html.WithClasses(false),
			html.PreventSurroundingPre(true),
		)
		if err := formatter.Format(&highlighted, styles.Get(docsCodeStyle), iterator); err != nil {
			return err
		}
		contentHTML = highlighted.String()
	}

	return highlightedCodeBlock(highlightedCodeBlockPreStyle(), codeClass, contentHTML).Render(ctx, w)
}

func highlightedCodeBlockPreStyle() string {
	style := styles.Get(docsCodeStyle)
	if style == nil {
		return "-webkit-text-size-adjust: none;"
	}

	background := style.Get(chroma.Background)
	styles := []string{
		html.StyleEntryToCSS(style.Get(chroma.PreWrapper).Sub(background)),
		html.StyleEntryToCSS(background),
		"-webkit-text-size-adjust: none;",
	}
	return strings.Join(nonEmptyStrings(styles), ";")
}

func nonEmptyStrings(values []string) []string {
	filtered := values[:0]
	for _, value := range values {
		if value != "" {
			filtered = append(filtered, value)
		}
	}
	return filtered
}
