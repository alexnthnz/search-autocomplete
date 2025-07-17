package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"

	"github.com/alexnthnz/search-autocomplete/pkg/models"
)

// RedisCache implements caching using Redis
type RedisCache struct {
	client *redis.Client
	ttl    time.Duration
	logger *logrus.Logger
}

// Config holds Redis configuration
type Config struct {
	Host     string
	Port     int
	Password string
	DB       int
	TTL      time.Duration
}

// NewRedisCache creates a new Redis cache instance
func NewRedisCache(config Config, logger *logrus.Logger) *RedisCache {
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", config.Host, config.Port),
		Password: config.Password,
		DB:       config.DB,
	})

	// Test connection
	ctx := context.Background()
	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		logger.WithError(err).Fatal("Failed to connect to Redis")
	}

	logger.Info("Successfully connected to Redis")

	return &RedisCache{
		client: rdb,
		ttl:    config.TTL,
		logger: logger,
	}
}

// Get retrieves suggestions from cache
func (r *RedisCache) Get(ctx context.Context, query string) ([]models.Suggestion, bool) {
	key := r.buildKey(query)

	val, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, false // Cache miss
	}
	if err != nil {
		r.logger.WithError(err).Error("Failed to get from cache")
		return nil, false
	}

	var suggestions []models.Suggestion
	if err := json.Unmarshal([]byte(val), &suggestions); err != nil {
		r.logger.WithError(err).Error("Failed to unmarshal cached suggestions")
		return nil, false
	}

	// Update access time for LRU
	r.client.Expire(ctx, key, r.ttl)

	return suggestions, true
}

// Set stores suggestions in cache
func (r *RedisCache) Set(ctx context.Context, query string, suggestions []models.Suggestion) error {
	key := r.buildKey(query)

	data, err := json.Marshal(suggestions)
	if err != nil {
		return fmt.Errorf("failed to marshal suggestions: %w", err)
	}

	err = r.client.Set(ctx, key, data, r.ttl).Err()
	if err != nil {
		r.logger.WithError(err).Error("Failed to set cache")
		return fmt.Errorf("failed to set cache: %w", err)
	}

	return nil
}

// Delete removes a query from cache
func (r *RedisCache) Delete(ctx context.Context, query string) error {
	key := r.buildKey(query)
	return r.client.Del(ctx, key).Err()
}

// Clear removes all cached queries matching a pattern
func (r *RedisCache) Clear(ctx context.Context, pattern string) error {
	if pattern == "" {
		pattern = "autocomplete:*"
	}

	keys, err := r.client.Keys(ctx, pattern).Result()
	if err != nil {
		return fmt.Errorf("failed to get keys: %w", err)
	}

	if len(keys) > 0 {
		return r.client.Del(ctx, keys...).Err()
	}

	return nil
}

// GetStats returns cache statistics
func (r *RedisCache) GetStats(ctx context.Context) (map[string]interface{}, error) {
	info, err := r.client.Info(ctx, "stats").Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get Redis stats: %w", err)
	}

	// Get key count for autocomplete
	keys, err := r.client.Keys(ctx, "autocomplete:*").Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get key count: %w", err)
	}

	stats := map[string]interface{}{
		"redis_info":        info,
		"autocomplete_keys": len(keys),
		"ttl_seconds":       r.ttl.Seconds(),
	}

	return stats, nil
}

// Warmup pre-loads common queries into cache
func (r *RedisCache) Warmup(ctx context.Context, commonQueries map[string][]models.Suggestion) error {
	r.logger.Info("Starting cache warmup")

	for query, suggestions := range commonQueries {
		if err := r.Set(ctx, query, suggestions); err != nil {
			r.logger.WithError(err).WithField("query", query).Error("Failed to warmup query")
			continue
		}
	}

	r.logger.WithField("count", len(commonQueries)).Info("Cache warmup completed")
	return nil
}

// buildKey creates a standardized cache key
func (r *RedisCache) buildKey(query string) string {
	return fmt.Sprintf("autocomplete:%s", query)
}

// Close closes the Redis connection
func (r *RedisCache) Close() error {
	return r.client.Close()
}

// InMemoryCache implements a simple in-memory cache as fallback
type InMemoryCache struct {
	data   map[string]cacheItem
	ttl    time.Duration
	logger *logrus.Logger
}

type cacheItem struct {
	suggestions []models.Suggestion
	expiry      time.Time
}

// NewInMemoryCache creates a new in-memory cache
func NewInMemoryCache(ttl time.Duration, logger *logrus.Logger) *InMemoryCache {
	cache := &InMemoryCache{
		data:   make(map[string]cacheItem),
		ttl:    ttl,
		logger: logger,
	}

	// Start cleanup routine
	go cache.cleanup()

	return cache
}

// Get retrieves suggestions from in-memory cache
func (c *InMemoryCache) Get(ctx context.Context, query string) ([]models.Suggestion, bool) {
	item, exists := c.data[query]
	if !exists || time.Now().After(item.expiry) {
		if exists {
			delete(c.data, query) // Clean expired item
		}
		return nil, false
	}

	return item.suggestions, true
}

// Set stores suggestions in in-memory cache
func (c *InMemoryCache) Set(ctx context.Context, query string, suggestions []models.Suggestion) error {
	c.data[query] = cacheItem{
		suggestions: suggestions,
		expiry:      time.Now().Add(c.ttl),
	}
	return nil
}

// Delete removes a query from in-memory cache
func (c *InMemoryCache) Delete(ctx context.Context, query string) error {
	delete(c.data, query)
	return nil
}

// cleanup removes expired items from cache
func (c *InMemoryCache) cleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		for key, item := range c.data {
			if now.After(item.expiry) {
				delete(c.data, key)
			}
		}
	}
}

// Cache interface defines the caching contract
type Cache interface {
	Get(ctx context.Context, query string) ([]models.Suggestion, bool)
	Set(ctx context.Context, query string, suggestions []models.Suggestion) error
	Delete(ctx context.Context, query string) error
}
