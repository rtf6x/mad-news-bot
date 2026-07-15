package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"mad-news-bot/internal/advicejobs"
	"mad-news-bot/internal/cache"
	"mad-news-bot/internal/config"
	"mad-news-bot/internal/handlers"
	"mad-news-bot/internal/telegram"
)

func main() {
	config.LoadDotEnv(".env")
	cfg := config.Load()

	redis, err := cache.New(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)
	if err != nil {
		log.Fatalf("redis: %v", err)
	}
	defer redis.Close()

	tg := telegram.NewClient(cfg.BotToken)
	adviceQueue := advicejobs.New(redis.Client(), time.Duration(cfg.AdviceJobTTLSec)*time.Second)
	router := telegram.NewRouter(cfg, tg, redis, adviceQueue)

	mux := handlers.NewMux(cfg, redis, router)
	srv := &http.Server{Addr: cfg.Addr, Handler: mux}

	go func() {
		log.Printf("mad-news-bot listening on %s", cfg.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	log.Printf("shutting down")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}
