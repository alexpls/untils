package robotstxt

import (
	"net/http"
)

func Handler(servesPublicPages bool) http.Handler {
	var robots string

	if !servesPublicPages {
		// when not serving public pages (i.e. in selfhosted mode) robots file
		// should disallow everything
		robots = "User-agent: *\nDisallow: /"
	} else {
		robots = "User-agent: *\nDisallow: /app/"
	}

	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Add("content-type", "text/plain")
		_, _ = w.Write([]byte(robots))
	})
}
