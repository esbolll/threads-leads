package config

import (
	"fmt"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

type Config struct {
	DBPath    string `env:"DB_PATH" envDefault:"data/leads.db"`
	OutputDir string `env:"OUTPUT_DIR" envDefault:"output"`
	LogLevel  string `env:"LOG_LEVEL" envDefault:"info"`

	Threads  Threads
	Browser  Browser
	Telegram Telegram
}

// Threads holds official API credentials. Token comes from `auth` (OAuth) and is written back to .env.
type Threads struct {
	Token       string `env:"THREADS_TOKEN"`
	AppID       string `env:"THREADS_APP_ID"`
	AppSecret   string `env:"THREADS_APP_SECRET"`
	SearchLimit int    `env:"THREADS_SEARCH_LIMIT" envDefault:"50"`
	// Redirect URI registered in the Threads use case settings.
	RedirectURI string `env:"THREADS_REDIRECT_URI" envDefault:"https://localhost:8443/callback"`
}

// Browser is the go-rod source: Google Chrome with a persistent profile.
type Browser struct {
	Chrome          string `env:"THREADS_CHROME"` // empty = system Chrome via launcher.LookPath
	ProfileDir      string `env:"BROWSER_PROFILE_DIR" envDefault:"threads_profile"`
	Headless        bool   `env:"BROWSER_HEADLESS" envDefault:"false"`
	ScrollsPerQuery int    `env:"BROWSER_SCROLLS" envDefault:"4"`
}

type Telegram struct {
	BotToken string `env:"TELEGRAM_BOT_TOKEN"`
	ChatID   string `env:"TELEGRAM_CHAT_ID"`
}

func (t Telegram) Enabled() bool { return t.BotToken != "" && t.ChatID != "" }

// Load reads .env (if present) and the environment.
func Load() (*Config, error) {
	_ = godotenv.Load()
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}
	return cfg, nil
}
