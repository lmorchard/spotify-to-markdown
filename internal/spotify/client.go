package spotify

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const apiBaseURL = "https://api.spotify.com/v1"

// TokenSource produces a current, valid access token. The Authenticator in
// internal/spotifyauth satisfies this interface.
type TokenSource interface {
	AccessToken(ctx context.Context) (string, error)
}

// Client is a thin Spotify Web API client.
type Client struct {
	http        *http.Client
	tokenSource TokenSource
}

// New constructs a Client. If httpClient is nil, a default with a sensible
// timeout is used.
func New(tokenSource TokenSource, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	return &Client{http: httpClient, tokenSource: tokenSource}
}

// GetRecentlyPlayed returns up to `limit` recently-played items (max 50).
// If after > 0, results are restricted to plays strictly after the given Unix
// millisecond timestamp.
func (c *Client) GetRecentlyPlayed(ctx context.Context, limit int, after int64) (*RecentlyPlayedResponse, error) {
	if limit <= 0 || limit > 50 {
		limit = 50
	}
	q := url.Values{}
	q.Set("limit", strconv.Itoa(limit))
	if after > 0 {
		q.Set("after", strconv.FormatInt(after, 10))
	}

	var out RecentlyPlayedResponse
	if err := c.doJSON(ctx, http.MethodGet, "/me/player/recently-played?"+q.Encode(), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// doJSON issues an authenticated request and decodes a JSON response into out.
func (c *Client) doJSON(ctx context.Context, method, path string, out interface{}) error {
	resp, err := c.do(ctx, method, path)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response body: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("spotify API %s %s returned %d: %s", method, path, resp.StatusCode, string(body))
	}
	if len(body) == 0 {
		return nil
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

// do issues a single authenticated request. It does not retry on 401 — token
// freshness is the TokenSource's responsibility.
func (c *Client) do(ctx context.Context, method, path string) (*http.Response, error) {
	token, err := c.tokenSource.AccessToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("get access token: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, method, apiBaseURL+path, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	return resp, nil
}
