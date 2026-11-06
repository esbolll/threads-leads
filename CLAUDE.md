# threads-leads

Go CLI that searches Threads for hiring/freelance IT posts, stores them in SQLite and sends new leads to Telegram.
Two data sources behind one interface: Google Chrome over CDP (go-rod, persistent profile) and the official Threads API.

Build: `make build` (or `go build -o threads-leads.exe ./cmd/app`). Tests: `make test`.

## Commands
- `threads-leads.exe run [-api] [-no-notify] [query ...]` — collect, classify, store, notify. No queries = all from `config/queries.go`.
- `threads-leads.exe login` — open Chrome with the profile, wait for manual login, exit.
- `threads-leads.exe explore "query"` — dump DOM facts (`explore.json`, `search.html`, `search.png`) into OUTPUT_DIR.
- `threads-leads.exe auth` — OAuth for a Threads API token with `threads_keyword_search`; writes THREADS_TOKEN into .env.
- `threads-leads.exe tg test | tg chats` — Telegram smoke test / find the channel id.
- `threads-leads.exe export -n 100`, `threads-leads.exe stats`, `threads-leads.exe reclassify`.

## Layout (same conventions as myor/backend and whatsapp-go)
- `cmd/app/main.go` — subcommands and flags only.
- `app/app.go` — wiring: config → repo → source → collector → notifier.
- `config/` — `caarlos0/env` + `godotenv` config, search queries, `.env` writer.
- `internal/lead/{model,repository,usecase}` — Post/Lead, SQLite (sqlx + modernc, migrations via golang-migrate iofs), Collector.
- `internal/filter/` — relevance rules (TECH + HIRING, negative patterns), category; tests.
- `internal/source/browser/` — go-rod client; `selectors.go` is the only file to touch when Threads changes markup.
- `internal/source/threadsapi/` — API client (`keyword_search`, `/me`, `debug_token`) and the OAuth flow.
- `internal/notify/telegram/` — bot client, message format.
- `migrations/` — embedded SQL. `data/leads.db` is gitignored.
- `docs/` — GitHub Pages (privacy, terms, data deletion), app icon, App Review notes.

## Data flow
Source.Search(query) → posts upserted into `posts` (first_seen/last_seen) → filter.Classify → `leads` (notified_at NULL)
→ right after EACH query every lead with notified_at NULL is printed, sent to Telegram (silent) and marked.
Batches go oldest → newest so the freshest lead is at the bottom of the channel. Reruns never resend (key = permalink).
`.env`: `LEAD_CURRENT_MONTH_ONLY=true` drops posts from earlier months, `LEAD_MAX_AGE_DAYS=N` drops older than N days.

## Filter
A hiring word must sit next to a tech word: intent then tech within 40 runes ("need a developer"), or tech then
intent within 15 ("developer needed"). Role words (StrongTech: разработчик, devops, python) are enough alone.
Thing words (WeakTech: сайт, бот, app, crm) are everyday talk, so they also need a SeekMarker: a person to hire or
money ("ищу", "кто сделает", "специалист", "бюджет", "срочно", "тг", "$"). "проект"/"заказ" are category words only.
Negative phrases reject job seekers, sellers advertising themselves and non-IT "engineer" posts.
Rules: `internal/filter/rules.go`. Every real false positive or missed lead goes into `filter_test.go` first.
Tune offline with `threads-leads.exe reclassify` (reruns rules over stored posts, no API calls, changes nothing):
2026-11-06 on 1873 stored posts the old rules accepted 551, the current ones 129, mostly real orders.
API source drops replies inside other threads (`LEAD_SKIP_REPLIES=true`).

Region filter (`internal/filter/region.go`, `LEAD_REGION_FILTER=true` by default): the target market is Russian- and
Kazakh-speaking. Russian and Kazakh posts always pass; Ukrainian/Belarusian (letters ї є ґ ў, or "і" without Kazakh
letters/words) are dropped; English passes only with a CIS marker: city/country word, +7/+998/+996 phone code, ₸/₽,
flag emoji, .kz/.uz domain, username segment like `kz`/`almaty`, or any Cyrillic word in the text. Posts carry no
location, so this is a heuristic. Kazakh is verb-final, so for Kazakh text the tech→intent window is also 60 runes.
Queries: `config/queries.go` = dev RU, devops RU, web RU, Kazakh, English pinned to the region ("developer Almaty").

## Hard-won facts (2026-10-09)
- Chrome with `--enable-automation` (go-rod default) gets a 404 shell from threads.com and Meta drops the session right after
  login. `launcher.Delete("enable-automation")` fixes it; keep `disable-blink-features=AutomationControlled`.
- Never call `launcher.Cleanup()`: it deletes the user data dir (the saved session). This caused three "lost session" rounds.
- Direct search URL works: `https://www.threads.com/search?q=<q>&serp_type=default` (logged-in only). Typing into the box is a fallback.
- Never mix browsers on one profile: Playwright Chromium cookies are unreadable by Google Chrome and get wiped.
- Instagram throttles confirmation codes after ~3 logins in an hour. Prove session persistence before asking for another login.
- The Claude Code Bash/PowerShell sandbox cannot spawn a headed Chrome and virtualises `%LOCALAPPDATA%` writes:
  run anything that opens a window from the user's terminal panel.
- Session check = cookies `sessionid` + `ds_user_id` on a `*threads*` domain. Instagram-only cookies mean the Threads step was not finished.
- Orphaned Chrome holds the profile lock → next launch fails with "Failed to get the debug url". Ctrl-C now closes Chrome.

## Threads API status
- App "Threads Leads" (Threads app id 1804019570933559), user @yesbolkonsbayev is a Threads Tester.
- The dashboard token generator never includes `threads_keyword_search`; use `auth` (OAuth, redirect `https://localhost:8443/callback`).
- With the scope, `/keyword_search` returns `{"data":[]}` for everything until App Review approves it (searches own posts only).
- 2026-10-15: business verification APPROVED (portfolio "ИП Konsbayev", registry match). Identity check needed a real camera photo of the physical ID; scans and resized photos were rejected twice.
  App settings done: privacy/terms/data-deletion at https://esbolll.github.io/threads-leads/, icon, category.
- 2026-10-20: App Review SUBMITTED for threads_keyword_search + threads_basic (status "in review", Meta says up to 20 days).
  Screencast had to show the OAuth consent (`auth` command) — the form requires it. Platform "Website" added
  (https://esbolll.github.io/threads-leads/) because reviewer instructions need a platform; Facebook Login = No.
  Test post "Test: looking for devops" on @yesbolkonsbayev makes keyword_search return 1 result for the demo.
- 2026-11-06: App Review APPROVED and the app is published (Live). `keyword_search` now returns public posts
  with the existing token (no re-auth needed). Token is long-lived, 60 days: rerun `auth` before it expires.
- API facts: `search_type=RECENT` returns only roughly the last 24 h (2–11 posts per query); `TOP` returns popular
  posts back to 2024 — useless for monitoring. Timestamps come as `+0000` and are normalised to RFC3339.
  Limit 2200 queries/24 h per user: with ~100 queries per pass the app raises `-every` to ~73 min automatically.
- API permalinks have the same shape as browser ones, so both sources dedupe against each other in one DB.
- `.env` holds THREADS_TOKEN / THREADS_APP_ID / THREADS_APP_SECRET / TELEGRAM_*; never print or commit it.

## Browser source status
- 2026-10-23: session persists across restarts (Google Chrome profile, no Cleanup). `explore` confirmed the markup:
  cards = `div[data-pressable-container]`, `time[datetime]`, `a[href^='/@']`, text leaves `div/span[dir=auto]`.
  First real run: 3 queries → 57 posts, 39 leads, real budgets ($350–$2500 sites, DevOps projects). ~20 s per query.
- Text extraction drops like/reply counters, carousel "1 / 2", bare username and relative dates.
- `run -every 15m` keeps running (pause between passes). A full browser pass over ~100 queries takes ~30 min.
- If Chrome was left running, launch fails once, the app kills Chrome processes bound to the profile and retries.

## Next
1. Main mode now: `threads-leads.exe run -api -every 60m` (no Chrome; ~80 queries → min interval ~58 min).
   Browser mode stays as a wider, slower source.
2. Filter tuning on real output; rule-based has a ceiling → LLM classification / scoring for the leads that pass rules.
3. API mode in Docker on a server (no desktop needed); token refresh before the 60-day expiry.
