// internal/scraper/ratelimiter.go
package scraper

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// Default configuration constants
const (
	DefaultBaseInterval         = 1 * time.Second
	DefaultBurstSize           = 5
	DefaultMaxInterval         = 30 * time.Second
	DefaultAdaptationRate      = 0.5
	DefaultStrategy            = StrategyHybrid
	DefaultBurstRefillRate     = 10 * time.Second
	DefaultHealthWindow        = 5 * time.Minute
	DefaultAdaptationThreshold = 1 * time.Second
	DefaultErrorRateThreshold  = 0.1  // 10% error rate
	DefaultConsecutiveErrLimit = 5
	DefaultMinChangeThreshold  = 0.1  // 10% minimum change
)

// Adaptation behavior constants
const (
	ErrorRateMultiplier        = 3.0  // Up to 4x slower at 100% error rate (1 + 3)
	BurstIncreaseThreshold     = 0.05 // 5% error rate - allow larger bursts
	BurstDecreaseThreshold     = 0.2  // 20% error rate - reduce bursts
	BurstIncreaseMultiplier    = 1.5  // Increase burst size by 50%
	BurstDecreaseMultiplier    = 0.5  // Decrease burst size by 50%
	// Caps the rate slowdown caused by consecutive errors to prevent excessive delays
	// in the adaptation algorithm.
	MaxConsecutiveMultiplier   = 10.0
)

// Health tracking efficiency constants
const (
	MaxHealthErrors           = 1000  // Maximum health errors to track (memory protection)
	HealthCleanupInterval     = 100   // Clean up after every N error reports
	HealthErrorsRetentionRatio = 0.5  // Retain 50% of entries when truncating to avoid frequent re-truncation
)

// AdaptiveRateLimiter provides enhanced rate limiting with burst control and adaptive delays
type AdaptiveRateLimiter struct {
	// Core rate limiting
	limiter *rate.Limiter
	mu      sync.RWMutex
	
	// Configuration
	baseInterval        time.Duration
	baseBurstSize       int
	maxInterval         time.Duration
	adaptationRate      float64
	adaptationThreshold time.Duration
	errorRateThreshold  float64
	consecutiveErrLimit int
	minChangeThreshold  float64
	
	// Adaptive behavior
	errorCount      int
	successCount    int
	consecutiveErrs int
	lastAdaptation  time.Time
	currentInterval time.Duration
	currentBurst    int
	
	// Burst control
	burstTokens     int
	burstRefillRate time.Duration
	lastBurstRefill time.Time
	burstMu         sync.Mutex
	
	// Rate limiting strategies
	strategy        RateLimitStrategy
	
	// Health tracking
	healthWindow    time.Duration
	healthErrors    []time.Time
	healthErrorCount int  // Counter for cleanup efficiency
	healthMu        sync.Mutex
}

// RateLimitStrategy defines different rate limiting approaches
type RateLimitStrategy int

const (
	StrategyFixed    RateLimitStrategy = iota // Fixed rate limiting
	StrategyAdaptive                          // Adaptive based on errors
	StrategyBurst                            // Burst-aware limiting
	StrategyHybrid                           // Combination of adaptive and burst
)

// RateLimiterConfig configures the adaptive rate limiter
type RateLimiterConfig struct {
	BaseInterval         time.Duration     `yaml:"base_interval" json:"base_interval"`
	BurstSize            int               `yaml:"burst_size" json:"burst_size"`
	MaxInterval          time.Duration     `yaml:"max_interval" json:"max_interval"`
	AdaptationRate       float64           `yaml:"adaptation_rate" json:"adaptation_rate"`
	Strategy             RateLimitStrategy `yaml:"strategy" json:"strategy"`
	BurstRefillRate      time.Duration     `yaml:"burst_refill_rate" json:"burst_refill_rate"`
	HealthWindow         time.Duration     `yaml:"health_window" json:"health_window"`
	
	// Adaptation sensitivity controls
	AdaptationThreshold  time.Duration     `yaml:"adaptation_threshold" json:"adaptation_threshold"`   // Minimum time between adaptations
	ErrorRateThreshold   float64           `yaml:"error_rate_threshold" json:"error_rate_threshold"`   // Error rate that triggers adaptation
	ConsecutiveErrLimit  int               `yaml:"consecutive_err_limit" json:"consecutive_err_limit"` // Consecutive errors threshold
	MinChangeThreshold   float64           `yaml:"min_change_threshold" json:"min_change_threshold"`   // Minimum rate change percentage
}

// getDefaultConfig returns a configuration with production-safe defaults
func getDefaultConfig() *RateLimiterConfig {
	return &RateLimiterConfig{
		BaseInterval:         DefaultBaseInterval,
		BurstSize:           DefaultBurstSize,
		MaxInterval:         DefaultMaxInterval,
		AdaptationRate:      DefaultAdaptationRate,
		Strategy:            DefaultStrategy,
		BurstRefillRate:     DefaultBurstRefillRate,
		HealthWindow:        DefaultHealthWindow,
		AdaptationThreshold: DefaultAdaptationThreshold,
		ErrorRateThreshold:  DefaultErrorRateThreshold,
		ConsecutiveErrLimit: DefaultConsecutiveErrLimit,
		MinChangeThreshold:  DefaultMinChangeThreshold,
	}
}

// applyDefaults fills in missing configuration values with production-safe defaults
func applyDefaults(config *RateLimiterConfig) {
	if config.AdaptationThreshold == 0 {
		config.AdaptationThreshold = DefaultAdaptationThreshold
	}
	if config.ErrorRateThreshold == 0 {
		config.ErrorRateThreshold = DefaultErrorRateThreshold
	}
	if config.ConsecutiveErrLimit == 0 {
		config.ConsecutiveErrLimit = DefaultConsecutiveErrLimit
	}
	if config.MinChangeThreshold == 0 {
		config.MinChangeThreshold = DefaultMinChangeThreshold
	}
}

// NewAdaptiveRateLimiter creates a new adaptive rate limiter
func NewAdaptiveRateLimiter(config *RateLimiterConfig) *AdaptiveRateLimiter {
	if config == nil {
		config = getDefaultConfig()
	} else {
		// Apply defaults for any missing values
		applyDefaults(config)
	}

	rl := &AdaptiveRateLimiter{
		baseInterval:        config.BaseInterval,
		baseBurstSize:       config.BurstSize,
		maxInterval:         config.MaxInterval,
		adaptationRate:      config.AdaptationRate,
		adaptationThreshold: config.AdaptationThreshold,
		errorRateThreshold:  config.ErrorRateThreshold,
		consecutiveErrLimit: config.ConsecutiveErrLimit,
		minChangeThreshold:  config.MinChangeThreshold,
		strategy:            config.Strategy,
		burstRefillRate:     config.BurstRefillRate,
		healthWindow:        config.HealthWindow,
		
		currentInterval: config.BaseInterval,
		currentBurst:    config.BurstSize,
		burstTokens:     config.BurstSize,
		lastBurstRefill: time.Now(),
		lastAdaptation:  time.Now(),
	}

	rl.limiter = rate.NewLimiter(rate.Every(config.BaseInterval), config.BurstSize)
	return rl
}

// Wait blocks until the rate limiter allows the operation
func (rl *AdaptiveRateLimiter) Wait(ctx context.Context) error {
	return rl.WaitN(ctx, 1)
}

// WaitN blocks until n operations are allowed
func (rl *AdaptiveRateLimiter) WaitN(ctx context.Context, n int) error {
	switch rl.strategy {
	case StrategyFixed:
		return rl.waitFixed(ctx, n)
	case StrategyAdaptive:
		return rl.waitAdaptive(ctx, n)
	case StrategyBurst:
		return rl.waitBurst(ctx, n)
	case StrategyHybrid:
		return rl.waitHybrid(ctx, n)
	default:
		return rl.waitFixed(ctx, n)
	}
}

// waitFixed implements fixed rate limiting
func (rl *AdaptiveRateLimiter) waitFixed(ctx context.Context, n int) error {
	rl.mu.RLock()
	limiter := rl.limiter
	rl.mu.RUnlock()
	
	return limiter.WaitN(ctx, n)
}

// waitAdaptive implements adaptive rate limiting based on error rates
func (rl *AdaptiveRateLimiter) waitAdaptive(ctx context.Context, n int) error {
	rl.updateAdaptiveRate()
	
	rl.mu.RLock()
	limiter := rl.limiter
	rl.mu.RUnlock()
	
	return limiter.WaitN(ctx, n)
}

// waitBurst implements burst-aware rate limiting
func (rl *AdaptiveRateLimiter) waitBurst(ctx context.Context, n int) error {
	// Try to consume burst tokens first
	if rl.tryConsumeBurstTokens(n) {
		return nil
	}
	
	// Fall back to regular rate limiting
	return rl.waitFixed(ctx, n)
}

// waitHybrid implements hybrid rate limiting (adaptive + burst)
func (rl *AdaptiveRateLimiter) waitHybrid(ctx context.Context, n int) error {
	// Update adaptive rate based on recent performance
	rl.updateAdaptiveRate()
	
	// Try burst tokens first if available
	if rl.tryConsumeBurstTokens(n) {
		return nil
	}
	
	// Use adaptive rate limiting
	rl.mu.RLock()
	limiter := rl.limiter
	rl.mu.RUnlock()
	
	return limiter.WaitN(ctx, n)
}

// Allow checks if an operation is allowed without blocking
func (rl *AdaptiveRateLimiter) Allow() bool {
	return rl.AllowN(1)
}

// AllowN checks if n operations are allowed without blocking
func (rl *AdaptiveRateLimiter) AllowN(n int) bool {
	switch rl.strategy {
	case StrategyBurst:
		// For pure burst strategy, only use burst tokens
		return rl.tryConsumeBurstTokens(n)
	case StrategyHybrid:
		// Try burst tokens first
		if rl.tryConsumeBurstTokens(n) {
			return true
		}
		// Fall back to rate limiter
		break
	}
	
	rl.mu.RLock()
	allowed := rl.limiter.AllowN(time.Now(), n)
	rl.mu.RUnlock()
	
	return allowed
}

// ReportSuccess reports a successful operation for adaptive behavior
func (rl *AdaptiveRateLimiter) ReportSuccess() {
	rl.mu.Lock()
	rl.successCount++
	rl.consecutiveErrs = 0
	rl.mu.Unlock()
}

// ReportError reports a failed operation for adaptive behavior
func (rl *AdaptiveRateLimiter) ReportError() {
	rl.mu.Lock()
	rl.errorCount++
	rl.consecutiveErrs++
	rl.mu.Unlock()
	
	// Track for health window with efficient memory management
	rl.healthMu.Lock()
	now := time.Now()
	
	// Add new error
	rl.healthErrors = append(rl.healthErrors, now)
	rl.healthErrorCount++
	
	// Implement memory protection: enforce maximum size
	if len(rl.healthErrors) > MaxHealthErrors {
		// Keep only the most recent entries based on retention ratio. This approach minimizes 
		// the need for frequent re-truncation, which can be computationally expensive, by 
		// proactively retaining only the most relevant entries.
		keepCount := int(float64(MaxHealthErrors) * HealthErrorsRetentionRatio)
		copy(rl.healthErrors, rl.healthErrors[len(rl.healthErrors)-keepCount:])
		rl.healthErrors = rl.healthErrors[:keepCount]
	}
	
	// Periodic cleanup based on counter (more efficient than every time)
	if rl.healthErrorCount%HealthCleanupInterval == 0 {
		rl.cleanupHealthErrors(now)
	}
	
	rl.healthMu.Unlock()
}

// cleanupHealthErrors removes expired errors from the health tracking slice
// Must be called with healthMu held
func (rl *AdaptiveRateLimiter) cleanupHealthErrors(now time.Time) {
	cutoff := now.Add(-rl.healthWindow)
	writeIndex := 0
	
	// Use in-place filtering to avoid slice allocation
	for readIndex := 0; readIndex < len(rl.healthErrors); readIndex++ {
		if rl.healthErrors[readIndex].After(cutoff) {
			rl.healthErrors[writeIndex] = rl.healthErrors[readIndex]
			writeIndex++
		}
	}
	
	// Truncate slice to new size and clear unused entries to prevent memory leaks
	for i := writeIndex; i < len(rl.healthErrors); i++ {
		rl.healthErrors[i] = time.Time{} // Zero value to help GC
	}
	rl.healthErrors = rl.healthErrors[:writeIndex]
}

// tryConsumeBurstTokens attempts to consume burst tokens
func (rl *AdaptiveRateLimiter) tryConsumeBurstTokens(n int) bool {
	rl.burstMu.Lock()
	defer rl.burstMu.Unlock()
	
	// Refill burst tokens if enough time has passed
	now := time.Now()
	if now.Sub(rl.lastBurstRefill) >= rl.burstRefillRate {
		rl.burstTokens = rl.currentBurst
		rl.lastBurstRefill = now
	}
	
	// Check if we have enough tokens
	if rl.burstTokens >= n {
		rl.burstTokens -= n
		return true
	}
	
	return false
}

// updateAdaptiveRate updates the rate limiter based on recent error patterns
func (rl *AdaptiveRateLimiter) updateAdaptiveRate() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	
	now := time.Now()
	if now.Sub(rl.lastAdaptation) < rl.adaptationThreshold {
		return // Don't adapt too frequently
	}
	
	rl.lastAdaptation = now
	
	totalOperations := rl.successCount + rl.errorCount
	if totalOperations == 0 {
		return
	}
	
	// Calculate error rate from total operations
	errorRate := float64(rl.errorCount) / float64(totalOperations)
	
	// Adjust rate based on error rate and consecutive errors
	var multiplier float64 = 1.0
	
	// Increase delays only if error rate exceeds threshold
	if errorRate > rl.errorRateThreshold {
		multiplier = 1 + (errorRate * ErrorRateMultiplier) // Up to 4x slower at 100% error rate
	}
	
	// Additional penalty for consecutive errors
	if rl.consecutiveErrs > rl.consecutiveErrLimit {
		// Calculate ratio and apply it as an additional multiplier
		consecutiveRatio := float64(rl.consecutiveErrs) / float64(rl.consecutiveErrLimit)
		consecutiveMultiplier := math.Min(consecutiveRatio, MaxConsecutiveMultiplier)
		multiplier *= consecutiveMultiplier
	}
	
	// Calculate new interval
	newInterval := time.Duration(float64(rl.baseInterval) * multiplier)
	if newInterval > rl.maxInterval {
		newInterval = rl.maxInterval
	}
	
	// Only update if change is significant enough
	changeRatio := math.Abs(float64(newInterval-rl.currentInterval)) / float64(rl.currentInterval)
	if changeRatio >= rl.minChangeThreshold {
		rl.currentInterval = newInterval
		rl.limiter.SetLimit(rate.Every(newInterval))
	}
	
	// Adjust burst size based on performance
	newBurst := rl.baseBurstSize
	if errorRate < BurstIncreaseThreshold { // Less than 5% errors - allow larger bursts
		newBurst = int(float64(rl.baseBurstSize) * BurstIncreaseMultiplier)
	} else if errorRate > BurstDecreaseThreshold { // More than 20% errors - reduce bursts
		newBurst = int(float64(rl.baseBurstSize) * BurstDecreaseMultiplier)
		if newBurst < 1 {
			newBurst = 1
		}
	}
	
	if newBurst != rl.currentBurst {
		rl.currentBurst = newBurst
		rl.limiter.SetBurst(newBurst)
	}
}

// GetStats returns current rate limiter statistics
func (rl *AdaptiveRateLimiter) GetStats() *RateLimiterStats {
	rl.mu.RLock()
	rl.healthMu.Lock()
	
	stats := &RateLimiterStats{
		Strategy:         rl.strategy,
		BaseInterval:     rl.baseInterval,
		CurrentInterval:  rl.currentInterval,
		BaseBurstSize:    rl.baseBurstSize,
		CurrentBurstSize: rl.currentBurst,
		SuccessCount:     rl.successCount,
		ErrorCount:       rl.errorCount,
		ConsecutiveErrs:  rl.consecutiveErrs,
		RecentErrors:     len(rl.healthErrors),
		BurstTokens:      rl.burstTokens,
	}
	
	if rl.successCount+rl.errorCount > 0 {
		stats.ErrorRate = float64(rl.errorCount) / float64(rl.successCount+rl.errorCount)
	}
	
	rl.healthMu.Unlock()
	rl.mu.RUnlock()
	return stats
}

// RateLimiterStats contains rate limiter performance statistics
type RateLimiterStats struct {
	Strategy         RateLimitStrategy `json:"strategy"`
	BaseInterval     time.Duration     `json:"base_interval"`
	CurrentInterval  time.Duration     `json:"current_interval"`
	BaseBurstSize    int               `json:"base_burst_size"`
	CurrentBurstSize int               `json:"current_burst_size"`
	SuccessCount     int               `json:"success_count"`
	ErrorCount       int               `json:"error_count"`
	ConsecutiveErrs  int               `json:"consecutive_errors"`
	RecentErrors     int               `json:"recent_errors"`
	ErrorRate        float64           `json:"error_rate"`
	BurstTokens      int               `json:"burst_tokens"`
}

// Reset resets the rate limiter statistics
func (rl *AdaptiveRateLimiter) Reset() {
	rl.mu.Lock()
	rl.errorCount = 0
	rl.successCount = 0
	rl.consecutiveErrs = 0
	rl.currentInterval = rl.baseInterval
	rl.currentBurst = rl.baseBurstSize
	rl.burstTokens = rl.baseBurstSize
	rl.limiter.SetLimit(rate.Every(rl.baseInterval))
	rl.limiter.SetBurst(rl.baseBurstSize)
	rl.mu.Unlock()
	
	rl.healthMu.Lock()
	// Clear health errors - use nil to free memory since this is a reset operation
	// and we don't expect frequent resets that would benefit from capacity retention
	rl.healthErrors = nil
	rl.healthErrorCount = 0
	rl.healthMu.Unlock()
}

// SetStrategy changes the rate limiting strategy
func (rl *AdaptiveRateLimiter) SetStrategy(strategy RateLimitStrategy) {
	rl.mu.Lock()
	rl.strategy = strategy
	rl.mu.Unlock()
}

// GetCurrentRate returns the current rate limit
func (rl *AdaptiveRateLimiter) GetCurrentRate() (interval time.Duration, burst int) {
	rl.mu.RLock()
	defer rl.mu.RUnlock()
	return rl.currentInterval, rl.currentBurst
}

// String returns a string representation of the rate limiter
func (rl *AdaptiveRateLimiter) String() string {
	stats := rl.GetStats()
	return fmt.Sprintf("AdaptiveRateLimiter(strategy=%d, interval=%v, burst=%d, errors=%d/%d, rate=%.2f%%)",
		stats.Strategy, stats.CurrentInterval, stats.CurrentBurstSize,
		stats.ErrorCount, stats.SuccessCount+stats.ErrorCount, stats.ErrorRate*100)
}