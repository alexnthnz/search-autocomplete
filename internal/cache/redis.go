package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"

	"github.com/alexnthnz/search-autocomplete/internal/metrics"
	"github.com/alexnthnz/search-autocomplete/pkg/models"
)

// RedisCache implements caching using Redis
type RedisCache struct {
	client  *redis.Client
	ttl     time.Duration
	logger  *logrus.Logger
	metrics *metrics.Metrics
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
func NewRedisCache(config Config, logger *logrus.Logger, metricsInstance *metrics.Metrics) *RedisCache {
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
		client:  rdb,
		ttl:     config.TTL,
		logger:  logger,
		metrics: metricsInstance,
	}
}

// Get retrieves suggestions from cache
func (r *RedisCache) Get(ctx context.Context, query string) ([]models.Suggestion, bool) {
	start := time.Now()
	key := r.buildKey(query)

	val, err := r.client.Get(ctx, key).Result()

	// Record cache operation duration
	r.metrics.RecordCacheOperation("get", "redis", time.Since(start))

	if err == redis.Nil {
		r.metrics.RecordCacheMiss("redis")
		return nil, false // Cache miss
	}
	if err != nil {
		r.logger.WithError(err).Error("Failed to get from cache")
		r.metrics.RecordError("cache", "get_failed")
		return nil, false
	}

	var suggestions []models.Suggestion
	if err := json.Unmarshal([]byte(val), &suggestions); err != nil {
		r.logger.WithError(err).Error("Failed to unmarshal cached suggestions")
		r.metrics.RecordError("cache", "unmarshal_failed")
		return nil, false
	}

	// Record cache hit
	r.metrics.RecordCacheHit("redis")

	// Update access time for LRU
	r.client.Expire(ctx, key, r.ttl)

	return suggestions, true
}

// Set stores suggestions in cache
func (r *RedisCache) Set(ctx context.Context, query string, suggestions []models.Suggestion) error {
	start := time.Now()
	key := r.buildKey(query)

	data, err := json.Marshal(suggestions)
	if err != nil {
		r.metrics.RecordError("cache", "marshal_failed")
		return err
	}

	err = r.client.Set(ctx, key, data, r.ttl).Err()

	// Record cache operation duration
	r.metrics.RecordCacheOperation("set", "redis", time.Since(start))

	if err != nil {
		r.logger.WithError(err).Error("Failed to set cache")
		r.metrics.RecordError("cache", "set_failed")
		return err
	}

	return nil
}

// Delete removes a query from cache
func (r *RedisCache) Delete(ctx context.Context, query string) error {
	start := time.Now()
	key := r.buildKey(query)

	err := r.client.Del(ctx, key).Err()

	// Record cache operation duration
	r.metrics.RecordCacheOperation("delete", "redis", time.Since(start))

	if err != nil {
		r.logger.WithError(err).Error("Failed to delete from cache")
		r.metrics.RecordError("cache", "delete_failed")
		return err
	}

	return nil
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
	data    map[string]cacheItem
	ttl     time.Duration
	logger  *logrus.Logger
	metrics *metrics.Metrics
}

type cacheItem struct {
	suggestions []models.Suggestion
	expiry      time.Time
}

// NewInMemoryCache creates a new in-memory cache
func NewInMemoryCache(ttl time.Duration, logger *logrus.Logger, metricsInstance *metrics.Metrics) *InMemoryCache {
	cache := &InMemoryCache{
		data:    make(map[string]cacheItem),
		ttl:     ttl,
		logger:  logger,
		metrics: metricsInstance,
	}

	// Start cleanup routine
	go cache.cleanup()

	return cache
}

// Get retrieves suggestions from in-memory cache
func (c *InMemoryCache) Get(ctx context.Context, query string) ([]models.Suggestion, bool) {
	start := time.Now()

	item, exists := c.data[query]

	// Record cache operation duration
	c.metrics.RecordCacheOperation("get", "memory", time.Since(start))

	if !exists || time.Now().After(item.expiry) {
		if exists {
			delete(c.data, query) // Clean expired item
		}
		c.metrics.RecordCacheMiss("memory")
		return nil, false
	}

	c.metrics.RecordCacheHit("memory")
	return item.suggestions, true
}

// Set stores suggestions in in-memory cache
func (c *InMemoryCache) Set(ctx context.Context, query string, suggestions []models.Suggestion) error {
	start := time.Now()

	c.data[query] = cacheItem{
		suggestions: suggestions,
		expiry:      time.Now().Add(c.ttl),
	}

	// Record cache operation duration
	c.metrics.RecordCacheOperation("set", "memory", time.Since(start))

	return nil
}

// Delete removes a query from in-memory cache
func (c *InMemoryCache) Delete(ctx context.Context, query string) error {
	start := time.Now()

	delete(c.data, query)

	// Record cache operation duration
	c.metrics.RecordCacheOperation("delete", "memory", time.Since(start))

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
