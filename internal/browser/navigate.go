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
	URL      string
	Title    string
	Contents string
}

func (p Page) String() string {
	return fmt.Sprintf("Title: %s\nURL: %s\n\n%s", p.Title, p.URL, p.Contents)
}

type NavigateResult struct {
	BrowserCtx    context.Context
	BrowserCancel context.CancelFunc
	Page          *Page
}

// BrowserCtx wraps a chromedp context
type BrowserCtx struct {
	context.Context
	ID     string
	logger *slog.Logger
}

func NewBrowser(parentCtx context.Context, logger *slog.Logger) (BrowserCtx, context.CancelFunc) {
	// TODO: Make sure browser is configured with timeout

	ctx, cancel := chromedp.NewContext(parentCtx)
	return BrowserCtx{Context: ctx, logger: logger}, cancel
}

func (ctx *BrowserCtx) Click(idStr string) (*Page, error) {
	ctx.logger.Debug("clicking node", slog.String("id", idStr))

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("parsing node id: %w", err)
	}
	backendNodeID := cdp.BackendNodeID(id)

	// Resolve the backend node ID to a JavaScript object and click it.
	// Other approaches would mean pushing the node into the frontend, which
	// is more complex.
	// Still, there must be an easier way?
	if err := chromedp.Run(ctx,
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

	page, err := pageResult(ctx)
	if err != nil {
		return nil, err
	}

	ctx.logger.Debug("clicked node", slog.String("id", idStr), slog.String("new_url", page.URL))

	return page, nil
}

func (ctx *BrowserCtx) Navigate(path string) (*Page, error) {
	ctx.logger.Debug("navigating to page", slog.String("url", path))

	if err := chromedp.Run(ctx,
		accessibility.Enable(),
		emulation.SetEmulatedMedia().WithMedia("print"),
		chromedp.Navigate(path),
		waitForNetworkIdle(5*time.Second),
	); err != nil {
		return nil, err
	}

	return pageResult(ctx)
}

func pageResult(ctx context.Context) (*Page, error) {
	var urlStr string

	if err := chromedp.Run(ctx,
		chromedp.Location(&urlStr),
	); err != nil {
		return nil, err
	}

	u, err := url.Parse(urlStr)
	if err != nil {
		return nil, fmt.Errorf("parsing url: %w", err)
	}

	var title string
	var tree axTree

	if err := chromedp.Run(ctx,
		tidyHTML(u),
		chromedp.Title(&title),
		accessibilityTree(&tree),
	); err != nil {
		return nil, err
	}

	pageContents := tree.String()

	return &Page{
		URL:      urlStr,
		Title:    title,
		Contents: pageContents,
	}, nil
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
		return nil
	}
}
