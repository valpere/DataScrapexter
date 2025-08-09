// Example tests to demonstrate performance utilities functionality
package utils

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestPerformanceMetrics_BasicUsage(t *testing.T) {
	pm := NewPerformanceMetrics()
	
	// Record some operations
	pm.RecordOperation(10*time.Millisecond, true)
	pm.RecordOperation(20*time.Millisecond, true)
	pm.RecordOperation(30*time.Millisecond, false)
	
	snapshot := pm.GetSnapshot()
	if snapshot.TotalOperations != 3 {
		t.Errorf("Expected 3 total operations, got %d", snapshot.TotalOperations)
	}
	if snapshot.SuccessfulOps != 2 {
		t.Errorf("Expected 2 successful operations, got %d", snapshot.SuccessfulOps)
	}
	if snapshot.FailedOps != 1 {
		t.Errorf("Expected 1 failed operation, got %d", snapshot.FailedOps)
	}
}

func TestLatencyHistogram_BasicUsage(t *testing.T) {
	lh := NewLatencyHistogram(100)
	
	// Record some latencies
	for i := 0; i < 100; i++ {
		lh.Record(time.Duration(i) * time.Millisecond)
	}
	
	// Test percentiles
	p50 := lh.GetPercentile(50)
	p95 := lh.GetPercentile(95)
	p99 := lh.GetPercentile(99)
	
	if p50 <= 0 {
		t.Error("P50 should be greater than 0")
	}
	if p95 <= p50 {
		t.Error("P95 should be greater than P50")
	}
	if p99 <= p95 {
		t.Error("P99 should be greater than P95")
	}
	
	if lh.GetTotalSamples() != 100 {
		t.Errorf("Expected 100 total samples, got %d", lh.GetTotalSamples())
	}
}

func TestAdaptiveBatchProcessor_BasicUsage(t *testing.T) {
	processCount := 0
	processFunc := func(items []int) error {
		processCount += len(items)
		return nil
	}
	
	bp := NewAdaptiveBatchProcessor(10, 5, 50, 100*time.Millisecond, 50*time.Millisecond, processFunc)
	
	// Add some items
	for i := 0; i < 100; i++ {
		if err := bp.Add(i); err != nil {
			t.Errorf("Failed to add item %d: %v", i, err)
		}
	}
	
	// Flush remaining items
	if err := bp.Flush(); err != nil {
		t.Errorf("Failed to flush: %v", err)
	}
	
	if processCount != 100 {
		t.Errorf("Expected to process 100 items, processed %d", processCount)
	}
	
	currentSize := bp.GetCurrentBatchSize()
	if currentSize < bp.minBatchSize || currentSize > bp.maxBatchSize {
		t.Errorf("Batch size %d is outside valid range [%d, %d]", currentSize, bp.minBatchSize, bp.maxBatchSize)
	}
}

func TestHighThroughputMemoryPool_BasicUsage(t *testing.T) {
	htp := NewHighThroughputMemoryPool(4,
		func() []byte { return make([]byte, 1024) },
		func(buf []byte) {
			for i := range buf {
				buf[i] = 0
			}
		},
	)
	
	// Get and put several times
	for i := 0; i < 10; i++ {
		buf := htp.Get()
		if len(buf) != 1024 {
			t.Errorf("Expected buffer size 1024, got %d", len(buf))
		}
		
		// Use the buffer
		buf[0] = 42
		
		htp.Put(buf)
	}
	
	stats := htp.GetStats()
	if stats.PoolCount != 4 {
		t.Errorf("Expected 4 pools, got %d", stats.PoolCount)
	}
}

func TestBufferPool_BasicUsage(t *testing.T) {
	bp := NewBufferPool()
	
	// Test different buffer sizes
	sizes := []int{512, 2048, 32768, 131072}
	
	for _, size := range sizes {
		buf := bp.GetBuffer(size)
		if len(buf) != size {
			t.Errorf("Expected buffer size %d, got %d", size, len(buf))
		}
		
		// Use the buffer
		buf[0] = byte(size % 256)
		
		bp.PutBuffer(buf)
	}
}

func TestStringPool_BasicUsage(t *testing.T) {
	sp := NewStringPool()
	
	result := sp.BuildString(func(sb *strings.Builder) {
		sb.WriteString("Hello")
		sb.WriteString(" ")
		sb.WriteString("World")
		sb.WriteString("!")
	})
	
	expected := "Hello World!"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestConcurrentMap_BasicUsage(t *testing.T) {
	cm := NewConcurrentMap[string, int](16)
	
	// Set some values
	cm.Set("key1", 1)
	cm.Set("key2", 2)
	cm.Set("key3", 3)
	
	// Get values
	if val, ok := cm.Get("key1"); !ok || val != 1 {
		t.Errorf("Expected key1 to have value 1, got %v (exists: %v)", val, ok)
	}
	
	if val, ok := cm.Get("key2"); !ok || val != 2 {
		t.Errorf("Expected key2 to have value 2, got %v (exists: %v)", val, ok)
	}
	
	// Check size
	if size := cm.Size(); size != 3 {
		t.Errorf("Expected size 3, got %d", size)
	}
	
	// Delete a key
	cm.Delete("key2")
	if _, ok := cm.Get("key2"); ok {
		t.Error("key2 should have been deleted")
	}
	
	// Check keys
	keys := cm.Keys()
	if len(keys) != 2 {
		t.Errorf("Expected 2 keys after deletion, got %d", len(keys))
	}
}

func TestExponentialBackoff_BasicUsage(t *testing.T) {
	eb := NewExponentialBackoff(10*time.Millisecond, 1*time.Second, 2.0, false)
	
	// Test a few delays
	delays := make([]time.Duration, 5)
	for i := 0; i < 5; i++ {
		delays[i] = eb.NextDelay()
	}
	
	// Check that delays increase
	for i := 1; i < len(delays); i++ {
		if delays[i] <= delays[i-1] {
			t.Errorf("Delay should increase: %v should be greater than %v", delays[i], delays[i-1])
		}
	}
	
	// Test reset
	eb.Reset()
	if attempt := eb.GetAttempt(); attempt != 0 {
		t.Errorf("Expected attempt to be 0 after reset, got %d", attempt)
	}
}

func TestResourceLimiter_BasicUsage(t *testing.T) {
	rl := NewResourceLimiter(2)
	ctx := context.Background()
	
	// Acquire resources
	if err := rl.Acquire(ctx); err != nil {
		t.Errorf("Failed to acquire first resource: %v", err)
	}
	
	if err := rl.Acquire(ctx); err != nil {
		t.Errorf("Failed to acquire second resource: %v", err)
	}
	
	// Should be at capacity
	if count := rl.GetCurrentCount(); count != 2 {
		t.Errorf("Expected current count to be 2, got %d", count)
	}
	
	// Try to acquire one more (should fail)
	if acquired := rl.TryAcquire(); acquired {
		t.Error("Should not have been able to acquire third resource")
	}
	
	// Release one
	rl.Release()
	
	// Should be able to acquire now
	if acquired := rl.TryAcquire(); !acquired {
		t.Error("Should have been able to acquire resource after release")
	}
}

func TestFastStringConcat(t *testing.T) {
	parts := []string{"Hello", " ", "World", "!", " ", "Test"}
	result := FastStringConcat(parts...)
	expected := "Hello World! Test"
	
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
	
	// Test edge cases
	if result := FastStringConcat(); result != "" {
		t.Errorf("Expected empty string for no parts, got %q", result)
	}
	
	if result := FastStringConcat("single"); result != "single" {
		t.Errorf("Expected 'single' for single part, got %q", result)
	}
}

// Example function demonstrating usage
func ExamplePerformanceMetrics() {
	pm := NewPerformanceMetrics()
	
	// Record some operations
	pm.RecordOperation(10*time.Millisecond, true)
	pm.RecordOperation(20*time.Millisecond, true)
	pm.RecordOperation(30*time.Millisecond, false)
	
	// Get a snapshot
	snapshot := pm.GetSnapshot()
	fmt.Printf("Total operations: %d\n", snapshot.TotalOperations)
	fmt.Printf("Success rate: %.2f%%\n", float64(snapshot.SuccessfulOps)/float64(snapshot.TotalOperations)*100)
	fmt.Printf("Average latency: %v\n", snapshot.AverageLatency)
	
	// Output:
	// Total operations: 3
	// Success rate: 66.67%
	// Average latency: 20ms
}