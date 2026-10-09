package telegram

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/esbolll/threads-leads/internal/lead/model"
)

// Client posts messages to a chat/channel where the bot is an admin.
type Client struct {
	token  string
	chatID string
	http   *http.Client
}

func New(token, chatID string) *Client {
	return &Client{token: token, chatID: chatID, http: &http.Client{Timeout: 30 * time.Second}}
}

func (c *Client) api(ctx context.Context, method string, params url.Values) (json.RawMessage, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.telegram.org/bot"+c.token+"/"+method, strings.NewReader(params.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := c.http.Do(req)
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
		return nil, fmt.Errorf("telegram: bad response (%d)", resp.StatusCode)
	}
	if !r.OK {
		if r.Parameters.RetryAfter > 0 {
			select {
			case <-time.After(time.Duration(r.Parameters.RetryAfter+1) * time.Second):
				return c.api(ctx, method, params)
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
		return nil, errors.New("telegram: " + r.Description)
	}
	return r.Result, nil
}

func (c *Client) SendText(ctx context.Context, text string) error {
	_, err := c.api(ctx, "sendMessage", url.Values{
		"chat_id":    {c.chatID},
		"text":       {text},
		"parse_mode": {"HTML"},
	})
	return err
}

func (c *Client) SendLead(ctx context.Context, l model.Lead) error {
	return c.SendText(ctx, Format(l))
}

// Format renders a lead as a Telegram HTML message.
func Format(l model.Lead) string {
	text := strings.TrimSpace(l.Text)
	if r := []rune(text); len(r) > 400 {
		text = string(r[:400]) + "…"
	}
	date := l.PostedAt
	if ts, err := time.Parse(time.RFC3339, l.PostedAt); err == nil {
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
	return b.String()
}

type Chat struct {
	ID       int64  `json:"id"`
	Type     string `json:"type"`
	Title    string `json:"title"`
	Username string `json:"username"`
}

// RecentChats lists chats seen in getUpdates; used once to find a channel id.
func (c *Client) RecentChats(ctx context.Context) ([]Chat, error) {
	res, err := c.api(ctx, "getUpdates", url.Values{"allowed_updates": {`["message","channel_post","my_chat_member"]`}})
	if err != nil {
		return nil, err
	}
	var updates []struct {
		Message     *struct{ Chat Chat } `json:"message"`
		ChannelPost *struct{ Chat Chat } `json:"channel_post"`
		MyChat      *struct{ Chat Chat } `json:"my_chat_member"`
	}
	if err := json.Unmarshal(res, &updates); err != nil {
		return nil, err
	}
	seen := map[int64]bool{}
	var out []Chat
	for _, u := range updates {
		for _, m := range []*struct{ Chat Chat }{u.Message, u.ChannelPost, u.MyChat} {
			if m == nil || seen[m.Chat.ID] {
				continue
			}
			seen[m.Chat.ID] = true
			out = append(out, m.Chat)
		}
	}
	return out, nil
}
