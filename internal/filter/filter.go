package filter

import (
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/esbolll/threads-leads/internal/lead/model"
)

var spaceRe = regexp.MustCompile(`\s+`)

// Classify returns the post enriched with match data; ok=false means not relevant.
func Classify(p model.Post, now time.Time) (model.Lead, bool) {
	text := strings.ToLower(spaceRe.ReplaceAllString(p.Text, " "))
	if text == "" {
		return model.Lead{}, false
	}
	for _, neg := range NegativePatterns {
		if strings.Contains(text, neg) {
			return model.Lead{}, false
		}
	}
	tech := matches(text, TechKeywords)
	intent := matches(text, HiringKeywords)
	if len(tech) == 0 || len(intent) == 0 {
		return model.Lead{}, false
	}
	category := "unknown"
	for _, rule := range CategoryRules {
		if len(matches(text, rule.Words)) > 0 {
			category = rule.Name
			break
		}
	}
	return model.Lead{
		Post:          p,
		Category:      category,
		MatchedTech:   tech,
		MatchedIntent: intent,
		CollectedAt:   now,
	}, true
}

func matches(text string, keywords []string) []string {
	var found []string
	for _, kw := range keywords {
		if wordMatch(text, kw) {
			found = append(found, kw)
		}
	}
	return found
}

// wordMatch reports whether kw occurs in text as a whole word (Cyrillic-aware, hyphen is a boundary).
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
