package middleware

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/ErenKarakus1/API-Gateway/internal/response"
	"github.com/gin-gonic/gin"
)

type RateLimiter struct {
	mu      sync.Mutex
	buckets map[string]bucket
	now     func() time.Time
}

type bucket struct {
	count     int
	expiresAt time.Time
}

func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		buckets: make(map[string]bucket),
		now:     time.Now,
	}
}

func (limiter *RateLimiter) Limit(routeID string, requests int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		if requests <= 0 || window <= 0 {
			c.Next()
			return
		}

		key := routeID + ":" + clientKey(c)
		allowed, remaining, resetAt := limiter.allow(key, requests, window)
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

func (limiter *RateLimiter) allow(key string, limit int, window time.Duration) (bool, int, time.Time) {
	limiter.mu.Lock()
	defer limiter.mu.Unlock()

	now := limiter.now()
	current := limiter.buckets[key]
	if current.expiresAt.IsZero() || !now.Before(current.expiresAt) {
		current = bucket{expiresAt: now.Add(window)}
	}

	if current.count >= limit {
		return false, 0, current.expiresAt
	}

	current.count++
	limiter.buckets[key] = current

	return true, limit - current.count, current.expiresAt
}

func clientKey(c *gin.Context) string {
	if userID := c.GetString(UserIDKey); userID != "" {
		return "user:" + userID
	}

	return "ip:" + c.ClientIP()
}
