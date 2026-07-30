package chatlog

import (
	"fmt"
	"strings"
)

type Entry struct {
	ChatID           int64
	ChatTitle        string
	ChatUsername     string
	ChatType         string
	SenderID         int64
	SenderUsername   string
	SenderFirstName  string
	SenderLastName   string
	Text             string
}

func formatChatLabel(e Entry) string {
	switch {
	case e.ChatTitle != "":
		return e.ChatTitle
	case e.ChatUsername != "":
		return "@" + e.ChatUsername
	default:
		return fmt.Sprintf("%d", e.ChatID)
	}
}

func formatSenderLabel(e Entry) string {
	if e.SenderUsername != "" {
		return "@" + e.SenderUsername
	}
	name := strings.TrimSpace(strings.Join([]string{e.SenderFirstName, e.SenderLastName}, " "))
	if name != "" {
		return name
	}
	if e.SenderID != 0 {
		return fmt.Sprintf("%d", e.SenderID)
	}
	return "unknown"
}

func formatLine(e Entry) string {
	return fmt.Sprintf("[%s | %d] %s (%d): %s",
		formatChatLabel(e), e.ChatID, formatSenderLabel(e), e.SenderID, e.Text)
}

// FormatLine renders a chat message entry for application logs.
func FormatLine(e Entry) string {
	return formatLine(e)
}
