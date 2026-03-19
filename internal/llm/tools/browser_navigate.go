package tools

import (
	"time"

	"github.com/alexpls/untils/internal/models"
)

var browserNavigateTool = tool[models.BrowserNavigateParams]{
	name:        models.LLMToolNameBrowserNavigate,
	description: "Use a web browser to navigate to the given URL and retrieve the page contents",
	usageBody: `- Use this tool to visit the websites from your search request and read their contents.
- The response of the tool will be a text representation of the webpage.
- Sometimes you'll land on 404 Not Found pages. This is normal as search results can be
  stale. When this happens, go back to your search results and try the next most appropriate
  link. DO NOT keep trying to request the same page over and over again, it will never work.
- If you have found enough information to determine the current value of the subject,
  DO NOT keep calling this tool to visit more URLs. Once you have your answer it's
  crucial to respond as quickly as possible.`,
	execute: func(tc *Context, p models.BrowserNavigateParams) (string, error) {
		tc.Logger.DebugContext(tc.Ctx, "browser_navigate started", "url", p.URL)
		start := time.Now()

		if tc.AddSiteVisited != nil {
			tc.AddSiteVisited(p.URL)
		}

		getBrowserStart := time.Now()
		b, err := tc.Browser()
		if err != nil {
			return "", err
		}
		tc.Logger.DebugContext(tc.Ctx, "browser_navigate got browser context", "duration", time.Since(getBrowserStart))

		navigateStart := time.Now()
		res, err := b.Navigate(p.URL)
		tc.Logger.DebugContext(tc.Ctx, "browser_navigate navigation completed", "duration", time.Since(navigateStart))

		if err != nil {
			return "", err
		}

		tc.Logger.DebugContext(tc.Ctx, "browser_navigate completed", "url", p.URL, "total_duration", time.Since(start))
		return res.String(), nil
	},
	validate: func(tc *Context, params models.BrowserNavigateParams) string {
		return noDuplicateCallsValidator(tc, params, "navigating to the same url multiple times is not allowed. try browsing to a different url")
	},
}
