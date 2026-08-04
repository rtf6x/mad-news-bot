package oraclequeue

import (
	"context"
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/google/uuid"
)

const (
	jobsExchange        = "jobs"
	eventsExchange      = "events"
	adviceJobKey        = "advice.job"
	adviceEventKey      = "advice.event"
	adviceJobsQueue     = "advice.jobs"
	madNewsEventsQueue  = "advice.events.mad-news"
)

// Queue publishes advice jobs and consumes advice events for mad-news.
type Queue struct {
	conn *amqp.Connection
	pub  *amqp.Channel
}

type EventSub struct {
	ch     chan Event
	cancel context.CancelFunc
	done   chan struct{}
}

func (s *EventSub) Channel() <-chan Event { return s.ch }
func (s *EventSub) Close() error {
	s.cancel()
	<-s.done
	return nil
}

func NewRabbit(amqpURL string) (*Queue, error) {
	if amqpURL == "" {
		return nil, fmt.Errorf("RABBIT_URL is empty")
	}
	conn, err := amqp.Dial(amqpURL)
	if err != nil {
		return nil, err
	}
	pub, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	q := &Queue{conn: conn, pub: pub}
	if err := q.declare(); err != nil {
		_ = q.Close()
		return nil, err
	}
	return q, nil
}

func (q *Queue) declare() error {
	if err := q.pub.ExchangeDeclare(jobsExchange, "direct", true, false, false, false, nil); err != nil {
		return err
	}
	if err := q.pub.ExchangeDeclare(eventsExchange, "topic", true, false, false, false, nil); err != nil {
		return err
	}
	if _, err := q.pub.QueueDeclare(adviceJobsQueue, true, false, false, false, nil); err != nil {
		return err
	}
	if err := q.pub.QueueBind(adviceJobsQueue, adviceJobKey, jobsExchange, false, nil); err != nil {
		return err
	}
	if _, err := q.pub.QueueDeclare(madNewsEventsQueue, true, false, false, false, nil); err != nil {
		return err
	}
	return q.pub.QueueBind(madNewsEventsQueue, adviceEventKey, eventsExchange, false, nil)
}

func (q *Queue) Close() error {
	if q.pub != nil {
		_ = q.pub.Close()
	}
	if q.conn != nil {
		return q.conn.Close()
	}
	return nil
}

func (q *Queue) Enqueue(ctx context.Context, prompt, lang string, chatID int64) (string, error) {
	id := uuid.NewString()
	job := Job{
		ID:      id,
		Status:  StatusQueued,
		Prompt:  prompt,
		Lang:    lang,
		Attempt: 1,
		ChatID:  chatID,
	}
	body, err := json.Marshal(job)
	if err != nil {
		return "", err
	}
	if err := q.pub.PublishWithContext(ctx, jobsExchange, adviceJobKey, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Body:         body,
		MessageId:    id,
	}); err != nil {
		return "", err
	}
	return id, nil
}

func (q *Queue) Subscribe(ctx context.Context) *EventSub {
	ctx, cancel := context.WithCancel(ctx)
	sub := &EventSub{ch: make(chan Event, 32), cancel: cancel, done: make(chan struct{})}
	go q.runEvents(ctx, sub)
	return sub
}

func (q *Queue) runEvents(ctx context.Context, sub *EventSub) {
	defer close(sub.done)
	defer close(sub.ch)
	ch, err := q.conn.Channel()
	if err != nil {
		return
	}
	defer ch.Close()
	deliveries, err := ch.Consume(madNewsEventsQueue, "", true, false, false, false, nil)
	if err != nil {
		return
	}
	for {
		select {
		case <-ctx.Done():
			return
		case d, ok := <-deliveries:
			if !ok {
				return
			}
			var ev Event
			if err := json.Unmarshal(d.Body, &ev); err != nil {
				continue
			}
			select {
			case sub.ch <- ev:
			case <-ctx.Done():
				return
			}
		}
	}
}
