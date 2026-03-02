package browser

import (
	"context"
	"encoding/json"
	"net/url"
	"testing"

	"github.com/alexpls/untils/internal/testhelper"
	"github.com/chromedp/cdproto/accessibility"
	"github.com/chromedp/chromedp"
	"github.com/stretchr/testify/require"
)

func TestFlattenNodesWherePreservesSiblingOrder(t *testing.T) {
	tree := axTree{
		RootID: "root",
		Nodes: map[string]*axNode{
			"root": {NodeID: "root", Role: "RootWebArea", ChildIDs: []string{"a", "g", "c"}},
			"a":    {NodeID: "a", ParentID: "root", Role: "heading", Name: "A"},
			"g":    {NodeID: "g", ParentID: "root", Role: "generic", ChildIDs: []string{"b1", "b2"}},
			"b1":   {NodeID: "b1", ParentID: "g", Role: "heading", Name: "B1"},
			"b2":   {NodeID: "b2", ParentID: "g", Role: "heading", Name: "B2"},
			"c":    {NodeID: "c", ParentID: "root", Role: "heading", Name: "C"},
		},
	}

	tree.flattenNodesWhere(func(n *axNode) bool { return n.Role == "generic" })
	require.Equal(t, []string{"a", "b1", "b2", "c"}, tree.Nodes["root"].ChildIDs)
}

func TestFormatEnergexAccessibilityTree(t *testing.T) {
	var tree axTree
	require.NoError(t, json.Unmarshal(energexAxTreeFixture(t), &tree))

	testhelper.SnapshotMatch(t, "accessibility_tree_energex.parsed.txt", tree.String())
}

func TestFormatWikipediaAccessibilityTree(t *testing.T) {
	var tree axTree
	require.NoError(t, json.Unmarshal(wikipediaAxTreeFixture(t), &tree))

	testhelper.SnapshotMatch(t, "accessibility_tree_wikipedia.parsed.txt", tree.String())
}

func wikipediaAxTreeFixture(t *testing.T) []byte {
	return axTreeFixture(t, "accessibility_tree_wikipedia.json", "https://en.wikipedia.org/wiki/Taylor_Swift_albums_discography")
}

func energexAxTreeFixture(t *testing.T) []byte {
	return axTreeFixture(t, "accessibility_tree_energex.json", "https://www.energex.com.au/outages/outage-finder/emergency-outages-text-view/")
}

func axTreeFixture(t *testing.T, name string, path string) []byte {
	t.Helper()

	s := testhelper.Snapshot(t, name, func() string {
		ctx, cancel := chromedp.NewContext(context.TODO())
		defer cancel()

		u, err := url.Parse(path)
		require.NoError(t, err)

		var tree axTree

		require.NoError(t, chromedp.Run(ctx,
			accessibility.Enable(),
			chromedp.Navigate(path),
			tidyHTML(u),
			accessibilityTree(&tree),
		))

		jsonStr, err := json.MarshalIndent(tree, "", "  ")
		require.NoError(t, err)

		return string(jsonStr)
	})

	return []byte(s)
}
