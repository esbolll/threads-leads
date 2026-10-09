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
- `threads-leads.exe export -n 100`, `threads-leads.exe stats`.

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
→ after all queries, every lead with notified_at NULL is printed, sent to Telegram if configured, and marked. Reruns never resend.

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
- 2026-10-09: business portfolio "ИП Konsbayev" created, identity + business verification submitted (up to 48 h / 5 business days).
  App settings done: privacy/terms/data-deletion at https://esbolll.github.io/threads-leads/, icon, category.
- Next after approval: App Review request using `docs/app-review.md` (needs a screencast; post a test "looking for devops" first).
- `.env` holds THREADS_TOKEN / THREADS_APP_ID / THREADS_APP_SECRET / TELEGRAM_*; never print or commit it.

## Next
1. Browser path: one more `login` (Instagram code throttling permitting), then `explore` to verify selectors on real cards, then `run`.
2. Telegram: user creates bot + channel, `tg chats` → TELEGRAM_CHAT_ID, `tg test`.
3. Scheduled runs every 10–20 min (API mode fits a Docker container; browser mode needs the desktop).
4. LLM classification and lead scoring after rules are tuned on real data.
