package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"mad-news-bot/internal/advicebridge"
	"mad-news-bot/internal/cache"
	"mad-news-bot/internal/config"
	"mad-news-bot/internal/oraclequeue"
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

	oracleRedis, err := cache.New(cfg.RedisAddr, cfg.RedisPassword, cfg.OracleRedisDB)
	if err != nil {
		log.Fatalf("oracle redis: %v", err)
	}
	defer oracleRedis.Close()

	tg := telegram.NewClient(cfg.BotToken)
	bridge := advicebridge.New(redis.Client(), time.Duration(cfg.AdviceJobTTLSec)*time.Second)
	queue := oraclequeue.New(oracleRedis.Client(), time.Duration(cfg.AdviceJobTTLSec)*time.Second)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		stop := make(chan os.Signal, 1)
		signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
		<-stop
		cancel()
	}()

	log.Printf("mad-news-bot worker listening on oracle redis db=%d", cfg.OracleRedisDB)
	sub := queue.Subscribe(ctx)
	defer sub.Close()

	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-sub.Channel():
			if !ok {
				return
			}
			handleEvent(ctx, tg, bridge, msg.Payload)
		}
	}
}

func handleEvent(ctx context.Context, tg *telegram.Client, bridge *advicebridge.Store, payload string) {
	var ev oraclequeue.Event
	if err := json.Unmarshal([]byte(payload), &ev); err != nil {
		log.Printf("advice event parse: %v", err)
		return
	}
	if ev.JobID == "" {
		return
	}
	if !bridge.Has(ctx, ev.JobID) {
		return
	}

	switch ev.Status {
	case oraclequeue.StatusRetrying:
		if ev.RetryJobID == "" {
			return
		}
		if err := bridge.Migrate(ctx, ev.JobID, ev.RetryJobID); err != nil {
			log.Printf("advice migrate %s -> %s: %v", ev.JobID, ev.RetryJobID, err)
		}
	case oraclequeue.StatusDone:
		chatID, err := bridge.Pop(ctx, ev.JobID)
		if err != nil {
			log.Printf("advice pop %s: %v", ev.JobID, err)
			return
		}
		text, err := oraclequeue.FormatAdvice(ev.Result)
		if err != nil {
			log.Printf("advice format %s: %v", ev.JobID, err)
			_ = tg.SendMessage(chatID, "Оракул вернул пустой ответ.")
			return
		}
		if err := tg.SendMessage(chatID, text); err != nil {
			log.Printf("advice send %s: %v", ev.JobID, err)
		}
	case oraclequeue.StatusFailed:
		chatID, err := bridge.Pop(ctx, ev.JobID)
		if err != nil {
			log.Printf("advice pop failed %s: %v", ev.JobID, err)
			return
		}
		_ = tg.SendMessage(chatID, "Оракул молчит. Попробуйте позже.")
	}
}
