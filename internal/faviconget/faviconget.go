package faviconget

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sync/singleflight"
)

const maxFaviconSize = 1 * 1024 * 1024 // 1MB

type favicon struct {
	Data        []byte
	ContentType string
}

type faviconGetter struct {
	logger   *slog.Logger
	client   *http.Client
	getGroup singleflight.Group
}

func Handler(logger *slog.Logger) http.Handler {
	getter := &faviconGetter{
		logger: logger,
		client: newHttpClient(),
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Query().Get("path")
		if path == "" {
			http.NotFound(w, r)
			return
		}
		hostname, err := extractHostname(path)
		if err != nil {
			http.Error(w, "invalid path", http.StatusBadRequest)
			return
		}
		favicon, err := getter.getFavicon(hostname)
		if err != nil {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("content-type", favicon.ContentType)

		twoWeeks := 14 * 24 * time.Hour
		w.Header().Set("cache-control", fmt.Sprintf("public, max-age=%.0f", twoWeeks.Seconds()))

		io.Copy(w, bytes.NewReader(favicon.Data))
	})
}

func (g *faviconGetter) getFavicon(hostname string) (*favicon, error) {
	faviconPaths := []string{
		"/favicon.svg", "/favicon.png", "/favicon.ico",
	}

	fav, err, _ := g.getGroup.Do(hostname, func() (any, error) {
		for _, fp := range faviconPaths {
			faviconMaybe := fmt.Sprintf("%s%s", hostname, fp)
			flogger := g.logger.With("url", faviconMaybe)

			if !g.shouldDownload(faviconMaybe) {
				continue
			}

			res, err := g.client.Get(faviconMaybe)
			if err != nil {
				flogger.Error("favicon request error", "error", err)
				continue
			}
			if res.StatusCode > 399 {
				flogger.Debug("favicon request unsuccessful status", "status", res.StatusCode)
				res.Body.Close()
				continue
			}
			if !strings.HasPrefix(res.Header.Get("content-type"), "image/") {
				flogger.Debug("favicon response not an image", "content_type", res.Header.Get("content-type"))
				res.Body.Close()
				continue
			}

			limit := io.LimitReader(res.Body, maxFaviconSize+1)
			data, err := io.ReadAll(limit)
			res.Body.Close()
			if err != nil {
				flogger.Error("favicon read error", "error", err)
				continue
			}

			return &favicon{
				Data:        data,
				ContentType: res.Header.Get("content-type"),
			}, nil
		}

		return nil, fmt.Errorf("favicon not found for %s", hostname)
	})

	if err != nil {
		return nil, err
	}

	if fav, ok := fav.(*favicon); ok {
		return fav, err
	}

	return nil, fmt.Errorf("unexpected result from singleflight: %w", err)
}

func extractHostname(path string) (string, error) {
	url, err := url.Parse(path)
	if err != nil {
		return "", fmt.Errorf("invalid url: %s", path)
	}
	return fmt.Sprintf("%s://%s", url.Scheme, url.Host), nil
}

func (g *faviconGetter) shouldDownload(path string) bool {
	logger := g.logger.With("url", path)
	headRes, err := g.client.Head(path)
	if err != nil {
		logger.Debug("favicon HEAD request error", "error", err)
		return false
	}
	defer headRes.Body.Close()

	if headRes.StatusCode > 399 {
		logger.Debug("favicon HEAD request unsuccessful status", "status", headRes.StatusCode)
		return false
	}

	if contentLength := headRes.Header.Get("Content-Length"); contentLength != "" {
		size, err := strconv.ParseInt(contentLength, 10, 64)
		if err == nil && size > maxFaviconSize {
			logger.Warn("favicon too large, skipping", "size", size, "max", maxFaviconSize)
			return false
		}
	}

	return true
}
