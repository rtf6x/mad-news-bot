package telegram

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"mad-news-bot/internal/commands/apod"
)

func TestDeliverAPOD_sendsPhotoWhenFramePresent(t *testing.T) {
	var gotPath, gotPhoto, gotCaption, gotText, gotVideo string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		gotPath = r.URL.Path
		gotPhoto = r.FormValue("photo")
		gotCaption = r.FormValue("caption")
		gotText = r.FormValue("text")
		gotVideo = r.FormValue("video")
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	r := &Router{tg: &Client{token: "test-token", http: srv.Client(), apiBase: srv.URL}}
	r.deliverAPOD(99, apod.Result{
		Photo:     "https://example.test/frame.png",
		Message:   "Watch: https://apod.nasa.gov/clip.mp4",
		MediaType: "video",
	})
	if gotPath != "/bottest-token/sendPhoto" {
		t.Fatalf("path: %q", gotPath)
	}
	if gotPhoto != "https://example.test/frame.png" {
		t.Fatalf("photo: %q", gotPhoto)
	}
	if gotCaption != "Watch: https://apod.nasa.gov/clip.mp4" {
		t.Fatalf("caption: %q", gotCaption)
	}
	if gotVideo != "" || gotText != "" {
		t.Fatalf("unexpected video=%q text=%q", gotVideo, gotText)
	}
}

func TestDeliverAPOD_sendsMessageWhenNoPhoto(t *testing.T) {
	var gotPath, gotText string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		gotPath = r.URL.Path
		gotText = r.FormValue("text")
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	r := &Router{tg: &Client{token: "test-token", http: srv.Client(), apiBase: srv.URL}}
	r.deliverAPOD(99, apod.Result{
		Message:   "Watch: https://apod.nasa.gov/clip.mp4",
		MediaType: "video",
	})
	if gotPath != "/bottest-token/sendMessage" {
		t.Fatalf("path: %q", gotPath)
	}
	if gotText != "Watch: https://apod.nasa.gov/clip.mp4" {
		t.Fatalf("text: %q", gotText)
	}
}

func TestDeliverAPOD_fallsBackToMessageIfSendPhotoFails(t *testing.T) {
	var calls []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.URL.Path)
		if r.URL.Path == "/bottest-token/sendPhoto" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	r := &Router{tg: &Client{token: "test-token", http: srv.Client(), apiBase: srv.URL}}
	r.deliverAPOD(99, apod.Result{
		Photo:     "https://example.test/frame.png",
		Message:   "Watch: https://apod.nasa.gov/clip.mp4",
		MediaType: "video",
	})
	if len(calls) != 2 || calls[0] != "/bottest-token/sendPhoto" || calls[1] != "/bottest-token/sendMessage" {
		t.Fatalf("calls: %v", calls)
	}
}
