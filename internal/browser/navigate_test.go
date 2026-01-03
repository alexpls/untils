//go:build integration

package browser_test

import (
	"regexp"
	"testing"

	"github.com/alexpls/untils/internal/browser"
	"github.com/alexpls/untils/internal/testhelper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNavigateMySite(t *testing.T) {
	tl := testhelper.TestLogger(t)
	b, closeB := browser.NewBrowser(t.Context(), tl)
	defer closeB()

	page, err := b.Navigate("https://alexplescan.com")
	require.NoError(t, err)

	assert.Equal(t, "Home | Alex Plescan", page.Title)
	assert.Equal(t, "https://alexplescan.com/favicon-32x32.png", page.FaviconURL)
	assert.Contains(t, page.Contents, "I make websites")

	re := regexp.MustCompile(`\[ABOUT\]\(click:(\d+)\)`)
	matches := re.FindStringSubmatch(page.Contents)
	require.Len(t, matches, 2, "expected to find a clickable link in the page contents")

	clickedPage, err := b.Click(matches[1])
	require.NoError(t, err)

	assert.Contains(t, clickedPage.Contents, "software engineer")
}

func TestNavigateSiteWithNoFavicon(t *testing.T) {
	tl := testhelper.TestLogger(t)
	b, closeB := browser.NewBrowser(t.Context(), tl)
	defer closeB()

	page, err := b.Navigate("https://example.org")
	require.NoError(t, err)

	assert.Equal(t, "Example Domain", page.Title)
	assert.Zero(t, page.FaviconURL)
}
