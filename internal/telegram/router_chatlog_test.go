package telegram

import (
	"encoding/json"
	"testing"

	"mad-news-bot/internal/chatlog"
)

func TestChatLogEntry_fromTelegramPayload(t *testing.T) {
	body := []byte(`{
		"message": {
			"text": "Нет ))",
			"chat": {
				"id": -1001570086074,
				"title": "Rootfox Family",
				"type": "supergroup"
			},
			"from": {
				"id": 98526006,
				"username": "vova",
				"first_name": "Vova",
				"last_name": "Example"
			}
		}
	}`)

	var update Update
	if err := json.Unmarshal(body, &update); err != nil {
		t.Fatal(err)
	}

	entry := chatLogEntry(update.Message)
	if entry.ChatTitle != "Rootfox Family" {
		t.Fatalf("chat title: %q", entry.ChatTitle)
	}
	if entry.SenderUsername != "vova" {
		t.Fatalf("sender username: %q", entry.SenderUsername)
	}

	line := chatlog.FormatLine(entry)
	want := "[Rootfox Family | -1001570086074] @vova (98526006): Нет ))"
	if line != want {
		t.Fatalf("got %q, want %q", line, want)
	}
}
