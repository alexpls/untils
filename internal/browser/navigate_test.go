package browser

import (
	"context"
	"testing"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/stretchr/testify/require"
)

func TestWaitForNetworkIdleAllowsSoftTimeout(t *testing.T) {
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	err := waitForNetworkIdle(10 * time.Millisecond)(ctx)
	require.NoError(t, err)
}
