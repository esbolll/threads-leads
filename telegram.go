package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"net/url"
	"strings"
	"time"
)

// Telegram notifications: one message per new lead to a channel/chat where the bot is admin.
// Config: TELEGRAM_BOT_TOKEN and TELEGRAM_CHAT_ID in env or .env.

type telegramClient struct {
	token  string
	chatID string
}

func newTelegramFromEnv() *telegramClient {
	tok := envOrDotenv("TELEGRAM_BOT_TOKEN")
	chat := envOrDotenv("TELEGRAM_CHAT_ID")
	if tok == "" || chat == "" {
		return nil
	}
	return &telegramClient{token: tok, chatID: chat}
}

func (t *telegramClient) api(method string, params url.Values) (json.RawMessage, error) {
	resp, err := httpClient.PostForm("https://api.telegram.org/bot"+t.token+"/"+method, params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	var r struct {
		OK          bool            `json:"ok"`
		Description string          `json:"description"`
		Result      json.RawMessage `json:"result"`
		Parameters  struct {
			RetryAfter int `json:"retry_after"`
		} `json:"parameters"`
	}
	if err := json.Unmarshal(body, &r); err != nil {
		return nil, fmt.Errorf("telegram: bad response (%d): %s", resp.StatusCode, truncate(string(body), 200))
	}
	if !r.OK {
		if r.Parameters.RetryAfter > 0 {
			time.Sleep(time.Duration(r.Parameters.RetryAfter+1) * time.Second)
			return t.api(method, params)
		}
		return nil, errors.New("telegram: " + r.Description)
	}
	return r.Result, nil
}

func (t *telegramClient) SendText(text string) error {
	_, err := t.api("sendMessage", url.Values{
		"chat_id":    {t.chatID},
		"text":       {text},
		"parse_mode": {"HTML"},
	})
	return err
}

func (t *telegramClient) SendLead(l Lead) error {
	text := strings.TrimSpace(l.Text)
	if r := []rune(text); len(r) > 400 {
		text = string(r[:400]) + "…"
	}
	date := l.Timestamp
	if ts, err := time.Parse(time.RFC3339, l.Timestamp); err == nil {
		date = ts.Local().Format("02.01.2006 15:04")
	} else if len(date) > 10 {
		date = date[:10]
	}
	var b strings.Builder
	fmt.Fprintf(&b, "🔎 <b>%s</b> · %s\n", html.EscapeString(l.Category), html.EscapeString(strings.Join(l.MatchedTech, ", ")))
	fmt.Fprintf(&b, "<a href=\"https://www.threads.com/@%s\">@%s</a>", url.PathEscape(l.Username), html.EscapeString(l.Username))
	if date != "" {
		fmt.Fprintf(&b, " · %s", html.EscapeString(date))
	}
	b.WriteString("\n\n")
	b.WriteString(html.EscapeString(text))
	b.WriteString("\n\n")
	b.WriteString(html.EscapeString(l.Permalink))
	fmt.Fprintf(&b, "\n<i>query: %s</i>", html.EscapeString(l.SearchQuery))
	return t.SendText(b.String())
}

// ListChats prints chats the bot has recently seen (via getUpdates), to find a channel id.
func (t *telegramClient) ListChats() error {
	res, err := t.api("getUpdates", url.Values{"allowed_updates": {`["message","channel_post","my_chat_member"]`}})
	if err != nil {
		return err
	}
	var updates []struct {
		Message     *struct{ Chat tgChat } `json:"message"`
		ChannelPost *struct{ Chat tgChat } `json:"channel_post"`
		MyChat      *struct{ Chat tgChat } `json:"my_chat_member"`
	}
	if err := json.Unmarshal(res, &updates); err != nil {
		return err
	}
	seen := map[int64]bool{}
	for _, u := range updates {
		for _, c := range []*struct{ Chat tgChat }{u.Message, u.ChannelPost, u.MyChat} {
			if c == nil || seen[c.Chat.ID] {
				continue
			}
			seen[c.Chat.ID] = true
			fmt.Printf("chat_id=%d type=%s title=%q username=%q\n", c.Chat.ID, c.Chat.Type, c.Chat.Title, c.Chat.Username)
		}
	}
	if len(seen) == 0 {
		fmt.Println("No chats seen yet. Add the bot as admin to the channel and post any message there, then rerun.")
	}
	return nil
}

type tgChat struct {
	ID       int64  `json:"id"`
	Type     string `json:"type"`
	Title    string `json:"title"`
	Username string `json:"username"`
}
