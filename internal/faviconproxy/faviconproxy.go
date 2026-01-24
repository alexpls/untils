package faviconproxy

import (
	"io"
	"log/slog"
	"net/http"
	"time"
)

func Handler(logger *slog.Logger) http.Handler {
	h := newHttpClient()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		targetFavicon := r.URL.Query().Get("url")

		res, err := h.Get(targetFavicon)
		if err != nil {
			http.Error(w, "Failed to fetch URL", http.StatusBadRequest)
			return
		}
		defer res.Body.Close() // nolint:errcheck

		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		w.Header().Set("Content-Type", res.Header.Get("Content-Type"))

		_, err = io.Copy(w, res.Body)
		if err != nil {
			logger.ErrorContext(r.Context(), "error writing favicon response", "error", err)
		}
	})
}

func newHttpClient() *http.Client {
	return &http.Client{
		Timeout: 5 * time.Second,
		Transport: &userAgentTransport{
			ua: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) " +
				"AppleWebKit/537.36 (KHTML, like Gecko) " +
				"Chrome/143.0.0.0 " +
				"Safari/537.36" +
				"Untils/1.0 (+https://untils.com; contact=alex@alexplescan.com)",
			rt: http.DefaultTransport,
		},
	}
}

type userAgentTransport struct {
	ua string
	rt http.RoundTripper
}

func (t *userAgentTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", t.ua)
	}
	return t.rt.RoundTrip(req)
}
