//go:build integration

package browser_test

import (
	"regexp"
	"testing"

	"github.com/alexpls/untils_go/internal/browser"
	"github.com/alexpls/untils_go/internal/testhelper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNavigate(t *testing.T) {
	tl := testhelper.TestLogger(t)
	b, closeB := browser.NewBrowser(t.Context(), tl)
	defer closeB()

	result, err := b.Navigate("https://example.org")
	require.NoError(t, err)

	assert.Equal(t, "Example Domain", result.Title)
	assert.Contains(t, result.Contents, "This domain is for use in documentation examples")
}

func TestNavigateClick(t *testing.T) {
	tl := testhelper.TestLogger(t)
	b, closeB := browser.NewBrowser(t.Context(), tl)
	defer closeB()

	page, err := b.Navigate("https://example.org")
	require.NoError(t, err)
	require.NotContains(t, page.Contents, "incidental traffic")

	re := regexp.MustCompile(`\[Learn more\]\(click:(\d+)\)`)
	matches := re.FindStringSubmatch(page.Contents)
	require.Len(t, matches, 2, "expected to find a clickable link in the page contents")

	clickedPage, err := b.Click(matches[1])
	require.NoError(t, err)

	assert.Contains(t, clickedPage.Contents, "incidental traffic")
}
