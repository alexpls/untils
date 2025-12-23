package faviconget

import (
	"net/http"
	"time"
)

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
