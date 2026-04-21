package useragent

import "net/http"

const DefaultAgent = "Untils/1.0 (+https://untils.com; contact=alex@alexplescan.com)"

type Transport struct {
	Agent        string
	RoundTripper http.RoundTripper
}

func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", t.Agent)
	}
	return t.RoundTripper.RoundTrip(req)
}
