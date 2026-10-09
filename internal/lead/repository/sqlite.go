package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/golang-migrate/migrate/v4"
	msqlite "github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"

	"github.com/esbolll/threads-leads/internal/lead/model"
	"github.com/esbolll/threads-leads/migrations"
)

type Repository struct {
	db *sqlx.DB
}

// Open opens (creating if needed) the SQLite database and applies migrations.
func Open(path string) (*Repository, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	dsn := fmt.Sprintf("file:%s?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)", filepath.ToSlash(path))
	db, err := sqlx.Connect("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	db.SetMaxOpenConns(1)
	if err := migrateUp(db.DB); err != nil {
		db.Close()
		return nil, err
	}
	return &Repository{db: db}, nil
}

func migrateUp(db *sql.DB) error {
	src, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return err
	}
	drv, err := msqlite.WithInstance(db, &msqlite.Config{})
	if err != nil {
		return err
	}
	m, err := migrate.NewWithInstance("iofs", src, "sqlite", drv)
	if err != nil {
		return err
	}
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migrate: %w", err)
	}
	return nil
}

func (r *Repository) Close() error { return r.db.Close() }

// UpsertPost stores the post; returns true when it was not seen before.
func (r *Repository) UpsertPost(ctx context.Context, p model.Post, now time.Time) (bool, error) {
	res, err := r.db.ExecContext(ctx, `
		INSERT INTO posts (key, source, username, display_name, text, permalink, post_id, posted_at, search_query, first_seen_at, last_seen_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(key) DO UPDATE SET last_seen_at = excluded.last_seen_at`,
		p.Key(), p.Source, p.Username, p.DisplayName, p.Text, p.Permalink, p.PostID, p.PostedAt, p.SearchQuery, now, now)
	if err != nil {
		return false, err
	}
	// SQLite reports 1 changed row for both insert and update; detect insert via first_seen_at.
	var first time.Time
	if err := r.db.GetContext(ctx, &first, `SELECT first_seen_at FROM posts WHERE key = ?`, p.Key()); err != nil {
		return false, err
	}
	_ = res
	return first.Equal(now), nil
}

// UpsertLead stores the classification; existing rows keep their notified_at.
func (r *Repository) UpsertLead(ctx context.Context, l model.Lead) error {
	tech, _ := json.Marshal(l.MatchedTech)
	intent, _ := json.Marshal(l.MatchedIntent)
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO leads (post_key, category, matched_tech, matched_intent, collected_at, notified_at)
		VALUES (?, ?, ?, ?, ?, NULL)
		ON CONFLICT(post_key) DO UPDATE SET
			category = excluded.category,
			matched_tech = excluded.matched_tech,
			matched_intent = excluded.matched_intent`,
		l.Key(), l.Category, string(tech), string(intent), l.CollectedAt)
	return err
}

type leadRow struct {
	model.Post
	Category      string       `db:"category"`
	MatchedTech   string       `db:"matched_tech"`
	MatchedIntent string       `db:"matched_intent"`
	CollectedAt   time.Time    `db:"collected_at"`
	NotifiedAt    sql.NullTime `db:"notified_at"`
}

func (row leadRow) toLead() model.Lead {
	l := model.Lead{Post: row.Post, Category: row.Category, CollectedAt: row.CollectedAt}
	_ = json.Unmarshal([]byte(row.MatchedTech), &l.MatchedTech)
	_ = json.Unmarshal([]byte(row.MatchedIntent), &l.MatchedIntent)
	if row.NotifiedAt.Valid {
		t := row.NotifiedAt.Time
		l.NotifiedAt = &t
	}
	return l
}

const leadSelect = `
	SELECT p.source, p.username, p.display_name, p.text, p.permalink, p.post_id, p.posted_at, p.search_query,
	       p.first_seen_at, p.last_seen_at,
	       l.category, l.matched_tech, l.matched_intent, l.collected_at, l.notified_at
	FROM leads l JOIN posts p ON p.key = l.post_key`

// Unnotified returns leads that were never sent anywhere, oldest first.
func (r *Repository) Unnotified(ctx context.Context) ([]model.Lead, error) {
	var rows []leadRow
	if err := r.db.SelectContext(ctx, &rows, leadSelect+` WHERE l.notified_at IS NULL ORDER BY p.first_seen_at`); err != nil {
		return nil, err
	}
	out := make([]model.Lead, 0, len(rows))
	for _, row := range rows {
		out = append(out, row.toLead())
	}
	return out, nil
}

func (r *Repository) MarkNotified(ctx context.Context, key string, now time.Time) error {
	_, err := r.db.ExecContext(ctx, `UPDATE leads SET notified_at = ? WHERE post_key = ?`, now, key)
	return err
}

// Recent returns the latest leads for export.
func (r *Repository) Recent(ctx context.Context, limit int) ([]model.Lead, error) {
	var rows []leadRow
	if err := r.db.SelectContext(ctx, &rows, leadSelect+` ORDER BY p.first_seen_at DESC LIMIT ?`, limit); err != nil {
		return nil, err
	}
	out := make([]model.Lead, 0, len(rows))
	for _, row := range rows {
		out = append(out, row.toLead())
	}
	return out, nil
}

type Stats struct {
	Posts      int `db:"posts"`
	Leads      int `db:"leads"`
	Unnotified int `db:"unnotified"`
}

func (r *Repository) Stats(ctx context.Context) (Stats, error) {
	var s Stats
	err := r.db.GetContext(ctx, &s, `
		SELECT (SELECT COUNT(*) FROM posts) AS posts,
		       (SELECT COUNT(*) FROM leads) AS leads,
		       (SELECT COUNT(*) FROM leads WHERE notified_at IS NULL) AS unnotified`)
	return s, err
}
