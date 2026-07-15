package madnews

import (
	"strings"
	"testing"
)

func TestGenerateNotEmpty(t *testing.T) {
	msg, err := Generate("ru")
	if err != nil {
		t.Fatal(err)
	}
	if msg == "" || len(msg) < 20 {
		t.Fatalf("unexpected: %q", msg)
	}
	if msg != strings.ToUpper(msg) {
		t.Fatalf("expected uppercase output: %q", msg)
	}
}

func TestReplaceSets(t *testing.T) {
	gen, err := newGenerator(ruDictionaryJSON)
	if err != nil {
		t.Fatal(err)
	}
	out := gen.replaceSets("[МУЖЧИНА]")
	if out == "" || out == "[МУЖЧИНА]" {
		t.Fatalf("replacement failed: %q", out)
	}
}
