package middleware

import (
	"net/http"
	"sync"
	"time"

	"panel-api/internal/config"

	"github.com/gin-gonic/gin"
)

type RateLimiter struct {
	requests map[string][]time.Time
	mu       sync.RWMutex
	limit    int
	window   time.Duration
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		requests: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}
	go rl.cleanup()
	return rl
}

func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for ip, times := range rl.requests {
			var validTimes []time.Time
			for _, t := range times {
				if now.Sub(t) <= rl.window {
					validTimes = append(validTimes, t)
				}
			}
			if len(validTimes) == 0 {
				delete(rl.requests, ip)
			} else {
				rl.requests[ip] = validTimes
			}
		}
		rl.mu.Unlock()
	}
}

func (rl *RateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-rl.window)

	var validTimes []time.Time
	for _, t := range rl.requests[ip] {
		if t.After(windowStart) {
			validTimes = append(validTimes, t)
		}
	}

	if len(validTimes) >= rl.limit {
		rl.requests[ip] = validTimes
		return false
	}

	rl.requests[ip] = append(validTimes, now)
	return true
}

type AuthRateLimiter struct {
	*RateLimiter
}

func NewAuthRateLimiter() *AuthRateLimiter {
	return &AuthRateLimiter{
		RateLimiter: NewRateLimiter(10, 1*time.Minute),
	}
}

func RateLimitAuth(cfg *config.Config) gin.HandlerFunc {
	limiter := NewAuthRateLimiter()

	return func(c *gin.Context) {
		ip := c.ClientIP()

		if !limiter.Allow(ip) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "rate_limit_exceeded",
				"message": "Too many authentication attempts. Please try again later.",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}