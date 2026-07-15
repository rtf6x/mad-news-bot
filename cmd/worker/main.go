package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"mad-news-bot/internal/advicejobs"
	"mad-news-bot/internal/badadvice"
	"mad-news-bot/internal/cache"
	"mad-news-bot/internal/config"
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
	adviceClient := badadvice.NewClient(cfg.BadAdviceURL)
	queue := advicejobs.New(redis.Client(), time.Duration(cfg.AdviceJobTTLSec)*time.Second)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		stop := make(chan os.Signal, 1)
		signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
		<-stop
		cancel()
	}()

	log.Printf("mad-news-bot worker listening (bad-advice %s)", cfg.BadAdviceURL)
	for {
		if err := ctx.Err(); err != nil {
			return
		}
		job, err := queue.Claim(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("claim: %v", err)
			time.Sleep(time.Second)
			continue
		}
		handleJob(ctx, tg, adviceClient, queue, job)
	}
}

func handleJob(ctx context.Context, tg *telegram.Client, advice *badadvice.Client, queue *advicejobs.Queue, job advicejobs.Job) {
	workCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Minute)
	defer cancel()

	text, err := advice.RequestAdvice(workCtx, job.Prompt, job.Lang)
	if err != nil {
		log.Printf("advice job %s: %v", job.ID, err)
		_ = tg.SendMessage(job.ChatID, "Оракул молчит. Попробуйте позже.")
		_ = queue.Ack(ctx, job.ID)
		return
	}
	if err := tg.SendMessage(job.ChatID, text); err != nil {
		log.Printf("advice send %s: %v", job.ID, err)
	}
	_ = queue.Ack(ctx, job.ID)
}
