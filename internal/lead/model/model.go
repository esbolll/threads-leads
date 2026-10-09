package model

import (
	"crypto/sha1"
	"encoding/hex"
	"strings"
	"time"
)

const SourceThreads = "threads"

// Post is a raw public post as returned by a source (browser or API).
type Post struct {
	Source      string    `db:"source" json:"source"`
	Username    string    `db:"username" json:"username"`
	DisplayName string    `db:"display_name" json:"display_name,omitempty"`
	Text        string    `db:"text" json:"text"`
	Permalink   string    `db:"permalink" json:"permalink"`
	PostID      string    `db:"post_id" json:"post_id,omitempty"`
	PostedAt    string    `db:"posted_at" json:"timestamp"` // RFC3339 from the source, may be empty
	SearchQuery string    `db:"search_query" json:"search_query"`
	FirstSeenAt time.Time `db:"first_seen_at" json:"-"`
	LastSeenAt  time.Time `db:"last_seen_at" json:"-"`
}

// Key is the dedupe key: permalink without query string, else a hash of author + text.
func (p Post) Key() string {
	if p.Permalink != "" {
		return strings.TrimRight(strings.SplitN(p.Permalink, "?", 2)[0], "/")
	}
	sum := sha1.Sum([]byte(p.Username + "|" + p.Text))
	return "hash:" + hex.EncodeToString(sum[:])
}

// Lead is a post that passed the relevance filter.
type Lead struct {
	Post
	Category      string     `json:"category"`
	MatchedTech   []string   `json:"matched_tech"`
	MatchedIntent []string   `json:"matched_intent"`
	CollectedAt   time.Time  `json:"collected_at"`
	NotifiedAt    *time.Time `json:"notified_at,omitempty"`
}
