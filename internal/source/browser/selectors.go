package browser

import "strings"

// DOM selectors. Threads changes markup; edit only this file when it breaks.
// Placeholders (__NAME__) are substituted into the JS below.
var selectors = map[string]string{
	"__POST_LINK__": "a[href*='/post/']",                // permalink inside a post card
	"__CONTAINER__": "div[data-pressable-container]",    // one post card
	"__TIME__":      "time[datetime]",                   // post timestamp
	"__AUTHOR__":    "a[href^='/@']",                    // profile link, href="/@username"
	"__TEXT__":      "div[dir='auto'],span[dir='auto']", // text lines inside a card
	// search box on /search
	"__SEARCH_INPUT__": "input[type='search'],input[placeholder*='Search'],input[placeholder*='Поиск'],input[aria-label*='Search'],input[aria-label*='Поиск']",
}

func sel(name string) string { return selectors[name] }

func withSelectors(js string) string {
	pairs := make([]string, 0, len(selectors)*2)
	for k, v := range selectors {
		pairs = append(pairs, k, v)
	}
	return strings.NewReplacer(pairs...).Replace(js)
}

// extractJS returns a JSON string: [{username, post_id, permalink, timestamp, text}].
// It anchors on permalinks, climbs to the nearest card and reads pieces by
// semantics (href patterns, <time>), never by generated class names.
var extractJS = withSelectors(`() => {
  const posts = [];
  const seen = new Set();
  for (const a of document.querySelectorAll("__POST_LINK__")) {
    const href = a.getAttribute("href") || "";
    const m = href.match(/^\/@([^/]+)\/post\/([A-Za-z0-9_-]+)/);
    if (!m) continue;
    const key = m[1] + "/" + m[2];
    if (seen.has(key)) continue;
    const card = a.closest("__CONTAINER__") || a.parentElement;
    if (!card) continue;
    seen.add(key);
    const t = card.querySelector("__TIME__");
    const lines = [];
    for (const el of card.querySelectorAll("__TEXT__")) {
      if (el.closest("a")) continue;                 // usernames, link rows
      if (el.querySelector("__TEXT__")) continue;    // leaves only
      const s = (el.innerText || "").trim();
      if (s) lines.push(s);
    }
    posts.push({
      username: m[1],
      post_id: m[2],
      permalink: "https://www.threads.com/@" + m[1] + "/post/" + m[2],
      timestamp: t ? (t.getAttribute("datetime") || "") : "",
      text: lines.join("\n").trim(),
    });
  }
  return JSON.stringify(posts);
}`)

// exploreJS dumps structural facts about the first result cards for manual inspection.
var exploreJS = withSelectors(`() => {
  const out = { url: location.href, title: document.title, samples: [] };
  out.post_links = document.querySelectorAll("__POST_LINK__").length;
  out.containers = document.querySelectorAll("__CONTAINER__").length;
  const seen = new Set();
  for (const a of document.querySelectorAll("__POST_LINK__")) {
    const href = a.getAttribute("href");
    if (seen.has(href)) continue;
    seen.add(href);
    let el = a, depth = 0, c = null;
    while (el && depth < 15) {
      if (el.hasAttribute && (el.hasAttribute("data-pressable-container") || el.tagName === "ARTICLE")) { c = el; break; }
      el = el.parentElement; depth++;
    }
    out.samples.push({
      href, depth_to_container: c ? depth : null,
      container_tag: c ? c.tagName : null,
      container_attrs: c ? [...c.attributes].map(x => x.name + "=" + x.value.slice(0, 40)) : null,
      container_text: c ? c.innerText.slice(0, 300) : null,
      times: c ? [...c.querySelectorAll("time")].map(t => t.getAttribute("datetime")) : null,
      author_links: c ? [...c.querySelectorAll("__AUTHOR__")].map(x => x.getAttribute("href")).slice(0, 5) : null,
      text_leaves: c ? [...c.querySelectorAll("__TEXT__")].filter(e => !e.querySelector("__TEXT__")).map(e => e.innerText.slice(0, 80)) : null,
    });
    if (out.samples.length >= 5) break;
  }
  return JSON.stringify(out, null, 2);
}`)
