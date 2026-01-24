package pushover

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	logger *slog.Logger
	key    string
	store  *Store
}

func NewPushoverClient(key string, logger *slog.Logger, store *Store) *Client {
	return &Client{
		logger: logger,
		key:    key,
		store:  store,
	}
}

type ErrInvalidToken struct {
	Reasons []string
}

func (e ErrInvalidToken) Error() string {
	return fmt.Sprintf("invalid token: %s", strings.Join(e.Reasons, ", "))
}

func (c *Client) Validate(ctx context.Context, userKey string) error {
	b := struct {
		Token string `json:"token"`
		User  string `json:"user"`
	}{
		Token: c.key,
		User:  userKey,
	}

	res, err := c.sendRequest(ctx, "https://api.pushover.net/1/users/validate.json", b)
	if err != nil {
		return err
	}

	if res.Status != 1 {
		return &ErrInvalidToken{Reasons: *res.Errors}
	}

	return nil
}

type SendParams struct {
	Title   string
	Message string
	UserID  int64
}

func (c *Client) Send(ctx context.Context, params SendParams) error {
	token, err := c.store.GetToken(ctx, params.UserID)
	if err != nil {
		if errors.Is(err, ErrNoPushoverUserToken) {
			c.logger.WarnContext(ctx, "tried to send pushover notification to a user without a pushover token", "user_id", params.UserID)
			return nil
		}
		return fmt.Errorf("getting pushover user token: %w", err)
	}

	b := struct {
		Token   string `json:"token"`
		User    string `json:"user"`
		Message string `json:"message"`
		Title   string `json:"title"`
	}{
		Token:   c.key,
		User:    token.Token,
		Message: params.Message,
		Title:   params.Title,
	}

	res, err := c.sendRequest(ctx, "https://api.pushover.net/1/messages.json", b)
	if err != nil {
		return fmt.Errorf("pushover send error: %w", err)
	}

	if !res.Success() {
		return fmt.Errorf("pushover returned error: %w", res)
	}

	return nil
}

type PushoverResponse struct {
	Status    int       `json:"status"`
	RequestID string    `json:"request"`
	Errors    *[]string `json:"errors"`
}

func (p PushoverResponse) Success() bool {
	return p.Status == 1
}

func (p PushoverResponse) Error() string {
	return strings.Join(*p.Errors, ", ")
}

func (c *Client) sendRequest(ctx context.Context, url string, payload any) (*PushoverResponse, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshaling json: %w", err)
	}

	start := time.Now()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}
	defer res.Body.Close() // nolint:errcheck

	c.logger.InfoContext(ctx, "pushover: request sent",
		"duration_ms", time.Since(start).Milliseconds(),
		"status_code", res.StatusCode)

	pRes := &PushoverResponse{}
	if err := json.NewDecoder(res.Body).Decode(pRes); err != nil {
		return nil, fmt.Errorf("unmarshaling response: %w", err)
	}

	return pRes, nil
}
