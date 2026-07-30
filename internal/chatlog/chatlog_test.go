package chatlog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLog_skipsPrivateChats(t *testing.T) {
	dir := t.TempDir()
	logger, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer logger.Close()

	if err := logger.Log(Entry{ChatID: 324702279, SenderID: 324702279, Text: "/nasaapod"}); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected no files for private chat, got %d", len(entries))
	}
}

func TestLog_writesGroupChatToFileByID(t *testing.T) {
	dir := t.TempDir()
	logger, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer logger.Close()

	if err := logger.Log(Entry{
		ChatID:         -1001570086074,
		ChatTitle:      "Rootfox Family",
		SenderID:       98526006,
		SenderUsername: "vova",
		Text:           "Нет ))",
	}); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "-1001570086074.log"))
	if err != nil {
		t.Fatal(err)
	}
	line := string(data)
	if !strings.Contains(line, "[Rootfox Family | -1001570086074] @vova (98526006): Нет ))") {
		t.Fatalf("unexpected line: %q", line)
	}
}

func TestLog_appendsMultipleLines(t *testing.T) {
	dir := t.TempDir()
	logger, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer logger.Close()

	chatID := int64(-498084874)
	if err := logger.Log(Entry{ChatID: chatID, SenderID: 1, Text: "first"}); err != nil {
		t.Fatal(err)
	}
	if err := logger.Log(Entry{ChatID: chatID, SenderID: 2, Text: "second"}); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "-498084874.log"))
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d: %q", len(lines), data)
	}
}
