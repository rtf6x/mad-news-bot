package apod

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"mad-news-bot/internal/cache"
)

const (
	cacheKey     = "nasa-apod"
	cacheTTL     = 10 * 24 * time.Hour
	fetchTimeout = 45 * time.Second
)

var (
	nasaAPODURL = "https://api.nasa.gov/planetary/apod"
	httpClient  = &http.Client{Timeout: fetchTimeout}
)

type APOD struct {
	Copyright   string `json:"copyright"`
	Date        string `json:"date"`
	Explanation string `json:"explanation"`
	HDURL       string `json:"hdurl"`
	MediaType   string `json:"media_type"`
	Title       string `json:"title"`
	URL         string `json:"url"`
}

type Result struct {
	Photo     string `json:"photo"`
	Message   string `json:"message"`
	MediaType string `json:"media_type,omitempty"`
}

type nasaErrorResponse struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

type nasaLegacyErrorResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

func FetchAndStore(ctx context.Context, redis *cache.Redis, apiKey string) error {
	item, err := fetchFromNASA(ctx, apiKey)
	if err != nil {
		if _, cacheErr := GetCached(ctx, redis); cacheErr == nil {
			return fmt.Errorf("nasa fetch failed, keeping stale cache: %w", err)
		}
		return err
	}
	raw, err := json.Marshal(item)
	if err != nil {
		return err
	}
	return redis.SetEX(ctx, cacheKey, string(raw), cacheTTL)
}

func GetCached(ctx context.Context, redis *cache.Redis) (Result, error) {
	raw, err := redis.Get(ctx, cacheKey)
	if err != nil {
		return Result{}, fmt.Errorf("apod cache miss: %w", err)
	}
	var item APOD
	if err := json.Unmarshal([]byte(raw), &item); err != nil {
		return Result{}, err
	}
	return format(item), nil
}

func Get(ctx context.Context, redis *cache.Redis, apiKey string) (Result, error) {
	if res, err := GetCached(ctx, redis); err == nil {
		return res, nil
	}
	if apiKey == "" {
		return Result{}, fmt.Errorf("nasa apod cache empty and NASA_APOD_KEY is not set")
	}
	item, err := fetchFromNASA(ctx, apiKey)
	if err != nil {
		return Result{}, err
	}
	raw, _ := json.Marshal(item)
	_ = redis.SetEX(ctx, cacheKey, string(raw), cacheTTL)
	return format(item), nil
}

func fetchFromNASA(ctx context.Context, apiKey string) (APOD, error) {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		loc = time.UTC
	}
	today := time.Now().In(loc)
	end := today.Format("2006-01-02")
	start := today.AddDate(0, 0, -2).Format("2006-01-02")
	return fetchFromNASARange(ctx, apiKey, start, end)
}

func fetchFromNASARange(ctx context.Context, apiKey, startDate, endDate string) (APOD, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, nasaAPODURL, nil)
	if err != nil {
		return APOD{}, err
	}
	q := req.URL.Query()
	q.Set("api_key", apiKey)
	q.Set("start_date", startDate)
	q.Set("end_date", endDate)
	req.URL.RawQuery = q.Encode()

	resp, err := httpClient.Do(req)
	if err != nil {
		return APOD{}, fmt.Errorf("nasa apod request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return APOD{}, fmt.Errorf("nasa apod read body: %w", err)
	}

	if resp.StatusCode >= 300 {
		return APOD{}, parseNASAError(resp.StatusCode, body)
	}

	var items []APOD
	if err := json.Unmarshal(body, &items); err != nil {
		return APOD{}, parseNASAError(resp.StatusCode, body)
	}
	return pickLatestAPOD(items)
}

func pickLatestAPOD(items []APOD) (APOD, error) {
	var latest APOD
	for _, item := range items {
		if item.URL == "" {
			continue
		}
		if item.Date > latest.Date {
			latest = item
		}
	}
	if latest.URL == "" {
		return APOD{}, fmt.Errorf("nasa apod: no usable items in range")
	}
	return latest, nil
}

func parseNASAError(status int, body []byte) error {
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return fmt.Errorf("nasa apod: status %d, empty body", status)
	}

	var apiErr nasaErrorResponse
	if json.Unmarshal(body, &apiErr) == nil && apiErr.Error.Message != "" {
		return fmt.Errorf("nasa apod: status %d: %s", status, apiErr.Error.Message)
	}

	var legacyErr nasaLegacyErrorResponse
	if json.Unmarshal(body, &legacyErr) == nil && legacyErr.Msg != "" {
		return fmt.Errorf("nasa apod: status %d: %s", status, legacyErr.Msg)
	}

	snippet := trimmed
	if len(snippet) > 160 {
		snippet = snippet[:160] + "..."
	}
	return fmt.Errorf("nasa apod: status %d, non-json body: %s", status, snippet)
}

func isVideoAPOD(item APOD) bool {
	if item.MediaType == "video" {
		return true
	}
	if item.MediaType == "image" {
		return false
	}
	return strings.Contains(item.URL, "youtube.com") || strings.Contains(item.URL, "youtu.be")
}

func format(item APOD) Result {
	firstSentence := item.Explanation
	if idx := strings.Index(item.Explanation, "."); idx >= 0 {
		firstSentence = item.Explanation[:idx]
	}
	copyright := item.Copyright
	if copyright == "" {
		copyright = "NASA"
	}

	if isVideoAPOD(item) {
		message := fmt.Sprintf("%s (%s)\n\n%s\n\nWatch: %s\n(c) %s\n",
			item.Title, item.Date, firstSentence, item.URL, copyright)
		return Result{Message: message, MediaType: "video"}
	}

	message := fmt.Sprintf("%s (%s)\n\n%s\n\nHi-Res: %s\n(c) %s\n",
		item.Title, item.Date, firstSentence, item.HDURL, copyright)
	return Result{Photo: item.URL, Message: message, MediaType: "image"}
}
