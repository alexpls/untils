package browser

import (
	"context"
	"fmt"
	"time"

	"github.com/chromedp/cdproto/accessibility"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

type Page struct {
	Title    string
	Contents string
}

type GoToResult struct {
	Page Page
}

func GoTo(ctx context.Context, path string) (*GoToResult, error) {
	browserCtx, browserCancel := chromedp.NewContext(ctx)
	defer browserCancel()

	timeoutCtx, timeoutCancel := context.WithTimeout(browserCtx, 30*time.Second)
	defer timeoutCancel()

	var title string
	var tree axTreeResponse

	if err := chromedp.Run(timeoutCtx,
		accessibility.Enable(),
		chromedp.Navigate(path),
		waitForNetworkIdle(),
		chromedp.Title(&title),
		accessibilityTree(&tree),
	); err != nil {
		return nil, err
	}

	pageContents := tree.String()

	return &GoToResult{
		Page: Page{
			Title:    title,
			Contents: pageContents,
		},
	}, nil
}

// waitForNetworkIdle waits until the event networkIdle is fired or the
// context timeout.
func waitForNetworkIdle() chromedp.ActionFunc {
	return func(ctx context.Context) error {
		ch := make(chan struct{})
		cctx, cancel := context.WithCancel(ctx)

		chromedp.ListenTarget(cctx, func(ev any) {
			switch e := ev.(type) {
			case *page.EventLifecycleEvent:
				if e.Name == "networkIdle" {
					cancel()
					close(ch)
				}
			}
		})

		select {
		case <-ch:
			return nil
		case <-ctx.Done():
			return fmt.Errorf("wait for event networkIdle: %w", ctx.Err())
		}
	}
}
