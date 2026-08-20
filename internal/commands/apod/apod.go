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
	Copyright    string `json:"copyright"`
	Date         string `json:"date"`
	Explanation  string `json:"explanation"`
	HDURL        string `json:"hdurl"`
	MediaType    string `json:"media_type"`
	ThumbnailURL string `json:"thumbnail_url"`
	Title        string `json:"title"`
	URL          string `json:"url"`
}

type Result struct {
	Photo     string `json:"photo"`
	Video     string `json:"video,omitempty"`
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
	return resultFrom(ctx, item), nil
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
	return resultFrom(ctx, item), nil
}

func resultFrom(ctx context.Context, item APOD) Result {
	res := format(item)
	if res.MediaType != "video" {
		return res
	}
	res.Video = ""
	res.Photo = ""
	for _, frameURL := range frameURLsFor(item) {
		if isTelegramImage(ctx, frameURL) {
			res.Photo = frameURL
			break
		}
	}
	return res
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
	q.Set("thumbs", "true")
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

func isHostedVideoFile(rawURL string) bool {
	path := strings.ToLower(rawURL)
	if q := strings.Index(path, "?"); q >= 0 {
		path = path[:q]
	}
	if strings.Contains(path, "youtube.com") || strings.Contains(path, "youtu.be") || strings.Contains(path, "vimeo.com") {
		return false
	}
	return strings.HasSuffix(path, ".mp4") || strings.HasSuffix(path, ".webm") || strings.HasSuffix(path, ".mov")
}

var frameURLsFor = candidateFrameURLs

func candidateFrameURLs(item APOD) []string {
	var out []string
	if item.ThumbnailURL != "" {
		out = append(out, item.ThumbnailURL)
	}
	if id := youtubeVideoID(item.URL); id != "" {
		out = append(out, "https://img.youtube.com/vi/"+id+"/hqdefault.jpg")
		return out
	}
	if !isHostedVideoFile(item.URL) {
		return out
	}
	day, err := time.Parse("2006-01-02", item.Date)
	if err != nil {
		return out
	}
	base := videoBaseName(item.URL)
	if base == "" {
		return out
	}
	root := fmt.Sprintf(
		"https://assets.science.nasa.gov/content/dam/science/cds/apod/apod/%d/%s/%s_frame.png",
		day.Year(), strings.ToLower(day.Month().String()), base,
	)
	out = append(out, root+"/jcr:content/renditions/cq5dam.web.1280.1280.png", root)
	return out
}

func videoBaseName(rawURL string) string {
	path := rawURL
	if q := strings.Index(path, "?"); q >= 0 {
		path = path[:q]
	}
	slash := strings.LastIndex(path, "/")
	if slash >= 0 {
		path = path[slash+1:]
	}
	if dot := strings.LastIndex(path, "."); dot >= 0 {
		path = path[:dot]
	}
	return path
}

func youtubeVideoID(rawURL string) string {
	u := rawURL
	switch {
	case strings.Contains(u, "youtube.com/embed/"):
		u = u[strings.Index(u, "/embed/")+len("/embed/"):]
	case strings.Contains(u, "youtube.com/watch"):
		idx := strings.Index(u, "v=")
		if idx < 0 {
			return ""
		}
		u = u[idx+2:]
	case strings.Contains(u, "youtu.be/"):
		u = u[strings.Index(u, "youtu.be/")+len("youtu.be/"):]
	default:
		return ""
	}
	if i := strings.IndexAny(u, "?&/#"); i >= 0 {
		u = u[:i]
	}
	return u
}

const telegramPhotoMaxBytes = 10 * 1024 * 1024

func isTelegramImage(ctx context.Context, imageURL string) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, imageURL, nil)
	if err != nil {
		return false
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return false
	}
	if resp.ContentLength > telegramPhotoMaxBytes {
		return false
	}
	ctype := strings.ToLower(resp.Header.Get("Content-Type"))
	if i := strings.Index(ctype, ";"); i >= 0 {
		ctype = strings.TrimSpace(ctype[:i])
	}
	return ctype == "image/jpeg" || ctype == "image/jpg" || ctype == "image/png" || ctype == "image/webp" || ctype == "image/gif"
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
		message := fmt.Sprintf("%s (%s)\n\n%s\n\nWatch: %s\n",
			item.Title, item.Date, firstSentence, item.URL)
		if item.HDURL != "" && item.HDURL != item.URL {
			message += fmt.Sprintf("Hi-Res: %s\n", item.HDURL)
		}
		message += fmt.Sprintf("(c) %s\n", copyright)
		res := Result{Message: message, MediaType: "video"}
		return res
	}

	message := fmt.Sprintf("%s (%s)\n\n%s\n\nHi-Res: %s\n(c) %s\n",
		item.Title, item.Date, firstSentence, item.HDURL, copyright)
	return Result{Photo: item.URL, Message: message, MediaType: "image"}
}
