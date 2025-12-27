package browser

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/chromedp/chromedp"

	_ "embed"
)

//go:embed tidyhtml_wikipedia.js
var wikipediaTidyScript string

func tidyHTML(u *url.URL) chromedp.ActionFunc {
	return func(ctx context.Context) error {
		if strings.HasSuffix(u.Hostname(), "wikipedia.org") {
			if err := chromedp.Evaluate(wikipediaTidyScript, nil).Do(ctx); err != nil {
				return fmt.Errorf("running wikipedia tidy script: %w", err)
			}
		}

		return nil
	}
}
