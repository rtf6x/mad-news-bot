package telegram

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"mad-news-bot/internal/commands/apod"
)

func TestDeliverAPOD_sendsVideo(t *testing.T) {
	var gotPath, gotVideo, gotCaption, gotPhoto string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		gotPath = r.URL.Path
		gotVideo = r.FormValue("video")
		gotCaption = r.FormValue("caption")
		gotPhoto = r.FormValue("photo")
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	r := &Router{tg: &Client{token: "test-token", http: srv.Client(), apiBase: srv.URL}}
	r.deliverAPOD(99, apod.Result{
		Video:     "https://apod.nasa.gov/clip.mp4",
		Message:   "Maybe Meteor caption",
		MediaType: "video",
	})
	if gotPath != "/bottest-token/sendVideo" {
		t.Fatalf("path: %q", gotPath)
	}
	if gotVideo != "https://apod.nasa.gov/clip.mp4" {
		t.Fatalf("video: %q", gotVideo)
	}
	if gotCaption != "Maybe Meteor caption" {
		t.Fatalf("caption: %q", gotCaption)
	}
	if gotPhoto != "" {
		t.Fatalf("should not send photo, got %q", gotPhoto)
	}
}

func TestDeliverAPOD_fallsBackToMessageIfSendVideoFails(t *testing.T) {
	var calls []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.URL.Path)
		_ = r.ParseForm()
		if r.URL.Path == "/bottest-token/sendVideo" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	r := &Router{tg: &Client{token: "test-token", http: srv.Client(), apiBase: srv.URL}}
	r.deliverAPOD(99, apod.Result{
		Video:     "https://apod.nasa.gov/clip.mp4",
		Message:   "Watch: https://apod.nasa.gov/clip.mp4",
		MediaType: "video",
	})
	if len(calls) != 2 || calls[0] != "/bottest-token/sendVideo" || calls[1] != "/bottest-token/sendMessage" {
		t.Fatalf("calls: %v", calls)
	}
}
