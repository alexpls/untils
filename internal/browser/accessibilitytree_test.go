package browser

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/alexpls/untils_go/internal/testhelper"
	"github.com/chromedp/cdproto/accessibility"
	"github.com/chromedp/chromedp"
	"github.com/stretchr/testify/require"
)

func TestFormatAccessibilityTree(t *testing.T) {
	var tree axTreeResponse
	require.NoError(t, json.Unmarshal(accessibilityTreeFixture(t), &tree))

	testhelper.SnapshotMatch(t, "accessibility_tree_energex.parsed.txt", tree.String())
}

func accessibilityTreeFixture(t *testing.T) []byte {
	s := testhelper.Snapshot(t, "accessibility_tree_energex.json", func() string {
		ctx, cancel := chromedp.NewContext(context.TODO())
		defer cancel()

		var tree *axTreeResponse

		require.NoError(t, chromedp.Run(ctx,
			accessibility.Enable(),
			chromedp.Navigate("https://www.energex.com.au/outages/outage-finder/emergency-outages-text-view/"),
			accessibilityTree(tree),
		))

		jsonStr, err := json.MarshalIndent(tree, "", "  ")
		require.NoError(t, err)

		return string(jsonStr)
	})

	return []byte(s)
}
