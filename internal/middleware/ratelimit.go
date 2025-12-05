package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type RateLimiter struct {
	requests map[string]*clientLimit
	mu       sync.RWMutex
	limit    int
	window   time.Duration
}

type clientLimit struct {
	count      int
	resetTime  time.Time
	lastUpdate time.Time
}

func NewRateLimiter(requestsPerWindow int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		requests: make(map[string]*clientLimit),
		limit:    requestsPerWindow,
		window:   window,
	}

	// Cleanup goroutine to remove stale entries
	go rl.cleanup()

	return rl
}

func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(rl.window)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for key, limit := range rl.requests {
			if now.After(limit.resetTime.Add(rl.window)) {
				delete(rl.requests, key)
			}
		}
		rl.mu.Unlock()
	}
}

func (rl *RateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := c.ClientIP()

		rl.mu.Lock()
		defer rl.mu.Unlock()

		now := time.Now()
		limit, exists := rl.requests[clientIP]

		if !exists || now.After(limit.resetTime) {
			// New client or window has expired
			rl.requests[clientIP] = &clientLimit{
				count:      1,
				resetTime:  now.Add(rl.window),
				lastUpdate: now,
			}
			c.Next()
			return
		}

		// Check if limit exceeded
		if limit.count >= rl.limit {
			retryAfter := int(limit.resetTime.Sub(now).Seconds())
			c.Header("X-RateLimit-Limit", string(rune(rl.limit)))
			c.Header("X-RateLimit-Remaining", "0")
			c.Header("X-RateLimit-Reset", string(rune(limit.resetTime.Unix())))
			c.Header("Retry-After", string(rune(retryAfter)))

			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "rate limit exceeded",
				"retry_after": retryAfter,
			})
			c.Abort()
			return
		}

		// Increment counter
		limit.count++
		limit.lastUpdate = now

		c.Header("X-RateLimit-Limit", string(rune(rl.limit)))
		c.Header("X-RateLimit-Remaining", string(rune(rl.limit-limit.count)))
		c.Header("X-RateLimit-Reset", string(rune(limit.resetTime.Unix())))

		c.Next()
	}
}
