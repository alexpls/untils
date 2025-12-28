package browser

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/chromedp/cdproto/accessibility"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

type Page struct {
	Title    string
	Contents string
}

type NavigateResult struct {
	BrowserCtx    context.Context
	BrowserCancel context.CancelFunc
	Page          *Page
}

type BrowserCtx struct {
	context.Context
}

func NewBrowser(parentCtx context.Context) (BrowserCtx, context.CancelFunc) {
	ctx, cancel := chromedp.NewContext(parentCtx)
	return BrowserCtx{ctx}, cancel
}

func (ctx *BrowserCtx) Click(idStr string) (*Page, error) {
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("parsing node id: %w", err)
	}
	nodeID := cdp.NodeID(id)

	timeoutCtx, timeoutCancel := context.WithTimeout(ctx, 30*time.Second)
	defer timeoutCancel()

	var nodes []*cdp.Node

	if err := chromedp.Run(
		timeoutCtx,
		chromedp.Nodes([]cdp.NodeID{nodeID}, &nodes, chromedp.ByNodeID),
	); err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nil, fmt.Errorf("no nodes found for id %d", id)
	}
	if len(nodes) > 1 {
		return nil, fmt.Errorf("more than one node found for id %d", id)
	}

	if err := chromedp.Run(timeoutCtx,
		chromedp.MouseClickNode(nodes[0]),
	); err != nil {
		return nil, err
	}

	return pageResult(timeoutCtx)
}

func (ctx *BrowserCtx) Navigate(path string) (*Page, error) {
	timeoutCtx, timeoutCancel := context.WithTimeout(ctx, 30*time.Second)
	defer timeoutCancel()

	if err := chromedp.Run(timeoutCtx,
		accessibility.Enable(),
		emulation.SetEmulatedMedia().WithMedia("print"),
		chromedp.Navigate(path),
		waitForNetworkIdle(5*time.Second),
	); err != nil {
		return nil, err
	}

	return pageResult(timeoutCtx)
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
