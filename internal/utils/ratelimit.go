package utils

import (
	"sync"
	"time"
)

// RateLimiter implements a simple token bucket rate limiter
type RateLimiter struct {
	rate       int // requests per second
	tokens     int // available tokens
	maxTokens  int // max burst size
	lastRefill time.Time
	mu         sync.Mutex
}

// NewRateLimiter creates a new rate limiter
// rate: requests per second (0 = unlimited)
func NewRateLimiter(rate int) *RateLimiter {
	if rate <= 0 {
		return nil
	}
	return &RateLimiter{
		rate:       rate,
		tokens:     rate,
		maxTokens:  rate,
		lastRefill: time.Now(),
	}
}

// Wait blocks until a token is available
func (rl *RateLimiter) Wait() {
	if rl == nil {
		return
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.refill()

	for rl.tokens <= 0 {
		waitTime := time.Second / time.Duration(rl.rate)
		time.Sleep(waitTime)
		rl.refill()
	}

	rl.tokens--
}

// TryWait tries to get a token without blocking, returns true if successful
func (rl *RateLimiter) TryWait() bool {
	if rl == nil {
		return true
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.refill()

	if rl.tokens > 0 {
		rl.tokens--
		return true
	}
	return false
}

// refill adds tokens based on elapsed time
func (rl *RateLimiter) refill() {
	now := time.Now()
	elapsed := now.Sub(rl.lastRefill)
	tokensToAdd := int(elapsed.Seconds() * float64(rl.rate))

	if tokensToAdd > 0 {
		rl.tokens += tokensToAdd
		if rl.tokens > rl.maxTokens {
			rl.tokens = rl.maxTokens
		}
		rl.lastRefill = now
	}
}
