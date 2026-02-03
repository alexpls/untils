package monitor

import (
	"context"
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

// sseReload sends javascript to the sse stream to reload the page
func sseReload(sse *datastar.ServerSentEventGenerator) error {
	js := "setTimeout(() => window.location.reload())"
	return sse.ExecuteScript(js)
}

// ConditionalPatchRenderer renders a [templ.Component] either as a patch within
// an SSE stream (updated each time [Updater] sends a message), or as a plain
// HTTP response.
type ConditionalPatchRenderer struct {
	Logger   *slog.Logger
	Renderer func(patch bool) (templ.Component, error)
	Updater  func(ctx context.Context) (<-chan struct{}, error)
}

// Handle responds to the [http.Request] by rendering the configured component. It
// either does this by opening an SSE stream when the query param ?sse=true is given, or by
// rendering straight to HTTP.
func (cpr *ConditionalPatchRenderer) Handle(w http.ResponseWriter, r *http.Request) {
	wantsPatches := r.URL.Query().Get("sse") == "true"

	if wantsPatches {
		sse := datastar.NewSSE(w, r)

		updates, err := cpr.Updater(sse.Context())
		if err != nil {
			cpr.Logger.ErrorContext(sse.Context(), "error subscribing for updates", "error", err)
			return
		}

		for {
			comp, err := cpr.Renderer(wantsPatches)
			if err != nil {
				cpr.Logger.ErrorContext(sse.Context(), "error rendering component", "error", err)
				return
			}
			if err := sse.PatchElementTempl(comp); err != nil {
				cpr.Logger.ErrorContext(sse.Context(), "error patching component", "error", err)
				return
			}

			select {
			case <-updates:
			case <-sse.Context().Done():
				return
			}
		}

	} else {
		comp, err := cpr.Renderer(wantsPatches)
		if err != nil {
			cpr.Logger.ErrorContext(r.Context(), "error rendering component", "error", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		if err := comp.Render(r.Context(), w); err != nil {
			cpr.Logger.ErrorContext(r.Context(), "error rendering component", "error", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	}
}
