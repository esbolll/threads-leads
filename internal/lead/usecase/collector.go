package usecase

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/esbolll/threads-leads/internal/filter"
	"github.com/esbolll/threads-leads/internal/lead/model"
	"github.com/esbolll/threads-leads/internal/lead/repository"
)

// Source is a data source: one query in, raw public posts out.
type Source interface {
	Name() string
	Search(ctx context.Context, query string) ([]model.Post, error)
}

// Notifier delivers a lead somewhere (Telegram).
type Notifier interface {
	SendLead(ctx context.Context, l model.Lead) error
}

type Collector struct {
	repo     *repository.Repository
	source   Source
	notifier Notifier // nil = print only
	log      *logrus.Logger
	out      io.Writer
}

func NewCollector(repo *repository.Repository, src Source, n Notifier, log *logrus.Logger, out io.Writer) *Collector {
	return &Collector{repo: repo, source: src, notifier: n, log: log, out: out}
}

type Summary struct {
	Queries  int
	Posts    int
	NewPosts int
	Leads    int
	NewLeads int
	Sent     int
}

// Run searches every query, stores posts, classifies leads and notifies the new ones.
func (c *Collector) Run(ctx context.Context, queries []string) (Summary, error) {
	var sum Summary
	now := time.Now().UTC()

	for _, q := range queries {
		if err := ctx.Err(); err != nil {
			return sum, err
		}
		fmt.Fprintf(c.out, "\nSearching: %s\n", q)
		posts, err := c.source.Search(ctx, q)
		if err != nil {
			c.log.WithError(err).WithField("query", q).Warn("search failed")
			continue
		}
		sum.Queries++
		relevant := 0
		for _, p := range posts {
			p.SearchQuery = q
			isNew, err := c.repo.UpsertPost(ctx, p, now)
			if err != nil {
				return sum, err
			}
			sum.Posts++
			if isNew {
				sum.NewPosts++
			}
			lead, ok := filter.Classify(p, now)
			if !ok {
				continue
			}
			relevant++
			if err := c.repo.UpsertLead(ctx, lead); err != nil {
				return sum, err
			}
		}
		sum.Leads += relevant
		fmt.Fprintf(c.out, "Found: %d posts\n", len(posts))
		fmt.Fprintf(c.out, "Relevant: %d\n", relevant)
	}

	fresh, err := c.repo.Unnotified(ctx)
	if err != nil {
		return sum, err
	}
	sum.NewLeads = len(fresh)

	fmt.Fprintln(c.out)
	for _, l := range fresh {
		printLead(c.out, l)
	}
	fmt.Fprintln(c.out, strings.Repeat("-", 27))
	fmt.Fprintf(c.out, "Posts: %d (new %d), leads this run: %d, new leads: %d\n", sum.Posts, sum.NewPosts, sum.Leads, sum.NewLeads)

	for _, l := range fresh {
		if c.notifier != nil {
			if err := c.notifier.SendLead(ctx, l); err != nil {
				c.log.WithError(err).WithField("permalink", l.Permalink).Warn("notify failed")
				continue
			}
			sum.Sent++
			time.Sleep(1100 * time.Millisecond) // Telegram channel limit is about 1 msg/s
		}
		if err := c.repo.MarkNotified(ctx, l.Key(), time.Now().UTC()); err != nil {
			return sum, err
		}
	}
	if c.notifier != nil {
		fmt.Fprintf(c.out, "Sent to Telegram: %d\n", sum.Sent)
	}
	return sum, nil
}

func printLead(w io.Writer, l model.Lead) {
	fmt.Fprintln(w, strings.Repeat("-", 27))
	fmt.Fprintln(w, "[LEAD]")
	fmt.Fprintf(w, "Author: @%s\n", l.Username)
	fmt.Fprintf(w, "Type: %s\n", l.Category)
	fmt.Fprintf(w, "Tech: %s\n", strings.Join(l.MatchedTech, ", "))
	fmt.Fprintln(w)
	text := l.Text
	if r := []rune(text); len(r) > 600 {
		text = string(r[:600]) + "..."
	}
	fmt.Fprintln(w, text)
	fmt.Fprintln(w)
	fmt.Fprintln(w, l.Permalink)
}
