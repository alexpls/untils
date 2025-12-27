//go:build integration

package browser_test

import (
	"context"
	"testing"
	"time"

	"github.com/alexpls/untils_go/internal/browser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNavigate(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := browser.Navigate(ctx, "https://example.org")
	require.NoError(t, err)

	assert.Equal(t, "Example Domain", result.Page.Title)
	assert.Contains(t, result.Page.Contents, "This domain is for use in documentation examples")
}

func TestNavigateBigPage(t *testing.T) {
	t.Skip("just for testing")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := browser.Navigate(ctx, "https://en.wikipedia.org/wiki/The_Life_of_a_Showgirl")
	require.NoError(t, err)

	t.Log(result)
}
