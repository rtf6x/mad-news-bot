package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWithCORS_preflight(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/webhooks/hire", func(w http.ResponseWriter, r *http.Request) {
		WriteJSON(w, http.StatusOK, map[string]string{"status": "success"})
	})

	srv := httptest.NewServer(WithCORS(mux))
	t.Cleanup(srv.Close)

	req, err := http.NewRequest(http.MethodOptions, srv.URL+"/api/webhooks/hire", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Origin", "https://hire.example")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "content-type")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusNoContent)
	}
	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want *", got)
	}
	if got := resp.Header.Get("Access-Control-Allow-Methods"); got != "GET, POST, OPTIONS" {
		t.Fatalf("Access-Control-Allow-Methods = %q", got)
	}
	if got := resp.Header.Get("Access-Control-Allow-Headers"); got != "content-type" {
		t.Fatalf("Access-Control-Allow-Headers = %q, want content-type", got)
	}
}

func TestWithCORS_postIncludesAllowOrigin(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/webhooks/hire", func(w http.ResponseWriter, r *http.Request) {
		WriteJSON(w, http.StatusOK, map[string]string{"status": "success"})
	})

	srv := httptest.NewServer(WithCORS(mux))
	t.Cleanup(srv.Close)

	req, err := http.NewRequest(http.MethodPost, srv.URL+"/api/webhooks/hire", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Origin", "https://hire.example")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want *", got)
	}
}
