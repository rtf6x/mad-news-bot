package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Addr            string
	BotToken        string
	BotUsername     string
	HireChatID      string
	RedisAddr       string
	RedisPassword   string
	RedisDB         int
	NASAAPODKey     string
	BadAdviceURL    string
	AdviceJobTTLSec int
}

func Load() Config {
	return Config{
		Addr:            getEnv("APP_ADDR", "127.0.0.1:8346"),
		BotToken:        os.Getenv("BOT_TOKEN"),
		BotUsername:     getEnv("BOT_USERNAME", "madnews_rtf6x_bot"),
		HireChatID:      getEnv("HIRE_CHAT_ID", "324702279"),
		RedisAddr:       getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:   os.Getenv("REDIS_PASSWORD"),
		RedisDB:         getEnvInt("REDIS_DB", 0),
		NASAAPODKey:     os.Getenv("NASA_APOD_KEY"),
		BadAdviceURL:    strings.TrimRight(getEnv("BAD_ADVICE_URL", "http://127.0.0.1:8088"), "/"),
		AdviceJobTTLSec: getEnvInt("ADVICE_JOB_TTL_SEC", 120),
	}
}

func LoadDotEnv(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	for _, line := range splitLines(string(data)) {
		if line == "" || line[0] == '#' {
			continue
		}
		key, val, ok := splitKV(line)
		if !ok {
			continue
		}
		if os.Getenv(key) == "" {
			_ = os.Setenv(key, val)
		}
	}
}

func getEnv(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			line := s[start:i]
			if len(line) > 0 && line[len(line)-1] == '\r' {
				line = line[:len(line)-1]
			}
			lines = append(lines, line)
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func splitKV(line string) (string, string, bool) {
	for i := 0; i < len(line); i++ {
		if line[i] == '=' {
			key := line[:i]
			val := line[i+1:]
			if len(val) >= 2 {
				if (val[0] == '"' && val[len(val)-1] == '"') || (val[0] == '\'' && val[len(val)-1] == '\'') {
					val = val[1 : len(val)-1]
				}
			}
			return key, val, key != ""
		}
	}
	return "", "", false
}
