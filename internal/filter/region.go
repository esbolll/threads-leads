package filter

import (
	"strings"
	"unicode"

	"github.com/esbolll/threads-leads/internal/lead/model"
)

// RegionFilter keeps only Russian/Kazakh posts, plus Latin-script posts that carry a CIS marker
// (city, country, phone code, currency, flag, domain, username). Posts have no location field,
// so this is a heuristic. Set from config (LEAD_REGION_FILTER).
var RegionFilter = true

const (
	LangRU    = "ru"
	LangKK    = "kk"
	LangEN    = "en"
	LangOther = "other"
)

// Letters that exist in Kazakh Cyrillic but not in Russian.
const kazakhLetters = "әғқңөұүһ"

// Frequent Kazakh words: they settle short texts that contain "і" but none of the Kazakh-only letters.
var kazakhWords = []string{
	"керек", "жасау", "жасап", "жасайды", "кім", "бізге", "маған", "жоқ", "емес", "болса", "бар ма",
	"беретін", "іздеймін", "іздейміз", "қалай", "неше", "сізге", "рахмет", "сәлем",
}

// Letters that mark Ukrainian / Belarusian rather than Russian or Kazakh.
const ukrainianLetters = "їєґў"

// Region words matched as whole words in lower-cased Latin text.
var cisWords = []string{
	"kazakhstan", "kazakh", "almaty", "astana", "shymkent", "karaganda", "aktau", "atyrau", "aktobe", "pavlodar",
	"uzbekistan", "tashkent", "samarkand", "kyrgyzstan", "bishkek", "tajikistan", "dushanbe",
	"turkmenistan", "ashgabat", "russia", "moscow", "petersburg", "novosibirsk", "kazan",
	"belarus", "minsk", "azerbaijan", "baku", "armenia", "yerevan", "tbilisi",
	"cis", "central asia", "kz", "kzt", "tenge", "rub", "uzs",
}

// Substrings: phone codes, currency signs, flags, domains.
var cisSubstrings = []string{
	"+7 ", "+7(", "+7-", "+77", "+79", "+998", "+996", "+375", "+994", "+374", "+992", "+993",
	"₸", "₽",
	"🇰🇿", "🇷🇺", "🇺🇿", "🇰🇬", "🇧🇾", "🇦🇿", "🇦🇲", "🇹🇯", "🇹🇲",
	".kz", ".ru/", ".uz", ".kg", ".by/",
}

// Username fragments that point to the region.
var cisUsernameParts = []string{
	"kz", "almaty", "astana", "qazaq", "kazakh", "tashkent", "bishkek", "uzb", "msk", "spb",
}

// DetectLang classifies the post language by script and letters.
func DetectLang(text string) string {
	var cyr, lat, kk, ua int
	for _, r := range strings.ToLower(text) {
		switch {
		case strings.ContainsRune(kazakhLetters, r):
			kk++
			cyr++
		case strings.ContainsRune(ukrainianLetters, r):
			ua++
			cyr++
		case unicode.Is(unicode.Cyrillic, r):
			cyr++
		case unicode.Is(unicode.Latin, r):
			lat++
		}
	}
	letters := cyr + lat
	if letters == 0 {
		return LangOther
	}
	if cyr*10 >= letters*3 { // at least 30% Cyrillic: a Russian/Kazakh text with English tech terms
		if ua > 0 {
			return LangOther
		}
		low := strings.ToLower(text)
		kkWords := len(findHits([]rune(low), kazakhWords)) > 0
		// "і" with neither a Kazakh-only letter nor a Kazakh word means Ukrainian or Belarusian.
		if strings.ContainsRune(low, 'і') && kk == 0 && !kkWords {
			return LangOther
		}
		if kk > 0 || kkWords {
			return LangKK
		}
		return LangRU
	}
	if lat*10 >= letters*7 {
		return LangEN
	}
	return LangOther
}

// InRegion reports the detected language and whether the post passes the region filter.
func InRegion(p model.Post) (string, bool) {
	lang := DetectLang(p.Text)
	switch lang {
	case LangRU, LangKK:
		return lang, true
	case LangEN:
		return lang, hasCISMarker(p)
	default:
		return lang, false
	}
}

func hasCISMarker(p model.Post) bool {
	low := strings.ToLower(p.Text)
	// Any Cyrillic word inside an English post is a strong hint.
	for _, r := range low {
		if unicode.Is(unicode.Cyrillic, r) {
			return true
		}
	}
	for _, sub := range cisSubstrings {
		if strings.Contains(low, sub) {
			return true
		}
	}
	if len(findHits([]rune(low), cisWords)) > 0 {
		return true
	}
	user := strings.ToLower(p.Username)
	for _, part := range cisUsernameParts {
		if usernameHas(user, part) {
			return true
		}
	}
	return false
}

// usernameHas matches part as a separate segment of the username (split by . and _) or as a
// prefix/suffix of a segment, so "himark.kz" and "almaty_dev" match but "jackzen" does not.
func usernameHas(user, part string) bool {
	for _, seg := range strings.FieldsFunc(user, func(r rune) bool { return r == '.' || r == '_' }) {
		if seg == part {
			return true
		}
		if len(part) >= 5 && strings.Contains(seg, part) {
			return true
		}
	}
	return false
}
