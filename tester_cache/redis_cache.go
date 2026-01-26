package tester_cache

import (
	"context"
	"fmt"
	"os"

	"github.com/redis/go-redis/v9"
)

// redisCache is the Redis implementation of the Cache interface
type redisCache struct {
	client  *redis.Client
	context context.Context
}

func newRedisCache() (*redisCache, error) {
	redisURL := os.Getenv("CODECRAFTERS_SECRET_TESTER_CACHE_REDIS_URL")

	if redisURL == "" {
		return nil, fmt.Errorf("CODECRAFTERS_SECRET_TESTER_CACHE_REDIS_URL not set")
	}

	redisOptions, err := redis.ParseURL(redisURL)

	if err != nil {
		return nil, fmt.Errorf("Failed to parse Redis URL: %s", err)
	}

	client := redis.NewClient(redisOptions)
	context := context.Background()

	if err := client.Ping(context).Err(); err != nil {
		return nil, fmt.Errorf("Failed to connect to Redis: %s", err)
	}

	return &redisCache{
		client:  client,
		context: context,
	}, nil
}

func (r *redisCache) Get(key string) ([]byte, bool) {
	val, err := r.client.Get(r.context, key).Result()

	if err != nil {
		return nil, false
	}

	return []byte(val), true
}

func (r *redisCache) Set(key string, value []byte) {
	r.client.Set(r.context, key, value, 0)
}

func (r *redisCache) Close() {
	if r.client != nil {
		r.client.Close()
	}
}
