package main

import (
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
)

type Post struct {
	Source      string `json:"source"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name,omitempty"`
	Text        string `json:"text"`
	Permalink   string `json:"permalink"`
	PostID      string `json:"post_id,omitempty"`
	Timestamp   string `json:"timestamp"`
	SearchQuery string `json:"search_query"`
}

type Lead struct {
	Post
	Category      string   `json:"category"`
	MatchedTech   []string `json:"matched_tech"`
	MatchedIntent []string `json:"matched_intent"`
	CollectedAt   string   `json:"collected_at"`
}

type Client struct {
	browser *rod.Browser
	page    *rod.Page
	l       *launcher.Launcher
	Verbose bool
}

const stepTimeout = 30 * time.Second

func (c *Client) logf(format string, a ...any) {
	if c.Verbose {
		fmt.Printf("  > "+format+"\n", a...)
	}
}

// p returns the page with a bounded context so no CDP call can hang forever.
func (c *Client) p() *rod.Page { return c.page.Timeout(stepTimeout) }

// chromeBinary picks a browser: $THREADS_CHROME, a system Chrome/Edge, else rod downloads one.
func chromeBinary() (string, error) {
	if p := os.Getenv("THREADS_CHROME"); p != "" {
		return p, nil
	}
	if p, ok := launcher.LookPath(); ok {
		return p, nil
	}
	fmt.Println("No Chrome found, downloading a Chromium build (one time)...")
	return launcher.NewBrowser().Get()
}

func NewClient(headless bool) (*Client, error) {
	bin, err := chromeBinary()
	if err != nil {
		return nil, err
	}
	dir, _ := filepath.Abs(profileDir)
	_ = os.MkdirAll(dir, 0o755)
	fmt.Println("Browser:", bin)
	fmt.Println("Profile:", dir)

	l := launcher.New().
		Bin(bin).
		UserDataDir(dir).
		Headless(headless).
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
	} else {
		page, err = b.Page(proto.TargetCreateTarget{URL: "about:blank"})
		if err != nil {
			return nil, err
		}
	}
	return &Client{browser: b, page: page, l: l}, nil
}

func (c *Client) Close() {
	// Never call launcher.Cleanup(): it deletes the user data dir (our saved session).
	_ = c.browser.Close()
}

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

func (c *Client) debugCookies() {
	cookies, err := c.browser.GetCookies()
	if err != nil {
		fmt.Println("cookies: error", err)
		return
	}
	var names []string
	for _, ck := range cookies {
		if strings.Contains(ck.Domain, "threads") || strings.Contains(ck.Domain, "instagram") {
			names = append(names, ck.Domain+":"+ck.Name)
		}
	}
	fmt.Printf("cookies seen (%d total): %s\n", len(cookies), strings.Join(names, " "))
}

func (c *Client) EnsureLoggedIn() error {
	if err := c.navigate(threadsURL); err != nil {
		return err
	}
	time.Sleep(2 * time.Second)
	if c.isLoggedIn() {
		fmt.Println("Threads session loaded")
		return nil
	}
	c.debugCookies()
	fmt.Println("No saved session. Log in to Threads in the opened browser window.")
	fmt.Printf("Waiting up to %d min...\n", int(loginTimeout.Minutes()))
	deadline := time.Now().Add(loginTimeout)
	for time.Now().Before(deadline) {
		time.Sleep(3 * time.Second)
		if c.isLoggedIn() {
			fmt.Println("Login detected, session saved to profile.")
			time.Sleep(3 * time.Second)
			return nil
		}
	}
	return errors.New("login timeout: no session cookies found")
}

func (c *Client) navigate(u string) error {
	c.logf("navigate %s", u)
	if err := c.p().Navigate(u); err != nil {
		return err
	}
	_ = c.page.Timeout(20 * time.Second).WaitLoad()
	c.logf("loaded %s", c.URL())
	return nil
}

func (c *Client) OpenSearch(query string) error {
	u := fmt.Sprintf(searchURLFmt, url.QueryEscape(query))
	if err := c.navigate(u); err != nil {
		return err
	}
	// Empty results or layout change are not fatal; the parser will report 0.
	_, err := c.page.Timeout(resultsTimeout).Element(selectors["__POST_LINK__"])
	c.logf("results wait: err=%v", err)
	time.Sleep(1500 * time.Millisecond)
	return nil
}

// SearchViaUI types the query into the search box and submits it.
func (c *Client) SearchViaUI(query string) error {
	if err := c.navigate(threadsURL + "/search"); err != nil {
		return err
	}
	time.Sleep(2 * time.Second)
	el, err := c.page.Timeout(10 * time.Second).Element(selectors["__SEARCH_INPUT__"])
	if err != nil {
		c.Screenshot(outputDir + "/no_search_input.png")
		return fmt.Errorf("search input not found (see output/no_search_input.png): %w", err)
	}
	c.logf("search input found")
	el = el.Timeout(stepTimeout)
	if err := el.Click(proto.InputMouseButtonLeft, 1); err != nil {
		return fmt.Errorf("click search input: %w", err)
	}
	c.logf("clicked")
	if err := el.Input(query); err != nil {
		return fmt.Errorf("type query: %w", err)
	}
	c.logf("typed")
	if err := c.p().Keyboard.Press(input.Enter); err != nil {
		return fmt.Errorf("press enter: %w", err)
	}
	c.logf("enter pressed")
	_, err = c.page.Timeout(resultsTimeout).Element(selectors["__POST_LINK__"])
	c.logf("results wait: err=%v", err)
	time.Sleep(1500 * time.Millisecond)
	return nil
}

func (c *Client) URL() string {
	info, err := c.p().Info()
	if err != nil {
		return ""
	}
	return info.URL
}

func (c *Client) ScrollOnce() {
	_ = c.p().Mouse.Scroll(0, float64(1500+rand.Intn(1000)), 5)
	time.Sleep(randDuration(scrollPauseMin, scrollPauseMax))
}

func (c *Client) ExtractPosts() ([]Post, error) {
	res, err := c.p().Eval(extractJS)
	if err != nil {
		return nil, err
	}
	var posts []Post
	if err := json.Unmarshal([]byte(res.Value.Str()), &posts); err != nil {
		return nil, err
	}
	for i := range posts {
		posts[i].Source = "threads"
	}
	return posts, nil
}

func (c *Client) Explore() (string, error) {
	res, err := c.p().Eval(exploreJS)
	if err != nil {
		return "", err
	}
	return res.Value.Str(), nil
}

func (c *Client) Screenshot(path string) {
	data, err := c.p().Screenshot(false, nil)
	if err == nil {
		_ = os.WriteFile(path, data, 0o644)
	}
}

func (c *Client) HTML() string {
	h, _ := c.p().HTML()
	return h
}

func PauseBetweenQueries() { time.Sleep(randDuration(queryPauseMin, queryPauseMax)) }

func randDuration(lo, hi time.Duration) time.Duration {
	return lo + time.Duration(rand.Int63n(int64(hi-lo)))
}
