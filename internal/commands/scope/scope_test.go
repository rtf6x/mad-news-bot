package scope

import "testing"

func TestWrapIndex(t *testing.T) {
	if got := wrapIndex(5, 10); got != 5 {
		t.Fatalf("expected 5, got %d", got)
	}
	if got := wrapIndex(12, 10); got != 2 {
		t.Fatalf("expected 2, got %d", got)
	}
	if got := wrapIndex(-1, 10); got != 0 {
		t.Fatalf("expected 0, got %d", got)
	}
}

func TestPrograscopeNotEmpty(t *testing.T) {
	msg := Prograscope(12345)
	if msg == "" || len(msg) < 20 {
		t.Fatalf("unexpected message: %q", msg)
	}
}
