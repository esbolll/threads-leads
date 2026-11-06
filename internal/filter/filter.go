package filter

import (
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/esbolll/threads-leads/internal/lead/model"
)

var spaceRe = regexp.MustCompile(`\s+`)

// A hiring word must sit close to a tech word, otherwise unrelated sentences match
// ("check the website ... things you need"). Intent normally precedes the role
// ("need a developer"); the reverse order is only accepted when adjacent ("developer needed").
const (
	gapIntentThenTech = 40 // runes between the end of the intent word and the start of the tech word
	gapTechThenIntent = 15
)

type hit struct {
	kw         string
	start, end int // rune offsets
}

// Classify returns the post enriched with match data; ok=false means not relevant.
func Classify(p model.Post, now time.Time) (model.Lead, bool) {
	text := strings.ToLower(spaceRe.ReplaceAllString(p.Text, " "))
	text = strings.TrimSuffix(text, " translate") // button label captured by older browser runs
	if text == "" {
		return model.Lead{}, false
	}
	lang := DetectLang(p.Text)
	if RegionFilter {
		if _, ok := InRegion(p); !ok {
			return model.Lead{}, false
		}
	}
	for _, neg := range NegativePatterns {
		if strings.Contains(text, neg) {
			return model.Lead{}, false
		}
	}
	runes := []rune(text)
	strong := findHits(runes, StrongTech)
	weak := findHits(runes, WeakTech)
	intent := findHits(runes, HiringKeywords)
	if len(intent) == 0 || len(strong)+len(weak) == 0 {
		return model.Lead{}, false
	}
	// Kazakh puts the verb last ("бағдарламашы керек"), so the reverse order gets the wide window too.
	symmetric := lang == LangKK
	switch {
	case near(intent, strong, symmetric):
		// asking for a specialist is enough
	case near(intent, weak, symmetric) && seeksPerson(text, runes):
		// asking for a site/bot/app only counts when a person or a budget is mentioned
	default:
		return model.Lead{}, false
	}
	tech := append(strong, weak...)
	category := "unknown"
	for _, rule := range CategoryRules {
		if len(findHits(runes, rule.Words)) > 0 {
			category = rule.Name
			break
		}
	}
	return model.Lead{
		Post:          p,
		Category:      category,
		MatchedTech:   keywords(tech),
		MatchedIntent: keywords(intent),
		CollectedAt:   now,
	}, true
}

// seeksPerson reports whether the text mentions a person to hire or money for the job.
func seeksPerson(text string, runes []rune) bool {
	if len(findHits(runes, SeekMarkers)) > 0 {
		return true
	}
	for _, sub := range SeekSubstrings {
		if strings.Contains(text, sub) {
			return true
		}
	}
	return false
}

// near reports whether some intent hit and some tech hit are close enough.
// symmetric widens the tech-then-intent window to the same size as intent-then-tech.
func near(intent, tech []hit, symmetric bool) bool {
	reverse := gapTechThenIntent
	if symmetric {
		reverse = gapIntentThenTech
	}
	for _, in := range intent {
		for _, t := range tech {
			if t.start >= in.end && t.start-in.end <= gapIntentThenTech {
				return true
			}
			if in.start >= t.end && in.start-t.end <= reverse {
				return true
			}
		}
	}
	return false
}

func keywords(hits []hit) []string {
	seen := map[string]bool{}
	var out []string
	for _, h := range hits {
		if !seen[h.kw] {
			seen[h.kw] = true
			out = append(out, h.kw)
		}
	}
	return out
}

// findHits returns every whole-word occurrence of every keyword (Cyrillic-aware, hyphen is a boundary).
func findHits(text []rune, kws []string) []hit {
	var hits []hit
	for _, kw := range kws {
		kr := []rune(kw)
		if len(kr) == 0 || len(kr) > len(text) {
			continue
		}
		for i := 0; i+len(kr) <= len(text); i++ {
			if !equalRunes(text[i:i+len(kr)], kr) {
				continue
			}
			before := i == 0 || !wordRune(text[i-1])
			after := i+len(kr) == len(text) || !wordRune(text[i+len(kr)])
			if before && after {
				hits = append(hits, hit{kw: kw, start: i, end: i + len(kr)})
			}
		}
	}
	return hits
}

func wordMatch(text, kw string) bool {
	return len(findHits([]rune(text), []string{kw})) > 0
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
