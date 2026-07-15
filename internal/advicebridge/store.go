package advicebridge

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

const chatKeyPrefix = "madnews:advice:chat:"

type Store struct {
	client *redis.Client
	ttl    time.Duration
}

func New(client *redis.Client, ttl time.Duration) *Store {
	if ttl <= 0 {
		ttl = 2 * time.Hour
	}
	return &Store{client: client, ttl: ttl}
}

func (s *Store) Bind(ctx context.Context, jobID string, chatID int64) error {
	return s.client.SetEx(ctx, chatKeyPrefix+jobID, strconv.FormatInt(chatID, 10), s.ttl).Err()
}

func (s *Store) Migrate(ctx context.Context, oldID, newID string) error {
	key := chatKeyPrefix + oldID
	chatID, err := s.client.Get(ctx, key).Result()
	if err != nil {
		return err
	}
	pipe := s.client.TxPipeline()
	pipe.SetEx(ctx, chatKeyPrefix+newID, chatID, s.ttl)
	pipe.Del(ctx, key)
	_, err = pipe.Exec(ctx)
	return err
}

func (s *Store) Pop(ctx context.Context, jobID string) (int64, error) {
	key := chatKeyPrefix + jobID
	chatID, err := s.client.Get(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	if err := s.client.Del(ctx, key).Err(); err != nil {
		return 0, err
	}
	id, err := strconv.ParseInt(chatID, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse chat id: %w", err)
	}
	return id, nil
}

func (s *Store) Has(ctx context.Context, jobID string) bool {
	n, err := s.client.Exists(ctx, chatKeyPrefix+jobID).Result()
	return err == nil && n > 0
}
