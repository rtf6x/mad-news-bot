package apod

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"mad-news-bot/internal/cache"
)

const (
	cacheKey     = "nasa-apod"
	cacheTTL     = 48 * time.Hour
	nasaAPODURL  = "https://api.nasa.gov/planetary/apod"
	fetchTimeout = 45 * time.Second
)

var httpClient = &http.Client{Timeout: fetchTimeout}

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

var errNoDataForDate = errors.New("nasa apod: no data for date")

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
	for daysAgo := 0; daysAgo < 4; daysAgo++ {
		date := today.AddDate(0, 0, -daysAgo).Format("2006-01-02")
		item, err := fetchFromNASAForDate(ctx, apiKey, date)
		if err == nil {
			return item, nil
		}
		if errors.Is(err, errNoDataForDate) {
			continue
		}
		return APOD{}, err
	}
	return APOD{}, fmt.Errorf("nasa apod: no data for recent dates")
}

func fetchFromNASAForDate(ctx context.Context, apiKey, date string) (APOD, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, nasaAPODURL, nil)
	if err != nil {
		return APOD{}, err
	}
	q := req.URL.Query()
	q.Set("api_key", apiKey)
	q.Set("date", date)
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

	if resp.StatusCode == http.StatusNotFound {
		return APOD{}, errNoDataForDate
	}
	if resp.StatusCode >= 300 {
		return APOD{}, parseNASAError(resp.StatusCode, body)
	}

	var item APOD
	if err := json.Unmarshal(body, &item); err != nil {
		return APOD{}, parseNASAError(resp.StatusCode, body)
	}
	if item.URL == "" {
		return APOD{}, fmt.Errorf("nasa apod: empty url in response")
	}
	return item, nil
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
