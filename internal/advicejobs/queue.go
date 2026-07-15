package advicejobs

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	queueKey      = "madnews:advice:queue"
	jobKeyPrefix  = "madnews:advice:job:"
	defaultJobTTL = 2 * time.Hour
)

type Job struct {
	ID     string `json:"id"`
	ChatID int64  `json:"chat_id"`
	Prompt string `json:"prompt"`
	Lang   string `json:"lang"`
}

type Queue struct {
	client *redis.Client
	ttl    time.Duration
}

func New(client *redis.Client, ttl time.Duration) *Queue {
	if ttl <= 0 {
		ttl = defaultJobTTL
	}
	return &Queue{client: client, ttl: ttl}
}

func (q *Queue) Enqueue(ctx context.Context, chatID int64, prompt, lang string) (string, error) {
	id := uuid.NewString()
	job := Job{ID: id, ChatID: chatID, Prompt: prompt, Lang: lang}
	raw, err := json.Marshal(job)
	if err != nil {
		return "", err
	}
	if err := q.client.SetEx(ctx, jobKeyPrefix+id, raw, q.ttl).Err(); err != nil {
		return "", err
	}
	if err := q.client.LPush(ctx, queueKey, id).Err(); err != nil {
		return "", err
	}
	return id, nil
}

func (q *Queue) Claim(ctx context.Context) (Job, error) {
	for {
		id, err := q.client.BRPopLPush(ctx, queueKey, queueKey+":processing", 0).Result()
		if err != nil {
			return Job{}, err
		}
		raw, err := q.client.Get(ctx, jobKeyPrefix+id).Result()
		if err != nil {
			_ = q.client.LRem(ctx, queueKey+":processing", 1, id).Err()
			continue
		}
		var job Job
		if err := json.Unmarshal([]byte(raw), &job); err != nil {
			_ = q.client.LRem(ctx, queueKey+":processing", 1, id).Err()
			continue
		}
		return job, nil
	}
}

func (q *Queue) Ack(ctx context.Context, jobID string) error {
	_ = q.client.LRem(ctx, queueKey+":processing", 1, jobID).Err()
	return q.client.Del(ctx, jobKeyPrefix+jobID).Err()
}

func (q *Queue) Fail(ctx context.Context, jobID string, reason string) error {
	_ = q.client.LRem(ctx, queueKey+":processing", 1, jobID).Err()
	return fmt.Errorf("job %s failed: %s", jobID, reason)
}
