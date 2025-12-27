package browser

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/chromedp/cdproto/accessibility"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/chromedp"
)

// parsers for the accessibility object model, see:
// https://wicg.github.io/aom/explainer.html
// heavily inspired by: https://github.com/ChromeDevTools/chrome-devtools-mcp/blob/main/src/formatters/snapshotFormatter.ts

func accessibilityTree(response *axTreeResponse) chromedp.ActionFunc {
	return func(ctx context.Context) error {
		if err := chromedp.Run(ctx, chromedp.ActionFunc(func(ctx context.Context) error {
			// should really use: accessibility.GetFullAXTree().Do(ctx)
			// but can't because json parsing the response breaks due to new fields
			// in the chrome devtools protocol that are not on cdproto yet. so... we do
			// things manually
			return cdp.Execute(ctx, accessibility.CommandGetFullAXTree, nil, &response)
		})); err != nil {
			return fmt.Errorf("getting accessibility tree: %w", err)
		}
		return nil
	}
}

type axTreeResponse struct {
	Nodes []*axNode `json:"nodes,omitempty,omitzero"`
}

func (t *axTreeResponse) Roots() []*axTreeNode {
	// nodeID -> node quick lookup
	tree := make(map[accessibility.NodeID]*axTreeNode)
	for _, node := range t.Nodes {
		treeNode := &axTreeNode{node: node}
		tree[treeNode.node.NodeID] = treeNode
	}

	var roots []*axTreeNode
	for _, node := range t.Nodes {
		treeNode := tree[node.NodeID]
		if node.ParentID == "" {
			roots = append(roots, treeNode)
		}
		for _, childID := range node.ChildIDs {
			if _, ok := tree[childID]; ok {
				tree[childID].parent = treeNode
				treeNode.children = append(treeNode.children, tree[childID])
			}
		}
	}

	return roots
}

func (t *axTreeResponse) String() string {
	var sb strings.Builder

	roots := t.Roots()
	for _, root := range roots {
		root.format(&sb, 0)
	}

	return strings.TrimSpace(sb.String())
}

type axTreeNode struct {
	node     *axNode
	parent   *axTreeNode
	children []*axTreeNode
}

func (n *axTreeNode) skip() bool {
	if n.node.Ignored {
		return true
	}

	if role := valueString(n.node.Role); role == "InlineTextBox" {
		if pRole := valueString(n.parent.node.Role); pRole == "StaticText" {
			return true
		}
	}

	return false
}

func (n *axTreeNode) format(sb *strings.Builder, depth int) {
	node := n.node

	if n.skip() {
		for _, child := range n.children {
			child.format(sb, depth)
		}
		return
	}

	attributes := node.attributes()

	indent := strings.Repeat("  ", depth)
	sb.WriteString(indent)
	sb.WriteString(strings.Join(attributes, " "))
	sb.WriteString("\n")

	childDepth := depth + 1
	for _, child := range n.children {
		child.format(sb, childDepth)
	}
}

// axNode is a node in the accessibility tree.
//
// Subset of accessibility.Node which intentionally skips ignoredReasons
// due to incompatibilities with cdproto.
type axNode struct {
	NodeID   accessibility.NodeID   `json:"nodeId"`
	ParentID accessibility.NodeID   `json:"parentId,omitempty,omitzero"`
	ChildIDs []accessibility.NodeID `json:"childIds,omitempty,omitzero"`
	Ignored  bool                   `json:"ignored"`
	Name     *accessibility.Value   `json:"name,omitempty,omitzero"`
	Role     *accessibility.Value   `json:"role,omitempty,omitzero"`
	Value    *accessibility.Value   `json:"value,omitempty,omitzero"`
}

func (n *axNode) attributes() []string {
	var attributes []string

	attributes = append(attributes, "id:"+n.NodeID.String())

	if role := valueString(n.Role); role != "" {
		if role == "none" {
			attributes = append(attributes, "ignored")
		} else {
			attributes = append(attributes, role)
		}
	}

	if name := valueString(n.Name); name != "" {
		attributes = append(attributes, name)
	}

	if value := valueString(n.Value); value != "" {
		attributes = append(attributes, value)
	}

	return attributes
}

func valueString(v *accessibility.Value) string {
	if v == nil {
		return ""
	}

	var s string
	if err := json.Unmarshal(v.Value, &s); err == nil {
		return s
	}

	var b bool
	if err := json.Unmarshal(v.Value, &b); err == nil {
		if b {
			return "true"
		} else {
			return "false"
		}
	}

	var n float64
	if err := json.Unmarshal(v.Value, &n); err == nil {
		return fmt.Sprintf("%v", n)
	}

	return s
}
