package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// Official Threads API data source. Needs a user access token with
// threads_basic + threads_keyword_search. Until the app passes App Review
// for threads_keyword_search, Meta only searches the token owner's own posts.

const apiBase = "https://graph.threads.net/v1.0"

type apiError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    int    `json:"code"`
	Subcode int    `json:"error_subcode"`
}

type apiSearchResponse struct {
	Data []struct {
		ID        string `json:"id"`
		Text      string `json:"text"`
		Username  string `json:"username"`
		Permalink string `json:"permalink"`
		Timestamp string `json:"timestamp"`
		MediaType string `json:"media_type"`
	} `json:"data"`
	Paging struct {
		Cursors struct {
			After string `json:"after"`
		} `json:"cursors"`
	} `json:"paging"`
	Error *apiError `json:"error"`
}

// loadToken reads THREADS_TOKEN from the environment or a local .env file.
func loadToken() (string, error) {
	if t := os.Getenv("THREADS_TOKEN"); t != "" {
		return t, nil
	}
	f, err := os.Open(".env")
	if err != nil {
		return "", errors.New("THREADS_TOKEN not set and no .env file found")
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if strings.HasPrefix(line, "THREADS_TOKEN=") {
			return strings.Trim(strings.TrimPrefix(line, "THREADS_TOKEN="), `"'`), nil
		}
	}
	return "", errors.New("THREADS_TOKEN not found in .env")
}

var httpClient = &http.Client{Timeout: 30 * time.Second}

func apiKeywordSearch(token, query string, limit int) ([]Post, error) {
	params := url.Values{}
	params.Set("q", query)
	params.Set("search_type", "RECENT")
	params.Set("fields", "id,text,username,permalink,timestamp,media_type")
	params.Set("limit", fmt.Sprint(limit))
	params.Set("access_token", token)

	resp, err := httpClient.Get(apiBase + "/keyword_search?" + params.Encode())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))

	var r apiSearchResponse
	if err := json.Unmarshal(body, &r); err != nil {
		return nil, fmt.Errorf("bad response (%d): %s", resp.StatusCode, truncate(string(body), 300))
	}
	if r.Error != nil {
		return nil, fmt.Errorf("api error %d/%d: %s", r.Error.Code, r.Error.Subcode, r.Error.Message)
	}
	posts := make([]Post, 0, len(r.Data))
	for _, d := range r.Data {
		posts = append(posts, Post{
			Source:      "threads",
			Username:    d.Username,
			Text:        d.Text,
			Permalink:   d.Permalink,
			PostID:      d.ID,
			Timestamp:   d.Timestamp,
			SearchQuery: query,
		})
	}
	return posts, nil
}

// apiMe verifies the token and reports whose it is.
func apiMe(token string) (string, error) {
	resp, err := httpClient.Get(apiBase + "/me?fields=id,username&access_token=" + url.QueryEscape(token))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	var r struct {
		ID       string    `json:"id"`
		Username string    `json:"username"`
		Error    *apiError `json:"error"`
	}
	if err := json.Unmarshal(body, &r); err != nil {
		return "", fmt.Errorf("bad response (%d): %s", resp.StatusCode, truncate(string(body), 300))
	}
	if r.Error != nil {
		return "", fmt.Errorf("api error %d: %s", r.Error.Code, r.Error.Message)
	}
	return r.Username, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
