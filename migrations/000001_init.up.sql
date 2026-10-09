CREATE TABLE IF NOT EXISTS posts (
    key            TEXT PRIMARY KEY,
    source         TEXT NOT NULL,
    username       TEXT NOT NULL,
    display_name   TEXT NOT NULL DEFAULT '',
    text           TEXT NOT NULL,
    permalink      TEXT NOT NULL,
    post_id        TEXT NOT NULL DEFAULT '',
    posted_at      TEXT NOT NULL DEFAULT '',
    search_query   TEXT NOT NULL,
    first_seen_at  TIMESTAMP NOT NULL,
    last_seen_at   TIMESTAMP NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_posts_username ON posts(username);
CREATE INDEX IF NOT EXISTS idx_posts_first_seen ON posts(first_seen_at);

CREATE TABLE IF NOT EXISTS leads (
    post_key        TEXT PRIMARY KEY REFERENCES posts(key) ON DELETE CASCADE,
    category        TEXT NOT NULL,
    matched_tech    TEXT NOT NULL,  -- JSON array
    matched_intent  TEXT NOT NULL,  -- JSON array
    collected_at    TIMESTAMP NOT NULL,
    notified_at     TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_leads_notified ON leads(notified_at);
