package middleware

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRateLimiter_Allow(t *testing.T) {
	limiter := NewRateLimiter(3, 1*time.Second)

	assert.True(t, limiter.Allow("192.168.1.1"), "first request should be allowed")
	assert.True(t, limiter.Allow("192.168.1.1"), "second request should be allowed")
	assert.True(t, limiter.Allow("192.168.1.1"), "third request should be allowed")
	assert.False(t, limiter.Allow("192.168.1.1"), "fourth request should be blocked")
}

func TestRateLimiter_DifferentIPs(t *testing.T) {
	limiter := NewRateLimiter(2, 1*time.Second)

	assert.True(t, limiter.Allow("192.168.1.1"))
	assert.True(t, limiter.Allow("192.168.1.1"))
	assert.False(t, limiter.Allow("192.168.1.1"))

	assert.True(t, limiter.Allow("10.0.0.1"), "different IP should have its own limit")
	assert.True(t, limiter.Allow("10.0.0.1"))
	assert.False(t, limiter.Allow("10.0.0.1"))
}

func TestRateLimiter_WindowExpiry(t *testing.T) {
	limiter := NewRateLimiter(2, 50*time.Millisecond)

	assert.True(t, limiter.Allow("192.168.1.1"))
	assert.True(t, limiter.Allow("192.168.1.1"))
	assert.False(t, limiter.Allow("192.168.1.1"))

	time.Sleep(60 * time.Millisecond)

	assert.True(t, limiter.Allow("192.168.1.1"), "request should be allowed after window expires")
}

func TestRateLimiter_Concurrent(t *testing.T) {
	limiter := NewRateLimiter(100, 1*time.Second)

	var wg sync.WaitGroup
	allowed := 0
	var mu sync.Mutex

	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if limiter.Allow("192.168.1.1") {
				mu.Lock()
				allowed++
				mu.Unlock()
			}
		}()
	}

	wg.Wait()
	assert.Equal(t, 100, allowed, "exactly 100 requests should be allowed")
}

func TestAuthRateLimiter_Limit(t *testing.T) {
	limiter := NewAuthRateLimiter()

	for i := 0; i < 5; i++ {
		assert.True(t, limiter.Allow("192.168.1.1"), "request %d should be allowed", i+1)
	}
	assert.False(t, limiter.Allow("192.168.1.1"), "6th request should be blocked")
}
