package advicebridge

import (
	"context"
	"encoding/json"
	"log"

	"mad-news-bot/internal/oraclequeue"
)

type Messenger interface {
	SendMessage(chatID int64, text string) error
}

func Listen(ctx context.Context, tg Messenger, bridge *Store, queue *oraclequeue.Queue, oracleRedisDB int) {
	log.Printf("advice listener on oracle redis db=%d", oracleRedisDB)
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

func handleEvent(ctx context.Context, tg Messenger, bridge *Store, payload string) {
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
