# threads-leads

Go CLI (go-rod over CDP) that searches Threads for hiring/freelance posts, filters them by rules and writes `output/leads.json`.
Single binary: `go build -o threads-leads.exe .`

## Commands
- `threads-leads.exe -login` — open Chrome with the persistent profile, wait for manual login, exit.
- `threads-leads.exe -explore "query"` — dump DOM facts to `output/explore.json`, `search.html`, `search.png`.
- `threads-leads.exe [-v] [-headless] [query ...]` — full run; no args = all queries from `config.go`.
- `-skip-login` — diagnostics only, do not wait for a session.

## Hard-won facts (2026-10-09)
- Chrome launched with `--enable-automation` (go-rod default) gets a 404 shell from threads.com and Meta drops the
  session server-side right after login. `launcher.Delete("enable-automation")` fixes it; keep `disable-blink-features=AutomationControlled`.
- Direct search URL works: `https://www.threads.com/search?q=<q>&serp_type=default` (needs a logged-in session; logged out shows "No results").
  Typing into the search box is only a fallback; the login modal covers the input when logged out.
- Never mix browsers on one profile: a cookie DB written by Playwright's Chromium is unreadable by Google Chrome
  (different encryption) and gets wiped. The profile is Google Chrome only (`launcher.LookPath()`), override with `THREADS_CHROME`.
- Instagram throttles confirmation codes after ~3 logins in an hour. Do not ask the user to re-login repeatedly;
  make sure session persistence is proven before asking for another login.
- The Claude Code Bash/PowerShell sandbox cannot spawn a headed Chrome (`spawn UNKNOWN`) and virtualises `%LOCALAPPDATA%`
  writes. Run anything that opens a window from the user's terminal panel.
- Session check = cookies `sessionid` + `ds_user_id` on a `*threads*` domain. Instagram-only cookies mean the Threads step of login was not finished.
- Ctrl-C handling closes Chrome; otherwise orphaned Chrome holds the profile lock and the next launch fails with "Failed to get the debug url".

## Threads API status (2026-10-09)
- App "Threads Leads" (Threads app id 1804019570933559), user @yesbolkonsbayev is a Threads Tester.
- The dashboard "User Token Generator" issues tokens WITHOUT `threads_keyword_search` (fixed scope set). Use `threads-leads.exe -auth`
  (OAuth, redirect `https://localhost:8443/callback`, needs THREADS_APP_ID/SECRET in .env) to get a token with the scope.
- With the scope, `/keyword_search` answers 200 but `{"data":[]}` for every query: until App Review approves
  `threads_keyword_search` it searches only the token owner's own posts. App Review needs: privacy policy URL, app icon,
  screencast of the call, and "Become a Tech Provider" = Meta business verification (legal entity documents).
- Token check: `GET graph.threads.net/v1.0/debug_token?input_token=T&access_token=T` -> `data.scopes`.
- `.env` holds THREADS_TOKEN / THREADS_APP_ID / THREADS_APP_SECRET; never print or commit it.

## Layout
- `config.go` — queries, keyword lists, pacing.
- `selectors.go` — every DOM selector and the in-page JS; the only file to touch when Threads changes markup.
- `browser.go` — launcher, login wait, navigation, scrolling, extraction calls.
- `filters.go` — relevance rules (TECH + HIRING, negative patterns), category, dedupe. Tests in `filters_test.go`.
- `api.go` — official API source (`-api`): keyword_search + /me check.
- `auth.go` — OAuth flow (`-auth`) with a self-signed localhost TLS callback; writes THREADS_TOKEN to .env.
- `main.go` — CLI; both sources feed `collectAll` (filter, print, save).

## Next
1. After login works: run `-explore`, verify `data-pressable-container` / `time[datetime]` / text leaves against real cards, adjust `selectors.go`.
2. First full run, tune negative patterns on real output.
3. Telegram: bot `sendMessage` to a channel with link + author + first 300 chars. Not before live leads are seen.
4. Threads API (`threads_keyword_search`, 2200 queries/day) as a replacement data source once App Review passes.
