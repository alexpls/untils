package tools

import (
	"fmt"
	"strings"
	"time"

	"github.com/alexpls/untils/internal/models"
	"github.com/alexpls/untils/internal/search"
)

var searchTool = tool[models.SearchRequestParams]{
	name:        models.LLMToolNameSearchRequest,
	description: "Use a web search engine to search for the given query and retrieve relevant results",
	usageBody: `- You will get up to 10 results from a web search about the subject. This will be
  enough for you to start navigating them and seeing if the results are suitable.
- Avoid calling this query more than once per check. You can do this by ensuring
  that the query you specify is likely to yield good results. Prefer spending extra
  time coming up with a good query rather than calling this tool multiple times.
- When applicable to the subject, prefer search queries for lists of things
  (e.g. list of taylor swift albums, or taylor swift discography). This will help find
  URLs that are more evergreen and likely to be useful for future checks as well.
- If the subject is about something that recurs (e.g. latest movie in a franchise, or
  latest game to be reviewed at 10/10 by a publisher), then craft your search query to
  return a list of the relevant things in that series. This will help you monitor it
  more easily in the future.
- Including words like "latest" or "most recent" in your search query does not yield
  better results. We are dealing with a text matching search engine (not a semantic one).`,
	execute: func(tc *Context, p models.SearchRequestParams) (string, error) {
		tc.Logger.DebugContext(tc.Ctx, "search_request started", "query", p.Query)
		start := time.Now()

		res, err := tc.Search(search.NewSearchParams(p.Query))
		tc.Logger.DebugContext(tc.Ctx, "search_request search completed", "duration", time.Since(start))

		if err != nil {
			return "", fmt.Errorf("performing search: %w", err)
		}

		var sb strings.Builder
		sb.WriteString("## Search results for query: " + p.Query + "\n\n")
		for _, result := range res.Results {
			sb.WriteString("- " + result.String() + "\n")
		}

		tc.Logger.DebugContext(tc.Ctx, "search_request completed", "query", p.Query, "total_duration", time.Since(start))
		return sb.String(), nil
	},
	validate: func(tc *Context, params models.SearchRequestParams) string {
		return noDuplicateCallsValidator(tc, params, "searching with the same query twice is not allowed. "+
			"try adjusting the query or using an existing result from a previous search")
	},
}
