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

	b, closeB := browser.NewBrowser(ctx)
	defer closeB()

	result, err := b.Navigate("https://example.org")
	require.NoError(t, err)

	assert.Equal(t, "Example Domain", result.Title)
	assert.Contains(t, result.Contents, "This domain is for use in documentation examples")
}

func TestNavigateBigPage(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	b, closeB := browser.NewBrowser(ctx)
	defer closeB()

	result, err := b.Navigate("https://www.ign.com/reviews/games")
	require.NoError(t, err)

	t.Log(result)
}
