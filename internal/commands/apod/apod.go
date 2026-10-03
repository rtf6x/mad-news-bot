package apod

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"mad-news-bot/internal/cache"
)

const (
	cacheKey     = "nasa-apod"
	cacheTTL     = 10 * 24 * time.Hour
	fetchTimeout = 45 * time.Second

	photoMaxSide = "1280"
)

var (
	feedURL    = "https://science.nasa.gov/feed/apod-basic/"
	httpClient = &http.Client{Timeout: fetchTimeout}

	tagRE    = regexp.MustCompile(`(?s)<[^>]*>`)
	spaceRE  = regexp.MustCompile(`\s+`)
	sourceRE = regexp.MustCompile(`<source[^>]+src="([^"]+)"`)
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

type feed struct {
	Items []feedItem `xml:"channel>item"`
}

type feedItem struct {
	Title       string `xml:"title"`
	PubDate     string `xml:"pubDate"`
	HDURL       string `xml:"https://science.nasa.gov/apod/ hdurl"`
	Explanation string `xml:"https://science.nasa.gov/apod/ explanation"`
	Copyright   string `xml:"https://science.nasa.gov/apod/ copyright"`
	Content     string `xml:"http://purl.org/rss/1.0/modules/content/ encoded"`
}

func FetchAndStore(ctx context.Context, redis *cache.Redis) error {
	item, err := fetchFromNASA(ctx)
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

func Get(ctx context.Context, redis *cache.Redis) (Result, error) {
	if res, err := GetCached(ctx, redis); err == nil {
		return res, nil
	}
	item, err := fetchFromNASA(ctx)
	if err != nil {
		return Result{}, err
	}
	raw, _ := json.Marshal(item)
	_ = redis.SetEX(ctx, cacheKey, string(raw), cacheTTL)
	return format(item), nil
}

func fetchFromNASA(ctx context.Context) (APOD, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, feedURL, nil)
	if err != nil {
		return APOD{}, err
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return APOD{}, fmt.Errorf("nasa apod request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return APOD{}, fmt.Errorf("nasa apod: feed status %d", resp.StatusCode)
	}

	var f feed
	if err := xml.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&f); err != nil {
		return APOD{}, fmt.Errorf("nasa apod: parse feed: %w", err)
	}

	items := make([]APOD, 0, len(f.Items))
	for _, it := range f.Items {
		item, err := it.toAPOD()
		if err != nil {
			return APOD{}, err
		}
		items = append(items, item)
	}
	return pickLatestAPOD(items)
}

func (it feedItem) toAPOD() (APOD, error) {
	published, err := time.Parse(time.RFC1123Z, strings.TrimSpace(it.PubDate))
	if err != nil {
		return APOD{}, fmt.Errorf("nasa apod: bad pubDate %q: %w", it.PubDate, err)
	}
	item := APOD{
		Title:       html.UnescapeString(strings.TrimSpace(it.Title)),
		Date:        published.UTC().Format("2006-01-02"),
		Explanation: strings.TrimPrefix(plainText(it.Explanation), "Explanation: "),
		Copyright:   plainText(it.Copyright),
		HDURL:       html.UnescapeString(strings.TrimSpace(it.HDURL)),
	}
	if m := sourceRE.FindStringSubmatch(it.Content); m != nil {
		item.MediaType = "video"
		item.URL = html.UnescapeString(m[1])
		return item, nil
	}
	item.MediaType = "image"
	item.URL = photoURL(item.HDURL)
	return item, nil
}

func plainText(markup string) string {
	text := html.UnescapeString(tagRE.ReplaceAllString(markup, ""))
	return strings.TrimSpace(spaceRE.ReplaceAllString(text, " "))
}

// photoURL asks the NASA image service for a Telegram-sized rendition; the
// feed's hdurl is the full-size original (tens of MB for some pictures).
func photoURL(hdurl string) string {
	u, err := url.Parse(hdurl)
	if err != nil || !strings.Contains(u.Path, "/dynamicimage/") {
		return hdurl
	}
	q := u.Query()
	q.Set("w", photoMaxSide)
	q.Set("h", photoMaxSide)
	q.Set("fit", "clip")
	q.Del("crop")
	u.RawQuery = q.Encode()
	return u.String()
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
		return APOD{}, fmt.Errorf("nasa apod: no usable items in feed")
	}
	return latest, nil
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
