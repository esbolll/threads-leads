package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"

	"github.com/sirupsen/logrus"

	"github.com/esbolll/threads-leads/app"
	"github.com/esbolll/threads-leads/config"
)

const usage = `threads-leads <command> [flags] [queries...]

Commands:
  run        collect leads with the browser source (default) and notify new ones
             -api        use the official Threads API instead of Chrome
             -no-notify  do not send to Telegram
  login      open Chrome with the persistent profile, wait for manual login
  explore    dump DOM facts for one query into the output dir
  auth       OAuth flow for a Threads API token (writes THREADS_TOKEN to .env)
  tg test    send a test message to the Telegram chat
  tg chats   list chats the bot has seen (to find the channel id)
  export     print latest leads as JSON  (-n 100)
  stats      print database counts
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	cfg, err := config.Load()
	if err != nil {
		fatal(err)
	}
	log := logrus.New()
	log.SetOutput(os.Stderr)
	if lvl, err := logrus.ParseLevel(cfg.LogLevel); err == nil {
		log.SetLevel(lvl)
	}
	a := app.New(cfg, log, os.Stdout)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	cmd, args := os.Args[1], os.Args[2:]
	switch cmd {
	case "run":
		fs := flag.NewFlagSet("run", flag.ExitOnError)
		useAPI := fs.Bool("api", false, "use the official Threads API")
		noNotify := fs.Bool("no-notify", false, "do not send to Telegram")
		_ = fs.Parse(args)
		err = a.Run(ctx, app.RunOptions{UseAPI: *useAPI, Notify: !*noNotify, Queries: fs.Args()})
	case "login":
		err = a.Login(ctx)
	case "explore":
		if len(args) < 1 {
			fatal(fmt.Errorf("explore needs a query"))
		}
		err = a.Explore(ctx, args[0])
	case "auth":
		err = a.Auth(ctx)
	case "tg":
		sub := ""
		if len(args) > 0 {
			sub = args[0]
		}
		switch sub {
		case "test":
			err = a.TelegramTest(ctx)
		case "chats":
			err = a.TelegramChats(ctx)
		default:
			fatal(fmt.Errorf("tg needs 'test' or 'chats'"))
		}
	case "export":
		fs := flag.NewFlagSet("export", flag.ExitOnError)
		n := fs.Int("n", 100, "number of leads")
		_ = fs.Parse(args)
		err = a.Export(ctx, *n)
	case "stats":
		err = a.Stats(ctx)
	case "help", "-h", "--help":
		fmt.Print(usage)
	default:
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	if err != nil {
		fatal(err)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}
