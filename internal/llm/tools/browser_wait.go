package tools

import (
	"time"

	"github.com/alexpls/untils/internal/models"
)

type browserWaitParams struct{}

func (p browserWaitParams) Equal(other any) bool {
	_, ok := other.(browserWaitParams)
	return ok
}

var browserWaitTool = tool[browserWaitParams]{
	name:        models.LLMToolNameBrowserWait,
	description: "Wait for the current page to finish loading. Use this when you suspect dynamic content may not have loaded yet. Returns the updated page contents after waiting.",
	usageBody: `- Use this tool when you suspect a page may not have fully loaded yet (e.g. dynamic
  content, JavaScript-rendered pages, or pages that load content asynchronously).
- This tool waits for 3 seconds before returning the updated page contents.
- IMPORTANT: You should only call ` + "`" + models.LLMToolNameBrowserWait + "`" + ` ONCE per page. Never call it two times
  in a row for the same page. If the content still hasn't loaded after one wait, move on
  and try a different approach or source.`,
	execute: func(tc *Context, p browserWaitParams) (string, error) {
		tc.Logger.DebugContext(tc.Ctx, "browser_wait started")
		start := time.Now()

		time.Sleep(3 * time.Second)

		b := tc.Browser()
		page, err := b.CurrentPage()
		if err != nil {
			tc.Logger.ErrorContext(tc.Ctx, "error getting current page after wait", "error", err)
			return "", err
		}

		tc.Logger.DebugContext(tc.Ctx, "browser_wait completed", "total_duration", time.Since(start))
		return page.String(), nil
	},
	validate: func(tc *Context, params browserWaitParams) string {
		l := len(*tc.PriorCalls)
		if l > 0 {
			last := (*tc.PriorCalls)[l-1]
			if params.Equal(last.Params()) {
				return "can't wait multiple times consecutively. try using another tool " +
					"to navigate to a new page."
			}
		}
		return ""
	},
}
