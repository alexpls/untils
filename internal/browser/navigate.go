package browser

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/chromedp/cdproto/accessibility"
	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

type Page struct {
	Title    string
	Contents string
}

type NavigateResult struct {
	Page Page
}

func Navigate(ctx context.Context, path string) (*NavigateResult, error) {
	u, err := url.Parse(path)
	if err != nil {
		return nil, fmt.Errorf("parsing url: %w", err)
	}

	browserCtx, browserCancel := chromedp.NewContext(ctx)
	defer browserCancel()

	timeoutCtx, timeoutCancel := context.WithTimeout(browserCtx, 30*time.Second)
	defer timeoutCancel()

	var title string
	var tree axTree

	if err := chromedp.Run(timeoutCtx,
		accessibility.Enable(),
		emulation.SetEmulatedMedia().WithMedia("print"),
		chromedp.Navigate(u.String()),
		waitForNetworkIdle(5*time.Second),
		tidyHTML(u),
		chromedp.Title(&title),
		accessibilityTree(&tree),
	); err != nil {
		return nil, err
	}

	pageContents := tree.String()

	return &NavigateResult{
		Page: Page{
			Title:    title,
			Contents: pageContents,
		},
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
