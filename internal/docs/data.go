package docs

import "strings"

type Page struct {
	Title        string
	SidebarTitle string
	Description  string
	Section      string
	Path         string
	LastUpdated  string
	ContentHTML  string
	Headings     []PageHeading
}

type PageHeading struct {
	Title string
	ID    string
}

type NavPage struct {
	SidebarTitle string
	Path         string
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

var generatedSite Site

func CurrentSite() Site {
	return generatedSite
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
