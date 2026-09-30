package limiter

import (
	"sync"
	"time"
)

type ClientBucket struct {
	tokens     float64
	lastRefill time.Time
}

type RateLimiter struct {
	mu       sync.Mutex
	clients  map[string]*ClientBucket
	rate     float64       // Cookies added per second
	capacity float64       // Max cookies the jar can hold
	cleanup  time.Duration // Time after which empty/inactive jars are thrown away
}

func NewRateLimiter(rate float64, capacity float64) *RateLimiter {
	rl := &RateLimiter{
		clients:  make(map[string]*ClientBucket),
		rate:     rate,
		capacity: capacity,
		cleanup:  3 * time.Minute,
	}

	// Background worker: clean up old jars so memory stays clean
	go rl.startCleanupDaemon()

	return rl
}

// Allow checks if the friend has enough cookies to ask a question
func (rl *RateLimiter) Allow(clientIP string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	bucket, exists := rl.clients[clientIP]

	if !exists {
		// First visit: give them a full jar minus the 1 cookie they eat right now
		rl.clients[clientIP] = &ClientBucket{
			tokens:     rl.capacity - 1.0,
			lastRefill: now,
		}
		return true
	}

	// 1. Refill cookies based on how much time has passed
	elapsed := now.Sub(bucket.lastRefill).Seconds()
	bucket.tokens += elapsed * rl.rate
	if bucket.tokens > rl.capacity {
		bucket.tokens = rl.capacity
	}
	bucket.lastRefill = now

	// 2. Eat 1 cookie if available
	if bucket.tokens >= 1.0 {
		bucket.tokens -= 1.0
		return true
	}

	// No cookies left! Blocked!
	return false
}

func (rl *RateLimiter) startCleanupDaemon() {
	ticker := time.NewTicker(1 * time.Minute)
	for range ticker.C {
		rl.mu.Lock()
		for ip, bucket := range rl.clients {
			if time.Since(bucket.lastRefill) > rl.cleanup {
				delete(rl.clients, ip)
			}
		}
		rl.mu.Unlock()
	}
}