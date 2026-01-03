package browser

import (
	"context"

	"github.com/chromedp/chromedp"

	_ "embed"
)

//go:embed favicon.js
var faviconJS string

func favicon(out *string) chromedp.ActionFunc {
	if out == nil {
		panic("out cannot be nil")
	}
	return func(ctx context.Context) error {
		return chromedp.Run(ctx, chromedp.Evaluate(faviconJS, out))
	}
}
