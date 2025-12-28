package search

type SearchParams struct {
	Query string
	Count int
}

func NewSearchParams(query string) *SearchParams {
	return &SearchParams{
		Query: query,
		Count: 10,
	}
}

func (sp *SearchParams) WithCount(count int) *SearchParams {
	sp.Count = count
	return sp
}

type SearchResult struct {
	Title       string
	URL         string
	Description string
}

func (r SearchResult) String() string {
	return r.Title + " - " + r.URL
}

type SearchResponse struct {
	Results []*SearchResult
}

type WebSearcher interface {
	Search(params *SearchParams) (*SearchResponse, error)
}
