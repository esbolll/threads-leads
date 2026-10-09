package main

import (
	"crypto/sha1"
	"encoding/hex"
	"regexp"
	"strings"
	"unicode"
)

// wordMatch reports whether kw occurs in text as a whole word (Cyrillic-aware).
func wordMatch(text, kw string) bool {
	tr, kr := []rune(text), []rune(kw)
	if len(kr) == 0 || len(kr) > len(tr) {
		return false
	}
	for i := 0; i+len(kr) <= len(tr); i++ {
		if !equalRunes(tr[i:i+len(kr)], kr) {
			continue
		}
		before := i == 0 || !wordRune(tr[i-1])
		after := i+len(kr) == len(tr) || !wordRune(tr[i+len(kr)])
		if before && after {
			return true
		}
	}
	return false
}

func equalRunes(a, b []rune) bool {
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func wordRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}

func findMatches(text string, keywords []string) []string {
	var found []string
	for _, kw := range keywords {
		if wordMatch(text, kw) {
			found = append(found, kw)
		}
	}
	return found
}

var spaceRe = regexp.MustCompile(`\s+`)

// classify enriches a post with match data; ok=false means not relevant.
func classify(p Post) (Lead, bool) {
	text := strings.ToLower(spaceRe.ReplaceAllString(p.Text, " "))
	if text == "" {
		return Lead{}, false
	}
	for _, neg := range negativePatterns {
		if strings.Contains(text, neg) {
			return Lead{}, false
		}
	}
	tech := findMatches(text, techKeywords)
	intent := findMatches(text, hiringKeywords)
	if len(tech) == 0 || len(intent) == 0 {
		return Lead{}, false
	}
	category := "unknown"
	for _, rule := range categoryRules {
		if len(findMatches(text, rule.words)) > 0 {
			category = rule.name
			break
		}
	}
	return Lead{Post: p, Category: category, MatchedTech: tech, MatchedIntent: intent}, true
}

func dedupeKey(p Post) string {
	if p.Permalink != "" {
		return strings.TrimRight(strings.SplitN(p.Permalink, "?", 2)[0], "/")
	}
	sum := sha1.Sum([]byte(p.Username + "|" + p.Text))
	return "hash:" + hex.EncodeToString(sum[:])
}

func dedupeLeads(in []Lead) []Lead {
	seen := map[string]bool{}
	var out []Lead
	for _, l := range in {
		k := dedupeKey(l.Post)
		if seen[k] {
			continue
		}
		seen[k] = true
		out = append(out, l)
	}
	return out
}

func dedupePosts(in []Post) []Post {
	seen := map[string]bool{}
	var out []Post
	for _, p := range in {
		k := dedupeKey(p)
		if seen[k] {
			continue
		}
		seen[k] = true
		out = append(out, p)
	}
	return out
}
