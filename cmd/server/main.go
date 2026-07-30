package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"mad-news-bot/internal/advicebridge"
	"mad-news-bot/internal/cache"
	"mad-news-bot/internal/chatlog"
	"mad-news-bot/internal/config"
	"mad-news-bot/internal/handlers"
	"mad-news-bot/internal/oraclequeue"
	"mad-news-bot/internal/telegram"
)

func main() {
	log.SetOutput(os.Stdout)
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
	oracleQueue := oraclequeue.New(oracleRedis.Client(), time.Duration(cfg.AdviceJobTTLSec)*time.Second)
	adviceBridge := advicebridge.New(redis.Client(), time.Duration(cfg.AdviceJobTTLSec)*time.Second)

	chatLogger, err := chatlog.New(cfg.ChatLogDir)
	if err != nil {
		log.Fatalf("chat log: %v", err)
	}
	defer chatLogger.Close()

	router := telegram.NewRouter(cfg, tg, redis, oracleQueue, adviceBridge, chatLogger)

	rootCtx, rootCancel := context.WithCancel(context.Background())
	defer rootCancel()
	go advicebridge.Listen(rootCtx, tg, adviceBridge, oracleQueue, cfg.OracleRedisDB)

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
	rootCancel()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
}
