package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"product-management/internal/models"

	"github.com/go-redis/redis/v8"
)

type RedisCache struct {
	client *redis.Client
}

func NewRedisCache(redisURL string) *RedisCache {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		panic(fmt.Errorf("failed to parse Redis URL: %w", err))
	}

	return &RedisCache{
		client: redis.NewClient(opt),
	}
}

func (c *RedisCache) Set(ctx context.Context, key string, value *models.Product, expiration time.Duration) error {
	// Serialize product
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal product: %w", err)
	}

	// Set in Redis
	return c.client.Set(ctx, key, data, expiration).Err()
}

func (c *RedisCache) Get(ctx context.Context, key string) (*models.Product, error) {
	// Get from Redis
	data, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		return nil, fmt.Errorf("failed to get from")