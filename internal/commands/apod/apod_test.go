package apod

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestCacheTTLIsTenDays(t *testing.T) {
	want := 10 * 24 * time.Hour
	if cacheTTL != want {
		t.Fatalf("cacheTTL: got %v, want %v", cacheTTL, want)
	}
}

func TestPickLatestAPOD(t *testing.T) {
	got, err := pickLatestAPOD([]APOD{
		{Date: "2026-08-08", Title: "A", URL: "https://a.example/a.jpg"},
		{Date: "2026-08-10", Title: "C", URL: "https://c.example/c.jpg"},
		{Date: "2026-08-09", Title: "B", URL: "https://b.example/b.jpg"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Date != "2026-08-10" || got.Title != "C" {
		t.Fatalf("got %+v", got)
	}
}

func TestPickLatestAPODEmpty(t *testing.T) {
	if _, err := pickLatestAPOD(nil); err == nil {
		t.Fatal("expected error")
	}
}

const imageItem = `<item>
<title>Harvest Moon &amp; Etna</title>
<pubDate>Thu, 01 Oct 2026 04:05:00 +0000</pubDate>
<apod:hdurl>https://assets.science.nasa.gov/dynamicimage/assets/science/cds/apod/apod/2026/october/moon.jpg?w=4455&#038;h=5592&#038;fit=clip&#038;crop=faces%2Cfocalpoint</apod:hdurl>
<apod:copyright><![CDATA[<strong>Image Credit:</strong> <a href="http://x">Jane Doe</a>]]></apod:copyright>
<apod:explanation><![CDATA[<strong>Explanation:</strong> A <a href="http://x">big</a> moon. More.]]></apod:explanation>
<content:encoded><![CDATA[<html><body><IMG src="x.jpg"></body></html>]]></content:encoded>
</item>`

const videoItem = `<item>
<title>Wink</title>
<pubDate>Fri, 02 Oct 2026 04:05:00 +0000</pubDate>
<apod:hdurl>https://assets.science.nasa.gov/dynamicimage/assets/science/cds/apod/apod/2026/october/wink.jpg?w=1920&#038;h=1088</apod:hdurl>
<apod:copyright><![CDATA[Nobody]]></apod:copyright>
<apod:explanation><![CDATA[<strong>Explanation:</strong> A star winks. More.]]></apod:explanation>
<content:encoded><![CDATA[<video controls>
<source src="https://assets.science.nasa.gov/content/dam/science/cds/apod/apod/2026/october/wink.mp4" type="video/mp4">
</video>]]></content:encoded>
</item>`

func serveFeed(t *testing.T, items ...string) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `<?xml version="1.0"?><rss version="2.0" xmlns:content="http://purl.org/rss/1.0/modules/content/" xmlns:apod="https://science.nasa.gov/apod/"><channel>%s</channel></rss>`, strings.Join(items, ""))
	}))
	t.Cleanup(srv.Close)
	prevURL, prevClient := feedURL, httpClient
	feedURL, httpClient = srv.URL, srv.Client()
	t.Cleanup(func() { feedURL, httpClient = prevURL, prevClient })
}

func TestFetchFromNASAParsesImageItem(t *testing.T) {
	serveFeed(t, imageItem)
	got, err := fetchFromNASA(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "Harvest Moon & Etna" || got.Date != "2026-10-01" || got.MediaType != "image" {
		t.Fatalf("got %+v", got)
	}
	if got.Explanation != "A big moon. More." {
		t.Fatalf("explanation: %q", got.Explanation)
	}
	if got.Copyright != "Image Credit: Jane Doe" {
		t.Fatalf("copyright: %q", got.Copyright)
	}
	if !strings.Contains(got.HDURL, "w=4455") {
		t.Fatalf("hdurl should stay original: %q", got.HDURL)
	}
	if !strings.Contains(got.URL, "w=1280") || !strings.Contains(got.URL, "h=1280") || strings.Contains(got.URL, "crop") {
		t.Fatalf("photo url should be a 1280 rendition: %q", got.URL)
	}
}

func TestFetchFromNASAPicksNewestAndDetectsVideo(t *testing.T) {
	serveFeed(t, imageItem, videoItem)
	got, err := fetchFromNASA(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "Wink" || got.MediaType != "video" {
		t.Fatalf("got %+v", got)
	}
	if got.URL != "https://assets.science.nasa.gov/content/dam/science/cds/apod/apod/2026/october/wink.mp4" {
		t.Fatalf("video url: %q", got.URL)
	}
}

func TestFetchFromNASAEmptyFeed(t *testing.T) {
	serveFeed(t)
	if _, err := fetchFromNASA(context.Background()); err == nil {
		t.Fatal("expected error")
	}
}

func TestFetchFromNASABadStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusBadGateway)
	}))
	t.Cleanup(srv.Close)
	prevURL, prevClient := feedURL, httpClient
	feedURL, httpClient = srv.URL, srv.Client()
	t.Cleanup(func() { feedURL, httpClient = prevURL, prevClient })
	if _, err := fetchFromNASA(context.Background()); err == nil || !strings.Contains(err.Error(), "502") {
		t.Fatalf("expected status error, got %v", err)
	}
}

func TestFormatImageAPOD(t *testing.T) {
	res := format(APOD{
		MediaType:   "image",
		Title:       "Red Sun",
		Date:        "2026-07-30",
		Explanation: "Smoke made the Sun red. More text follows.",
		URL:         "https://apod.nasa.gov/apod/image/2607/red_sun_1024.jpg",
		HDURL:       "https://apod.nasa.gov/apod/image/2607/red_sun.jpg",
	})
	if res.MediaType != "image" {
		t.Fatalf("media type: got %q", res.MediaType)
	}
	if res.Photo != "https://apod.nasa.gov/apod/image/2607/red_sun_1024.jpg" {
		t.Fatalf("photo: got %q", res.Photo)
	}
	if !strings.Contains(res.Message, "Red Sun") || !strings.Contains(res.Message, "Hi-Res:") {
		t.Fatalf("message: %q", res.Message)
	}
}

func TestFormatVideoAPOD(t *testing.T) {
	res := format(APOD{
		MediaType:   "video",
		Title:       "Psyche Assist",
		Date:        "2026-07-29",
		Explanation: "Gravity assist from Mars. Extra details.",
		URL:         "https://www.youtube.com/embed/6_cH5-daLjg",
	})
	if res.MediaType != "video" {
		t.Fatalf("media type: got %q", res.MediaType)
	}
	if res.Photo != "" {
		t.Fatalf("photo should be empty for video, got %q", res.Photo)
	}
	if !strings.Contains(res.Message, "Watch:") || !strings.Contains(res.Message, "https://www.youtube.com/embed/6_cH5-daLjg") {
		t.Fatalf("message: %q", res.Message)
	}
}

func TestFormatHostedVideoAPODKeepsWatchURL(t *testing.T) {
	low := "https://apod.nasa.gov/apod/image/2608/perseids_eclipse_mystery.mp4"
	res := format(APOD{
		MediaType:   "video",
		Title:       "Maybe Meteor",
		Date:        "2026-08-19",
		Explanation: "Whatdunit? Extra details.",
		URL:         low,
	})
	if res.MediaType != "video" {
		t.Fatalf("media type: got %q", res.MediaType)
	}
	if res.Photo != "" {
		t.Fatalf("photo should be empty for video, got %q", res.Photo)
	}
	if !strings.Contains(res.Message, "Watch: "+low) {
		t.Fatalf("message: %q", res.Message)
	}
}

func TestIsVideoAPOD(t *testing.T) {
	if !isVideoAPOD(APOD{MediaType: "video", URL: "https://youtu.be/x"}) {
		t.Fatal("expected video by media_type")
	}
	if isVideoAPOD(APOD{MediaType: "image", URL: "https://apod.nasa.gov/x.jpg"}) {
		t.Fatal("expected image")
	}
	if !isVideoAPOD(APOD{URL: "https://www.youtube.com/embed/x"}) {
		t.Fatal("expected video by youtube url in legacy cache")
	}
}

func TestFormatVideoIsTextOnly(t *testing.T) {
	res := format(APOD{
		MediaType:   "video",
		Title:       "Maybe Meteor",
		Date:        "2026-08-19",
		Explanation: "Whatdunit? Extra.",
		URL:         "https://apod.nasa.gov/clip.mp4",
	})
	if res.Photo != "" {
		t.Fatalf("photo should be empty, got %q", res.Photo)
	}
	if !strings.Contains(res.Message, "Watch: https://apod.nasa.gov/clip.mp4") {
		t.Fatalf("message: %q", res.Message)
	}
}
