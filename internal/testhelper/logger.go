package testhelper

import (
	"log/slog"
	"testing"
)

// testWriter adapts testing.TB to io.Writer
type testWriter struct {
	t testing.TB
}

func (w testWriter) Write(p []byte) (n int, err error) {
	w.t.Helper()
	// Trim trailing newline since t.Log adds one
	msg := string(p)
	if len(msg) > 0 && msg[len(msg)-1] == '\n' {
		msg = msg[:len(msg)-1]
	}
	w.t.Log(msg)
	return len(p), nil
}

// TestLogger returns a slog.Logger that writes to the test's log output.
// Log messages will only appear when running with `go test -v` or when the test fails.
func TestLogger(t testing.TB) *slog.Logger {
	t.Helper()

	handler := slog.NewTextHandler(testWriter{t}, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})

	return slog.New(handler)
}
