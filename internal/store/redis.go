package store

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Store is the interface for job state persistence.
type Store interface {
	Ping(ctx context.Context) error
	InitJob(ctx context.Context, key string, stepNames []string) error
	SetStep(ctx context.Context, key, stepName, status, data, errMsg string) error
	SetOverallStatus(ctx context.Context, key, status, errMsg string) error
	GetAll(ctx context.Context, key string) (map[string]string, error)
}

// RedisStore manages job state using Redis Hashes.
// Each field is updated atomically via HSET, avoiding read-modify-write races
// between concurrent step goroutines.
type RedisStore struct {
	client *redis.Client
	ttl    time.Duration
}

func NewRedisStore(addr string, ttl time.Duration) *RedisStore {
	return &RedisStore{
		client: redis.NewClient(&redis.Options{Addr: addr}),
		ttl:    ttl,
	}
}

// Ping checks Redis connectivity.
func (s *RedisStore) Ping(ctx context.Context) error {
	return s.client.Ping(ctx).Err()
}

// InitJob sets the overall_status to "pending" and pre-populates all step statuses.
func (s *RedisStore) InitJob(ctx context.Context, key string, stepNames []string) error {
	pipe := s.client.Pipeline()
	pipe.HSet(ctx, key, "overall_status", "pending")
	pipe.HSet(ctx, key, "overall_error", "")
	for _, name := range stepNames {
		pipe.HSet(ctx, key, fmt.Sprintf("step:%s:status", name), "pending")
	}
	pipe.Expire(ctx, key, s.ttl)
	_, err := pipe.Exec(ctx)
	return err
}

// SetStep atomically updates a step's status, data, and error fields.
func (s *RedisStore) SetStep(ctx context.Context, key, stepName, status, data, errMsg string) error {
	pipe := s.client.Pipeline()
	pipe.HSet(ctx, key, fmt.Sprintf("step:%s:status", stepName), status)
	if data != "" {
		pipe.HSet(ctx, key, fmt.Sprintf("step:%s:data", stepName), data)
	}
	if errMsg != "" {
		pipe.HSet(ctx, key, fmt.Sprintf("step:%s:error", stepName), errMsg)
	}
	pipe.Expire(ctx, key, s.ttl)
	_, err := pipe.Exec(ctx)
	return err
}

// SetOverallStatus updates the top-level job status.
func (s *RedisStore) SetOverallStatus(ctx context.Context, key, status, errMsg string) error {
	pipe := s.client.Pipeline()
	pipe.HSet(ctx, key, "overall_status", status)
	if errMsg != "" {
		pipe.HSet(ctx, key, "overall_error", errMsg)
	}
	pipe.Expire(ctx, key, s.ttl)
	_, err := pipe.Exec(ctx)
	return err
}

// GetAll returns the full Redis hash for a job key.
func (s *RedisStore) GetAll(ctx context.Context, key string) (map[string]string, error) {
	return s.client.HGetAll(ctx, key).Result()
}
