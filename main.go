package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"time"
)

// searchFunc is a data source: one query in, raw posts out.
type searchFunc func(query string) ([]Post, error)

func main() {
	useAPI := flag.Bool("api", false, "use the official Threads API (THREADS_TOKEN from env or .env) instead of the browser")
	auth := flag.Bool("auth", false, "run the OAuth flow to obtain a token with threads_keyword_search and save it to .env")
	headless := flag.Bool("headless", false, "run browser without a window (only after login is saved)")
	login := flag.Bool("login", false, "open Threads, wait for manual login, exit")
	explore := flag.String("explore", "", "open a search for this query and dump DOM facts to output/")
	skipLogin := flag.Bool("skip-login", false, "do not wait for login (diagnostics only)")
	verbose := flag.Bool("v", false, "verbose step log")
	flag.Parse()

	_ = os.MkdirAll(outputDir, 0o755)

	var err error
	if *auth {
		err = runAuth()
	} else if *useAPI {
		err = runAPI(flag.Args())
	} else {
		err = runBrowser(*headless, *login, *skipLogin, *verbose, *explore, flag.Args())
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func runAPI(queries []string) error {
	token, err := loadToken()
	if err != nil {
		return err
	}
	user, err := apiMe(token)
	if err != nil {
		return fmt.Errorf("token check failed: %w", err)
	}
	fmt.Printf("Threads API token OK, user @%s\n", user)

	src := func(q string) ([]Post, error) {
		posts, err := apiKeywordSearch(token, q, 50)
		time.Sleep(700 * time.Millisecond) // polite spacing, far below the 2200/day limit
		return posts, err
	}
	return collectAll(queries, src)
}

func runBrowser(headless, loginOnly, skipLogin, verbose bool, explore string, queries []string) error {
	c, err := NewClient(headless)
	if err != nil {
		return err
	}
	defer c.Close()
	c.Verbose = verbose

	// Ctrl-C must close Chrome too, otherwise it keeps the profile locked.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	go func() {
		<-sig
		fmt.Println("interrupted, closing browser")
		c.Close()
		os.Exit(130)
	}()

	if skipLogin {
		if err := c.navigate(threadsURL); err != nil {
			return err
		}
	} else if err := c.EnsureLoggedIn(); err != nil {
		return err
	}
	if loginOnly {
		fmt.Println("Done. You can close the browser.")
		return nil
	}

	if explore != "" {
		return exploreDOM(c, explore)
	}

	return collectAll(queries, func(q string) ([]Post, error) {
		posts, err := collectForQuery(c, q)
		PauseBetweenQueries()
		return posts, err
	})
}

func exploreDOM(c *Client, query string) error {
	if err := c.OpenSearch(query); err != nil {
		return err
	}
	fmt.Println("URL after direct search:", c.URL())
	time.Sleep(2 * time.Second)
	c.Screenshot(outputDir + "/search_direct.png")
	if err := c.SearchViaUI(query); err != nil {
		fmt.Println("UI search failed:", err)
	} else {
		fmt.Println("URL after UI search:", c.URL())
	}
	time.Sleep(3 * time.Second)
	info, err := c.Explore()
	if err != nil {
		return err
	}
	fmt.Println(info)
	_ = os.WriteFile(outputDir+"/explore.json", []byte(info), 0o644)
	_ = os.WriteFile(outputDir+"/search.html", []byte(c.HTML()), 0o644)
	c.Screenshot(outputDir + "/search.png")
	fmt.Println("saved output/explore.json, output/search.html, output/search.png")
	return nil
}

// collectAll runs every query through src, filters, prints and saves leads.
func collectAll(queries []string, src searchFunc) error {
	if len(queries) == 0 {
		queries = searchQueries
	}
	collectedAt := time.Now().UTC().Format(time.RFC3339)
	var allRaw []Post
	var leads []Lead

	for _, q := range queries {
		fmt.Printf("\nSearching: %s\n", q)
		posts, err := src(q)
		if err != nil {
			fmt.Printf("  error: %v\n", err)
			continue
		}
		allRaw = append(allRaw, posts...)
		n := 0
		for _, p := range posts {
			if lead, ok := classify(p); ok {
				lead.CollectedAt = collectedAt
				leads = append(leads, lead)
				n++
			}
		}
		fmt.Printf("Found: %d posts\n", len(posts))
		fmt.Printf("Relevant: %d\n", n)
	}

	leads = dedupeLeads(leads)
	fmt.Println()
	for _, l := range leads {
		printLead(l)
	}
	fmt.Println(strings.Repeat("-", 27))
	fmt.Printf("Total leads: %d\n", len(leads))

	if err := writeJSON(leadsFile, leads); err != nil {
		return err
	}
	if err := writeJSON(rawFile, dedupePosts(allRaw)); err != nil {
		return err
	}
	fmt.Println("Saved", leadsFile)
	return nil
}

func collectForQuery(c *Client, query string) ([]Post, error) {
	if err := c.OpenSearch(query); err != nil {
		return nil, err
	}
	// Direct URL is the primary path; fall back to typing into the search box
	// if the page rendered without any post links.
	if first, _ := c.ExtractPosts(); len(first) == 0 {
		c.logf("no posts via direct URL, trying UI search")
		if err := c.SearchViaUI(query); err != nil {
			c.logf("UI search failed: %v", err)
		}
	}
	found := map[string]Post{}
	order := []string{}
	stale := 0
	for i := 0; i <= scrollsPerQuery; i++ {
		before := len(found)
		posts, err := c.ExtractPosts()
		if err != nil {
			return nil, err
		}
		for _, p := range posts {
			p.SearchQuery = query
			k := dedupeKey(p)
			if _, ok := found[k]; !ok {
				found[k] = p
				order = append(order, k)
			}
		}
		if len(found) == before {
			stale++
			if stale >= 2 {
				break
			}
		} else {
			stale = 0
		}
		if i < scrollsPerQuery {
			c.ScrollOnce()
		}
	}
	out := make([]Post, 0, len(order))
	for _, k := range order {
		out = append(out, found[k])
	}
	return out, nil
}

func printLead(l Lead) {
	fmt.Println(strings.Repeat("-", 27))
	fmt.Println("[LEAD]")
	fmt.Printf("Author: @%s\n", l.Username)
	fmt.Printf("Type: %s\n", l.Category)
	fmt.Printf("Tech: %s\n", strings.Join(l.MatchedTech, ", "))
	fmt.Println()
	text := l.Text
	if len(text) > 600 {
		text = text[:600] + "..."
	}
	fmt.Println(text)
	fmt.Println()
	fmt.Println(l.Permalink)
}

func writeJSON(path string, v any) error {
	if v == nil {
		v = []any{}
	}
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
