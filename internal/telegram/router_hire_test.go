package telegram

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"mad-news-bot/internal/config"
)

func TestHandleHire_sendsIPToHireChat(t *testing.T) {
	var gotText string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		gotText = r.FormValue("text")
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	router := &Router{
		cfg: config.Config{HireChatID: "324702279"},
		tg: &Client{
			token:   "test-token",
			http:    srv.Client(),
			apiBase: srv.URL,
		},
	}

	body := []byte(`{"points":42,"history":[{"question":"Q","answer":"A"}]}`)
	reply := router.HandleHire(body, "203.0.113.10")
	if reply.Status != "success" || reply.Code != 0 {
		t.Fatalf("got %+v, want success", reply)
	}
	if !strings.Contains(gotText, "IP: 203.0.113.10") {
		t.Fatalf("text = %q, want IP line", gotText)
	}
}
