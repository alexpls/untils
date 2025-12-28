package browser

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/chromedp/cdproto/accessibility"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/chromedp"
)

// Parsing and formatting of Chrome accessibility trees.
//
// See: https://wicg.github.io/aom/explainer.html
// See: https://chromedevtools.github.io/devtools-protocol/tot/Accessibility
// Approach inspired by: https://github.com/ChromeDevTools/chrome-devtools-mcp/blob/main/src/formatters/snapshotFormatter.ts

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
	RootID string             `json:"root_id"`
	Nodes  map[string]*axNode `json:"nodes"`
}

// String returns a string representation of the tree, ready for
// LLM consumption.
func (t *axTree) String() string {
	var sb strings.Builder

	root := t.Nodes[t.RootID]
	root.format(&sb, t, 0, formatStyleDefault) // recursively formats the whole tree

	return sb.String()
}

func (t *axTree) tidy() {
	t.flattenNodesWhere(func(n *axNode) bool { return n.Ignored || n.Role == "generic" })
	t.removeNodesWhere(func(n *axNode) bool { return n.Role == "InlineTextBox" || n.Role == "ListMarker" })
	t.removeRedundantStaticTextChildren()
}

// flattenNodesWhere removes nodes matching the predicate, reparenting their
// children to their parent.
func (t *axTree) flattenNodesWhere(predicate func(*axNode) bool) {
	matchingIDs := make([]string, 0)
	for id, node := range t.Nodes {
		if predicate(node) {
			matchingIDs = append(matchingIDs, id)
		}
	}

	for _, id := range matchingIDs {
		node := t.Nodes[id]
		if node == nil {
			continue // already removed
		}
		parent := node.parent(t)
		if parent == nil {
			continue // don't remove root
		}
		t.reparentChildren(node, parent)
		t.remove(node)
	}
}

// removeNodesWhere removes nodes matching the predicate along with all their children.
func (t *axTree) removeNodesWhere(predicate func(*axNode) bool) {
	for _, node := range t.Nodes {
		if predicate(node) {
			t.remove(node)
		}
	}
}

// removeRedundantStaticTextChildren removes StaticText child nodes when
// the parent already contains the same text as its name
func (t *axTree) removeRedundantStaticTextChildren() {
	for _, node := range t.Nodes {
		if node.Name == "" {
			continue
		}

		for _, child := range node.children(t) {
			if child.Role == "StaticText" && strings.Contains(node.Name, child.Name) {
				t.remove(child)
			}
		}
	}
}

func (t *axTree) remove(node *axNode) {
	for _, child := range node.children(t) {
		t.remove(child)
	}

	parent := t.Nodes[node.ParentID]
	parent.ChildIDs = slices.DeleteFunc(parent.ChildIDs, func(a string) bool {
		return a == node.NodeID
	})
	delete(t.Nodes, node.NodeID)
}

func (t *axTree) reparentChildren(node *axNode, newParent *axNode) {
	for _, child := range node.children(t) {
		child.ParentID = newParent.NodeID
	}

	newParent.ChildIDs = append(newParent.ChildIDs, node.ChildIDs...)
	node.ChildIDs = []string{}
}

// axNode is a node in our internal accessibility tree representation
// which has been parsed from a Chrome representation (chromeAxNode).
type axNode struct {
	// Identifies the node in the DOM domain
	BackendNodeID int64 `json:"backend_node_id,omitempty,omitzero"`
	// Identifies the node in the accessibility domain
	NodeID string `json:"node_id"`
	// Identifies the parent node in the accessibility domain
	ParentID string `json:"parent_id,omitempty,omitzero"`
	// Identifies the child nodes in the accessibility domain
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
			c = append(c, child)
		}
	}
	return c
}

func (n *axNode) parent(t *axTree) *axNode {
	return t.Nodes[n.ParentID]
}

const (
	formatStyleDefault = iota
	formatStyleCompact
)

func (n *axNode) format(sb *strings.Builder, t *axTree, depth int, formatStyle int) {
	// Special handling for table rows - format on a single line
	if n.Role == "row" {
		n.formatTableRow(sb, t, depth)
		return
	}

	// Special handling for lists - format compactly
	if n.Role == "list" {
		n.formatList(sb, t, depth)
		return
	}

	var text string

	switch n.Role {
	case "link", "button":
		text = n.formatAsClickable()
	case "StaticText", "superscript":
		text = n.Name
	case "table":
		text = fmt.Sprintf("table: %s", n.Name)
	default:
		attrs := n.attributes()
		text = strings.Join(attrs, ", ")
	}

	switch formatStyle {
	case formatStyleCompact:
		sb.WriteString(text)
	default:
		indent := strings.Repeat("  ", depth)
		sb.WriteString(indent)
		sb.WriteString(text)
		sb.WriteString("\n")
	}

	childDepth := depth + 1
	childStyle := formatStyle
	if n.Role == "paragraph" {
		childStyle = formatStyleCompact
		sb.WriteString(strings.Repeat("  ", childDepth))
	}

	for _, child := range n.children(t) {
		child.format(sb, t, childDepth, childStyle)
	}

	if n.Role == "paragraph" {
		sb.WriteString("\n")
	}
}

func (n *axNode) isTableCell() bool {
	return n.Role == "cell" || n.Role == "columnheader" || n.Role == "rowheader"
}

func (n *axNode) formatAsClickable() string {
	return fmt.Sprintf("[%s](click:%d)", n.Name, n.BackendNodeID)
}

func (n *axNode) formatTableRow(sb *strings.Builder, t *axTree, depth int) {
	indent := strings.Repeat("  ", depth)
	sb.WriteString(indent)
	sb.WriteString("| ")

	children := n.children(t)
	for i, child := range children {
		sb.WriteString(child.collectText(t))
		if i < len(children)-1 {
			sb.WriteString(" | ")
		}
	}
	sb.WriteString(" |\n")
}

// formatList formats a list compactly with each item on its own line prefixed with "-"
func (n *axNode) formatList(sb *strings.Builder, t *axTree, depth int) {
	indent := strings.Repeat("  ", depth)
	for _, child := range n.children(t) {
		if child.Role == "listitem" {
			sb.WriteString(indent)
			sb.WriteString("- ")
			sb.WriteString(child.collectText(t))
			sb.WriteString("\n")
		} else {
			// Non-listitem children get formatted normally
			child.format(sb, t, depth, formatStyleDefault)
		}
	}
}

// collectText recursively collects text content from a node and its children,
// preserving link/button formatting
func (n *axNode) collectText(t *axTree) string {
	var parts []string
	n.collectTextParts(t, &parts)
	return strings.Join(parts, " ")
}

func (n *axNode) collectTextParts(t *axTree, parts *[]string) {
	switch n.Role {
	case "link", "button":
		*parts = append(*parts, n.formatAsClickable())
		return
	case "StaticText":
		if n.Name != "" {
			*parts = append(*parts, n.Name)
		}
	default:
		// For table cells, don't add the name directly since it's a summary
		// of children - we want to traverse children to preserve formatting
		if !n.isTableCell() && n.Name != "" {
			*parts = append(*parts, n.Name)
		}
	}

	for _, child := range n.children(t) {
		child.collectTextParts(t, parts)
	}
}

func (n *axNode) attributes() []string {
	var attributes []string

	if n.Role != "" {
		attributes = append(attributes, n.Role)
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
	nodes := make(map[string]*axNode)
	for _, chromeNode := range t.Nodes {
		node := chromeNode.axNode()
		nodes[chromeNode.NodeID.String()] = &node
	}

	for _, node := range nodes {
		if node.ParentID == "" {
			t := axTree{
				RootID: node.NodeID,
				Nodes:  nodes,
			}
			t.tidy()
			return t, nil
		}
	}

	return axTree{}, fmt.Errorf("no root node found in accessibility tree")
}

// chromeAxNode is a node in the Chrome accessibility tree.
//
// Subset of accessibility.Node which intentionally skips ignoredReasons
// due to incompatibilities with cdproto.
type chromeAxNode struct {
	BackendNodeID cdp.BackendNodeID      `json:"backendDOMNodeId,omitempty,omitzero"`
	NodeID        accessibility.NodeID   `json:"nodeId"`
	ParentID      accessibility.NodeID   `json:"parentId,omitempty,omitzero"`
	ChildIDs      []accessibility.NodeID `json:"childIds,omitempty,omitzero"`
	Ignored       bool                   `json:"ignored"`
	Name          *accessibility.Value   `json:"name,omitempty,omitzero"`
	Role          *accessibility.Value   `json:"role,omitempty,omitzero"`
	Value         *accessibility.Value   `json:"value,omitempty,omitzero"`
}

func (n *chromeAxNode) axNode() axNode {
	return axNode{
		BackendNodeID: int64(n.BackendNodeID),
		NodeID:        n.NodeID.String(),
		ParentID:      n.ParentID.String(),
		ChildIDs: func() []string {
			var ids []string
			for _, id := range n.ChildIDs {
				ids = append(ids, id.String())
			}
			return ids
		}(),
		Ignored: n.Ignored,
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
