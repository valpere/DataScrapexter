// internal/errors/service_test.go
package errors

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"
)

// Test configuration constants
const (
	TestCircuitBreakerResetTimeout = 100 * time.Millisecond // Short timeout for circuit breaker tests
	TestSlowOperationTimeout       = 100 * time.Millisecond // Timeout for slow operation simulation
)

func TestService_ExecuteWithRecovery_Success(t *testing.T) {
	service := NewService()
	ctx := context.Background()

	operation := func() (interface{}, error) {
		return "success", nil
	}

	result := service.ExecuteWithRecovery(ctx, "test_operation", operation)

	if !result.Success {
		t.Error("Expected operation to succeed")
	}
	if result.UsedFallback {
		t.Error("Expected no fallback to be used")
	}
	if result.AttemptCount != 1 {
		t.Errorf("Expected 1 attempt, got %d", result.AttemptCount)
	}
	if result.Result != "success" {
		t.Errorf("Expected result 'success', got %v", result.Result)
	}
}

func TestService_ExecuteWithRecovery_RetrySuccess(t *testing.T) {
	service := NewService()
	ctx := context.Background()

	attemptCount := 0
	operation := func() (interface{}, error) {
		attemptCount++
		if attemptCount < 3 {
			return nil, fmt.Errorf("temporary error")
		}
		return "eventual_success", nil
	}

	result := service.ExecuteWithRecovery(ctx, "retry_test", operation)

	if !result.Success {
		t.Error("Expected operation to eventually succeed")
	}
	if result.UsedFallback {
		t.Error("Expected no fallback to be used")
	}
	if result.AttemptCount != 3 {
		t.Errorf("Expected 3 attempts, got %d", result.AttemptCount)
	}
	if result.Result != "eventual_success" {
		t.Errorf("Expected result 'eventual_success', got %v", result.Result)
	}
}

func TestService_ExecuteWithRecovery_FallbackUsed(t *testing.T) {
	service := NewService()
	ctx := context.Background()

	// Configure fallback
	service.ConfigureFallback("fallback_test", FallbackConfig{
		Strategy:     FallbackDefault,
		DefaultValue: "fallback_result",
	})

	operation := func() (interface{}, error) {
		return nil, fmt.Errorf("persistent error")
	}

	result := service.ExecuteWithRecovery(ctx, "fallback_test", operation)

	if !result.Success {
		t.Error("Expected operation to succeed via fallback")
	}
	if !result.UsedFallback {
		t.Error("Expected fallback to be used")
	}
	if result.FallbackType != "retry_exhausted_fallback" {
		t.Errorf("Expected fallback type 'retry_exhausted_fallback', got %s", result.FallbackType)
	}
	if result.Result != "fallback_result" {
		t.Errorf("Expected fallback result 'fallback_result', got %v", result.Result)
	}
}

func TestService_ExecuteWithRecovery_CachedFallback(t *testing.T) {
	service := NewService()
	ctx := context.Background()

	// Configure cached fallback
	service.ConfigureFallback("cached_test", FallbackConfig{
		Strategy:     FallbackCached,
		CacheTimeout: time.Minute,
	})

	// First, execute successfully to populate cache
	successOperation := func() (interface{}, error) {
		return "cached_data", nil
	}
	
	result := service.ExecuteWithRecovery(ctx, "cached_test", successOperation)
	if !result.Success {
		t.Fatal("Expected first operation to succeed")
	}

	// Now fail and use cached result
	failOperation := func() (interface{}, error) {
		return nil, fmt.Errorf("service unavailable")
	}

	result = service.ExecuteWithRecovery(ctx, "cached_test", failOperation)

	if !result.Success {
		t.Error("Expected operation to succeed via cached fallback")
	}
	if !result.UsedFallback {
		t.Error("Expected fallback to be used")
	}
	if result.Result != "cached_data" {
		t.Errorf("Expected cached result 'cached_data', got %v", result.Result)
	}
}

func TestCircuitBreaker_BasicOperation(t *testing.T) {
	service := NewService()
	ctx := context.Background()

	// Configure circuit breaker with low threshold for testing
	service.ConfigureCircuitBreaker("circuit_test", CircuitBreakerConfig{
		MaxFailures:  2,
		ResetTimeout: time.Second,
	})

	failOperation := func() (interface{}, error) {
		return nil, fmt.Errorf("service error")
	}

	// First two attempts should proceed normally but fail
	for i := 0; i < 2; i++ {
		result := service.ExecuteWithRecovery(ctx, "circuit_test", failOperation)
		if result.Success {
			t.Errorf("Expected failure on attempt %d", i+1)
		}
		if result.UsedFallback {
			t.Errorf("Expected no fallback on attempt %d", i+1)
		}
	}

	// Third attempt should be blocked by circuit breaker
	result := service.ExecuteWithRecovery(ctx, "circuit_test", failOperation)
	if result.Success && !result.UsedFallback {
		// Circuit breaker should either block (fail) or allow fallback
		t.Error("Expected circuit breaker to affect third attempt")
	}
	if result.OriginalError == nil || !strings.Contains(result.OriginalError.Error(), "circuit breaker is open") {
		t.Error("Expected circuit breaker error")
	}
}

func TestCircuitBreaker_Recovery(t *testing.T) {
	service := NewService()
	ctx := context.Background()

	// Configure circuit breaker with short reset timeout
	service.ConfigureCircuitBreaker("recovery_test", CircuitBreakerConfig{
		MaxFailures:  1,
		ResetTimeout: TestCircuitBreakerResetTimeout,
	})

	// Fail once to open circuit
	failOperation := func() (interface{}, error) {
		return nil, fmt.Errorf("service error")
	}
	
	result := service.ExecuteWithRecovery(ctx, "recovery_test", failOperation)
	if result.Success {
		t.Error("Expected first operation to fail")
	}

	// Wait for circuit breaker reset
	time.Sleep(150 * time.Millisecond)

	// Should now allow execution and succeed
	successOperation := func() (interface{}, error) {
		return "recovered", nil
	}

	result = service.ExecuteWithRecovery(ctx, "recovery_test", successOperation)
	if !result.Success {
		t.Error("Expected operation to succeed after circuit breaker reset")
	}
	if result.Result != "recovered" {
		t.Errorf("Expected result 'recovered', got %v", result.Result)
	}
}

func TestService_ConfigureFallback_AllStrategies(t *testing.T) {
	service := NewService()
	ctx := context.Background()

	testCases := []struct {
		name     string
		config   FallbackConfig
		expected interface{}
	}{
		{
			name: "default_strategy",
			config: FallbackConfig{
				Strategy:     FallbackDefault,
				DefaultValue: "default_value",
			},
			expected: "default_value",
		},
		{
			name: "alternative_strategy",
			config: FallbackConfig{
				Strategy:    FallbackAlternative,
				Alternative: "mobile_version",
			},
			expected: map[string]interface{}{
				"source": "mobile_fallback",
				"message": "Using mobile version as fallback",
				"operation": "alternative_strategy_test",
			},
		},
		{
			name: "degrade_strategy",
			config: FallbackConfig{
				Strategy: FallbackDegrade,
				Degraded: map[string]interface{}{"status": "degraded", "data": "limited"},
			},
			expected: map[string]interface{}{"status": "degraded", "data": "limited"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			operationName := tc.name + "_test"
			service.ConfigureFallback(operationName, tc.config)

			failOperation := func() (interface{}, error) {
				return nil, fmt.Errorf("operation failed")
			}

			result := service.ExecuteWithRecovery(ctx, operationName, failOperation)

			if !result.Success {
				t.Error("Expected fallback to succeed")
			}
			if !result.UsedFallback {
				t.Error("Expected fallback to be used")
			}

			// For alternative strategy, we expect a structured result
			if tc.name == "alternative_strategy" {
				resultMap, ok := result.Result.(map[string]interface{})
				if !ok {
					t.Errorf("Expected map result for alternative strategy, got %T", result.Result)
					return
				}
				expectedMap := tc.expected.(map[string]interface{})
				for key, expectedValue := range expectedMap {
					if resultMap[key] != expectedValue {
						t.Errorf("Expected %s = %v, got %v", key, expectedValue, resultMap[key])
					}
				}
			} else {
				// For other strategies, check the configured expected value
				if fmt.Sprintf("%v", result.Result) != fmt.Sprintf("%v", tc.expected) {
					t.Errorf("Expected %v, got %v", tc.expected, result.Result)
				}
			}
		})
	}
}

func TestService_CacheManagement(t *testing.T) {
	service := NewService()

	// Test cache population
	service.cacheResult("test_op", "cached_value")

	stats := service.GetCacheStats()
	totalEntries := stats["total_entries"].(int)
	if totalEntries != 1 {
		t.Errorf("Expected 1 cache entry, got %d", totalEntries)
	}

	// Test cache retrieval
	result, err := service.getCachedResult("test_op", time.Minute)
	if err != nil {
		t.Errorf("Expected successful cache retrieval, got error: %v", err)
	}
	if result != "cached_value" {
		t.Errorf("Expected 'cached_value', got %v", result)
	}

	// Test cache expiration
	_, err = service.getCachedResult("test_op", time.Nanosecond)
	if err == nil {
		t.Error("Expected cache expiration error")
	}

	// Test cache clearing
	service.ClearCache()
	stats = service.GetCacheStats()
	totalEntries = stats["total_entries"].(int)
	if totalEntries != 0 {
		t.Errorf("Expected 0 cache entries after clear, got %d", totalEntries)
	}
}

func TestService_CircuitBreakerStats(t *testing.T) {
	service := NewService()
	ctx := context.Background()

	// Create circuit breaker by executing operation
	failOperation := func() (interface{}, error) {
		return nil, fmt.Errorf("error")
	}

	service.ExecuteWithRecovery(ctx, "stats_test", failOperation)

	stats := service.GetCircuitBreakerStats()
	if len(stats) != 1 {
		t.Errorf("Expected 1 circuit breaker, got %d", len(stats))
	}

	if _, exists := stats["stats_test"]; !exists {
		t.Error("Expected circuit breaker stats for 'stats_test'")
	}
}

func TestService_ResetCircuitBreaker(t *testing.T) {
	service := NewService()
	ctx := context.Background()

	// Configure and trigger circuit breaker
	service.ConfigureCircuitBreaker("reset_test", CircuitBreakerConfig{
		MaxFailures:  1,
		ResetTimeout: time.Hour, // Long timeout
	})

	failOperation := func() (interface{}, error) {
		return nil, fmt.Errorf("error")
	}

	// Trigger circuit breaker opening
	service.ExecuteWithRecovery(ctx, "reset_test", failOperation)

	// Manually reset
	err := service.ResetCircuitBreaker("reset_test")
	if err != nil {
		t.Errorf("Expected successful reset, got error: %v", err)
	}

	// Should now allow execution
	successOperation := func() (interface{}, error) {
		return "success", nil
	}

	result := service.ExecuteWithRecovery(ctx, "reset_test", successOperation)
	if !result.Success {
		t.Error("Expected operation to succeed after manual reset")
	}
}

func TestService_ContextCancellation(t *testing.T) {
	service := NewService()
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	slowOperation := func() (interface{}, error) {
		// Simulate slow operation that respects context
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(TestSlowOperationTimeout):
			return "too_late", nil
		}
	}

	result := service.ExecuteWithRecovery(ctx, "timeout_test", slowOperation)

	if result.Success {
		t.Error("Expected operation to fail due to context cancellation")
	}
	if result.OriginalError == nil {
		t.Error("Expected context cancellation error")
	}
}

func TestService_AlternativeOperation_Strategies(t *testing.T) {
	service := NewService()

	testCases := []struct {
		name        string
		operation   string
		alternative string
		expectError bool
	}{
		{
			name:        "mobile_version",
			operation:   "test_op",
			alternative: "mobile_version",
			expectError: false,
		},
		{
			name:        "api_fallback",
			operation:   "test_op",
			alternative: "api_fallback",
			expectError: false,
		},
		{
			name:        "cached_alternative",
			operation:   "test_op",
			alternative: "cached_alternative",
			expectError: true, // No cached data initially
		},
		{
			name:        "generic_alternative",
			operation:   "test_op",
			alternative: "custom_strategy",
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := service.executeAlternativeOperation(tc.operation, tc.alternative)

			if tc.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tc.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
			if !tc.expectError && result == nil {
				t.Error("Expected result but got nil")
			}
		})
	}
}

func TestService_DefaultCircuitBreakerConfiguration(t *testing.T) {
	service := NewService()
	
	// Trigger circuit breaker creation by executing an operation
	ctx := context.Background()
	failOperation := func() (interface{}, error) {
		return nil, fmt.Errorf("test error")
	}
	
	service.ExecuteWithRecovery(ctx, "default_config_test", failOperation)
	
	// Get circuit breaker stats to verify default configuration
	stats := service.GetCircuitBreakerStats()
	cbStats, exists := stats["default_config_test"]
	if !exists {
		t.Fatal("Expected circuit breaker to be created")
	}
	
	cbStatsMap := cbStats.(map[string]interface{})
	
	// Verify default max failures
	maxFailures := cbStatsMap["max_failures"].(int)
	if maxFailures != DefaultCircuitBreakerMaxFailures {
		t.Errorf("Expected max failures %d, got %d", DefaultCircuitBreakerMaxFailures, maxFailures)
	}
	
	// Verify default reset timeout
	resetTimeout := cbStatsMap["reset_timeout"].(time.Duration)
	if resetTimeout != DefaultCircuitBreakerResetTimeout {
		t.Errorf("Expected reset timeout %v, got %v", DefaultCircuitBreakerResetTimeout, resetTimeout)
	}
}

func TestService_RetryableErrorPatterns(t *testing.T) {
	service := NewService()
	
	testCases := []struct {
		errorMessage string
		shouldRetry  bool
	}{
		{"temporary failure", true},        // Should match "temporary"
		{"temporary error occurred", true}, // Should match "temporary" (not redundant pattern)
		{"connection timeout", true},       // Should match "timeout"
		{"503 service unavailable", true},  // Should match both "503" and "service unavailable"
		{"permanent failure", false},       // Should not match any pattern
		{"invalid request", false},         // Should not match any pattern
	}
	
	for _, tc := range testCases {
		t.Run(tc.errorMessage, func(t *testing.T) {
			err := fmt.Errorf("%s", tc.errorMessage)
			shouldRetry := service.shouldRetry(err, 0) // First attempt
			
			if shouldRetry != tc.shouldRetry {
				t.Errorf("Error '%s' - expected shouldRetry=%t, got %t", 
					tc.errorMessage, tc.shouldRetry, shouldRetry)
			}
		})
	}
}

// Benchmark tests
func BenchmarkService_ExecuteWithRecovery_Success(b *testing.B) {
	service := NewService()
	ctx := context.Background()

	operation := func() (interface{}, error) {
		return "result", nil
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.ExecuteWithRecovery(ctx, "bench_test", operation)
	}
}

func BenchmarkService_ExecuteWithRecovery_WithFallback(b *testing.B) {
	service := NewService()
	ctx := context.Background()

	service.ConfigureFallback("bench_fallback", FallbackConfig{
		Strategy:     FallbackDefault,
		DefaultValue: "fallback",
	})

	operation := func() (interface{}, error) {
		return nil, fmt.Errorf("error")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.ExecuteWithRecovery(ctx, "bench_fallback", operation)
	}
}