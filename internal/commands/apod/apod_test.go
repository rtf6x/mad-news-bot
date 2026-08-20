package apod

import (
	"context"
	"encoding/json"
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

func TestFetchFromNASAUsesThreeDayRangeOnce(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Query().Get("api_key") != "test-key" {
			t.Errorf("api_key: %q", r.URL.Query().Get("api_key"))
		}
		if r.URL.Query().Get("date") != "" {
			t.Errorf("unexpected date param: %q", r.URL.Query().Get("date"))
		}
		start := r.URL.Query().Get("start_date")
		end := r.URL.Query().Get("end_date")
		if start == "" || end == "" {
			t.Errorf("missing range: start=%q end=%q", start, end)
		}
		loc, err := time.LoadLocation("America/New_York")
		if err != nil {
			t.Fatal(err)
		}
		today := time.Now().In(loc)
		wantEnd := today.Format("2006-01-02")
		wantStart := today.AddDate(0, 0, -2).Format("2006-01-02")
		if start != wantStart || end != wantEnd {
			t.Errorf("range: got %s..%s want %s..%s", start, end, wantStart, wantEnd)
		}
		_ = json.NewEncoder(w).Encode([]APOD{
			{Date: wantStart, Title: "Old", URL: "https://example/old.jpg", MediaType: "image"},
			{Date: wantEnd, Title: "New", URL: "https://example/new.jpg", MediaType: "image"},
		})
	}))
	t.Cleanup(srv.Close)

	prevURL, prevClient := nasaAPODURL, httpClient
	nasaAPODURL = srv.URL
	httpClient = srv.Client()
	t.Cleanup(func() {
		nasaAPODURL = prevURL
		httpClient = prevClient
	})

	item, err := fetchFromNASA(context.Background(), "test-key")
	if err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("calls: got %d want 1", calls)
	}
	if item.Title != "New" {
		t.Fatalf("expected latest day, got %+v", item)
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

func TestParseNASAErrorNonJSON(t *testing.T) {
	err := parseNASAError(502, []byte("upstream connect error or disconnect/reset before headers"))
	if err == nil || err.Error() == "" {
		t.Fatal("expected error")
	}
	if got := err.Error(); got == "" || len(got) < 20 {
		t.Fatalf("unexpected error: %q", got)
	}
}

func TestParseNASAErrorAPIJSON(t *testing.T) {
	body := []byte(`{"error":{"code":"OVER_RATE_LIMIT","message":"You have exceeded your rate limit."}}`)
	err := parseNASAError(429, body)
	if err == nil || err.Error() == "" {
		t.Fatal("expected error")
	}
}

func TestParseNASAErrorLegacyJSON(t *testing.T) {
	body := []byte(`{"code":500,"msg":"Internal Service Error","service_version":"v1"}`)
	err := parseNASAError(500, body)
	if err == nil {
		t.Fatal("expected error")
	}
	if got := err.Error(); !strings.Contains(got, "Internal Service Error") {
		t.Fatalf("unexpected error: %q", got)
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
