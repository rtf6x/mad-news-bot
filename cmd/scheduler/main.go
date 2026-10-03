package main

import (
	"context"
	"log"
	"os"
	"strings"
	"time"

	"mad-news-bot/internal/cache"
	"mad-news-bot/internal/commands/apod"
	"mad-news-bot/internal/config"
)

func main() {
	config.LoadDotEnv(".env")
	cfg := config.Load()

	redis, err := cache.New(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)
	if err != nil {
		log.Fatalf("redis: %v", err)
	}
	defer redis.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	if err := apod.FetchAndStore(ctx, redis); err != nil {
		if strings.Contains(err.Error(), "keeping stale cache") {
			log.Printf("warning: %v", err)
			return
		}
		log.Fatalf("apod prefetch failed: %v", err)
	}
	log.Printf("nasa apod cached successfully")
	os.Exit(0)
}
