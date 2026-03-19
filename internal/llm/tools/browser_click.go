package tools

import (
	"time"

	"github.com/alexpls/untils/internal/models"
)

type browserClickParams struct {
	NodeID string `json:"node_id"`
}

func (p browserClickParams) Equal(other any) bool {
	o, ok := other.(browserClickParams)
	if !ok {
		return false
	}
	return p.NodeID == o.NodeID
}

var browserClickTool = tool[browserClickParams]{
	name:        models.LLMToolNameBrowserClick,
	description: "Use a web browser to click on an element on the current page, identified by its unique ID (e.g. [learn more](click:123) - the ID is 123)",
	usageBody: `- You must specify a valid node ID from the latest ` + "`" + models.LLMToolNameBrowserNavigate + "`" + ` response.
  These are in the format: ` + "`[Next page](click:123)`" + ` - where "123" is the node ID. The text in
  square brackets is the name of the element you will be clicking on, and the text in parentheses
  is the node ID prefixed with "click:".
- Use this to click on elements that you want to navigate to. This tool may be useful for
  expanding sections of a webpage, paginating through results, or navigating to different
  parts of a site.`,
	execute: func(tc *Context, p browserClickParams) (string, error) {
		tc.Logger.DebugContext(tc.Ctx, "browser_click started", "node_id", p.NodeID)
		start := time.Now()

		b, err := tc.Browser()
		if err != nil {
			return "", err
		}
		clickStart := time.Now()
		page, err := b.Click(p.NodeID)
		tc.Logger.DebugContext(tc.Ctx, "browser_click click completed", "duration", time.Since(clickStart))

		if err != nil {
			tc.Logger.ErrorContext(tc.Ctx, "error performing click", "node_id", p.NodeID, "error", err)
			return "", err
		}

		tc.Logger.DebugContext(tc.Ctx, "browser_click completed", "node_id", p.NodeID, "total_duration", time.Since(start))
		return page.String(), nil
	},
	validate: func(tc *Context, params browserClickParams) string {
		return ""
	},
}
