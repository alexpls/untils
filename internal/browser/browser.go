package browser

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/chromedp/chromedp"
)

const (
	DefaultSessionTimeout = 5 * time.Minute
)

var ErrBrowserSessionLimitExceeded = errors.New("browser session limit exceeded")

type BrowserSession struct {
	context.Context
	logger *slog.Logger
}

type BrowserSessionConfig struct {
	ChromeDevToolsURL string
	SessionTimeout    time.Duration
}

type Manager struct {
	config BrowserSessionConfig
	logger *slog.Logger
	slots  chan struct{}
}

func NewManager(maxConcurrentSessions int, config BrowserSessionConfig, logger *slog.Logger) *Manager {
	if maxConcurrentSessions <= 0 {
		panic("maxConcurrentSessions must be greater than zero")
	}

	if config.SessionTimeout == 0 {
		config.SessionTimeout = DefaultSessionTimeout
	}

	return &Manager{
		config: config,
		logger: logger,
		slots:  make(chan struct{}, maxConcurrentSessions),
	}
}

func (m *Manager) NewSession(parentCtx context.Context) (BrowserSession, context.CancelFunc, error) {
	select {
	case m.slots <- struct{}{}:
	default:
		return BrowserSession{}, nil, ErrBrowserSessionLimitExceeded
	}

	sessionCtx, sessionTimeoutCancel := context.WithTimeout(parentCtx, m.config.SessionTimeout)

	releaseSlot := func() {
		<-m.slots
	}

	cleanupOnError := func(err error) (BrowserSession, context.CancelFunc, error) {
		sessionTimeoutCancel()
		releaseSlot()
		return BrowserSession{}, nil, err
	}

	var (
		ctx         context.Context
		cancel      context.CancelFunc
		allocCancel context.CancelFunc
	)

	if m.config.ChromeDevToolsURL == "" {
		ctx, cancel = chromedp.NewContext(sessionCtx)
	} else {
		allocCtx, remoteAllocCancel := chromedp.NewRemoteAllocator(sessionCtx, m.config.ChromeDevToolsURL)
		allocCancel = remoteAllocCancel
		ctx, cancel = chromedp.NewContext(allocCtx)
	}

	if ctx == nil || cancel == nil {
		return cleanupOnError(fmt.Errorf("initializing browser session"))
	}

	session := BrowserSession{
		Context: ctx,
		logger:  m.logger,
	}

	var once sync.Once

	cleanup := func() {
		once.Do(func() {
			cancel()
			if allocCancel != nil {
				allocCancel()
			}
			sessionTimeoutCancel()
			releaseSlot()
		})
	}

	go func() {
		<-sessionCtx.Done()
		cleanup()
	}()

	return session, cleanup, nil
}
