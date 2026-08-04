package advicebridge

import (
	"context"
	"log"

	"mad-news-bot/internal/oraclequeue"
)

type Messenger interface {
	SendMessage(chatID int64, text string) error
}

// Listen consumes advice events from RabbitMQ and replies in Telegram.
// chat_id travels in the event payload (no Redis bridge).
func Listen(ctx context.Context, tg Messenger, queue *oraclequeue.Queue) {
	log.Printf("advice listener on rabbit advice.events.mad-news")
	sub := queue.Subscribe(ctx)
	defer sub.Close()

	for {
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-sub.Channel():
			if !ok {
				return
			}
			handleEvent(tg, ev)
		}
	}
}

func handleEvent(tg Messenger, ev oraclequeue.Event) {
	if ev.JobID == "" || ev.ChatID == 0 {
		return
	}
	switch ev.Status {
	case oraclequeue.StatusRetrying:
		// chat_id stays on the new job via oracle ScheduleRetry; nothing to migrate
		return
	case oraclequeue.StatusDone:
		text, err := oraclequeue.FormatAdvice(ev.Result)
		if err != nil {
			log.Printf("advice format %s: %v", ev.JobID, err)
			_ = tg.SendMessage(ev.ChatID, "Оракул вернул пустой ответ.")
			return
		}
		if err := tg.SendMessage(ev.ChatID, text); err != nil {
			log.Printf("advice send %s: %v", ev.JobID, err)
		}
	case oraclequeue.StatusFailed:
		_ = tg.SendMessage(ev.ChatID, "Оракул молчит. Попробуйте позже.")
	}
}
