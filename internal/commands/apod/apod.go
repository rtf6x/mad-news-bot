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
	cacheKey = "nasa-apod"
	cacheTTL = 48 * time.Hour
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
	Photo   string `json:"photo"`
	Message string `json:"message"`
}

func FetchAndStore(ctx context.Context, redis *cache.Redis, apiKey string) error {
	item, err := fetchFromNASA(ctx, apiKey)
	if err != nil {
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
	url := fmt.Sprintf("https://api.nasa.gov/planetary/apod?api_key=%s", apiKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return APOD{}, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return APOD{}, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return APOD{}, err
	}
	var item APOD
	if err := json.Unmarshal(body, &item); err != nil {
		return APOD{}, err
	}
	if item.URL == "" {
		return APOD{}, fmt.Errorf("nasa apod: empty response")
	}
	return item, nil
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
	message := fmt.Sprintf("%s (%s)\n\n%s\n\nHi-Res: %s\n(c) %s\n",
		item.Title, item.Date, firstSentence, item.HDURL, copyright)
	return Result{Photo: item.URL, Message: message}
}
