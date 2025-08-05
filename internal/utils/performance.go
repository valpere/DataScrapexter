// Package utils provides performance optimization utilities
// for critical code paths in the DataScrapexter platform.
package utils

import (
	"context"
	"errors"
	"runtime"
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

// TokenBucketRateLimiter provides token bucket rate limiting
type TokenBucketRateLimiter struct {
	tokens   int64
	maxTokens int64
	refillRate time.Duration
	lastRefill int64
	mutex      sync.Mutex
}

// NewTokenBucketRateLimiter creates a new token bucket rate limiter
func NewTokenBucketRateLimiter(maxTokens int64, refillRate time.Duration) *TokenBucketRateLimiter {
	return &TokenBucketRateLimiter{
		tokens:     maxTokens,
		maxTokens:  maxTokens,
		refillRate: refillRate,
		lastRefill: time.Now().UnixNano(),
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
		logger := GetLogger("performance")
		logger.Warnf("TokenBucketRateLimiter: system time went backwards by %v "+
			"(possible causes: NTP correction, VM migration, manual clock change). "+
			"Timing reference reset to avoid negative calculations. "+
			"Impact: rate limiting may be temporarily inaccurate, but will self-correct. "+
			"If this occurs frequently, investigate system clock stability.", -elapsed)
		trl.lastRefill = now
		elapsed = 0
	}

	// Validate token state and detect potential bugs in the implementation.
	// Tokens should never be negative with the current logic, so if they are,
	// it indicates a bug that should be investigated and fixed.
	if trl.tokens < 0 {
		// This should never happen with correct implementation - log for debugging
		logger := GetLogger("performance")
		logger.Errorf("TokenBucketRateLimiter: tokens became negative (%d), this indicates a bug in the implementation. Resetting to 0.", trl.tokens)
		trl.tokens = 0
	} else if trl.tokens > trl.maxTokens {
		// This could happen due to calculation precision, clamp to max
		logger := GetLogger("performance")
		logger.Warnf("TokenBucketRateLimiter: tokens (%d) exceeded maximum (%d), clamping to maximum", trl.tokens, trl.maxTokens)
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
				logger := GetLogger("performance")
				logger.Errorf("TokenBucketRateLimiter: tokens became negative (%d) after refill, resetting to 0", trl.tokens)
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
			logger := GetLogger("performance")
			logger.Errorf("TokenBucketRateLimiter: tokens became negative (%d) after decrement, resetting to 0", trl.tokens)
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

// BatchProcessor processes items in batches for better performance
type BatchProcessor[T any] struct {
	batchSize    int
	flushTimeout time.Duration
	processFunc  func([]T) error
	batch        []T
	mutex        sync.Mutex
	lastFlush    time.Time
}

// NewBatchProcessor creates a new batch processor
func NewBatchProcessor[T any](batchSize int, flushTimeout time.Duration, processFunc func([]T) error) *BatchProcessor[T] {
	return &BatchProcessor[T]{
		batchSize:    batchSize,
		flushTimeout: flushTimeout,
		processFunc:  processFunc,
		batch:        make([]T, 0, batchSize),
		lastFlush:    time.Now(),
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
	
	err := bp.processFunc(bp.batch)
	bp.batch = bp.batch[:0] // Reset batch
	bp.lastFlush = time.Now()
	return err
}
