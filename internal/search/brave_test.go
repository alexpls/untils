//go:build integration

package search

import (
	"os"
	"strings"
	"testing"

	"github.com/alexpls/untils/internal/testhelper"
	"github.com/stretchr/testify/require"
)

func TestBraveSearch(t *testing.T) {
	tl := testhelper.TestLogger(t)
	c := NewBraveClient(os.Getenv("BRAVE_KEY"), tl)
	res, err := c.Search(NewSearchParams("latest ign game reviews").WithCount(5))

	require.NoError(t, err)
	require.Len(t, res.Results, 5)

	gotIGN := false

	for _, r := range res.Results {
		if strings.Contains(r.URL, "ign.com") {
			gotIGN = true
		}

		// t.Log(r)
	}

	require.True(t, gotIGN, "at least one result should be from ign.com")
}
