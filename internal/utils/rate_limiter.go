// internal/utils/rate_limiter.go
package utils

import (
	"context"

	"golang.org/x/time/rate"
)

// RateLimiter wraps the golang.org/x/time/rate limiter
type RateLimiter struct {
	limiter *rate.Limiter
}

// NewRateLimiter creates a new rate limiter with the given rate (requests per second)
func NewRateLimiter(requestsPerSecond float64) *RateLimiter {
	return &RateLimiter{
		limiter: rate.NewLimiter(rate.Limit(requestsPerSecond), 1),
	}
}

// Wait blocks until the rate limiter allows the next request
func (rl *RateLimiter) Wait(ctx context.Context) error {
	return rl.limiter.Wait(ctx)
}

// Allow reports whether an event may happen now
func (rl *RateLimiter) Allow() bool {
	return rl.limiter.Allow()
}

// Reserve returns a Reservation that indicates how long the caller must wait before the next request
func (rl *RateLimiter) Reserve() *rate.Reservation {
	return rl.limiter.Reserve()
}

// SetLimit changes the rate limit
func (rl *RateLimiter) SetLimit(newLimit rate.Limit) {
	rl.limiter.SetLimit(newLimit)
}

// SetBurst changes the burst size
func (rl *RateLimiter) SetBurst(newBurst int) {
	rl.limiter.SetBurst(newBurst)
}
