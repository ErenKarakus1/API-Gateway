package middleware

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/ErenKarakus1/API-Gateway/internal/response"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type RateLimiter struct {
	store RateLimitStore
}

type RateLimitStore interface {
	Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, int, time.Time, error)
}

type MemoryRateLimitStore struct {
	mu      sync.Mutex
	buckets map[string]bucket
	now     func() time.Time
}

type RedisRateLimitStore struct {
	client *redis.Client
	now    func() time.Time
}

type bucket struct {
	count     int
	expiresAt time.Time
}

func NewRateLimiter() *RateLimiter {
	return NewRateLimiterWithStore(NewMemoryRateLimitStore())
}

func NewRateLimiterWithStore(store RateLimitStore) *RateLimiter {
	return &RateLimiter{store: store}
}

func NewMemoryRateLimitStore() *MemoryRateLimitStore {
	return &MemoryRateLimitStore{
		buckets: make(map[string]bucket),
		now:     time.Now,
	}
}

func NewRedisRateLimitStore(client *redis.Client) *RedisRateLimitStore {
	return &RedisRateLimitStore{
		client: client,
		now:    time.Now,
	}
}

func (limiter *RateLimiter) Limit(routeID string, requests int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		if requests <= 0 || window <= 0 {
			c.Next()
			return
		}

		key := routeID + ":" + clientKey(c)
		allowed, remaining, resetAt, err := limiter.store.Allow(c.Request.Context(), key, requests, window)
		if err != nil {
			response.Error(c, http.StatusServiceUnavailable, "rate_limiter_unavailable", "rate limiter unavailable")
			return
		}
		c.Writer.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", requests))
		c.Writer.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
		c.Writer.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", resetAt.Unix()))

		if !allowed {
			response.Error(c, http.StatusTooManyRequests, "rate_limit_exceeded", "rate limit exceeded")
			return
		}

		c.Next()
	}
}

func (store *MemoryRateLimitStore) Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, int, time.Time, error) {
	store.mu.Lock()
	defer store.mu.Unlock()

	now := store.now()
	current := store.buckets[key]
	if current.expiresAt.IsZero() || !now.Before(current.expiresAt) {
		current = bucket{expiresAt: now.Add(window)}
	}

	if current.count >= limit {
		return false, 0, current.expiresAt, nil
	}

	current.count++
	store.buckets[key] = current

	return true, limit - current.count, current.expiresAt, nil
}

func (store *RedisRateLimitStore) Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, int, time.Time, error) {
	redisKey := "rate_limit:" + key
	count, err := store.client.Incr(ctx, redisKey).Result()
	if err != nil {
		return false, 0, time.Time{}, err
	}

	if count == 1 {
		if err := store.client.Expire(ctx, redisKey, window).Err(); err != nil {
			return false, 0, time.Time{}, err
		}
	}

	ttl, err := store.client.TTL(ctx, redisKey).Result()
	if err != nil {
		return false, 0, time.Time{}, err
	}
	if ttl < 0 {
		ttl = window
	}

	remaining := limit - int(count)
	if remaining < 0 {
		remaining = 0
	}

	return count <= int64(limit), remaining, store.now().Add(ttl), nil
}

func clientKey(c *gin.Context) string {
	if userID := c.GetString(UserIDKey); userID != "" {
		return "user:" + userID
	}

	return "ip:" + c.ClientIP()
}
