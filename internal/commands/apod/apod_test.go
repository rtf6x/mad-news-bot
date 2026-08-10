package apod

import (
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
