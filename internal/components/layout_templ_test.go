package components

import (
	"context"
	"io"
	"testing"

	"github.com/alexpls/untils/internal/reqcontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLayoutRendersPlausibleSnippet(t *testing.T) {
	render := func(ctx context.Context) (string, error) {
		r, w := io.Pipe()

		go func() {
			_ = Layout("Test Title").Render(ctx, w)
			_ = w.Close()
		}()

		d, err := io.ReadAll(r)
		if err != nil {
			return "", err
		}

		return string(d), nil
	}

	ctx := context.Background()

	doc, err := render(ctx)
	require.NoError(t, err)
	assert.NotContains(t, doc, "plausible.io/js/hello.js")

	ctx = reqcontext.ContextWithPlausibleSnippetTag(context.Background(), "hello")
	doc, err = render(ctx)
	require.NoError(t, err)
	assert.Contains(t, doc, "plausible.io/js/hello.js")
}
