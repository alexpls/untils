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

func accessibilityTree(response *axTree) chromedp.ActionFunc {
	return func(ctx context.Context) error {
		if err := chromedp.Run(ctx, chromedp.ActionFunc(func(ctx context.Context) (err error) {
			var resp chromeAxTree
			if err = cdp.Execute(ctx, accessibility.CommandGetFullAXTree, nil, &resp); err != nil {
				return err
			}
			if *response, err = resp.axTree(); err != nil {
				return err
			}
			return nil
		})); err != nil {
			return fmt.Errorf("getting accessibility tree: %w", err)
		}
		return nil
	}
}

// axTree is our internal representation of an accessibility tree
type axTree struct {
	RootID string            `json:"root_id"`
	Nodes  map[string]axNode `json:"nodes"`
}

// String returns a string representation of the tree, ready for
// LLM consumption.
func (t *axTree) String() string {
	var sb strings.Builder

	root := t.Nodes[t.RootID]
	root.format(&sb, t, 0) // recursively formats the whole tree

	return sb.String()
}

// axNode is a node in our internal accessibility tree representation
// which has been parsed from a Chrome representation (chromeAxNode).
type axNode struct {
	NodeID   string   `json:"node_id"`
	ParentID string   `json:"parent_id,omitempty,omitzero"`
	ChildIDs []string `json:"child_ids,omitempty,omitzero"`
	Ignored  bool     `json:"ignored"`
	Name     string   `json:"name,omitempty,omitzero"`
	Role     string   `json:"role,omitempty,omitzero"`
	Value    string   `json:"value,omitempty,omitzero"`
}

func (n *axNode) children(t *axTree) []*axNode {
	c := make([]*axNode, 0, len(n.ChildIDs))
	for _, childID := range n.ChildIDs {
		if child, ok := t.Nodes[childID]; ok {
			c = append(c, &child)
		}
	}
	return c
}

func (n *axNode) format(sb *strings.Builder, t *axTree, depth int) {
	attrs := n.attributes()

	indent := strings.Repeat("  ", depth)
	sb.WriteString(indent)
	sb.WriteString(strings.Join(attrs, ", "))
	sb.WriteString("\n")

	childDepth := depth + 1
	for _, child := range n.children(t) {
		child.format(sb, t, childDepth)
	}
}

func (n *axNode) attributes() []string {
	var attributes []string

	attributes = append(attributes, "id:"+n.NodeID)

	if n.Role != "" {
		if n.Role == "none" {
			attributes = append(attributes, "ignored")
		} else {
			attributes = append(attributes, n.Role)
		}
	}

	if n.Name != "" {
		attributes = append(attributes, n.Name)
	}

	if n.Value != "" {
		attributes = append(attributes, n.Value)
	}

	return attributes
}

// chromeAxTree represents the full accessibility tree returned by Chrome.
type chromeAxTree struct {
	Nodes []*chromeAxNode `json:"nodes,omitempty,omitzero"`
}

// axTree converts the Chrome accessibility tree to our internal representation
// which does away with a lot of the Chrome-specific details we don't use.
func (t *chromeAxTree) axTree() (axTree, error) {
	// nodeID -> node quick lookup
	nodes := make(map[string]axNode)
	for _, chromeNode := range t.Nodes {
		node := chromeNode.axNode()
		nodes[chromeNode.NodeID.String()] = node
	}

	for _, node := range nodes {
		if node.ParentID == "" {
			return axTree{
				RootID: node.NodeID,
				Nodes:  nodes,
			}, nil
		}
	}

	return axTree{}, fmt.Errorf("no root node found in accessibility tree")
}

// chromeAxNode is a node in the Chrome accessibility tree.
//
// Subset of accessibility.Node which intentionally skips ignoredReasons
// due to incompatibilities with cdproto.
type chromeAxNode struct {
	NodeID   accessibility.NodeID   `json:"nodeId"`
	ParentID accessibility.NodeID   `json:"parentId,omitempty,omitzero"`
	ChildIDs []accessibility.NodeID `json:"childIds,omitempty,omitzero"`
	Ignored  bool                   `json:"ignored"`
	Name     *accessibility.Value   `json:"name,omitempty,omitzero"`
	Role     *accessibility.Value   `json:"role,omitempty,omitzero"`
	Value    *accessibility.Value   `json:"value,omitempty,omitzero"`
}

func (n *chromeAxNode) axNode() axNode {
	return axNode{
		NodeID:   n.NodeID.String(),
		ParentID: n.ParentID.String(),
		ChildIDs: func() []string {
			var ids []string
			for _, id := range n.ChildIDs {
				ids = append(ids, id.String())
			}
			return ids
		}(),
		Ignored: true,
		Name:    valueString(n.Name),
		Role:    valueString(n.Role),
		Value:   valueString(n.Value),
	}
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
