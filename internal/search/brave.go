package search

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
)

type BraveClient struct {
	apiKey string
	logger *slog.Logger
}

func NewBraveClient(apiKey string, logger *slog.Logger) *BraveClient {
	return &BraveClient{
		apiKey: apiKey,
		logger: logger,
	}
}

type braveSearchSearchResult struct {
	Title       string `json:"title"`
	Type        string `json:"type"`
	URL         string `json:"url"`
	Description string `json:"description"`
}

func (b braveSearchSearchResult) toSearchResult() *SearchResult {
	return &SearchResult{
		Title:       b.Title,
		URL:         b.URL,
		Description: b.Description,
	}
}

type braveSearchSearch struct {
	Type    string                    `json:"type"`
	Results []braveSearchSearchResult `json:"results"`
}

type braveSearchAPIResponse struct {
	Type string            `json:"type"`
	Web  braveSearchSearch `json:"web"`
}

func (b braveSearchAPIResponse) toSearchResponse() *SearchResponse {
	results := make([]*SearchResult, len(b.Web.Results))
	for i, r := range b.Web.Results {
		results[i] = r.toSearchResult()
	}
	return &SearchResponse{Results: results}
}

func (c *BraveClient) Search(params *SearchParams) (*SearchResponse, error) {
	ps := url.Values{}
	ps.Add("q", params.Query)
	ps.Add("count", strconv.Itoa(params.Count))
	ps.Add("result_filter", "web")

	req, err := http.NewRequest(
		http.MethodGet,
		"https://api.search.brave.com/res/v1/web/search?"+ps.Encode(),
		nil,
	)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-Subscription-Token", c.apiKey)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close() // nolint:errcheck

	var apiRes braveSearchAPIResponse
	if err = json.NewDecoder(res.Body).Decode(&apiRes); err != nil {
		return nil, err
	}

	return apiRes.toSearchResponse(), nil
}
