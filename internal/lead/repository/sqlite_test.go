package repository

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/esbolll/threads-leads/internal/lead/model"
)

func TestPostsAndLeads(t *testing.T) {
	repo, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	p := model.Post{Source: "threads", Username: "u", Text: "need devops", Permalink: "https://www.threads.com/@u/post/abc", SearchQuery: "need devops"}
	isNew, err := repo.UpsertPost(ctx, p, now)
	if err != nil || !isNew {
		t.Fatalf("first upsert: new=%v err=%v", isNew, err)
	}
	isNew, err = repo.UpsertPost(ctx, p, now.Add(time.Minute))
	if err != nil || isNew {
		t.Fatalf("second upsert: new=%v err=%v", isNew, err)
	}

	lead := model.Lead{Post: p, Category: "project", MatchedTech: []string{"devops"}, MatchedIntent: []string{"need"}, CollectedAt: now}
	if err := repo.UpsertLead(ctx, lead); err != nil {
		t.Fatal(err)
	}
	fresh, err := repo.Unnotified(ctx)
	if err != nil || len(fresh) != 1 {
		t.Fatalf("unnotified: n=%d err=%v", len(fresh), err)
	}
	if fresh[0].Category != "project" || len(fresh[0].MatchedTech) != 1 || fresh[0].Username != "u" {
		t.Fatalf("round trip mismatch: %+v", fresh[0])
	}
	if err := repo.MarkNotified(ctx, p.Key(), now); err != nil {
		t.Fatal(err)
	}
	fresh, _ = repo.Unnotified(ctx)
	if len(fresh) != 0 {
		t.Fatalf("expected 0 unnotified, got %d", len(fresh))
	}
	// Re-upserting the lead must keep notified_at.
	if err := repo.UpsertLead(ctx, lead); err != nil {
		t.Fatal(err)
	}
	fresh, _ = repo.Unnotified(ctx)
	if len(fresh) != 0 {
		t.Fatalf("re-upsert reset notified_at")
	}
	s, err := repo.Stats(ctx)
	if err != nil || s.Posts != 1 || s.Leads != 1 || s.Unnotified != 0 {
		t.Fatalf("stats: %+v err=%v", s, err)
	}
}
