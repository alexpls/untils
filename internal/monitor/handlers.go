package monitor

import (
	"bytes"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/a-h/templ"
	"github.com/starfederation/datastar-go/datastar"
)

// Handlers contains the HTTP handlers for monitor routes
type Handlers struct {
	service *Service
	events  *DBEventHandler
	logger  *slog.Logger
}

// NewHandlers creates a new Handlers instance
func NewHandlers(service *Service, events *DBEventHandler, logger *slog.Logger) *Handlers {
	return &Handlers{
		service: service,
		events:  events,
		logger:  logger,
	}
}

// monitorIDFromPath extracts monitor ID from the path
func monitorIDFromPath(r *http.Request) int64 {
	return idFromPath(r, "monitor_id")
}

// resultIDFromPath extracts result ID from the path
func resultIDFromPath(r *http.Request) int64 {
	return idFromPath(r, "result_id")
}

// checkIDFromPath extracts check ID from the path
func checkIDFromPath(r *http.Request) int64 {
	return idFromPath(r, "check_id")
}

// idFromPath extracts an int64 ID from the path of the request
func idFromPath(r *http.Request, name string) int64 {
	id, err := strconv.ParseInt(r.PathValue(name), 10, 64)
	if err != nil {
		return 0
	}
	return id
}

// ssePatchElementTemplFragment sends HTML to the sse stream for the given templ component and fragment
func ssePatchElementTemplFragment(sse *datastar.ServerSentEventGenerator, c templ.Component, fragmentIDs ...any) error {
	var buf bytes.Buffer
	if err := templ.RenderFragments(sse.Context(), &buf, c, fragmentIDs...); err != nil {
		return fmt.Errorf("failed to patch element: %w", err)
	}
	if err := sse.PatchElements(buf.String()); err != nil {
		return fmt.Errorf("failed to patch element: %w", err)
	}
	return nil
}

// sseReload sends javascript to the sse stream to reload the page
func sseReload(sse *datastar.ServerSentEventGenerator) error {
	js := "setTimeout(() => window.location.reload())"
	return sse.ExecuteScript(js)
}
