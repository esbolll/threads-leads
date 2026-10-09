// Package browser is the go-rod data source: Google Chrome driven over CDP with a persistent profile.
package browser

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/input"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/sirupsen/logrus"

	"github.com/esbolll/threads-leads/internal/lead/model"
)

const (
	threadsURL   = "https://www.threads.com"
	searchURLFmt = threadsURL + "/search?q=%s&serp_type=default"

	stepTimeout    = 30 * time.Second
	resultsTimeout = 15 * time.Second
	loginTimeout   = 10 * time.Minute
)

type Options struct {
	Chrome          string // empty = system Chrome
	ProfileDir      string
	Headless        bool
	ScrollsPerQuery int
}

type Client struct {
	opts    Options
	log     *logrus.Logger
	browser *rod.Browser
	page    *rod.Page
}

func (c *Client) Name() string { return "threads-browser" }

// chromeBinary picks a browser: explicit path, a system Chrome/Edge, else rod downloads one.
func chromeBinary(explicit string) (string, error) {
	if explicit != "" {
		return explicit, nil
	}
	if p, ok := launcher.LookPath(); ok {
		return p, nil
	}
	return launcher.NewBrowser().Get()
}

func New(opts Options, log *logrus.Logger) (*Client, error) {
	bin, err := chromeBinary(opts.Chrome)
	if err != nil {
		return nil, err
	}
	dir, _ := filepath.Abs(opts.ProfileDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	log.WithFields(logrus.Fields{"browser": bin, "profile": dir}).Info("launching chrome")

	l := launcher.New().
		Bin(bin).
		UserDataDir(dir).
		Headless(opts.Headless).
		Leakless(false).
		Set("disable-blink-features", "AutomationControlled").
		Set("window-size", "1280,900").
		// Meta serves a 404 shell and drops the session when it sees automation flags.
		Delete("enable-automation").
		Delete("disable-extensions")

	u, err := l.Launch()
	if err != nil {
		return nil, fmt.Errorf("launch browser: %w", err)
	}
	b := rod.New().ControlURL(u)
	if err := b.Connect(); err != nil {
		return nil, fmt.Errorf("connect browser: %w", err)
	}
	pages, err := b.Pages()
	if err != nil {
		return nil, err
	}
	var page *rod.Page
	if len(pages) > 0 {
		page = pages[0]
	} else if page, err = b.Page(proto.TargetCreateTarget{URL: "about:blank"}); err != nil {
		return nil, err
	}
	return &Client{opts: opts, log: log, browser: b, page: page}, nil
}

// Close shuts Chrome down. Never call launcher.Cleanup(): it deletes the profile (our session).
func (c *Client) Close() error { return c.browser.Close() }

// p returns the page with a bounded context so no CDP call can hang forever.
func (c *Client) p() *rod.Page { return c.page.Timeout(stepTimeout) }

func (c *Client) URL() string {
	info, err := c.p().Info()
	if err != nil {
		return ""
	}
	return info.URL
}

// -- auth ---------------------------------------------------------------------

func (c *Client) isLoggedIn() bool {
	cookies, err := c.browser.GetCookies()
	if err != nil {
		return false
	}
	have := map[string]bool{}
	for _, ck := range cookies {
		if strings.Contains(ck.Domain, "threads") {
			have[ck.Name] = true
		}
	}
	return have["sessionid"] && have["ds_user_id"]
}

func (c *Client) cookieNames() []string {
	cookies, _ := c.browser.GetCookies()
	var names []string
	for _, ck := range cookies {
		if strings.Contains(ck.Domain, "threads") || strings.Contains(ck.Domain, "instagram") {
			names = append(names, ck.Domain+":"+ck.Name)
		}
	}
	return names
}

// EnsureLoggedIn opens Threads and, if there is no session, waits for the user to log in.
func (c *Client) EnsureLoggedIn(ctx context.Context) error {
	if err := c.navigate(threadsURL); err != nil {
		return err
	}
	time.Sleep(2 * time.Second)
	if c.isLoggedIn() {
		fmt.Println("Threads session loaded")
		return nil
	}
	c.log.WithField("cookies", strings.Join(c.cookieNames(), " ")).Debug("no session")
	fmt.Println("No saved session. Log in to Threads in the opened browser window.")
	fmt.Printf("Waiting up to %d min...\n", int(loginTimeout.Minutes()))
	deadline := time.Now().Add(loginTimeout)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(3 * time.Second):
		}
		if c.isLoggedIn() {
			fmt.Println("Login detected, session saved to profile.")
			time.Sleep(3 * time.Second)
			return nil
		}
	}
	return errors.New("login timeout: no session cookies found")
}

// -- navigation ---------------------------------------------------------------

func (c *Client) navigate(u string) error {
	c.log.WithField("url", u).Debug("navigate")
	if err := c.p().Navigate(u); err != nil {
		return err
	}
	_ = c.page.Timeout(20 * time.Second).WaitLoad()
	return nil
}

func (c *Client) openSearch(query string) error {
	if err := c.navigate(fmt.Sprintf(searchURLFmt, url.QueryEscape(query))); err != nil {
		return err
	}
	_, err := c.page.Timeout(resultsTimeout).Element(sel("__POST_LINK__"))
	c.log.WithError(err).Debug("results wait")
	time.Sleep(1500 * time.Millisecond)
	return nil
}

// searchViaUI types the query into the search box; fallback when the direct URL shows nothing.
func (c *Client) searchViaUI(query string) error {
	if err := c.navigate(threadsURL + "/search"); err != nil {
		return err
	}
	time.Sleep(2 * time.Second)
	el, err := c.page.Timeout(10 * time.Second).Element(sel("__SEARCH_INPUT__"))
	if err != nil {
		return fmt.Errorf("search input not found: %w", err)
	}
	el = el.Timeout(stepTimeout)
	if err := el.Click(proto.InputMouseButtonLeft, 1); err != nil {
		return fmt.Errorf("click search input: %w", err)
	}
	if err := el.Input(query); err != nil {
		return fmt.Errorf("type query: %w", err)
	}
	if err := c.p().Keyboard.Press(input.Enter); err != nil {
		return fmt.Errorf("press enter: %w", err)
	}
	_, _ = c.page.Timeout(resultsTimeout).Element(sel("__POST_LINK__"))
	time.Sleep(1500 * time.Millisecond)
	return nil
}

func (c *Client) scrollOnce() {
	_ = c.p().Mouse.Scroll(0, float64(1500+rand.Intn(1000)), 5)
	time.Sleep(time.Duration(1500+rand.Intn(1500)) * time.Millisecond)
}

func (c *Client) extract() ([]model.Post, error) {
	res, err := c.p().Eval(extractJS)
	if err != nil {
		return nil, err
	}
	var raw []struct {
		Username  string `json:"username"`
		PostID    string `json:"post_id"`
		Permalink string `json:"permalink"`
		Timestamp string `json:"timestamp"`
		Text      string `json:"text"`
	}
	if err := json.Unmarshal([]byte(res.Value.Str()), &raw); err != nil {
		return nil, err
	}
	posts := make([]model.Post, 0, len(raw))
	for _, r := range raw {
		posts = append(posts, model.Post{
			Source: model.SourceThreads, Username: r.Username, Text: r.Text,
			Permalink: r.Permalink, PostID: r.PostID, PostedAt: r.Timestamp,
		})
	}
	return posts, nil
}

// Search opens the results page, scrolls a few times and returns the unique posts seen.
func (c *Client) Search(ctx context.Context, query string) ([]model.Post, error) {
	if err := c.openSearch(query); err != nil {
		return nil, err
	}
	if first, _ := c.extract(); len(first) == 0 {
		c.log.Debug("no posts via direct URL, trying UI search")
		if err := c.searchViaUI(query); err != nil {
			c.log.WithError(err).Debug("UI search failed")
		}
	}
	found := map[string]model.Post{}
	var order []string
	stale := 0
	for i := 0; i <= c.opts.ScrollsPerQuery; i++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		before := len(found)
		posts, err := c.extract()
		if err != nil {
			return nil, err
		}
		for _, p := range posts {
			p.SearchQuery = query
			if _, ok := found[p.Key()]; !ok {
				found[p.Key()] = p
				order = append(order, p.Key())
			}
		}
		if len(found) == before {
			if stale++; stale >= 2 {
				break
			}
		} else {
			stale = 0
		}
		if i < c.opts.ScrollsPerQuery {
			c.scrollOnce()
		}
	}
	out := make([]model.Post, 0, len(order))
	for _, k := range order {
		out = append(out, found[k])
	}
	// Pause between queries so the session looks human.
	select {
	case <-time.After(time.Duration(3000+rand.Intn(3000)) * time.Millisecond):
	case <-ctx.Done():
	}
	return out, nil
}

// Explore opens a search and writes DOM facts, HTML and a screenshot into dir.
func (c *Client) Explore(query, dir string) (string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	if err := c.openSearch(query); err != nil {
		return "", err
	}
	time.Sleep(3 * time.Second)
	res, err := c.p().Eval(exploreJS)
	if err != nil {
		return "", err
	}
	info := res.Value.Str()
	_ = os.WriteFile(filepath.Join(dir, "explore.json"), []byte(info), 0o644)
	if h, err := c.p().HTML(); err == nil {
		_ = os.WriteFile(filepath.Join(dir, "search.html"), []byte(h), 0o644)
	}
	if png, err := c.p().Screenshot(false, nil); err == nil {
		_ = os.WriteFile(filepath.Join(dir, "search.png"), png, 0o644)
	}
	return info, nil
}
