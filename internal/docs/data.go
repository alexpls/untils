package docs

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/a-h/templ"
	docscontent "github.com/alexpls/untils/docs"
	"github.com/alexpls/untils/internal/codehighlight"
	"github.com/alexpls/untils/internal/must"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	mdhtml "github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
	"gopkg.in/yaml.v3"
)

type Page struct {
	Title               string
	SidebarTitle        string
	Description         string
	Section             string
	Path                string
	LastUpdated         string
	HideTableOfContents bool
	Content             templ.Component
	Headings            []PageHeading
}

type PageHeading struct {
	Title string
	ID    string
}

type NavPage struct {
	SidebarTitle string
	Path         string
	Children     []NavPage
}

type NavSection struct {
	Title string
	Pages []NavPage
}

type Site struct {
	IndexPath   string
	PagesByPath map[string]Page
	NavSections []NavSection
}

type frontmatter struct {
	Title               string `yaml:"title"`
	SidebarTitle        string `yaml:"sidebar_title"`
	URL                 string `yaml:"url"`
	Section             string `yaml:"section"`
	Description         string `yaml:"description"`
	LastUpdated         string `yaml:"last_updated"`
	HideTableOfContents bool   `yaml:"hide_table_of_contents"`
}

const (
	apiReferencePath   = "/docs/api/reference"
	openAPISpecPath    = "openapi.yml"
	openAPIMethodsTag  = "{{< openapi:methods >}}"
	openAPIWebhooksTag = "{{< openapi:webhooks >}}"
	openAPITypesTag    = "{{< openapi:types >}}"
)

var docsRootPath = filepath.Join("docs", "public")
var currentDocsFS fs.FS = docscontent.PublicFS()

var currentSite Site
var siteLoaded bool

func SetDevMode() {
	if siteLoaded {
		panic("can't set dev mode after site has already been loaded")
	}
	currentDocsFS = os.DirFS(docsRootPath)
}

// Load eagerly loads the docs site, including the OpenAPI reference.
// Intended to be called once at startup so any docs misconfiguration fails
// fast rather than crashing the first request goroutine.
func Load() {
	if siteLoaded {
		panic("docs have already been loaded")
	}
	docsFS := currentDocsFS
	currentSite = must.NoErrVal(loadSiteFS(docsFS))
	siteLoaded = true
}

func loadSiteFS(docsFS fs.FS) (Site, error) {
	files := make([]string, 0)
	err := fs.WalkDir(docsFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || filepath.Ext(path) != ".md" {
			return nil
		}
		files = append(files, path)
		return nil
	})
	if err != nil {
		return Site{}, err
	}
	if len(files) == 0 {
		return Site{}, fmt.Errorf("no docs markdown files found")
	}
	slices.Sort(files)

	renderer := markdownRenderer()

	openAPIReference, err := loadOpenAPIReferenceFS(docsFS, openAPISpecPath, renderer)
	if err != nil {
		return Site{}, err
	}
	openAPINav := APIReferenceNav(openAPIReference)

	indexPath := ""
	sections := make([]NavSection, 0)
	pagesByPath := make(map[string]Page, len(files))
	sectionIndexes := make(map[string]int)
	for _, path := range files {
		page, err := loadPage(docsFS, path, renderer)
		if err != nil {
			return Site{}, err
		}
		if indexPath == "" {
			indexPath = page.Path
		}

		navChildren := []NavPage(nil)
		if page.Path == apiReferencePath {
			page.Content = OpenAPIReferenceContent(openAPIReference)
			navChildren = openAPINav
		}
		if err := registerPage(pagesByPath, &sections, sectionIndexes, page, navChildren); err != nil {
			return Site{}, err
		}
	}

	return Site{
		IndexPath:   indexPath,
		PagesByPath: pagesByPath,
		NavSections: sections,
	}, nil
}

func (s Site) Page(path string) (Page, bool) {
	page, ok := s.PagesByPath[NormalizePath(path)]
	return page, ok
}

func (s Site) MustIndex() Page {
	page, ok := s.Page(s.IndexPath)
	if !ok {
		panic("docs index page missing")
	}
	return page
}

func NormalizePath(path string) string {
	if path == "" || path == "/" {
		return "/docs"
	}

	path = strings.TrimSpace(path)
	path = strings.TrimPrefix(path, "/")
	path = strings.TrimSuffix(path, "/")

	if path == "" || path == "docs" {
		return "/docs"
	}

	path = strings.TrimPrefix(path, "docs/")
	return "/docs/" + path
}

func registerPage(pagesByPath map[string]Page, sections *[]NavSection, sectionIndexes map[string]int, page Page, navChildren []NavPage) error {
	if _, exists := pagesByPath[page.Path]; exists {
		return fmt.Errorf("duplicate docs path %q", page.Path)
	}

	pagesByPath[page.Path] = page

	navPage := NavPage{
		SidebarTitle: page.SidebarTitle,
		Path:         page.Path,
		Children:     navChildren,
	}

	idx, ok := sectionIndexes[page.Section]
	if !ok {
		idx = len(*sections)
		sectionIndexes[page.Section] = idx
		*sections = append(*sections, NavSection{Title: page.Section})
	}
	(*sections)[idx].Pages = append((*sections)[idx].Pages, navPage)
	return nil
}

func loadPage(docsFS fs.FS, path string, renderer goldmark.Markdown) (Page, error) {
	content, err := fs.ReadFile(docsFS, path)
	if err != nil {
		return Page{}, err
	}
	fm, body, err := splitFrontmatter(string(content))
	if err != nil {
		return Page{}, fmt.Errorf("%s: %w", path, err)
	}

	if fm.Title == "" {
		return Page{}, fmt.Errorf("%s: missing title frontmatter", path)
	}
	if fm.SidebarTitle == "" {
		return Page{}, fmt.Errorf("%s: missing sidebar_title frontmatter", path)
	}
	if fm.URL == "" {
		return Page{}, fmt.Errorf("%s: missing url frontmatter", path)
	}
	if fm.LastUpdated == "" {
		return Page{}, fmt.Errorf("%s: missing last_updated frontmatter", path)
	}

	lastUpdated, err := time.Parse("2 January 2006", fm.LastUpdated)
	if err != nil {
		return Page{}, fmt.Errorf("%s: invalid last_updated frontmatter: %w", path, err)
	}

	body = strings.TrimSpace(strings.ReplaceAll(body, "\r\n", "\n"))
	hasOpenAPIContent := hasOpenAPITags(body)
	body = removeTemplateTags(body)

	contentHTML, headings, err := renderMarkdownHTML(renderer, body)
	if err != nil {
		return Page{}, fmt.Errorf("%s: render markdown: %w", path, err)
	}
	contentComponent := templ.Raw(contentHTML)
	if hasOpenAPIContent {
		contentComponent = templ.NopComponent
	}

	return Page{
		Title:               fm.Title,
		SidebarTitle:        fm.SidebarTitle,
		Description:         fm.Description,
		Section:             fm.Section,
		Path:                normalizeRoutePath(fm.URL),
		LastUpdated:         lastUpdated.Format("2 January 2006"),
		HideTableOfContents: fm.HideTableOfContents,
		Content:             contentComponent,
		Headings:            headings,
	}, nil
}

func splitFrontmatter(content string) (frontmatter, string, error) {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	if !strings.HasPrefix(content, "---\n") {
		return frontmatter{}, "", fmt.Errorf("docs file missing frontmatter opening delimiter")
	}

	rest := strings.TrimPrefix(content, "---\n")
	idx := strings.Index(rest, "\n---\n")
	if idx == -1 {
		return frontmatter{}, "", fmt.Errorf("docs file missing frontmatter closing delimiter")
	}

	var fm frontmatter
	if err := yaml.Unmarshal([]byte(rest[:idx]), &fm); err != nil {
		return frontmatter{}, "", err
	}

	return fm, rest[idx+len("\n---\n"):], nil
}

func hasOpenAPITags(body string) bool {
	return strings.Contains(body, openAPIMethodsTag) ||
		strings.Contains(body, openAPIWebhooksTag) ||
		strings.Contains(body, openAPITypesTag)
}

func removeTemplateTags(body string) string {
	body = strings.ReplaceAll(body, openAPIMethodsTag, "")
	body = strings.ReplaceAll(body, openAPIWebhooksTag, "")
	body = strings.ReplaceAll(body, openAPITypesTag, "")
	return body
}

func extractH2Headings(source []byte, doc ast.Node) []PageHeading {
	headings := []PageHeading{}

	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		heading, ok := n.(*ast.Heading)
		if !ok || heading.Level != 2 {
			return ast.WalkContinue, nil
		}

		idValue, ok := heading.AttributeString("id")
		if !ok {
			return ast.WalkContinue, nil
		}

		id, ok := idValue.([]byte)
		if !ok {
			return ast.WalkContinue, nil
		}

		title := strings.TrimSpace(string(heading.Lines().Value(source)))
		if title == "" || len(id) == 0 {
			return ast.WalkContinue, nil
		}

		headings = append(headings, PageHeading{
			Title: title,
			ID:    string(id),
		})

		return ast.WalkContinue, nil
	})

	return headings
}

func normalizeRoutePath(url string) string {
	url = strings.TrimSpace(url)
	url = strings.TrimPrefix(url, "/")
	url = strings.TrimSuffix(url, "/")
	if url == "" {
		return "/docs"
	}
	return "/docs/" + url
}

func markdownRenderer() goldmark.Markdown {
	return goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithParserOptions(parser.WithAutoHeadingID(), parser.WithAttribute()),
		goldmark.WithRendererOptions(mdhtml.WithUnsafe()),
	)
}
func renderMarkdownHTML(renderer goldmark.Markdown, markdown string) (string, []PageHeading, error) {
	source := markdownSource(markdown)
	contentHTML, doc, err := renderMarkdownSourceHTML(renderer, source)
	if err != nil {
		return "", nil, err
	}
	return contentHTML, extractH2Headings(source, doc), nil
}

func markdownSource(markdown string) []byte {
	return []byte(strings.TrimSpace(strings.ReplaceAll(markdown, "\r\n", "\n")))
}

func renderMarkdownSourceHTML(renderer goldmark.Markdown, source []byte) (string, ast.Node, error) {
	context := parser.NewContext()
	doc := renderer.Parser().Parse(text.NewReader(source), parser.WithContext(context))

	var rendered bytes.Buffer
	if err := renderer.Renderer().Render(&rendered, source, doc); err != nil {
		return "", nil, err
	}

	contentHTML, err := codehighlight.HTML(rendered.String())
	if err != nil {
		return "", nil, err
	}

	return contentHTML, doc, nil
}
