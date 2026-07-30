package chatlog

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Logger struct {
	dir string
	mu  sync.Mutex
}

func New(dir string) (*Logger, error) {
	if dir == "" {
		return nil, fmt.Errorf("chat log dir is empty")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("chat log mkdir: %w", err)
	}
	return &Logger{dir: dir}, nil
}

func (l *Logger) Log(e Entry) error {
	if e.ChatID >= 0 {
		return nil
	}

	line := fmt.Sprintf("%s %s\n", time.Now().Format("2006/01/02 15:04:05"), formatLine(e))

	l.mu.Lock()
	defer l.mu.Unlock()

	f, err := os.OpenFile(filepath.Join(l.dir, fmt.Sprintf("%d.log", e.ChatID)), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("chat log open: %w", err)
	}
	defer f.Close()

	if _, err := f.WriteString(line); err != nil {
		return fmt.Errorf("chat log write: %w", err)
	}
	return nil
}

func (l *Logger) Close() error {
	return nil
}
