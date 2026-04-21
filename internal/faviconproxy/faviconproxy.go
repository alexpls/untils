package faviconproxy

import (
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/alexpls/untils/internal/useragent"
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
		Transport: &useragent.Transport{
			Agent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) " +
				"AppleWebKit/537.36 (KHTML, like Gecko) " +
				"Chrome/143.0.0.0 " +
				"Safari/537.36 " +
				useragent.DefaultAgent,
			RoundTripper: http.DefaultTransport,
		},
	}
}
