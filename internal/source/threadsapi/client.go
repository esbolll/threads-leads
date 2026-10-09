// Package threadsapi is the official Threads API data source.
// Public keyword search needs threads_keyword_search approved via App Review;
// before that the endpoint only searches the token owner's own posts.
package threadsapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/esbolll/threads-leads/internal/lead/model"
)

const (
	BaseURL      = "https://graph.threads.net/v1.0"
	AuthorizeURL = "https://threads.net/oauth/authorize"
	TokenURL     = "https://graph.threads.net/oauth/access_token"
)

type Client struct {
	token string
	limit int
	http  *http.Client
}

func New(token string, limit int) *Client {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	return &Client{token: token, limit: limit, http: &http.Client{Timeout: 30 * time.Second}}
}

func (c *Client) Name() string { return "threads-api" }

type apiError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    int    `json:"code"`
	Subcode int    `json:"error_subcode"`
}

func (e *apiError) Error() string {
	return fmt.Sprintf("threads api error %d/%d: %s", e.Code, e.Subcode, e.Message)
}

func (c *Client) get(ctx context.Context, path string, params url.Values, out any) error {
	params.Set("access_token", c.token)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, BaseURL+path+"?"+params.Encode(), nil)
	if err != nil {
		return err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))

	var envelope struct {
		Error *apiError `json:"error"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return fmt.Errorf("bad response (%d): %.200s", resp.StatusCode, body)
	}
	if envelope.Error != nil {
		return envelope.Error
	}
	return json.Unmarshal(body, out)
}

// Search runs GET /keyword_search with search_type=RECENT.
func (c *Client) Search(ctx context.Context, query string) ([]model.Post, error) {
	var r struct {
		Data []struct {
			ID        string `json:"id"`
			Text      string `json:"text"`
			Username  string `json:"username"`
			Permalink string `json:"permalink"`
			Timestamp string `json:"timestamp"`
		} `json:"data"`
	}
	params := url.Values{
		"q":           {query},
		"search_type": {"RECENT"},
		"fields":      {"id,text,username,permalink,timestamp,media_type"},
		"limit":       {strconv.Itoa(c.limit)},
	}
	if err := c.get(ctx, "/keyword_search", params, &r); err != nil {
		return nil, err
	}
	posts := make([]model.Post, 0, len(r.Data))
	for _, d := range r.Data {
		posts = append(posts, model.Post{
			Source:      model.SourceThreads,
			Username:    d.Username,
			Text:        d.Text,
			Permalink:   d.Permalink,
			PostID:      d.ID,
			PostedAt:    d.Timestamp,
			SearchQuery: query,
		})
	}
	// Polite spacing; the documented limit is 2200 queries per 24 h.
	select {
	case <-time.After(700 * time.Millisecond):
	case <-ctx.Done():
	}
	return posts, nil
}

// Me verifies the token and returns the username it belongs to.
func (c *Client) Me(ctx context.Context) (string, error) {
	var r struct {
		Username string `json:"username"`
	}
	if err := c.get(ctx, "/me", url.Values{"fields": {"id,username"}}, &r); err != nil {
		return "", err
	}
	return r.Username, nil
}

// Scopes returns the permissions granted to the token.
func (c *Client) Scopes(ctx context.Context) ([]string, error) {
	var r struct {
		Data struct {
			Scopes []string `json:"scopes"`
		} `json:"data"`
	}
	if err := c.get(ctx, "/debug_token", url.Values{"input_token": {c.token}}, &r); err != nil {
		return nil, err
	}
	return r.Data.Scopes, nil
}
