package webhook

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/alexpls/untils/internal/errortypes"
	"github.com/alexpls/untils/internal/useragent"
)

type httpClient struct {
	client *http.Client
}

func newHttpClient() *httpClient {
	cl := &http.Client{
		Timeout: 15 * time.Second,
		Transport: &useragent.Transport{
			Agent:        useragent.DefaultAgent,
			RoundTripper: http.DefaultTransport,
		},
	}

	return &httpClient{
		client: cl,
	}
}

type HttpResponse struct {
	Latency    time.Duration
	StatusCode int
}

func (h *httpClient) Request(ctx context.Context, url string, body io.Reader) (HttpResponse, error) {
	var respData HttpResponse

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		return respData, err
	}
	req.Header.Set("Content-Type", "application/json")

	start := time.Now()
	resp, err := h.client.Do(req)
	respData.Latency = time.Since(start)

	if err != nil {
		dnsError, ok := errors.AsType[*net.DNSError](err)
		if ok && dnsError.IsNotFound {
			return respData, &errortypes.ErrWebhookRequest{
				Reason: "host not found",
			}
		}
		return respData, err
	}

	defer resp.Body.Close()

	respData.StatusCode = resp.StatusCode

	if resp.StatusCode > 399 {
		return respData, &errortypes.ErrWebhookRequest{
			Reason: fmt.Sprintf("unexpected status: %d", resp.StatusCode),
		}
	}

	return respData, nil
}
