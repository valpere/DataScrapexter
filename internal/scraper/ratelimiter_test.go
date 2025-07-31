// internal/scraper/ratelimiter_test.go
package scraper

import (
	"context"
	"testing"
	"time"
)

func TestAdaptiveRateLimiter_FixedStrategy(t *testing.T) {
	config := &RateLimiterConfig{
		BaseInterval:    100 * time.Millisecond,
		BurstSize:       3,
		Strategy:        StrategyFixed,
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
		BaseInterval:         50 * time.Millisecond,
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
		BaseInterval:         50 * time.Millisecond,
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
		BaseInterval:         100 * time.Millisecond,
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