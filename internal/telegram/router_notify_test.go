package telegram

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"mad-news-bot/internal/config"
)

func TestHandleNotify_rejectsInvalidPayload(t *testing.T) {
	router := NewRouter(config.Config{HireChatID: "123"}, NewClient("token"), nil, nil, nil, nil)

	for _, body := range []string{`{}`, `{"text":""}`, `{"text":"   "}`, `invalid`} {
		reply := router.HandleNotify([]byte(body))
		if reply.Status != "error" || reply.Code != 1 {
			t.Fatalf("body %q: got %+v, want error", body, reply)
		}
	}
}

func TestHandleNotify_sendsTextToHireChat(t *testing.T) {
	var gotChatID, gotText string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		gotChatID = r.FormValue("chat_id")
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

	reply := router.HandleNotify([]byte(`{"text":"hello from webhook"}`))
	if reply.Status != "success" || reply.Code != 0 {
		t.Fatalf("got %+v, want success", reply)
	}
	if gotChatID != "324702279" {
		t.Fatalf("chat_id = %q, want 324702279", gotChatID)
	}
	if gotText != "hello from webhook" {
		t.Fatalf("text = %q, want hello from webhook", gotText)
	}
}
