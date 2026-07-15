package badadvice

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

type Client struct {
	baseURL string
	http    *http.Client
}

type Advice struct {
	Advice     string `json:"advice"`
	HarmLevel  int    `json:"harm_level"`
	Disclaimer string `json:"disclaimer"`
	Mode       string `json:"mode"`
	Lang       string `json:"lang"`
}

type enqueueResponse struct {
	JobID         string `json:"job_id"`
	QueuePosition int    `json:"queue_position"`
}

type wsEvent struct {
	JobID      string  `json:"job_id"`
	Status     string  `json:"status"`
	RetryJobID string  `json:"retry_job_id"`
	Error      string  `json:"error"`
	Result     *Advice `json:"result"`
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		http:    &http.Client{Timeout: 15 * time.Second},
	}
}

func (c *Client) RequestAdvice(ctx context.Context, prompt, lang string) (string, error) {
	if lang == "" {
		lang = "ru"
	}
	jobID, err := c.enqueue(ctx, prompt, lang)
	if err != nil {
		return "", err
	}
	result, err := c.waitForJob(ctx, jobID)
	if err != nil {
		return "", err
	}
	if result == nil || result.Advice == "" {
		return "", fmt.Errorf("empty advice result")
	}
	text := result.Advice
	if result.Disclaimer != "" {
		text += "\n\n" + result.Disclaimer
	}
	return text, nil
}

func (c *Client) enqueue(ctx context.Context, prompt, lang string) (string, error) {
	body, _ := json.Marshal(map[string]string{
		"prompt": prompt,
		"lang":   lang,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/advice", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusAccepted {
		return "", fmt.Errorf("bad advice enqueue: %s: %s", resp.Status, strings.TrimSpace(string(raw)))
	}
	var data enqueueResponse
	if err := json.Unmarshal(raw, &data); err != nil {
		return "", err
	}
	if data.JobID == "" {
		return "", fmt.Errorf("bad advice enqueue: empty job_id")
	}
	return data.JobID, nil
}

func (c *Client) waitForJob(ctx context.Context, jobID string) (*Advice, error) {
	wsURL := strings.Replace(c.baseURL, "https://", "wss://", 1)
	wsURL = strings.Replace(wsURL, "http://", "ws://", 1)
	wsURL += "/ws/jobs/" + jobID

	dialer := websocket.Dialer{HandshakeTimeout: 10 * time.Second}
	conn, _, err := dialer.DialContext(ctx, wsURL, nil)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	timeout := time.NewTimer(2 * time.Minute)
	defer timeout.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timeout.C:
			return nil, fmt.Errorf("bad advice timeout")
		default:
		}

		if err := conn.SetReadDeadline(time.Now().Add(95 * time.Second)); err != nil {
			return nil, err
		}
		_, msg, err := conn.ReadMessage()
		if err != nil {
			return nil, err
		}
		var ev wsEvent
		if err := json.Unmarshal(msg, &ev); err != nil {
			continue
		}
		switch ev.Status {
		case "retrying":
			if ev.RetryJobID != "" {
				return c.waitForJob(ctx, ev.RetryJobID)
			}
		case "done":
			return ev.Result, nil
		case "failed":
			if ev.Error != "" {
				return nil, fmt.Errorf("%s", ev.Error)
			}
			return nil, fmt.Errorf("bad advice failed")
		}
	}
}
