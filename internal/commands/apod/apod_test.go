package apod

import (
	"strings"
	"testing"
)

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
