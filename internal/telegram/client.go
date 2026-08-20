package telegram

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type Client struct {
	token   string
	http    *http.Client
	apiBase string
}

func NewClient(token string) *Client {
	return &Client{
		token: token,
		http:  &http.Client{},
	}
}

func (c *Client) SendMessage(chatID int64, text string) error {
	if c.token == "" {
		return fmt.Errorf("bot token is empty")
	}
	endpoint := fmt.Sprintf("%s/bot%s/sendMessage", c.apiBaseOrDefault(), c.token)
	resp, err := c.http.PostForm(endpoint, url.Values{
		"chat_id": {fmt.Sprintf("%d", chatID)},
		"text":    {text},
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("telegram sendMessage: %s", resp.Status)
	}
	return nil
}

func (c *Client) SendPhoto(chatID int64, photoURL, caption string) error {
	if c.token == "" {
		return fmt.Errorf("bot token is empty")
	}
	endpoint := fmt.Sprintf("%s/bot%s/sendPhoto", c.apiBaseOrDefault(), c.token)
	resp, err := c.http.PostForm(endpoint, url.Values{
		"chat_id": {fmt.Sprintf("%d", chatID)},
		"photo":   {photoURL},
		"caption": {caption},
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("telegram sendPhoto: %s", resp.Status)
	}
	return nil
}

func (c *Client) apiBaseOrDefault() string {
	if c.apiBase != "" {
		return c.apiBase
	}
	return "https://api.telegram.org"
}

func NormalizeCommand(text, botUsername string) string {
	text = strings.TrimSpace(text)
	if botUsername == "" {
		return text
	}
	suffix := "@" + botUsername
	if idx := strings.Index(text, " "); idx > 0 {
		cmd := text[:idx]
		args := strings.TrimSpace(text[idx+1:])
		if strings.HasSuffix(cmd, suffix) {
			cmd = strings.TrimSuffix(cmd, suffix)
		}
		if args != "" {
			return cmd + " " + args
		}
		return cmd
	}
	return strings.TrimSuffix(text, suffix)
}

func CommandName(text string) string {
	text = strings.TrimSpace(text)
	if idx := strings.Index(text, " "); idx > 0 {
		return text[:idx]
	}
	return text
}

func CommandArgs(text string) string {
	text = strings.TrimSpace(text)
	if idx := strings.Index(text, " "); idx > 0 {
		return strings.TrimSpace(text[idx+1:])
	}
	return ""
}
