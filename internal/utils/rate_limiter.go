// internal/utils/rate_limiter.go

package utils

import (
	"context"
	"sync"
	"time"
)

// RateLimiter provides a simple token bucket rate limiter implementation.
type RateLimiter struct {
	rate       float64
	burst      int
	tokens     float64
	lastUpdate time.Time
	mu         sync.Mutex
}

// NewRateLimiter creates a new rate limiter with the specified rate (requests per second).
func NewRateLimiter(rate float64) *RateLimiter {
	if rate <= 0 {
		rate = 1
	}
	
	return &RateLimiter{
		rate:       rate,
		burst:      int(rate),
		tokens:     rate,
		lastUpdate: time.Now(),
	}
}

// Wait blocks until a token is available or the context is cancelled.
func (r *RateLimiter) Wait(ctx context.Context) error {
	for {
		if r.TryAcquire() {
			return nil
		}
		
		waitTime := r.waitTime()
		
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(waitTime):
			// Try again
		}
	}
}

// TryAcquire attempts to acquire a token without blocking.
func (r *RateLimiter) TryAcquire() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	r.updateTokens()
	
	if r.tokens >= 1 {
		r.tokens--
		return true
	}
	
	return false
}

// updateTokens adds tokens based on time elapsed since last update.
func (r *RateLimiter) updateTokens() {
	now := time.Now()
	elapsed := now.Sub(r.lastUpdate).Seconds()
	
	r.tokens += elapsed * r.rate
	
	if r.tokens > float64(r.burst) {
		r.tokens = float64(r.burst)
	}
	
	r.lastUpdate = now
}

// waitTime calculates how long to wait for the next token.
func (r *RateLimiter) waitTime() time.Duration {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if r.tokens >= 1 {
		return 0
	}
	
	tokensNeeded := 1 - r.tokens
	secondsToWait := tokensNeeded / r.rate
	
	return time.Duration(secondsToWait * float64(time.Second))
}

// SetRate updates the rate limiter's rate (requests per second).
func (r *RateLimiter) SetRate(rate float64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if rate <= 0 {
		rate = 1
	}
	
	r.rate = rate
	r.burst = int(rate)
}
