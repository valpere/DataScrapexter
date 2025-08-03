// internal/scraper/ratelimiter_test.go
package scraper

import (
	"context"
	"testing"
	"time"
)

func TestAdaptiveRateLimiter_FixedStrategy(t *testing.T) {
	config := &RateLimiterConfig{
		BaseInterval: 100 * time.Millisecond,
		BurstSize:    3,
		Strategy:     StrategyFixed,
	}

	rl := NewAdaptiveRateLimiter(config)

	// Should allow burst initially
	start := time.Now()
	for i := 0; i < 3; i++ {
		if !rl.Allow() {
			t.Errorf("Expected burst token %d to be allowed", i)
		}
	}

	// Next request should be limited
	if rl.Allow() {
		t.Error("Expected request to be rate limited after burst")
	}

	// Wait for interval and try again
	time.Sleep(120 * time.Millisecond)
	if !rl.Allow() {
		t.Error("Expected request to be allowed after interval")
	}

	duration := time.Since(start)
	if duration < 100*time.Millisecond {
		t.Errorf("Expected some delay, got %v", duration)
	}
}

func TestAdaptiveRateLimiter_AdaptiveStrategy(t *testing.T) {
	config := &RateLimiterConfig{
		BaseInterval:        50 * time.Millisecond,
		BurstSize:           2,
		Strategy:            StrategyAdaptive,
		MaxInterval:         500 * time.Millisecond,
		AdaptationRate:      0.5,
		AdaptationThreshold: 10 * time.Millisecond, // Fast adaptation for testing
		ErrorRateThreshold:  0.0,                   // Any errors trigger adaptation for testing
		ConsecutiveErrLimit: 2,                     // Lower threshold for testing
		MinChangeThreshold:  0.0,                   // No minimum change for testing
	}

	rl := NewAdaptiveRateLimiter(config)

	// Initial rate should be base rate
	stats := rl.GetStats()
	if stats.CurrentInterval != config.BaseInterval {
		t.Errorf("Expected initial interval %v, got %v", config.BaseInterval, stats.CurrentInterval)
	}

	// Report several errors to trigger adaptation
	for i := 0; i < 5; i++ {
		rl.ReportError()
	}

	// Allow time for adaptation and force it by calling Wait
	time.Sleep(120 * time.Millisecond) // Wait longer than adaptation threshold
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	rl.Wait(ctx) // Trigger adaptation

	stats = rl.GetStats()
	if stats.CurrentInterval <= config.BaseInterval {
		t.Errorf("Expected adapted interval to be higher than base %v, got %v",
			config.BaseInterval, stats.CurrentInterval)
	}

	// Report successes to improve rate
	for i := 0; i < 10; i++ {
		rl.ReportSuccess()
	}

	time.Sleep(60 * time.Millisecond)
	rl.Wait(context.Background()) // Trigger adaptation

	newStats := rl.GetStats()
	if newStats.ErrorRate >= stats.ErrorRate {
		t.Error("Expected error rate to decrease after successes")
	}
}

func TestAdaptiveRateLimiter_BurstStrategy(t *testing.T) {
	config := &RateLimiterConfig{
		BaseInterval:    200 * time.Millisecond,
		BurstSize:       5,
		Strategy:        StrategyBurst,
		BurstRefillRate: 300 * time.Millisecond,
	}

	rl := NewAdaptiveRateLimiter(config)

	// Should allow full burst immediately
	allowed := 0
	for i := 0; i < 10; i++ {
		if rl.Allow() {
			allowed++
		} else {
			break
		}
	}

	if allowed != config.BurstSize {
		t.Errorf("Expected %d burst tokens, got %d", config.BurstSize, allowed)
	}

	// Should not allow more until refill
	if rl.Allow() {
		t.Error("Expected request to be denied after burst exhaustion")
	}

	// Wait for burst refill
	time.Sleep(config.BurstRefillRate + 50*time.Millisecond)

	// Should allow burst again
	if !rl.Allow() {
		t.Error("Expected request to be allowed after burst refill")
	}
}

func TestAdaptiveRateLimiter_HybridStrategy(t *testing.T) {
	config := &RateLimiterConfig{
		BaseInterval:    100 * time.Millisecond,
		BurstSize:       3,
		Strategy:        StrategyHybrid,
		BurstRefillRate: 200 * time.Millisecond,
		MaxInterval:     1 * time.Second,
	}

	rl := NewAdaptiveRateLimiter(config)

	// Should use burst tokens first
	burstAllowed := 0
	for i := 0; i < config.BurstSize; i++ {
		if rl.Allow() {
			burstAllowed++
		}
	}

	if burstAllowed != config.BurstSize {
		t.Errorf("Expected %d burst requests, got %d", config.BurstSize, burstAllowed)
	}

	// Next requests should use adaptive rate limiting
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	err := rl.Wait(ctx)
	if err != nil {
		t.Errorf("Expected wait to succeed, got error: %v", err)
	}
}

func TestAdaptiveRateLimiter_ConsecutiveErrors(t *testing.T) {
	config := &RateLimiterConfig{
		BaseInterval:        50 * time.Millisecond,
		BurstSize:           2,
		Strategy:            StrategyAdaptive,
		MaxInterval:         500 * time.Millisecond,
		AdaptationRate:      0.5,
		AdaptationThreshold: 10 * time.Millisecond, // Fast adaptation for testing
		ErrorRateThreshold:  0.0,                   // Any errors trigger adaptation for testing
		ConsecutiveErrLimit: 2,                     // Lower threshold for testing
		MinChangeThreshold:  0.0,                   // No minimum change for testing
	}

	rl := NewAdaptiveRateLimiter(config)

	// Report many consecutive errors
	for i := 0; i < 8; i++ {
		rl.ReportError()
	}

	// Force adaptation with longer wait
	time.Sleep(120 * time.Millisecond)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	rl.Wait(ctx)

	stats := rl.GetStats()
	if stats.ConsecutiveErrs != 8 {
		t.Errorf("Expected 8 consecutive errors, got %d", stats.ConsecutiveErrs)
	}

	// Should have significantly increased interval
	if stats.CurrentInterval <= config.BaseInterval*2 {
		t.Errorf("Expected significant interval increase, got %v", stats.CurrentInterval)
	}

	// One success should reset consecutive errors
	rl.ReportSuccess()

	stats = rl.GetStats()
	if stats.ConsecutiveErrs != 0 {
		t.Errorf("Expected consecutive errors to reset after success, got %d", stats.ConsecutiveErrs)
	}
}

func TestAdaptiveRateLimiter_ContextCancellation(t *testing.T) {
	config := &RateLimiterConfig{
		BaseInterval: 1 * time.Second, // Long interval
		BurstSize:    1,
		Strategy:     StrategyFixed,
	}

	rl := NewAdaptiveRateLimiter(config)

	// Exhaust burst
	rl.Allow()

	// Create context with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	start := time.Now()
	err := rl.Wait(ctx)
	duration := time.Since(start)

	if err == nil {
		t.Error("Expected context cancellation error")
	}

	if duration > 200*time.Millisecond {
		t.Errorf("Expected quick cancellation, took %v", duration)
	}
}

func TestAdaptiveRateLimiter_Stats(t *testing.T) {
	config := &RateLimiterConfig{
		BaseInterval: 100 * time.Millisecond,
		BurstSize:    3,
		Strategy:     StrategyHybrid,
	}

	rl := NewAdaptiveRateLimiter(config)

	// Generate some activity
	rl.ReportSuccess()
	rl.ReportSuccess()
	rl.ReportError()
	rl.Allow() // Consume burst token

	stats := rl.GetStats()

	if stats.Strategy != StrategyHybrid {
		t.Errorf("Expected strategy %d, got %d", StrategyHybrid, stats.Strategy)
	}

	if stats.BaseInterval != config.BaseInterval {
		t.Errorf("Expected base interval %v, got %v", config.BaseInterval, stats.BaseInterval)
	}

	if stats.SuccessCount != 2 {
		t.Errorf("Expected 2 successes, got %d", stats.SuccessCount)
	}

	if stats.ErrorCount != 1 {
		t.Errorf("Expected 1 error, got %d", stats.ErrorCount)
	}

	expectedErrorRate := 1.0 / 3.0
	if stats.ErrorRate != expectedErrorRate {
		t.Errorf("Expected error rate %.2f, got %.2f", expectedErrorRate, stats.ErrorRate)
	}

	if stats.BurstTokens != 2 { // Started with 3, consumed 1
		t.Errorf("Expected 2 burst tokens remaining, got %d", stats.BurstTokens)
	}
}

func TestAdaptiveRateLimiter_Reset(t *testing.T) {
	config := &RateLimiterConfig{
		BaseInterval:        100 * time.Millisecond,
		BurstSize:           3,
		Strategy:            StrategyAdaptive,
		AdaptationThreshold: 10 * time.Millisecond, // Fast adaptation for testing
		ErrorRateThreshold:  0.0,                   // Any errors trigger adaptation for testing
		ConsecutiveErrLimit: 2,                     // Lower threshold for testing
		MinChangeThreshold:  0.0,                   // No minimum change for testing
	}

	rl := NewAdaptiveRateLimiter(config)

	// Generate some activity
	for i := 0; i < 5; i++ {
		rl.ReportError()
	}
	rl.Allow() // Consume burst token

	// Force adaptation
	time.Sleep(120 * time.Millisecond)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	rl.Wait(ctx)

	stats := rl.GetStats()
	if stats.ErrorCount == 0 || stats.CurrentInterval == config.BaseInterval {
		t.Error("Expected some activity before reset")
	}

	// Reset
	rl.Reset()

	newStats := rl.GetStats()
	if newStats.ErrorCount != 0 {
		t.Errorf("Expected error count to be reset to 0, got %d", newStats.ErrorCount)
	}

	if newStats.SuccessCount != 0 {
		t.Errorf("Expected success count to be reset to 0, got %d", newStats.SuccessCount)
	}

	if newStats.CurrentInterval != config.BaseInterval {
		t.Errorf("Expected interval to be reset to base %v, got %v",
			config.BaseInterval, newStats.CurrentInterval)
	}

	if newStats.BurstTokens != config.BurstSize {
		t.Errorf("Expected burst tokens to be reset to %d, got %d",
			config.BurstSize, newStats.BurstTokens)
	}
}

func TestAdaptiveRateLimiter_StrategyChange(t *testing.T) {
	config := &RateLimiterConfig{
		BaseInterval: 100 * time.Millisecond,
		BurstSize:    3,
		Strategy:     StrategyFixed,
	}

	rl := NewAdaptiveRateLimiter(config)

	stats := rl.GetStats()
	if stats.Strategy != StrategyFixed {
		t.Errorf("Expected initial strategy %d, got %d", StrategyFixed, stats.Strategy)
	}

	// Change strategy
	rl.SetStrategy(StrategyAdaptive)

	newStats := rl.GetStats()
	if newStats.Strategy != StrategyAdaptive {
		t.Errorf("Expected new strategy %d, got %d", StrategyAdaptive, newStats.Strategy)
	}
}

func BenchmarkAdaptiveRateLimiter_Allow(b *testing.B) {
	config := &RateLimiterConfig{
		BaseInterval: 1 * time.Millisecond,
		BurstSize:    1000,
		Strategy:     StrategyFixed,
	}

	rl := NewAdaptiveRateLimiter(config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rl.Allow()
	}
}

func BenchmarkAdaptiveRateLimiter_Wait(b *testing.B) {
	config := &RateLimiterConfig{
		BaseInterval: 1 * time.Microsecond, // Very fast for benchmarking
		BurstSize:    b.N,
		Strategy:     StrategyFixed,
	}

	rl := NewAdaptiveRateLimiter(config)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rl.Wait(ctx)
	}
}

func TestAdaptiveRateLimiter_DefaultConstants(t *testing.T) {
	// Test nil config uses defaults
	rl := NewAdaptiveRateLimiter(nil)
	stats := rl.GetStats()

	if stats.BaseInterval != DefaultBaseInterval {
		t.Errorf("Expected base interval %v, got %v", DefaultBaseInterval, stats.BaseInterval)
	}
	if stats.BaseBurstSize != DefaultBurstSize {
		t.Errorf("Expected burst size %d, got %d", DefaultBurstSize, stats.BaseBurstSize)
	}
	if stats.Strategy != DefaultStrategy {
		t.Errorf("Expected strategy %d, got %d", DefaultStrategy, stats.Strategy)
	}
}

func TestAdaptiveRateLimiter_ApplyDefaults(t *testing.T) {
	// Test partial config gets defaults applied
	config := &RateLimiterConfig{
		BaseInterval: 500 * time.Millisecond,
		BurstSize:    3,
		// Other fields should get defaults
	}

	rl := NewAdaptiveRateLimiter(config)

	if rl.errorRateThreshold != DefaultErrorRateThreshold {
		t.Errorf("Expected error rate threshold %v, got %v", DefaultErrorRateThreshold, rl.errorRateThreshold)
	}
	if rl.consecutiveErrLimit != DefaultConsecutiveErrLimit {
		t.Errorf("Expected consecutive error limit %d, got %d", DefaultConsecutiveErrLimit, rl.consecutiveErrLimit)
	}
	if rl.minChangeThreshold != DefaultMinChangeThreshold {
		t.Errorf("Expected min change threshold %v, got %v", DefaultMinChangeThreshold, rl.minChangeThreshold)
	}
}

func TestAdaptiveRateLimiter_ResetMemoryManagement(t *testing.T) {
	rl := NewAdaptiveRateLimiter(nil)

	// Add some errors to create the slice
	for i := 0; i < 10; i++ {
		rl.ReportError()
	}

	stats := rl.GetStats()
	if stats.ErrorCount == 0 {
		t.Error("Expected some errors before reset")
	}

	// Reset should clear everything and free memory
	rl.Reset()

	// Verify reset worked
	newStats := rl.GetStats()
	if newStats.ErrorCount != 0 {
		t.Errorf("Expected error count to be 0 after reset, got %d", newStats.ErrorCount)
	}
	if newStats.RecentErrors != 0 {
		t.Errorf("Expected recent errors to be 0 after reset, got %d", newStats.RecentErrors)
	}

	// Health errors slice should be nil (memory freed)
	rl.healthMu.Lock()
	if rl.healthErrors != nil {
		t.Error("Expected health errors slice to be nil after reset for memory efficiency")
	}
	if rl.healthErrorCount != 0 {
		t.Errorf("Expected health error count to be 0 after reset, got %d", rl.healthErrorCount)
	}
	rl.healthMu.Unlock()
}

func TestAdaptiveRateLimiter_HealthErrorsMemoryProtection(t *testing.T) {
	rl := NewAdaptiveRateLimiter(nil)

	// Report more errors than the maximum to test memory protection
	errorCount := MaxHealthErrors + 100
	for i := 0; i < errorCount; i++ {
		rl.ReportError()
	}

	rl.healthMu.Lock()
	actualCount := len(rl.healthErrors)
	rl.healthMu.Unlock()

	if actualCount > MaxHealthErrors {
		t.Errorf("Expected health errors to be capped at %d, got %d", MaxHealthErrors, actualCount)
	}

	// Should be around MaxHealthErrors * HealthErrorsRetentionRatio due to truncation logic
	expectedCount := int(float64(MaxHealthErrors) * HealthErrorsRetentionRatio)
	if actualCount < expectedCount-10 || actualCount > MaxHealthErrors {
		t.Errorf("Expected health errors count around %d, got %d", expectedCount, actualCount)
	}
}

func TestAdaptiveRateLimiter_PeriodicCleanup(t *testing.T) {
	config := &RateLimiterConfig{
		BaseInterval: 100 * time.Millisecond,
		HealthWindow: 50 * time.Millisecond, // Very short window for testing
	}
	rl := NewAdaptiveRateLimiter(config)

	// Add some errors
	for i := 0; i < 50; i++ {
		rl.ReportError()
	}

	// Wait for errors to expire
	time.Sleep(100 * time.Millisecond)

	// Add one more error to trigger cleanup (should hit cleanup interval)
	for i := 0; i < HealthCleanupInterval; i++ {
		rl.ReportError()
	}

	rl.healthMu.Lock()
	recentCount := len(rl.healthErrors)
	rl.healthMu.Unlock()

	// Should only have recent errors (within the health window)
	if recentCount > HealthCleanupInterval+10 { // Allow some tolerance
		t.Errorf("Expected cleanup to remove old errors, still have %d errors", recentCount)
	}
}

func TestAdaptiveRateLimiter_ConsecutiveErrorMultiplier(t *testing.T) {
	config := &RateLimiterConfig{
		BaseInterval:        100 * time.Millisecond,
		BurstSize:           2,
		Strategy:            StrategyAdaptive,
		MaxInterval:         10 * time.Second,
		AdaptationRate:      0.5,
		AdaptationThreshold: 10 * time.Millisecond, // Fast adaptation for testing
		ErrorRateThreshold:  0.0,                   // Any errors trigger adaptation for testing
		ConsecutiveErrLimit: 5,                     // Test threshold
		MinChangeThreshold:  0.0,                   // No minimum change for testing
	}

	rl := NewAdaptiveRateLimiter(config)

	// Report more consecutive errors than the limit to test multiplier logic
	for i := 0; i < 15; i++ { // 3x the consecutive error limit
		rl.ReportError()
	}

	// Force adaptation
	time.Sleep(50 * time.Millisecond)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	rl.Wait(ctx)

	stats := rl.GetStats()

	// Verify consecutive errors are tracked correctly
	if stats.ConsecutiveErrs != 15 {
		t.Errorf("Expected 15 consecutive errors, got %d", stats.ConsecutiveErrs)
	}

	// Verify that the interval has increased significantly due to consecutive errors
	// With 15 consecutive errors and limit of 5, we should have a ratio of 3.0
	// which should be applied as a multiplier
	expectedMinInterval := time.Duration(float64(config.BaseInterval) * 3.0) // At least 3x slower
	if stats.CurrentInterval < expectedMinInterval {
		t.Errorf("Expected interval >= %v due to consecutive errors, got %v",
			expectedMinInterval, stats.CurrentInterval)
	}
}
