package browser

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"strconv"
	"time"

	"github.com/chromedp/cdproto/accessibility"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/dom"
	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
)

type Page struct {
	URL        string
	Title      string
	Contents   string
	FaviconURL string
}

func (p Page) String() string {
	return fmt.Sprintf("Title: %s\n"+
		"URL: %s\n"+
		"Favicon URL: %s\n"+
		"Contents:\n%s\n",
		p.Title,
		p.URL,
		p.FaviconURL,
		p.Contents)
}

type NavigateResult struct {
	BrowserSession context.Context
	BrowserCancel  context.CancelFunc
	Page           *Page
}

func (s *BrowserSession) Click(idStr string) (*Page, error) {
	start := time.Now()
	s.logger.DebugContext(s.Context, "clicking node", slog.String("id", idStr))

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("parsing node id: %w", err)
	}
	backendNodeID := cdp.BackendNodeID(id)

	// Resolve the backend node ID to a JavaScript object and click it.
	// Other approaches would mean pushing the node into the frontend, which
	// is more complex.
	// Still, there must be an easier way?
	clickActionStart := time.Now()
	if err := chromedp.Run(s,
		chromedp.ActionFunc(func(ctx context.Context) error {
			remoteObj, err := dom.ResolveNode().WithBackendNodeID(backendNodeID).Do(ctx)
			if err != nil {
				return err
			}
			_, _, err = runtime.CallFunctionOn(`function() {
				this.scrollIntoViewIfNeeded();
				this.click();
			}`).WithObjectID(remoteObj.ObjectID).Do(ctx)
			return err
		}),
		waitForNetworkIdle(5*time.Second),
	); err != nil {
		return nil, fmt.Errorf("clicking node: %w", err)
	}
	s.logger.DebugContext(s.Context, "click action and network idle completed", "duration", time.Since(clickActionStart))

	pageResultStart := time.Now()
	page, err := pageResult(s)
	if err != nil {
		return nil, err
	}
	s.logger.DebugContext(s.Context, "page result extracted after click", "duration", time.Since(pageResultStart))

	s.logger.DebugContext(s.Context, "clicked node", slog.String("id", idStr), slog.String("new_url", page.URL), "total_duration", time.Since(start))

	return page, nil
}

func (s *BrowserSession) Navigate(path string) (*Page, error) {
	start := time.Now()
	s.logger.DebugContext(s.Context, "navigating to page", slog.String("url", path))

	navigateStart := time.Now()
	if err := chromedp.Run(s,
		accessibility.Enable(),
		emulation.SetEmulatedMedia().WithMedia("print"),
		chromedp.Navigate(path),
		waitForNetworkIdle(5*time.Second),
	); err != nil {
		return nil, err
	}
	s.logger.DebugContext(s.Context, "navigation and network idle completed", "duration", time.Since(navigateStart))

	pageResultStart := time.Now()
	page, err := pageResult(s)
	if err != nil {
		return nil, err
	}
	s.logger.DebugContext(s.Context, "page result extracted", "duration", time.Since(pageResultStart), "total_duration", time.Since(start))

	return page, nil
}

func pageResult(s *BrowserSession) (*Page, error) {
	start := time.Now()
	var urlStr string

	locationStart := time.Now()
	if err := chromedp.Run(s,
		chromedp.Location(&urlStr),
	); err != nil {
		return nil, err
	}
	s.logger.DebugContext(s.Context, "got page location", "duration", time.Since(locationStart))

	u, err := url.Parse(urlStr)
	if err != nil {
		return nil, fmt.Errorf("parsing url: %w", err)
	}

	var title string
	var tree axTree
	var f string

	pageDataStart := time.Now()
	if err := chromedp.Run(s,
		tidyHTML(u),
		chromedp.Title(&title),
		favicon(&f),
		accessibilityTree(&tree),
	); err != nil {
		return nil, err
	}
	s.logger.DebugContext(s.Context, "got page data (tidy, title, favicon, accessibility tree)", "duration", time.Since(pageDataStart))

	treeStringStart := time.Now()
	pageContents := tree.String()
	s.logger.DebugContext(s.Context, "accessibility tree stringified", "duration", time.Since(treeStringStart), "content_length", len(pageContents))

	s.logger.DebugContext(s.Context, "page result completed", "total_duration", time.Since(start))

	return &Page{
		URL:        urlStr,
		Title:      title,
		Contents:   pageContents,
		FaviconURL: f,
	}, nil
}

// CurrentPage returns the current page contents without navigating
func (s *BrowserSession) CurrentPage() (*Page, error) {
	s.logger.DebugContext(s.Context, "getting current page")
	start := time.Now()

	page, err := pageResult(s)
	if err != nil {
		return nil, err
	}

	s.logger.DebugContext(s.Context, "current page retrieved", "url", page.URL, "total_duration", time.Since(start))
	return page, nil
}

// waitForNetworkIdle waits until the event networkIdle is fired or the
// timeout is reached
func waitForNetworkIdle(timeout time.Duration) chromedp.ActionFunc {
	return func(ctx context.Context) error {
		dctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		chromedp.ListenTarget(dctx, func(ev any) {
			switch e := ev.(type) {
			case *page.EventLifecycleEvent:
				if e.Name == "networkIdle" {
					cancel()
				}
			}
		})

		<-dctx.Done()

		// whether we got 'networkIdle' or the context timed out, return
		// nil either way, so the rest of the actions can continue with a
		// hopefully interactable page.

		return nil
	}
}
