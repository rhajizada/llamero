package redisstore

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"

	"github.com/rhajizada/llamero/internal/config"
)

// Store wraps a Redis client for backend state management.
type Store struct {
	client *redis.Client
}

// New constructs a Store from the cache configuration.
func New(cfg *config.RedisConfig) (*Store, error) {
	if cfg == nil {
		return nil, fmt.Errorf("redis config is required")
	}
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Username: cfg.Username,
		Password: cfg.Password,
		DB:       cfg.DB,
		Protocol: 2,
	})
	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, err
	}
	return &Store{client: client}, nil
}

// Client exposes the underlying Redis client when direct access is needed.
func (s *Store) Client() *redis.Client {
	return s.client
}
