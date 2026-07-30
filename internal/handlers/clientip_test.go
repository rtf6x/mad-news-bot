package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientIP_fromXForwardedFor(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/api/webhooks/hire", nil)
	r.Header.Set("X-Forwarded-For", "203.0.113.10, 10.0.0.1")
	r.RemoteAddr = "127.0.0.1:12345"
	if got := ClientIP(r); got != "203.0.113.10" {
		t.Fatalf("ClientIP() = %q, want 203.0.113.10", got)
	}
}

func TestClientIP_fromXRealIP(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/api/webhooks/hire", nil)
	r.Header.Set("X-Real-IP", "198.51.100.5")
	r.RemoteAddr = "127.0.0.1:12345"
	if got := ClientIP(r); got != "198.51.100.5" {
		t.Fatalf("ClientIP() = %q, want 198.51.100.5", got)
	}
}

func TestClientIP_fromRemoteAddr(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/api/webhooks/hire", nil)
	r.RemoteAddr = "192.0.2.1:54321"
	if got := ClientIP(r); got != "192.0.2.1" {
		t.Fatalf("ClientIP() = %q, want 192.0.2.1", got)
	}
}
