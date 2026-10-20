package cache

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

const taskListVersionKey = "tasks:list:version"

type RedisTaskListCache struct {
	redisClient *redis.Client
}

func NewRedisTaskListCache(redisClient *redis.Client) *RedisTaskListCache {
	return &RedisTaskListCache{redisClient: redisClient}
}

func (c *RedisTaskListCache) Version(ctx context.Context) (int64, error) {
	version, err := c.redisClient.Get(ctx, taskListVersionKey).Int64()
	if errors.Is(err, redis.Nil) {
		return 0, nil
	}
	return version, err
}

func (c *RedisTaskListCache) Get(ctx context.Context, key string) ([]byte, bool, error) {
	value, err := c.redisClient.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return value, true, nil
}

func (c *RedisTaskListCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return c.redisClient.Set(ctx, key, value, ttl).Err()
}

func (c *RedisTaskListCache) Invalidate(ctx context.Context) error {
	return c.redisClient.Incr(ctx, taskListVersionKey).Err()
}

func (c *RedisTaskListCache) Close() error {
	return c.redisClient.Close()
}
