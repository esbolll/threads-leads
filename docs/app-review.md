# App Review submission notes (threads_keyword_search)

Texts to paste into the Meta App Review form. Keep them in English.

## Use case description

Threads Leads is an internal tool used by Individual Entrepreneur Yesbol Konsbayev, a software
development business, to find contract and freelance work on Threads.

How it works:
1. The operator (the only user of the app) authenticates with their own Threads account.
2. The app calls GET /keyword_search with a fixed list of hiring-related phrases such as
   "looking for a DevOps engineer", "hiring backend developer", "need python developer".
3. Returned public posts are filtered locally with keyword rules (must mention an IT skill and a hiring intent)
   and shown to the operator with the author username and the permalink, so the operator can open the post
   on Threads and reply manually.

Why threads_keyword_search is needed:
Without this permission keyword search only returns the operator's own posts, so the app cannot find
public posts from people who are looking for a contractor.

Data handling:
- Only public posts returned by the API are processed.
- Data is stored locally on the operator's computer, retained for at most 90 days, never shared or sold.
- The app does not publish, reply, like or follow. It only reads search results.
- Privacy policy: <PAGES_URL>/privacy-policy.html
- Data deletion: <PAGES_URL>/data-deletion.html

## Step-by-step test instructions for the reviewer

1. The app is a desktop command-line tool; there is no public UI. See the screencast.
2. Run `threads-leads.exe run -api "looking for devops"`.
3. The tool verifies the token with GET /me, then calls GET /keyword_search?q=looking%20for%20devops&search_type=RECENT.
4. Matching public posts are printed with username, text excerpt and permalink and stored in SQLite (data/leads.db); `threads-leads.exe export` prints them as JSON.

## Screencast checklist (1–2 minutes, no cuts)

1. Show the app dashboard with the Threads use case and the threads_keyword_search permission.
2. Show the terminal: run `threads-leads.exe run -api "looking for devops"`.
3. Show the output: `Threads API token OK`, the search line, results (with a test account they will be the
   operator's own posts: post something like "Test: looking for devops" before recording so the search is not empty).
4. Run `threads-leads.exe export` and show the JSON.
5. Show the privacy policy page in a browser.

## Before submitting

- Business verification completed (Tech Provider) for the IE.
- App icon 1024x1024 uploaded (docs/icon.png).
- Privacy Policy URL, Data Deletion URL, App category ("Business and Pages" or "Productivity"), contact email set in
  App settings → Basic.
- At least one successful keyword_search call made with the test token (the dashboard "Test use cases" page
  counts it).
