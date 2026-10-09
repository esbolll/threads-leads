// Package app wires config, storage, sources and notifiers together.
package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/esbolll/threads-leads/config"
	"github.com/esbolll/threads-leads/internal/lead/repository"
	"github.com/esbolll/threads-leads/internal/lead/usecase"
	"github.com/esbolll/threads-leads/internal/notify/telegram"
	"github.com/esbolll/threads-leads/internal/source/browser"
	"github.com/esbolll/threads-leads/internal/source/threadsapi"
)

type App struct {
	cfg *config.Config
	log *logrus.Logger
	out io.Writer
}

func New(cfg *config.Config, log *logrus.Logger, out io.Writer) *App {
	return &App{cfg: cfg, log: log, out: out}
}

type RunOptions struct {
	UseAPI  bool
	Notify  bool
	Queries []string
}

// Run collects leads from the chosen source into SQLite and notifies new ones.
func (a *App) Run(ctx context.Context, opts RunOptions) error {
	repo, err := repository.Open(a.cfg.DBPath)
	if err != nil {
		return err
	}
	defer repo.Close()

	var src usecase.Source
	if opts.UseAPI {
		api, err := a.apiClient(ctx)
		if err != nil {
			return err
		}
		src = api
	} else {
		b, err := a.browserClient(ctx)
		if err != nil {
			return err
		}
		defer b.Close()
		src = b
	}

	var notifier usecase.Notifier
	if opts.Notify && a.cfg.Telegram.Enabled() {
		notifier = telegram.New(a.cfg.Telegram.BotToken, a.cfg.Telegram.ChatID)
	}

	queries := opts.Queries
	if len(queries) == 0 {
		queries = config.SearchQueries()
	}
	collector := usecase.NewCollector(repo, src, notifier, a.log, a.out)
	sum, err := collector.Run(ctx, queries)
	if err != nil {
		return err
	}
	a.log.WithFields(logrus.Fields{
		"queries": sum.Queries, "posts": sum.Posts, "new_posts": sum.NewPosts,
		"leads": sum.Leads, "new_leads": sum.NewLeads, "sent": sum.Sent,
	}).Info("run finished")
	return nil
}

func (a *App) apiClient(ctx context.Context) (*threadsapi.Client, error) {
	if a.cfg.Threads.Token == "" {
		return nil, fmt.Errorf("THREADS_TOKEN is empty: run `auth` first")
	}
	api := threadsapi.New(a.cfg.Threads.Token, a.cfg.Threads.SearchLimit)
	user, err := api.Me(ctx)
	if err != nil {
		return nil, fmt.Errorf("token check failed: %w", err)
	}
	scopes, _ := api.Scopes(ctx)
	fmt.Fprintf(a.out, "Threads API token OK, user @%s, scopes: %s\n", user, strings.Join(scopes, ","))
	return api, nil
}

func (a *App) browserClient(ctx context.Context) (*browser.Client, error) {
	b, err := browser.New(browser.Options{
		Chrome:          a.cfg.Browser.Chrome,
		ProfileDir:      a.cfg.Browser.ProfileDir,
		Headless:        a.cfg.Browser.Headless,
		ScrollsPerQuery: a.cfg.Browser.ScrollsPerQuery,
	}, a.log)
	if err != nil {
		return nil, err
	}
	if err := b.EnsureLoggedIn(ctx); err != nil {
		b.Close()
		return nil, err
	}
	return b, nil
}

// Login opens Chrome, waits for a manual login and exits.
func (a *App) Login(ctx context.Context) error {
	b, err := a.browserClient(ctx)
	if err != nil {
		return err
	}
	defer b.Close()
	fmt.Fprintln(a.out, "Done. You can close the browser.")
	return nil
}

// Explore dumps DOM facts for a search query into the output dir.
func (a *App) Explore(ctx context.Context, query string) error {
	b, err := a.browserClient(ctx)
	if err != nil {
		return err
	}
	defer b.Close()
	info, err := b.Explore(query, a.cfg.OutputDir)
	if err != nil {
		return err
	}
	fmt.Fprintln(a.out, info)
	fmt.Fprintf(a.out, "saved explore.json, search.html, search.png in %s\n", a.cfg.OutputDir)
	return nil
}

// Auth runs the OAuth flow and writes THREADS_TOKEN into .env.
func (a *App) Auth(ctx context.Context) error {
	tok, err := threadsapi.OAuth{
		AppID:       a.cfg.Threads.AppID,
		AppSecret:   a.cfg.Threads.AppSecret,
		RedirectURI: a.cfg.Threads.RedirectURI,
		Out:         a.out,
	}.Authorize(ctx)
	if err != nil {
		return err
	}
	if err := config.WriteEnvValue(".env", "THREADS_TOKEN", tok.AccessToken); err != nil {
		return err
	}
	fmt.Fprintf(a.out, "Long-lived token saved to .env (expires in %d days).\n", tok.ExpiresIn/86400)
	return nil
}

func (a *App) telegram() (*telegram.Client, error) {
	if !a.cfg.Telegram.Enabled() {
		return nil, fmt.Errorf("set TELEGRAM_BOT_TOKEN and TELEGRAM_CHAT_ID in .env")
	}
	return telegram.New(a.cfg.Telegram.BotToken, a.cfg.Telegram.ChatID), nil
}

func (a *App) TelegramTest(ctx context.Context) error {
	tg, err := a.telegram()
	if err != nil {
		return err
	}
	if err := tg.SendText(ctx, "threads-leads: test message OK"); err != nil {
		return err
	}
	fmt.Fprintln(a.out, "Telegram test message sent")
	return nil
}

// TelegramChats lists chats the bot has seen, to find the channel id. Needs only the bot token.
func (a *App) TelegramChats(ctx context.Context) error {
	if a.cfg.Telegram.BotToken == "" {
		return fmt.Errorf("set TELEGRAM_BOT_TOKEN in .env")
	}
	chats, err := telegram.New(a.cfg.Telegram.BotToken, "").RecentChats(ctx)
	if err != nil {
		return err
	}
	if len(chats) == 0 {
		fmt.Fprintln(a.out, "No chats seen yet. Add the bot as admin to the channel, post any message there, then rerun.")
		return nil
	}
	for _, c := range chats {
		fmt.Fprintf(a.out, "chat_id=%d type=%s title=%q username=%q\n", c.ID, c.Type, c.Title, c.Username)
	}
	return nil
}

// Export prints the latest leads as JSON.
func (a *App) Export(ctx context.Context, limit int) error {
	repo, err := repository.Open(a.cfg.DBPath)
	if err != nil {
		return err
	}
	defer repo.Close()
	leads, err := repo.Recent(ctx, limit)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(a.out)
	enc.SetIndent("", "  ")
	return enc.Encode(leads)
}

// Stats prints row counts.
func (a *App) Stats(ctx context.Context) error {
	repo, err := repository.Open(a.cfg.DBPath)
	if err != nil {
		return err
	}
	defer repo.Close()
	s, err := repo.Stats(ctx)
	if err != nil {
		return err
	}
	fmt.Fprintf(a.out, "db: %s\nposts: %d\nleads: %d\nunnotified: %d\n", a.cfg.DBPath, s.Posts, s.Leads, s.Unnotified)
	return nil
}
