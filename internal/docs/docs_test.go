package docs

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizePath(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"":                        "/docs",
		"/":                       "/docs",
		"docs":                    "/docs",
		"/docs":                   "/docs",
		"/docs/":                  "/docs",
		"self-hosting/quickstart": "/docs/self-hosting/quickstart",
		"/docs/self-hosting/":     "/docs/self-hosting",
	}

	for input, want := range cases {
		if got := NormalizePath(input); got != want {
			t.Fatalf("NormalizePath(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestCurrentSiteIndexPath(t *testing.T) {
	t.Parallel()

	site := CurrentSite()
	require.Equal(t, "/docs/introduction/welcome", site.IndexPath)

	page, ok := site.Page(site.IndexPath)
	require.True(t, ok, "CurrentSite().Page(%q) not found", site.IndexPath)
	require.Equal(t, "/docs/introduction/welcome", page.Path)
	require.Equal(t, "Welcome", page.Title)
	require.Equal(t, "Welcome", page.SidebarTitle)
	require.NotEmpty(t, page.LastUpdated)
}
