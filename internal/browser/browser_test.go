package browser_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alexpls/untils/internal/browser"
	"github.com/alexpls/untils/internal/testhelper"
	"github.com/stretchr/testify/require"
)

func TestManagerReturnsLimitExceededWhenAllSessionsInUse(t *testing.T) {
	tl := testhelper.TestLogger(t)
	manager := browser.NewManager(1, browser.BrowserSessionConfig{}, tl)

	_, cancel, err := manager.NewSession(context.Background())
	require.NoError(t, err)
	defer cancel()

	_, _, err = manager.NewSession(context.Background())
	require.Error(t, err)
	require.True(t, errors.Is(err, browser.ErrBrowserSessionLimitExceeded))
}

func TestManagerReleasesSlotOnCancel(t *testing.T) {
	tl := testhelper.TestLogger(t)
	manager := browser.NewManager(1, browser.BrowserSessionConfig{}, tl)

	_, cancel, err := manager.NewSession(context.Background())
	require.NoError(t, err)

	cancel()

	_, cancel2, err := manager.NewSession(context.Background())
	require.NoError(t, err)
	defer cancel2()
}

func TestManagerReleasesSlotWhenSessionTimesOut(t *testing.T) {
	tl := testhelper.TestLogger(t)
	manager := browser.NewManager(1, browser.BrowserSessionConfig{
		SessionTimeout: 10 * time.Millisecond,
	}, tl)

	_, cancel, err := manager.NewSession(context.Background())
	require.NoError(t, err)
	defer cancel()

	time.Sleep(30 * time.Millisecond)

	_, cancel2, err := manager.NewSession(context.Background())
	require.NoError(t, err)
	defer cancel2()
}
