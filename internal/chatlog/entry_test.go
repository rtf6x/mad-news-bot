package chatlog

import "testing"

func TestFormatLine_withChatTitleAndUsername(t *testing.T) {
	line := formatLine(Entry{
		ChatID:         -1001570086074,
		ChatTitle:      "Rootfox Family",
		SenderID:       98526006,
		SenderUsername: "vova",
		Text:           "Нет ))",
	})
	want := "[Rootfox Family | -1001570086074] @vova (98526006): Нет ))"
	if line != want {
		t.Fatalf("got %q, want %q", line, want)
	}
}

func TestFormatLine_fallsBackToNamesAndIDs(t *testing.T) {
	line := formatLine(Entry{
		ChatID:          -498084874,
		ChatUsername:    "mad_chat",
		SenderID:        467175108,
		SenderFirstName: "Max",
		SenderLastName:  "Example",
		Text:            "спс",
	})
	want := "[@mad_chat | -498084874] Max Example (467175108): спс"
	if line != want {
		t.Fatalf("got %q, want %q", line, want)
	}
}

func TestFormatLine_senderIDOnly(t *testing.T) {
	line := formatLine(Entry{
		ChatID:   -100,
		ChatType: "supergroup",
		SenderID: 42,
		Text:     "hi",
	})
	want := "[-100 | -100] 42 (42): hi"
	if line != want {
		t.Fatalf("got %q, want %q", line, want)
	}
}
