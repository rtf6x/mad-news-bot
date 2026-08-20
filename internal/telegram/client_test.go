package telegram

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSendVideo_postsVideoURL(t *testing.T) {
	var gotPath, gotChatID, gotVideo, gotCaption string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		gotPath = r.URL.Path
		gotChatID = r.FormValue("chat_id")
		gotVideo = r.FormValue("video")
		gotCaption = r.FormValue("caption")
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	c := &Client{token: "test-token", http: srv.Client(), apiBase: srv.URL}
	err := c.SendVideo(42, "https://apod.nasa.gov/clip.mp4", "caption text")
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != "/bottest-token/sendVideo" {
		t.Fatalf("path: %q", gotPath)
	}
	if gotChatID != "42" {
		t.Fatalf("chat_id: %q", gotChatID)
	}
	if gotVideo != "https://apod.nasa.gov/clip.mp4" {
		t.Fatalf("video: %q", gotVideo)
	}
	if gotCaption != "caption text" {
		t.Fatalf("caption: %q", gotCaption)
	}
}
