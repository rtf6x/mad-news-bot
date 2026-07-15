package oraclequeue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type Queue struct {
	client *redis.Client
	ttl    time.Duration
}

func New(client *redis.Client, ttl time.Duration) *Queue {
	if ttl <= 0 {
		ttl = time.Hour
	}
	return &Queue{client: client, ttl: ttl}
}

func (q *Queue) Enqueue(ctx context.Context, prompt, lang string) (string, error) {
	id := uuid.NewString()
	job := Job{
		ID:      id,
		Status:  StatusQueued,
		Prompt:  prompt,
		Lang:    lang,
		Attempt: 1,
	}
	if err := q.saveJob(ctx, job); err != nil {
		return "", err
	}
	if err := q.client.LPush(ctx, QueueKey, id).Err(); err != nil {
		return "", err
	}
	pos, err := q.queuePosition(ctx, id)
	if err != nil {
		return "", err
	}
	if err := q.publish(ctx, Event{
		JobID:         id,
		Status:        StatusQueued,
		QueuePosition: pos,
		Attempt:       1,
	}); err != nil {
		return "", err
	}
	return id, nil
}

func (q *Queue) Subscribe(ctx context.Context) *redis.PubSub {
	return q.client.Subscribe(ctx, EventsChannel)
}

func (q *Queue) saveJob(ctx context.Context, job Job) error {
	raw, err := json.Marshal(job)
	if err != nil {
		return err
	}
	return q.client.Set(ctx, JobKeyPrefix+job.ID, raw, q.jobTTL(job.Status)).Err()
}

func (q *Queue) jobTTL(status string) time.Duration {
	if status == StatusQueued || status == StatusProcessing || status == StatusRetrying {
		if q.ttl < 120*time.Second {
			return 120 * time.Second
		}
	}
	return q.ttl
}

func (q *Queue) queuePosition(ctx context.Context, id string) (int, error) {
	ids, err := q.client.LRange(ctx, QueueKey, 0, -1).Result()
	if err != nil {
		return 0, err
	}
	for i, jobID := range ids {
		if jobID == id {
			return len(ids) - i, nil
		}
	}
	return 0, nil
}

func (q *Queue) publish(ctx context.Context, ev Event) error {
	raw, err := json.Marshal(ev)
	if err != nil {
		return err
	}
	return q.client.Publish(ctx, EventsChannel, raw).Err()
}

func FormatAdvice(result *Advice) (string, error) {
	if result == nil || result.Advice == "" {
		return "", fmt.Errorf("empty advice result")
	}
	return result.Advice, nil
}
