package limiter

import (
	"sync"
	"testing"
	"time"
)

func TestRateLimiter_BurstCapacity(t *testing.T) {
	// Refill: 1 cookie/sec, Capacity: 3 cookies
	rl := NewRateLimiter(1.0, 3.0)
	client := "192.168.1.1"

	// First 3 questions must pass (eats all 3 cookies)
	for i := 1; i <= 3; i++ {
		if !rl.Allow(client) {
			t.Fatalf("request %d should have been allowed", i)
		}
	}

	// 4th question right away must fail (jar is empty!)
	if rl.Allow(client) {
		t.Fatalf("request 4 should have been blocked")
	}

	// Wait 1.1 seconds for 1 new cookie to bake
	time.Sleep(1100 * time.Millisecond)

	// Now it should pass!
	if !rl.Allow(client) {
		t.Fatalf("request after waiting should have succeeded")
	}
}

func TestRateLimiter_ConcurrentSafety(t *testing.T) {
	// 50 cookies capacity
	rl := NewRateLimiter(10.0, 50.0)
	var wg sync.WaitGroup
	var mu sync.Mutex
	allowedCount := 0

	// 100 people grab cookies at the exact same millisecond
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if rl.Allow("10.0.0.1") {
				mu.Lock()
				allowedCount++
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	// Only 50 should have gotten a cookie
	if allowedCount != 50 {
		t.Errorf("expected 50 allowed requests, got %d", allowedCount)
	}
}