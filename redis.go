package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Cache defines the interface for cache operations
type Cache interface {
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Get(ctx context.Context, key string, wanted interface{}) error
	Del(ctx context.Context, keys ...string) error
	DelWildCard(ctx context.Context, wildcard string) error
}

// redisCache implements the Cache interface using Redis
type redisCache struct {
	client *redis.Client
}

// RedisConfig holds the configuration for Redis connection
type RedisConfig struct {
	Host     string
	Port     int
	Password string
	Database int
}

// NewRedisCache creates a new Redis cache instance
// It establishes a connection to Redis and verifies it with a ping
func NewRedisCache(cfg RedisConfig) (Cache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.Database,
	})

	// Verify connection
	_, err := client.Ping(context.Background()).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return &redisCache{
		client: client,
	}, nil
}

// Set stores a value in the cache with the given key and TTL
func (r *redisCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	return r.client.Set(ctx, key, value, ttl).Err()
}

// Get retrieves a value from the cache and unmarshal it into the wanted interface
func (r *redisCache) Get(ctx context.Context, key string, wanted interface{}) error {
	result, err := r.client.Get(ctx, key).Result()
	if err != nil {
		return err
	}

	// If wanted is *[]byte, return raw data
	if ptr, ok := wanted.(*[]byte); ok {
		*ptr = []byte(result)
		return nil
	}

	return json.Unmarshal([]byte(result), wanted)
}

// Del deletes keys from the cache
func (r *redisCache) Del(ctx context.Context, keys ...string) error {
	return r.client.Del(ctx, keys...).Err()
}

// DelWildCard deletes all keys matching the wildcard pattern
// Example: DelWildCard(ctx, "user:*") deletes all keys starting with "user:"
func (r *redisCache) DelWildCard(ctx context.Context, wildcard string) error {
	keys, err := r.client.Keys(ctx, wildcard).Result()
	if err != nil {
		return err
	}

	if err := r.Del(ctx, keys...); err != nil {
		return err
	}

	return nil
}
