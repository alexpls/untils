package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"

	"github.com/alexpls/untils/internal/apimessage"
	"github.com/alexpls/untils/internal/errortypes"
	"github.com/alexpls/untils/internal/monitor"
	"github.com/alexpls/untils/internal/pagination"
	"github.com/alexpls/untils/internal/reqcontext"
)

const latestResultsLimit = 30
const defaultResultsLimit = 50
const maxResultsLimit = 100

type Handlers struct {
	service  *Service
	monitors *monitor.Service
	logger   *slog.Logger
}

func NewHandlers(service *Service, monitors *monitor.Service, logger *slog.Logger) *Handlers {
	if monitors == nil {
		panic("api: monitor service is required")
	}

	return &Handlers{
		service:  service,
		monitors: monitors,
		logger:   logger,
	}
}

type TokenLogEvent struct {
	ID   int64
	Name string
}

func (e *TokenLogEvent) Key() string {
	return "api_token"
}

func (e *TokenLogEvent) SlogAttr() slog.Attr {
	return slog.Group(e.Key(),
		slog.Int64("id", e.ID),
		slog.String("name", e.Name),
	)
}

func (h *Handlers) ListLatestResults(w http.ResponseWriter, r *http.Request) {
	token, ok := reqcontext.APITokenFromContext(r.Context())
	if !ok {
		writeUnauthorized(w)
		return
	}

	rows, err := h.monitors.ListLatestResults(r.Context(), token.UserID, latestResultsLimit)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "listing latest api results", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "An internal error occurred.")
		return
	}

	results := make([]apimessage.ResultSummary, len(rows))
	for i, row := range rows {
		msg, err := apimessage.BuildResultSummaryMessage(row.Monitor, row.MonitorResult)
		if err != nil {
			h.logger.ErrorContext(r.Context(), "building latest api result summary", "error", err, "result_id", row.MonitorResult.ID)
			writeError(w, http.StatusInternalServerError, "internal_error", "An internal error occurred.")
			return
		}
		results[i] = msg
	}

	writeJSON(w, http.StatusOK, apimessage.NewListLatestResultsResponse(results))
}

func (h *Handlers) GetMonitor(w http.ResponseWriter, r *http.Request) {
	monitorID, ok := h.requiredInt64Query(w, r, "monitor_id")
	if !ok {
		return
	}
	token, ok := reqcontext.APITokenFromContext(r.Context())
	if !ok {
		writeUnauthorized(w)
		return
	}

	mon, err := h.monitors.GetMonitor(r.Context(), token.UserID, monitorID)
	if err != nil {
		if errors.Is(err, &errortypes.ResourceNotFoundError{}) {
			writeError(w, http.StatusNotFound, "not_found", "This monitor does not exist.")
			return
		}
		h.logger.ErrorContext(r.Context(), "getting api monitor", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "An internal error occurred.")
		return
	}

	writeJSON(w, http.StatusOK, apimessage.NewGetMonitorResponse(apimessage.BuildMonitorMessage(*mon)))
}

func (h *Handlers) ListResults(w http.ResponseWriter, r *http.Request) {
	params, ok := h.listResultsParams(w, r)
	if !ok {
		return
	}
	token, ok := reqcontext.APITokenFromContext(r.Context())
	if !ok {
		writeUnauthorized(w)
		return
	}
	params.UserID = token.UserID

	page, err := h.monitors.ListResults(r.Context(), params)
	if err != nil {
		if errors.Is(err, &errortypes.ResourceNotFoundError{}) {
			writeError(w, http.StatusNotFound, "not_found", "This monitor does not exist.")
			return
		}
		h.logger.ErrorContext(r.Context(), "listing api results", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "An internal error occurred.")
		return
	}

	// An empty page past the first means the caller has walked past the end of
	// the data. Treat this as a 404 rather than returning an empty page with a
	// prev link that may itself point past the end.
	if len(page.Results) == 0 && params.Pagination.HasPrev() {
		writeError(w, http.StatusNotFound, "not_found", "This page does not exist.")
		return
	}

	results := make([]apimessage.Result, len(page.Results))
	for i, row := range page.Results {
		msg, err := apimessage.BuildResultMessage(*row)
		if err != nil {
			h.logger.ErrorContext(r.Context(), "building api result message", "error", err, "result_id", row.ID)
			writeError(w, http.StatusInternalServerError, "internal_error", "An internal error occurred.")
			return
		}
		results[i] = msg
	}

	writeJSON(w, http.StatusOK, apimessage.NewListResultsResponse(results, h.listResultsLinks(r, params, page)))
}

func (h *Handlers) NotFound(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotFound, "not_found", "This API path does not exist. Please refer to the documentation and try again.")
}

func (h *Handlers) RequireMethodGet(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			next.ServeHTTP(w, r)
			return
		}

		// RFC 7231 requires an Allow header on 405 responses listing the
		// methods the resource supports.
		w.Header().Set("Allow", "GET")
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "This API request must use GET.")
	})
}

func (h *Handlers) listResultsParams(w http.ResponseWriter, r *http.Request) (monitor.ListResultsParams, bool) {
	monitorID, ok := h.requiredInt64Query(w, r, "monitor_id")
	if !ok {
		return monitor.ListResultsParams{}, false
	}

	pag, err := pagination.PaginationFromAPIRequest(r, defaultResultsLimit, maxResultsLimit)
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error()+".")
		return monitor.ListResultsParams{}, false
	}

	return monitor.ListResultsParams{
		MonitorID:  monitorID,
		Pagination: pag,
	}, true
}

func (h *Handlers) listResultsLinks(r *http.Request, params monitor.ListResultsParams, page *monitor.ListResultsPage) apimessage.PaginationLinks {
	var links apimessage.PaginationLinks
	baseURL := reqcontext.BaseURLFromContext(r.Context())
	if params.Pagination.HasPrev() {
		prev := h.listResultsLink(baseURL, r.URL.Path, params, params.Pagination.CurrentPageOneBased()-1)
		links.Prev = &prev
	}

	if page.Pagination.HasNext() {
		next := h.listResultsLink(baseURL, r.URL.Path, params, params.Pagination.CurrentPageOneBased()+1)
		links.Next = &next
	}

	return links
}

func (h *Handlers) listResultsLink(baseURL, path string, params monitor.ListResultsParams, page int) string {
	values := url.Values{}
	values.Set("monitor_id", strconv.FormatInt(params.MonitorID, 10))
	values.Set("page", strconv.Itoa(page))
	if params.Pagination.PageSize != defaultResultsLimit {
		values.Set("per_page", strconv.Itoa(params.Pagination.PageSize))
	}
	// Emit a fully-qualified absolute URL so clients calling the API directly
	// can use the link without resolving against an OpenAPI server URL.
	// baseURL has no trailing slash; path starts with "/".
	return baseURL + path + "?" + values.Encode()
}

func (h *Handlers) requiredInt64Query(w http.ResponseWriter, r *http.Request, name string) (int64, bool) {
	value := r.URL.Query().Get(name)
	if value == "" {
		writeError(w, http.StatusBadRequest, "bad_request", name+" is required.")
		return 0, false
	}

	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "bad_request", name+" must be a positive integer.")
		return 0, false
	}

	return id, true
}

func writeError(w http.ResponseWriter, status int, code string, message string) {
	writeJSON(w, status, apimessage.NewErrorResponse(code, message))
}

func writeUnauthorized(w http.ResponseWriter) {
	writeError(w, http.StatusUnauthorized, "unauthorized", "A valid API token is required.")
}

func writeJSON(w http.ResponseWriter, status int, response any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response)
}
