package docs

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizePath(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"":                        "/docs",
		"/":                       "/docs",
		"docs":                    "/docs",
		"/docs":                   "/docs",
		"/docs/":                  "/docs",
		"self-hosting/quickstart": "/docs/self-hosting/quickstart",
		"/docs/self-hosting/":     "/docs/self-hosting",
	}

	for input, want := range cases {
		if got := NormalizePath(input); got != want {
			t.Fatalf("NormalizePath(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestCurrentSiteIndexPath(t *testing.T) {
	t.Parallel()

	site, err := loadSiteFS(os.DirFS("../../docs/public"))
	require.NoError(t, err)
	require.Equal(t, "/docs/introduction/welcome", site.IndexPath)

	page, ok := site.Page(site.IndexPath)
	require.True(t, ok, "site.Page(%q) not found", site.IndexPath)
	require.Equal(t, "/docs/introduction/welcome", page.Path)
	require.Equal(t, "Welcome", page.Title)
	require.Equal(t, "Welcome", page.SidebarTitle)
	require.NotEmpty(t, page.LastUpdated)
}

func TestLoadSiteExtractsH2Headings(t *testing.T) {
	t.Parallel()

	site, err := loadSiteFS(os.DirFS("../../docs/public"))
	require.NoError(t, err)

	page, ok := site.Page("/docs/self-hosting/quickstart")
	require.True(t, ok)
	require.NotEmpty(t, page.Headings)

	firstHeading := page.Headings[0]
	require.Equal(t, "Prerequisites", firstHeading.Title)
	require.Equal(t, "prerequisites", firstHeading.ID)
}

func TestLoadSiteLoadsOpenAPIReferenceContent(t *testing.T) {
	t.Parallel()

	site, err := loadSiteFS(os.DirFS("../../docs/public"))
	require.NoError(t, err)

	page, ok := site.Page(apiReferencePath)
	require.True(t, ok)
	content := renderContent(t, page)
	require.NotEmpty(t, content)
	require.NotContains(t, content, openAPIMethodsTag)
	require.NotContains(t, content, openAPIWebhooksTag)
	require.NotContains(t, content, openAPITypesTag)
	require.True(t, page.HideTableOfContents)
	require.NotNil(t, page.Content)
	require.Contains(t, page.Headings, PageHeading{Title: "Methods", ID: "methods"})
	require.Contains(t, page.Headings, PageHeading{Title: "Webhooks", ID: "webhooks"})
	require.Contains(t, page.Headings, PageHeading{Title: "Types", ID: "types"})
	require.Contains(t, navChildPaths(site.NavSections, apiReferencePath), apiReferenceURL("results.list"))
}

func TestLoadOpenAPIReferenceBuildsMethods(t *testing.T) {
	t.Parallel()

	reference, err := LoadOpenAPIReference("../../docs/public/openapi.yml")
	require.NoError(t, err)

	monitorGet := requireAPIMethod(t, reference, "monitor.get")
	require.NotEmpty(t, monitorGet.Name)
	require.Equal(t, "GET", monitorGet.HTTPMethod)
	require.Equal(t, "/api/monitor.get", monitorGet.Endpoint)
	require.Equal(t, "Bearer token", monitorGet.Authorization)
	require.Equal(t, []string{"monitor_id"}, apiParameterNames(monitorGet.Parameters))
	require.True(t, monitorGet.Parameters[0].Required)
	require.Equal(t, "query", monitorGet.Parameters[0].In)
	require.Equal(t, "integer", monitorGet.Parameters[0].Schema.Type)
	require.Equal(t, "int64", monitorGet.Parameters[0].Schema.Format)

	listResults := requireAPIMethod(t, reference, "results.list")
	require.NotEmpty(t, listResults.Name)
	require.Equal(t, "GET", listResults.HTTPMethod)
	require.Equal(t, "/api/results.list", listResults.Endpoint)
	require.Equal(t, []string{"monitor_id", "page", "per_page"}, apiParameterNames(listResults.Parameters))

	listResultsResponse := requireAPIResponse(t, listResults, "200")
	require.Contains(t, apiSchemaRowFields(listResultsResponse.SchemaRows), "data.results")
	require.Contains(t, apiSchemaRowFields(listResultsResponse.SchemaRows), "data.links")
	require.NotContains(t, apiSchemaRowFields(listResultsResponse.SchemaRows), "data.results[].fields[].value")

	listLatest := requireAPIMethod(t, reference, "results.list_latest")
	require.NotEmpty(t, listLatest.Name)
	require.Equal(t, "GET", listLatest.HTTPMethod)
	require.Equal(t, "/api/results.list_latest", listLatest.Endpoint)
	require.Empty(t, listLatest.Parameters)

	listLatestResponse := requireAPIResponse(t, listLatest, "200")
	resultSummaries := requireAPISchemaRow(t, listLatestResponse.SchemaRows, "data.result_summaries")
	require.NotNil(t, resultSummaries.Schema.MaxItems)
	require.Equal(t, 30, *resultSummaries.Schema.MaxItems)
	require.NotNil(t, resultSummaries.Schema.Items)
	require.Equal(t, "ResultSummary", resultSummaries.Schema.Items.RefName)
	require.NotContains(t, apiSchemaRowFields(listLatestResponse.SchemaRows), "data.result_summaries[].fields[].value")
}

func TestLoadOpenAPIReferenceBuildsTypes(t *testing.T) {
	t.Parallel()

	reference, err := LoadOpenAPIReference("../../docs/public/openapi.yml")
	require.NoError(t, err)

	typeNames := apiReferenceTypeNames(reference)
	require.Contains(t, typeNames, "APIError")
	require.Contains(t, typeNames, "ResultSummary")
	require.Contains(t, typeNames, "ResultField")
	require.NotContains(t, typeNames, "ListLatestResultsResponse")
	require.NotContains(t, typeNames, "WebhookNewResultsPayload")

	resultSummary := requireAPIType(t, reference, "ResultSummary")
	require.NotEmpty(t, resultSummary.ExampleJSON)
	require.Contains(t, apiSchemaRowFields(resultSummary.SchemaRows), "monitor_id")
	require.Contains(t, apiSchemaRowFields(resultSummary.SchemaRows), "created_at")
	require.Contains(t, apiSchemaRowFields(resultSummary.SchemaRows), "fields")

	createdAt := requireAPISchemaRow(t, resultSummary.SchemaRows, "created_at")
	require.Equal(t, "string", createdAt.Schema.Type)
	require.Equal(t, "date-time", createdAt.Schema.Format)

	fields := requireAPISchemaRow(t, resultSummary.SchemaRows, "fields")
	require.Equal(t, "array", fields.Schema.Type)
	require.NotNil(t, fields.Schema.Items)
	require.Equal(t, "ResultField", fields.Schema.Items.RefName)

	monitor := requireAPIType(t, reference, "Monitor")
	status := requireAPISchemaRow(t, monitor.SchemaRows, "status")
	require.Equal(t, []string{"validating", "previewing", "rejected", "ready", "active", "paused"}, status.Schema.EnumValues)
}

func TestLoadOpenAPIReferenceBuildsWebhookPayloads(t *testing.T) {
	t.Parallel()

	reference, err := LoadOpenAPIReference("../../docs/public/openapi.yml")
	require.NoError(t, err)

	require.Len(t, reference.Webhooks, 1)
	webhook := reference.Webhooks[0]
	require.Equal(t, "webhook.delivery", webhook.Anchor)
	require.Equal(t, []string{"WebhookTestPayload", "WebhookNewResultsPayload"}, apiWebhookPayloadNames(webhook.RequestPayloads))

	newResults := requireAPIWebhookPayload(t, webhook, "WebhookNewResultsPayload")
	require.NotEmpty(t, newResults.ExampleJSON)
	require.Contains(t, apiSchemaRowFields(newResults.SchemaRows), "message.monitor")
	require.Contains(t, apiSchemaRowFields(newResults.SchemaRows), "message.new_results")
	require.Contains(t, apiSchemaRowFields(newResults.SchemaRows), "message.old_result")

	newResultsRow := requireAPISchemaRow(t, newResults.SchemaRows, "message.new_results")
	require.Equal(t, "array", newResultsRow.Schema.Type)
	require.NotNil(t, newResultsRow.Schema.MinItems)
	require.Equal(t, 1, *newResultsRow.Schema.MinItems)
	require.NotNil(t, newResultsRow.Schema.Items)
	require.Equal(t, "Result", newResultsRow.Schema.Items.RefName)
}

func TestRenderOpenAPIReference(t *testing.T) {
	t.Parallel()

	reference, err := LoadOpenAPIReference("../../docs/public/openapi.yml")
	require.NoError(t, err)

	var html bytes.Buffer
	err = OpenAPIReferenceContent(reference).Render(context.Background(), &html)
	require.NoError(t, err)

	content := html.String()
	require.Contains(t, content, `id="methods"`)
	require.Contains(t, content, `id="webhooks"`)
	require.Contains(t, content, `id="types"`)
}

func TestOpenAPIDescriptionMarkdown(t *testing.T) {
	t.Parallel()

	descriptionHTML, _, err := renderMarkdownHTML(markdownRenderer(), "First paragraph with `inline code`.\n\n```json\n{\"ok\": true}\n```")
	require.NoError(t, err)

	var html bytes.Buffer
	err = openAPIDescription(descriptionHTML, "text-sm").Render(context.Background(), &html)
	require.NoError(t, err)

	content := html.String()
	require.Contains(t, content, `<p>First paragraph with <code>inline code</code>.</p>`)
	require.Contains(t, content, `<pre style=`)
	require.Contains(t, content, `ok`)
	require.NotContains(t, content, "```")
}

func TestOpenAPIExampleJSONPreservesYAMLFieldOrder(t *testing.T) {
	t.Parallel()

	reference, err := LoadOpenAPIReference("../../docs/public/openapi.yml")
	require.NoError(t, err)

	listLatest := requireAPIMethod(t, reference, "results.list_latest")
	listLatestResponse := requireAPIResponse(t, listLatest, "200")
	requireSubstringsInOrder(t, listLatestResponse.ExampleJSON, `"error"`, `"data"`, `"type"`, `"result_summaries"`)

	listResults := requireAPIMethod(t, reference, "results.list")
	listResultsResponse := requireAPIResponse(t, listResults, "200")
	require.Contains(t, listResultsResponse.ExampleJSON, "?monitor_id=42&page=2")
	require.NotContains(t, listResultsResponse.ExampleJSON, "\\u0026")

	resultSummary := requireAPIType(t, reference, "ResultSummary")
	requireSubstringsInOrder(t, resultSummary.ExampleJSON, `"type"`, `"id"`, `"monitor_id"`, `"monitor_subject"`, `"created_at"`)
	requireSubstringsInOrder(t, resultSummary.ExampleJSON, `"fields"`, `"type"`, `"name"`, `"value"`)
	require.NotContains(t, resultSummary.ExampleJSON, `"monitor"`)
}

func TestAPIReferenceNav(t *testing.T) {
	t.Parallel()

	reference, err := LoadOpenAPIReference("../../docs/public/openapi.yml")
	require.NoError(t, err)

	nav := APIReferenceNav(reference)
	require.Len(t, nav, 3)
	require.NotEmpty(t, nav[0].Children)
	require.NotEmpty(t, nav[1].Children)
	require.NotEmpty(t, nav[2].Children)

	require.Contains(t, apiReferenceNavPaths(nav[0].Children), "/docs/api/reference#monitor.get")
	require.Contains(t, apiReferenceNavPaths(nav[0].Children), "/docs/api/reference#results.list")
	require.Contains(t, apiReferenceNavPaths(nav[0].Children), "/docs/api/reference#results.list_latest")
	require.Contains(t, apiReferenceNavPaths(nav[1].Children), "/docs/api/reference#WebhookTestPayload")
	require.Contains(t, apiReferenceNavPaths(nav[1].Children), "/docs/api/reference#WebhookNewResultsPayload")
	require.NotContains(t, apiReferenceNavPaths(nav[1].Children), "/docs/api/reference#webhook.delivery")
	require.Contains(t, apiReferenceNavPaths(nav[2].Children), "/docs/api/reference#APIError")
	require.Contains(t, apiReferenceNavPaths(nav[2].Children), "/docs/api/reference#ResultSummary")
	require.NotContains(t, apiReferenceNavPaths(nav[2].Children), "/docs/api/reference#WebhookNewResultsPayload")
	require.NotContains(t, nav[2].Children[0].Path, "ListLatestResultsResponse")
}

func apiReferenceTypeNames(reference APIReference) []string {
	names := make([]string, len(reference.Types))
	for i, apiType := range reference.Types {
		names[i] = apiType.Name
	}
	return names
}

func apiReferenceNavPaths(pages []NavPage) []string {
	paths := make([]string, len(pages))
	for i, page := range pages {
		paths[i] = page.Path
	}
	return paths
}

func apiParameterNames(params []APIParameter) []string {
	names := make([]string, len(params))
	for i, param := range params {
		names[i] = param.Name
	}
	return names
}

func apiWebhookPayloadNames(payloads []APIWebhookPayload) []string {
	names := make([]string, len(payloads))
	for i, payload := range payloads {
		names[i] = payload.Name
	}
	return names
}

func apiSchemaRowFields(rows []APISchemaRow) []string {
	fields := make([]string, len(rows))
	for i, row := range rows {
		fields[i] = row.Field
	}
	return fields
}

func requireAPIMethod(t *testing.T, reference APIReference, anchor string) APIMethod {
	t.Helper()

	for _, method := range reference.Methods {
		if method.Anchor == anchor {
			return method
		}
	}
	require.FailNowf(t, "missing api method", "method with anchor %q not found", anchor)
	return APIMethod{}
}

func requireAPIResponse(t *testing.T, method APIMethod, status string) APIResponse {
	t.Helper()

	for _, response := range method.Responses {
		if response.Status == status {
			return response
		}
	}
	require.FailNowf(t, "missing api response", "response with status %q not found for %q", status, method.Anchor)
	return APIResponse{}
}

func requireAPIType(t *testing.T, reference APIReference, name string) APIType {
	t.Helper()

	for _, apiType := range reference.Types {
		if apiType.Name == name {
			return apiType
		}
	}
	require.FailNowf(t, "missing api type", "type %q not found", name)
	return APIType{}
}

func requireAPIWebhookPayload(t *testing.T, webhook APIWebhook, name string) APIWebhookPayload {
	t.Helper()

	for _, payload := range webhook.RequestPayloads {
		if payload.Name == name {
			return payload
		}
	}
	require.FailNowf(t, "missing webhook payload", "payload %q not found for %q", name, webhook.Anchor)
	return APIWebhookPayload{}
}

func requireAPISchemaRow(t *testing.T, rows []APISchemaRow, field string) APISchemaRow {
	t.Helper()

	for _, row := range rows {
		if row.Field == field {
			return row
		}
	}
	require.FailNowf(t, "missing api schema row", "schema row %q not found", field)
	return APISchemaRow{}
}

func requireSubstringsInOrder(t *testing.T, content string, substrings ...string) {
	t.Helper()

	offset := 0
	for _, substring := range substrings {
		idx := strings.Index(content[offset:], substring)
		require.NotEqualf(t, -1, idx, "%q not found after byte %d in %q", substring, offset, content)
		offset += idx + len(substring)
	}
}

func navChildPaths(sections []NavSection, parentPath string) []string {
	for _, section := range sections {
		for _, page := range section.Pages {
			if page.Path == parentPath {
				return allNavPaths(page.Children)
			}
		}
	}
	return nil
}

func allNavPaths(pages []NavPage) []string {
	paths := make([]string, 0, len(pages))
	for _, page := range pages {
		paths = append(paths, page.Path)
		paths = append(paths, allNavPaths(page.Children)...)
	}
	return paths
}

func renderContent(t *testing.T, page Page) string {
	t.Helper()

	var html bytes.Buffer
	require.NoError(t, page.Content.Render(context.Background(), &html))
	return html.String()
}
