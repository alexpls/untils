package codehighlight

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBlockUsesTemplWrapper(t *testing.T) {
	t.Parallel()

	var html bytes.Buffer
	err := renderBlock(context.Background(), &html, "json", `{"name":"untils"}`)
	require.NoError(t, err)

	content := html.String()
	require.Contains(t, content, `<pre style=`)
	require.Contains(t, content, `<code class="language-json">`)
	require.Contains(t, content, `<span style=`)
	require.Contains(t, content, `&#34;untils&#34;`)
	require.NotContains(t, content, `<code class="">`)
}
