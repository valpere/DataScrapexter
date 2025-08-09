// Package utils provides performance optimization utilities
// for critical code paths in the DataScrapexter platform.
package utils

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// PerformanceMetrics tracks performance statistics
type PerformanceMetrics struct {
	TotalOperations   int64         `json:"total_operations"`
	SuccessfulOps     int64         `json:"successful_operations"`
	FailedOps         int64         `json:"failed_operations"`
	AverageLatency    time.Duration `json:"average_latency"`
	MinLatency        time.Duration `json:"min_latency"`
	MaxLatency        time.Duration `json:"max_latency"`
	TotalLatency      time.Duration `json:"total_latency"`
	OperationsPerSec  float64       `json:"operations_per_second"`
	StartTime         time.Time     `json:"start_time"`
	LastOperationTime time.Time     `json:"last_operation_time"`
	mutex             sync.RWMutex
}

// NewPerformanceMetrics creates a new performance metrics tracker
func NewPerformanceMetrics() *PerformanceMetrics {
	now := time.Now()
	return &PerformanceMetrics{
		StartTime:         now,
		LastOperationTime: now,
		MinLatency:        time.Duration(1<<63 - 1), // Max duration initially
	}
}

// RecordOperation records the result of an operation
func (pm *PerformanceMetrics) RecordOperation(duration time.Duration, success bool) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	pm.TotalOperations++
	if success {
		pm.SuccessfulOps++
	} else {
		pm.FailedOps++
	}

	// Update latency statistics
	pm.TotalLatency += duration
	if duration < pm.MinLatency {
		pm.MinLatency = duration
	}
	if duration > pm.MaxLatency {
		pm.MaxLatency = duration
	}
	// Use regular field access for thread-safe access to TotalOperations (mutex is held)
	if pm.TotalOperations > 0 {
		pm.AverageLatency = pm.TotalLatency / time.Duration(pm.TotalOperations)
	} else {
		pm.AverageLatency = 0
	}
	pm.LastOperationTime = time.Now()

	// Calculate operations per second
	elapsed := pm.LastOperationTime.Sub(pm.StartTime)
	if elapsed > 0 {
		pm.OperationsPerSec = float64(pm.TotalOperations) / elapsed.Seconds()
	}
}

// GetSnapshot returns a copy of current metrics
func (pm *PerformanceMetrics) GetSnapshot() PerformanceMetrics {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()
	
	return PerformanceMetrics{
		TotalOperations:   pm.TotalOperations,
		SuccessfulOps:     pm.SuccessfulOps,
		FailedOps:         pm.FailedOps,
		AverageLatency:    pm.AverageLatency,
		MinLatency:        pm.MinLatency,
		MaxLatency:        pm.MaxLatency,
		TotalLatency:      pm.TotalLatency,
		OperationsPerSec:  pm.OperationsPerSec,
		StartTime:         pm.StartTime,
		LastOperationTime: pm.LastOperationTime,
	}
}

// LatencyHistogram provides histogram-based latency tracking for percentile analysis
type LatencyHistogram struct {
	buckets      []time.Duration // Sorted bucket boundaries
	counts       []int64         // Count for each bucket
	samples      []time.Duration // Circular buffer of recent samples for exact percentiles
	sampleIndex  int            // Current position in samples buffer
	totalSamples int64          // Total number of samples recorded
	mutex        sync.RWMutex
}

// NewLatencyHistogram creates a new latency histogram with predefined buckets
func NewLatencyHistogram(sampleBufferSize int) *LatencyHistogram {
	// Define common latency buckets (in nanoseconds converted to Duration)
	buckets := []time.Duration{
		1 * time.Millisecond,
		5 * time.Millisecond,
		10 * time.Millisecond,
		25 * time.Millisecond,
		50 * time.Millisecond,
		100 * time.Millisecond,
		250 * time.Millisecond,
		500 * time.Millisecond,
		1 * time.Second,
		2 * time.Second,
		5 * time.Second,
		10 * time.Second,
	}
	
	return &LatencyHistogram{
		buckets: buckets,
		counts:  make([]int64, len(buckets)+1), // +1 for overflow bucket
		samples: make([]time.Duration, sampleBufferSize),
	}
}

// Record records a latency measurement
func (lh *LatencyHistogram) Record(latency time.Duration) {
	lh.mutex.Lock()
	defer lh.mutex.Unlock()
	
	// Find appropriate bucket
	bucketIndex := len(lh.buckets) // Default to overflow bucket
	for i, boundary := range lh.buckets {
		if latency <= boundary {
			bucketIndex = i
			break
		}
	}
	
	// Increment bucket count
	lh.counts[bucketIndex]++
	
	// Store sample for exact percentile calculation
	if len(lh.samples) > 0 {
		lh.samples[lh.sampleIndex] = latency
		lh.sampleIndex = (lh.sampleIndex + 1) % len(lh.samples)
	}
	
	lh.totalSamples++
}

// GetPercentile calculates the exact percentile from recent samples
func (lh *LatencyHistogram) GetPercentile(percentile float64) time.Duration {
	lh.mutex.RLock()
	defer lh.mutex.RUnlock()
	
	if lh.totalSamples == 0 {
		return 0
	}
	
	// Get valid samples (handle case where buffer isn't full yet)
	validSamples := int(lh.totalSamples)
	if validSamples > len(lh.samples) {
		validSamples = len(lh.samples)
	}
	
	if validSamples == 0 {
		return 0
	}
	
	// Copy samples to avoid holding lock during sort
	samples := make([]time.Duration, validSamples)
	if lh.totalSamples >= int64(len(lh.samples)) {
		// Buffer is full, copy all samples
		copy(samples, lh.samples)
	} else {
		// Buffer not full, copy only recorded samples
		copy(samples, lh.samples[:validSamples])
	}
	
	// Sort samples to calculate percentile
	sort.Slice(samples, func(i, j int) bool {
		return samples[i] < samples[j]
	})
	
	// Calculate percentile index
	index := int(float64(len(samples)-1) * percentile / 100.0)
	if index < 0 {
		index = 0
	}
	if index >= len(samples) {
		index = len(samples) - 1
	}
	
	return samples[index]
}

// GetHistogramSnapshot returns a snapshot of bucket counts
func (lh *LatencyHistogram) GetHistogramSnapshot() ([]time.Duration, []int64) {
	lh.mutex.RLock()
	defer lh.mutex.RUnlock()
	
	buckets := make([]time.Duration, len(lh.buckets))
	counts := make([]int64, len(lh.counts))
	copy(buckets, lh.buckets)
	copy(counts, lh.counts)
	
	return buckets, counts
}

// GetTotalSamples returns the total number of samples recorded
func (lh *LatencyHistogram) GetTotalSamples() int64 {
	lh.mutex.RLock()
	defer lh.mutex.RUnlock()
	return lh.totalSamples
}

// Reset resets all histogram data
func (lh *LatencyHistogram) Reset() {
	lh.mutex.Lock()
	defer lh.mutex.Unlock()
	
	for i := range lh.counts {
		lh.counts[i] = 0
	}
	for i := range lh.samples {
		lh.samples[i] = 0
	}
	lh.sampleIndex = 0
	lh.totalSamples = 0
}

// Reset resets all metrics
func (pm *PerformanceMetrics) Reset() {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()
	
	now := time.Now()
	pm.TotalOperations = 0
	pm.SuccessfulOps = 0
	pm.FailedOps = 0
	pm.AverageLatency = 0
	pm.MinLatency = time.Duration(1<<63 - 1)
	pm.MaxLatency = 0
	pm.TotalLatency = 0
	pm.OperationsPerSec = 0
	pm.StartTime = now
	pm.LastOperationTime = now
}

// Timer provides high-precision timing for operations
type Timer struct {
	start time.Time
	name  string
}

// NewTimer creates a new timer
func NewTimer(name string) *Timer {
	return &Timer{
		start: time.Now(),
		name:  name,
	}
}

// Stop stops the timer and returns the elapsed duration
func (t *Timer) Stop() time.Duration {
	return time.Since(t.start)
}

// Elapsed returns the elapsed time without stopping the timer
func (t *Timer) Elapsed() time.Duration {
	return time.Since(t.start)
}

// Name returns the timer name
func (t *Timer) Name() string {
	return t.name
}

// Pool provides efficient object pooling for reducing allocations
type Pool[T any] struct {
	pool    sync.Pool
	newFunc func() T
	resetFunc func(T)
}

// NewPool creates a new typed pool
func NewPool[T any](newFunc func() T, resetFunc func(T)) *Pool[T] {
	return &Pool[T]{
		pool: sync.Pool{
			New: func() interface{} {
				return newFunc()
			},
		},
		newFunc:   newFunc,
		resetFunc: resetFunc,
	}
}

// Get retrieves an object from the pool with type safety
func (p *Pool[T]) Get() T {
	obj := p.pool.Get()
	if typedObj, ok := obj.(T); ok {
		return typedObj
	}
	
	// This should never happen with proper pool usage, but provides safety
	// Create a new object using the newFunc function as fallback
	if p.newFunc != nil {
		return p.newFunc()
	}
	
	// Last resort: return zero value of T
	var zero T
	return zero
}

// Put returns an object to the pool after resetting it
func (p *Pool[T]) Put(obj T) {
	if p.resetFunc != nil {
		p.resetFunc(obj)
	}
	p.pool.Put(obj)
}

// WorkerPool manages a pool of workers for concurrent processing
type WorkerPool[T any] struct {
	workerCount int
	inputChan   chan T
	outputChan  chan interface{}
	errorChan   chan error
	workerFunc  func(T) (interface{}, error)
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	metrics     *PerformanceMetrics
}

// NewWorkerPool creates a new worker pool
func NewWorkerPool[T any](workerCount int, bufferSize int, workerFunc func(T) (interface{}, error)) *WorkerPool[T] {
	ctx, cancel := context.WithCancel(context.Background())
	return &WorkerPool[T]{
		workerCount: workerCount,
		inputChan:   make(chan T, bufferSize),
		outputChan:  make(chan interface{}, bufferSize),
		errorChan:   make(chan error, bufferSize),
		workerFunc:  workerFunc,
		ctx:         ctx,
		cancel:      cancel,
		metrics:     NewPerformanceMetrics(),
	}
}

// Start starts the worker pool
func (wp *WorkerPool[T]) Start() {
	for i := 0; i < wp.workerCount; i++ {
		wp.wg.Add(1)
		go wp.worker()
	}
}

// worker is the worker goroutine function
func (wp *WorkerPool[T]) worker() {
	defer wp.wg.Done()
	
	for {
		select {
		case <-wp.ctx.Done():
			return
		case input, ok := <-wp.inputChan:
			if !ok {
				return
			}
			
			timer := NewTimer("worker_operation")
			result, err := wp.workerFunc(input)
			duration := timer.Stop()
			
			if err != nil {
				wp.metrics.RecordOperation(duration, false)
				select {
				case wp.errorChan <- err:
				case <-wp.ctx.Done():
					return
				}
			} else {
				wp.metrics.RecordOperation(duration, true)
				select {
				case wp.outputChan <- result:
				case <-wp.ctx.Done():
					return
				}
			}
		}
	}
}

// Submit submits work to the pool
func (wp *WorkerPool[T]) Submit(input T) error {
	select {
	case wp.inputChan <- input:
		return nil
	case <-wp.ctx.Done():
		return wp.ctx.Err()
	}
}

// Results returns the output channel
func (wp *WorkerPool[T]) Results() <-chan interface{} {
	return wp.outputChan
}

// Errors returns the error channel
func (wp *WorkerPool[T]) Errors() <-chan error {
	return wp.errorChan
}

// Close closes the worker pool
func (wp *WorkerPool[T]) Close() {
	close(wp.inputChan)
	wp.wg.Wait()
	wp.cancel()
	close(wp.outputChan)
	close(wp.errorChan)
}

// GetMetrics returns performance metrics
func (wp *WorkerPool[T]) GetMetrics() PerformanceMetrics {
	return wp.metrics.GetSnapshot()
}

// LogFunc represents a logging function for decoupled logging
type LogFunc func(level string, format string, args ...interface{})

// TokenBucketRateLimiter provides token bucket rate limiting with configurable logging
type TokenBucketRateLimiter struct {
	tokens     int64
	maxTokens  int64
	refillRate time.Duration
	lastRefill int64
	mutex      sync.Mutex
	logFunc    LogFunc // Dependency injection for logging concerns
}

// NewTokenBucketRateLimiter creates a new token bucket rate limiter with default no-op logging
// For logging functionality, use NewTokenBucketRateLimiterWithLogger with a custom LogFunc
func NewTokenBucketRateLimiter(maxTokens int64, refillRate time.Duration) *TokenBucketRateLimiter {
	return NewTokenBucketRateLimiterWithLogger(maxTokens, refillRate, nil)
}

// NewTokenBucketRateLimiterWithPerformanceLogging is DEPRECATED and will be removed in v2.0.0
// 
// DEPRECATION NOTICE: This function has been deprecated due to circular import issues.
// It previously attempted to use utils.GetLogger which creates import cycles since
// performance utilities are used by the logging system itself.
//
// MIGRATION PATH:
// Replace this:
//   limiter := NewTokenBucketRateLimiterWithPerformanceLogging(1000, time.Second)
// With this:
//   logger := utils.GetLogger("rate-limiter") // Get logger from calling code
//   logFunc := func(level, format string, args ...interface{}) {
//     switch level {
//     case "error":
//       logger.Errorf(format, args...)
//     case "warn":  
//       logger.Warnf(format, args...)
//     default:
//       logger.Infof(format, args...)
//     }
//   }
//   limiter := NewTokenBucketRateLimiterWithLogger(1000, time.Second, logFunc)
//
// TODO: REMOVE this function in v2.0.0 after migration period
//
// Deprecated: Use NewTokenBucketRateLimiterWithLogger with a custom LogFunc instead.
func NewTokenBucketRateLimiterWithPerformanceLogging(maxTokens int64, refillRate time.Duration) *TokenBucketRateLimiter {
	// Return no-op logger to avoid circular import - users should use NewTokenBucketRateLimiterWithLogger
	return NewTokenBucketRateLimiterWithLogger(maxTokens, refillRate, nil)
}

// NewTokenBucketRateLimiterWithLogger creates a new token bucket rate limiter with custom logging
func NewTokenBucketRateLimiterWithLogger(maxTokens int64, refillRate time.Duration, logFunc LogFunc) *TokenBucketRateLimiter {
	// Default logging implementation if none provided - use no-op to avoid circular imports
	if logFunc == nil {
		logFunc = func(level string, format string, args ...interface{}) {
			// No-op logger to avoid circular import with utils.GetLogger
			// Users should provide their own logger if they want logging functionality
		}
	}
	
	return &TokenBucketRateLimiter{
		tokens:     maxTokens,
		maxTokens:  maxTokens,
		refillRate: refillRate,
		lastRefill: time.Now().UnixNano(),
		logFunc:    logFunc,
	}
}

// Allow checks if an operation is allowed
func (trl *TokenBucketRateLimiter) Allow() bool {
	trl.mutex.Lock()
	defer trl.mutex.Unlock()

	now := time.Now().UnixNano()
	elapsed := time.Duration(now - trl.lastRefill)

	// Handle system time going backwards (e.g., NTP corrections, system clock changes)
	if elapsed < 0 {
		// System time went backwards, reset timing reference to avoid negative calculations
		trl.logFunc("warn", `TokenBucketRateLimiter: system time went backwards by %v.
Possible causes: NTP correction, VM migration, manual clock change.
Timing reference reset to avoid negative calculations.
Impact: rate limiting may be temporarily inaccurate, but will self-correct.
If this occurs frequently, investigate system clock stability.`, -elapsed)
		trl.lastRefill = now
		elapsed = 0
	}

	// Validate token state and detect potential bugs in the implementation.
	// Tokens should never be negative with the current logic, so if they are,
	// it indicates a bug that should be investigated and fixed.
	if trl.tokens < 0 {
		// This should never happen with correct implementation - log for debugging
		trl.logFunc("error", "TokenBucketRateLimiter: tokens became negative (%d), this indicates a bug in the implementation. Resetting to 0.", trl.tokens)
		trl.tokens = 0
	} else if trl.tokens > trl.maxTokens {
		// This could happen due to calculation precision, clamp to max
		trl.logFunc("warn", "TokenBucketRateLimiter: tokens (%d) exceeded maximum (%d), clamping to maximum", trl.tokens, trl.maxTokens)
		trl.tokens = trl.maxTokens
	}

	// Refill tokens based on elapsed time
	if elapsed >= trl.refillRate {
		tokensToAdd := int64(elapsed / trl.refillRate)
		if tokensToAdd > 0 {
			// Prevent overflow by checking if addition would exceed maximum
			if trl.tokens+tokensToAdd > trl.maxTokens {
				trl.tokens = trl.maxTokens
			} else {
				trl.tokens += tokensToAdd
			}
			
			// Additional safety check to ensure tokens are valid
			if trl.tokens < 0 {
				trl.logFunc("error", "TokenBucketRateLimiter: tokens became negative (%d) after refill, resetting to 0", trl.tokens)
				trl.tokens = 0
			}
		}
		trl.lastRefill = now
	}

	// Safely decrement tokens with validation
	if trl.tokens > 0 {
		trl.tokens--
		// Final validation to ensure tokens didn't go negative
		if trl.tokens < 0 {
			trl.logFunc("error", "TokenBucketRateLimiter: tokens became negative (%d) after decrement, resetting to 0", trl.tokens)
			trl.tokens = 0
			return false
		}
		return true
	}
	return false
}

// Wait waits until a token is available
func (trl *TokenBucketRateLimiter) Wait(ctx context.Context) error {
	for {
		if trl.Allow() {
			return nil
		}
		
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(trl.refillRate / 10): // Check again after a short delay
			continue
		}
	}
}

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	maxFailures    int64
	resetTimeout   time.Duration
	failureCount   int64
	lastFailureTime int64
	state          int32 // 0: Closed, 1: Open, 2: Half-Open
	mutex          sync.RWMutex
}

// Circuit breaker states
const (
	StateClosed   = 0
	StateOpen     = 1
	StateHalfOpen = 2
)

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(maxFailures int64, resetTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		maxFailures:  maxFailures,
		resetTimeout: resetTimeout,
		state:        StateClosed,
	}
}

// Execute executes a function with circuit breaker protection
func (cb *CircuitBreaker) Execute(fn func() error) error {
	if !cb.allowRequest() {
		return errors.New("circuit breaker is open")
	}
	
	err := fn()
	if err != nil {
		cb.recordFailure()
		return err
	}
	
	cb.recordSuccess()
	return nil
}

// allowRequest checks if a request should be allowed
func (cb *CircuitBreaker) allowRequest() bool {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	
	state := atomic.LoadInt32(&cb.state)
	
	switch state {
	case StateClosed:
		return true
	case StateOpen:
		// Check if we should transition to half-open
		lastFailure := atomic.LoadInt64(&cb.lastFailureTime)
		if time.Since(time.Unix(0, lastFailure)) > cb.resetTimeout {
			atomic.CompareAndSwapInt32(&cb.state, StateOpen, StateHalfOpen)
			return true
		}
		return false
	case StateHalfOpen:
		return true
	default:
		return false
	}
}

// recordFailure records a failure
func (cb *CircuitBreaker) recordFailure() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()
	
	failures := atomic.AddInt64(&cb.failureCount, 1)
	atomic.StoreInt64(&cb.lastFailureTime, time.Now().UnixNano())
	
	if failures >= cb.maxFailures {
		atomic.StoreInt32(&cb.state, StateOpen)
	}
}

// recordSuccess records a success
func (cb *CircuitBreaker) recordSuccess() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()
	
	atomic.StoreInt64(&cb.failureCount, 0)
	atomic.StoreInt32(&cb.state, StateClosed)
}

// GetState returns the current circuit breaker state
func (cb *CircuitBreaker) GetState() int32 {
	return atomic.LoadInt32(&cb.state)
}

// MemoryManager helps manage memory usage and GC pressure
type MemoryManager struct {
	maxMemoryBytes uint64
	gcThreshold    uint64
	lastGC         time.Time
	gcInterval     time.Duration
}

// NewMemoryManager creates a new memory manager
func NewMemoryManager(maxMemoryBytes uint64, gcInterval time.Duration) *MemoryManager {
	return &MemoryManager{
		maxMemoryBytes: maxMemoryBytes,
		gcThreshold:    maxMemoryBytes / 2, // Trigger GC at 50% of max memory
		lastGC:         time.Now(),
		gcInterval:     gcInterval,
	}
}

// CheckMemoryUsage checks current memory usage and triggers GC if needed
func (mm *MemoryManager) CheckMemoryUsage() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	// Check if we should trigger GC
	if m.Alloc > mm.gcThreshold && time.Since(mm.lastGC) > mm.gcInterval {
		runtime.GC()
		mm.lastGC = time.Now()
	}
}

// GetMemoryStats returns current memory statistics
func (mm *MemoryManager) GetMemoryStats() runtime.MemStats {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return m
}

// IsMemoryPressureHigh checks if memory pressure is high
func (mm *MemoryManager) IsMemoryPressureHigh() bool {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return m.Alloc > mm.gcThreshold
}

// Helper functions

// MeasureOperation measures the performance of an operation
func MeasureOperation(name string, operation func() error) (time.Duration, error) {
	timer := NewTimer(name)
	err := operation()
	duration := timer.Stop()
	return duration, err
}

// BatchProcessor processes items in batches for better performance with adaptive sizing
type BatchProcessor[T any] struct {
	batchSize       int
	originalSize    int               // Original batch size for reset
	maxBatchSize    int               // Maximum allowed batch size
	minBatchSize    int               // Minimum allowed batch size
	flushTimeout    time.Duration
	processFunc     func([]T) error
	batch           []T
	mutex           sync.Mutex
	lastFlush       time.Time
	// Adaptive sizing fields
	processingTimes []time.Duration   // Recent processing times
	timeIndex       int               // Current index in processing times
	avgProcessTime  time.Duration     // Average processing time
	targetLatency   time.Duration     // Target processing latency
	adaptiveEnabled bool              // Whether adaptive sizing is enabled
}

// NewBatchProcessor creates a new batch processor with fixed batch size
func NewBatchProcessor[T any](batchSize int, flushTimeout time.Duration, processFunc func([]T) error) *BatchProcessor[T] {
	return &BatchProcessor[T]{
		batchSize:       batchSize,
		originalSize:    batchSize,
		maxBatchSize:    batchSize,
		minBatchSize:    batchSize,
		flushTimeout:    flushTimeout,
		processFunc:     processFunc,
		batch:           make([]T, 0, batchSize),
		lastFlush:       time.Now(),
		adaptiveEnabled: false,
	}
}

// NewAdaptiveBatchProcessor creates a new batch processor with adaptive sizing
func NewAdaptiveBatchProcessor[T any](initialSize int, minSize int, maxSize int, flushTimeout time.Duration, targetLatency time.Duration, processFunc func([]T) error) *BatchProcessor[T] {
	// Validate parameters
	if minSize <= 0 {
		minSize = 1
	}
	if maxSize < minSize {
		maxSize = minSize
	}
	if initialSize < minSize {
		initialSize = minSize
	}
	if initialSize > maxSize {
		initialSize = maxSize
	}
	
	return &BatchProcessor[T]{
		batchSize:       initialSize,
		originalSize:    initialSize,
		maxBatchSize:    maxSize,
		minBatchSize:    minSize,
		flushTimeout:    flushTimeout,
		processFunc:     processFunc,
		batch:           make([]T, 0, maxSize), // Use max size for capacity
		lastFlush:       time.Now(),
		processingTimes: make([]time.Duration, 10), // Keep track of last 10 processing times
		targetLatency:   targetLatency,
		adaptiveEnabled: true,
	}
}

// Add adds an item to the batch
func (bp *BatchProcessor[T]) Add(item T) error {
	bp.mutex.Lock()
	defer bp.mutex.Unlock()
	
	bp.batch = append(bp.batch, item)
	
	// Check if we should flush
	if len(bp.batch) >= bp.batchSize || time.Since(bp.lastFlush) > bp.flushTimeout {
		return bp.flush()
	}
	
	return nil
}

// Flush processes the current batch
func (bp *BatchProcessor[T]) Flush() error {
	bp.mutex.Lock()
	defer bp.mutex.Unlock()
	return bp.flush()
}

// flush internal flush method (assumes mutex is held)
func (bp *BatchProcessor[T]) flush() error {
	if len(bp.batch) == 0 {
		return nil
	}
	
	// Measure processing time for adaptive sizing
	startTime := time.Now()
	err := bp.processFunc(bp.batch)
	processingTime := time.Since(startTime)
	
	// Update batch size if adaptive sizing is enabled
	if bp.adaptiveEnabled {
		bp.updateBatchSize(processingTime, len(bp.batch))
	}
	
	bp.batch = bp.batch[:0] // Reset batch
	bp.lastFlush = time.Now()
	return err
}

// updateBatchSize adjusts the batch size based on processing performance
func (bp *BatchProcessor[T]) updateBatchSize(processingTime time.Duration, batchSize int) {
	// Record processing time
	bp.processingTimes[bp.timeIndex] = processingTime
	bp.timeIndex = (bp.timeIndex + 1) % len(bp.processingTimes)
	
	// Calculate average processing time
	var totalTime time.Duration
	validSamples := 0
	for _, t := range bp.processingTimes {
		if t > 0 {
			totalTime += t
			validSamples++
		}
	}
	
	if validSamples == 0 {
		return
	}
	
	bp.avgProcessTime = totalTime / time.Duration(validSamples)
	
	// Adaptive sizing logic
	if bp.avgProcessTime > bp.targetLatency {
		// Processing is too slow, reduce batch size
		newSize := int(float64(bp.batchSize) * 0.8)
		if newSize < bp.minBatchSize {
			newSize = bp.minBatchSize
		}
		bp.batchSize = newSize
	} else if bp.avgProcessTime < bp.targetLatency/2 {
		// Processing is fast, increase batch size
		newSize := int(float64(bp.batchSize) * 1.2)
		if newSize > bp.maxBatchSize {
			newSize = bp.maxBatchSize
		}
		bp.batchSize = newSize
	}
}

// GetCurrentBatchSize returns the current adaptive batch size
func (bp *BatchProcessor[T]) GetCurrentBatchSize() int {
	bp.mutex.Lock()
	defer bp.mutex.Unlock()
	return bp.batchSize
}

// GetAverageProcessingTime returns the current average processing time
func (bp *BatchProcessor[T]) GetAverageProcessingTime() time.Duration {
	bp.mutex.Lock()
	defer bp.mutex.Unlock()
	return bp.avgProcessTime
}

// ResetBatchSize resets the batch size to the original value
func (bp *BatchProcessor[T]) ResetBatchSize() {
	bp.mutex.Lock()
	defer bp.mutex.Unlock()
	bp.batchSize = bp.originalSize
	// Clear processing time history
	for i := range bp.processingTimes {
		bp.processingTimes[i] = 0
	}
	bp.timeIndex = 0
	bp.avgProcessTime = 0
}

// HighThroughputMemoryPool provides optimized memory pooling for high-throughput scenarios
type HighThroughputMemoryPool[T any] struct {
	pools       []*Pool[T]        // Multiple pools to reduce contention
	poolIndex   *int64            // Atomic pool selection index
	newFunc     func() T
	resetFunc   func(T)
	poolCount   int
}

// NewHighThroughputMemoryPool creates a memory pool optimized for high-throughput scenarios
// It uses multiple underlying pools to reduce lock contention
func NewHighThroughputMemoryPool[T any](poolCount int, newFunc func() T, resetFunc func(T)) *HighThroughputMemoryPool[T] {
	if poolCount <= 0 {
		poolCount = runtime.GOMAXPROCS(0) // Use number of CPUs as default
	}
	
	pools := make([]*Pool[T], poolCount)
	for i := 0; i < poolCount; i++ {
		pools[i] = NewPool(newFunc, resetFunc)
	}
	
	index := int64(0)
	return &HighThroughputMemoryPool[T]{
		pools:     pools,
		poolIndex: &index,
		newFunc:   newFunc,
		resetFunc: resetFunc,
		poolCount: poolCount,
	}
}

// Get retrieves an object from one of the pools with minimal contention
func (htp *HighThroughputMemoryPool[T]) Get() T {
	// Use atomic increment to distribute load across pools
	index := atomic.AddInt64(htp.poolIndex, 1)
	poolIdx := int(index % int64(htp.poolCount))
	return htp.pools[poolIdx].Get()
}

// Put returns an object to one of the pools
func (htp *HighThroughputMemoryPool[T]) Put(obj T) {
	// Use same distribution strategy as Get
	index := atomic.LoadInt64(htp.poolIndex)
	poolIdx := int(index % int64(htp.poolCount))
	htp.pools[poolIdx].Put(obj)
}

// GetStats returns statistics about pool utilization
func (htp *HighThroughputMemoryPool[T]) GetStats() PoolStats {
	// This would require instrumenting the pools, which is a simplification
	// In a production system, you'd want to track gets/puts per pool
	return PoolStats{
		PoolCount: htp.poolCount,
		// Additional stats would be implemented based on requirements
	}
}

// PoolStats provides statistics about pool performance
type PoolStats struct {
	PoolCount int `json:"pool_count"`
	// Additional fields would be added based on monitoring requirements
}

// BufferPool provides a specialized pool for byte slices with size categories
type BufferPool struct {
	pools map[int]*Pool[[]byte] // Keyed by buffer size category
	mutex sync.RWMutex
}

// NewBufferPool creates a new buffer pool with predefined size categories
func NewBufferPool() *BufferPool {
	bp := &BufferPool{
		pools: make(map[int]*Pool[[]byte]),
	}
	
	// Common buffer sizes: 1KB, 4KB, 16KB, 64KB, 256KB, 1MB
	sizes := []int{1024, 4096, 16384, 65536, 262144, 1048576}
	
	for _, size := range sizes {
		bp.pools[size] = NewPool(
			func() []byte {
				return make([]byte, size)
			},
			func(buf []byte) {
				// Clear the buffer (security best practice)
				for i := range buf {
					buf[i] = 0
				}
			},
		)
	}
	
	return bp
}

// GetBuffer gets a buffer of at least the specified size
func (bp *BufferPool) GetBuffer(minSize int) []byte {
	bp.mutex.RLock()
	defer bp.mutex.RUnlock()
	
	// Find the smallest buffer size that meets the requirement
	for size := 1024; size <= 1048576; size *= 4 {
		if size >= minSize {
			if pool, exists := bp.pools[size]; exists {
				buf := pool.Get()
				return buf[:minSize] // Return slice of requested size
			}
		}
	}
	
	// If no suitable pool exists, create a new buffer
	return make([]byte, minSize)
}

// PutBuffer returns a buffer to the appropriate pool
func (bp *BufferPool) PutBuffer(buf []byte) {
	// Determine the original capacity
	capacity := cap(buf)
	
	bp.mutex.RLock()
	defer bp.mutex.RUnlock()
	
	if pool, exists := bp.pools[capacity]; exists {
		// Restore to full capacity before returning to pool
		fullBuf := buf[:capacity]
		pool.Put(fullBuf)
	}
	// If no matching pool, let GC handle it
}

// StringPool provides efficient pooling for strings and string builders
type StringPool struct {
	builderPool *Pool[*strings.Builder]
}

// NewStringPool creates a new string pool
func NewStringPool() *StringPool {
	return &StringPool{
		builderPool: NewPool(
			func() *strings.Builder {
				return &strings.Builder{}
			},
			func(sb *strings.Builder) {
				sb.Reset()
			},
		),
	}
}

// GetBuilder gets a string builder from the pool
func (sp *StringPool) GetBuilder() *strings.Builder {
	return sp.builderPool.Get()
}

// PutBuilder returns a string builder to the pool
func (sp *StringPool) PutBuilder(sb *strings.Builder) {
	sp.builderPool.Put(sb)
}

// BuildString is a convenience method that gets a builder, executes the function, and returns the result
func (sp *StringPool) BuildString(fn func(*strings.Builder)) string {
	builder := sp.GetBuilder()
	defer sp.PutBuilder(builder)
	fn(builder)
	return builder.String()
}

// Additional utility functions for common performance optimizations

// FastStringConcat performs efficient string concatenation using a pre-allocated buffer
func FastStringConcat(parts ...string) string {
	if len(parts) == 0 {
		return ""
	}
	if len(parts) == 1 {
		return parts[0]
	}
	
	// Calculate total length
	totalLen := 0
	for _, part := range parts {
		totalLen += len(part)
	}
	
	// Pre-allocate buffer with exact size
	var builder strings.Builder
	builder.Grow(totalLen)
	
	for _, part := range parts {
		builder.WriteString(part)
	}
	
	return builder.String()
}

// ConcurrentMap provides a concurrent-safe map with read-write separation for better performance
type ConcurrentMap[K comparable, V any] struct {
	shards   []*mapShard[K, V]
	shardMask uint64
}

type mapShard[K comparable, V any] struct {
	items map[K]V
	mutex sync.RWMutex
}

// NewConcurrentMap creates a new concurrent map with the specified number of shards
func NewConcurrentMap[K comparable, V any](shardCount int) *ConcurrentMap[K, V] {
	if shardCount <= 0 {
		shardCount = 32 // Default shard count
	}
	
	// Ensure shard count is a power of 2 for efficient hashing
	if shardCount&(shardCount-1) != 0 {
		// Round up to next power of 2
		shardCount = 1
		for shardCount < 32 {
			shardCount <<= 1
		}
	}
	
	shards := make([]*mapShard[K, V], shardCount)
	for i := range shards {
		shards[i] = &mapShard[K, V]{
			items: make(map[K]V),
		}
	}
	
	return &ConcurrentMap[K, V]{
		shards:    shards,
		shardMask: uint64(shardCount - 1),
	}
}

// hash computes a hash for the key to determine shard
func (cm *ConcurrentMap[K, V]) hash(key K) uint64 {
	// Simple hash function - in production, use a more sophisticated hash
	var h uint64
	switch k := any(key).(type) {
	case string:
		for _, c := range k {
			h = h*31 + uint64(c)
		}
	case int:
		h = uint64(k)
	case int64:
		h = uint64(k)
	default:
		// Fallback for other types - convert to string
		h = uint64(len(fmt.Sprintf("%v", key)))
	}
	return h
}

// getShard returns the shard for the given key
func (cm *ConcurrentMap[K, V]) getShard(key K) *mapShard[K, V] {
	hash := cm.hash(key)
	return cm.shards[hash&cm.shardMask]
}

// Set stores a key-value pair
func (cm *ConcurrentMap[K, V]) Set(key K, value V) {
	shard := cm.getShard(key)
	shard.mutex.Lock()
	defer shard.mutex.Unlock()
	shard.items[key] = value
}

// Get retrieves a value by key
func (cm *ConcurrentMap[K, V]) Get(key K) (V, bool) {
	shard := cm.getShard(key)
	shard.mutex.RLock()
	defer shard.mutex.RUnlock()
	value, exists := shard.items[key]
	return value, exists
}

// Delete removes a key-value pair
func (cm *ConcurrentMap[K, V]) Delete(key K) {
	shard := cm.getShard(key)
	shard.mutex.Lock()
	defer shard.mutex.Unlock()
	delete(shard.items, key)
}

// Size returns the approximate number of items in the map
func (cm *ConcurrentMap[K, V]) Size() int {
	total := 0
	for _, shard := range cm.shards {
		shard.mutex.RLock()
		total += len(shard.items)
		shard.mutex.RUnlock()
	}
	return total
}

// Clear removes all items from the map
func (cm *ConcurrentMap[K, V]) Clear() {
	for _, shard := range cm.shards {
		shard.mutex.Lock()
		shard.items = make(map[K]V)
		shard.mutex.Unlock()
	}
}

// Keys returns a slice of all keys in the map
func (cm *ConcurrentMap[K, V]) Keys() []K {
	var keys []K
	for _, shard := range cm.shards {
		shard.mutex.RLock()
		for key := range shard.items {
			keys = append(keys, key)
		}
		shard.mutex.RUnlock()
	}
	return keys
}

// ExponentialBackoff implements exponential backoff for retry logic
type ExponentialBackoff struct {
	initialDelay time.Duration
	maxDelay     time.Duration
	multiplier   float64
	jitter       bool
	attempt      int
}

// NewExponentialBackoff creates a new exponential backoff configuration
func NewExponentialBackoff(initialDelay, maxDelay time.Duration, multiplier float64, jitter bool) *ExponentialBackoff {
	return &ExponentialBackoff{
		initialDelay: initialDelay,
		maxDelay:     maxDelay,
		multiplier:   multiplier,
		jitter:       jitter,
		attempt:      0,
	}
}

// NextDelay calculates the next delay duration
func (eb *ExponentialBackoff) NextDelay() time.Duration {
	delay := time.Duration(float64(eb.initialDelay) * eb.multiplier * float64(eb.attempt))
	if delay > eb.maxDelay {
		delay = eb.maxDelay
	}
	
	if eb.jitter {
		// Add up to 10% jitter to prevent thundering herd
		jitterAmount := time.Duration(float64(delay) * 0.1 * float64(time.Now().UnixNano()%100) / 100.0)
		delay += jitterAmount
	}
	
	eb.attempt++
	return delay
}

// Reset resets the backoff to initial state
func (eb *ExponentialBackoff) Reset() {
	eb.attempt = 0
}

// GetAttempt returns the current attempt number
func (eb *ExponentialBackoff) GetAttempt() int {
	return eb.attempt
}

// ResourceLimiter provides resource-based limiting (memory, connections, etc.)
type ResourceLimiter struct {
	maxResources int64
	current      int64
	waiters      chan struct{}
	mutex        sync.Mutex
}

// NewResourceLimiter creates a new resource limiter
func NewResourceLimiter(maxResources int64) *ResourceLimiter {
	return &ResourceLimiter{
		maxResources: maxResources,
		waiters:      make(chan struct{}, maxResources),
	}
}

// Acquire acquires a resource, blocking if necessary
func (rl *ResourceLimiter) Acquire(ctx context.Context) error {
	select {
	case rl.waiters <- struct{}{}:
		atomic.AddInt64(&rl.current, 1)
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// TryAcquire attempts to acquire a resource without blocking
func (rl *ResourceLimiter) TryAcquire() bool {
	select {
	case rl.waiters <- struct{}{}:
		atomic.AddInt64(&rl.current, 1)
		return true
	default:
		return false
	}
}

// Release releases a resource
func (rl *ResourceLimiter) Release() {
	select {
	case <-rl.waiters:
		atomic.AddInt64(&rl.current, -1)
	default:
		// Should not happen in correct usage
	}
}

// GetCurrentCount returns the current number of acquired resources
func (rl *ResourceLimiter) GetCurrentCount() int64 {
	return atomic.LoadInt64(&rl.current)
}

// GetMaxResources returns the maximum number of resources
func (rl *ResourceLimiter) GetMaxResources() int64 {
	return rl.maxResources
}
