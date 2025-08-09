// Comprehensive benchmarks for performance utilities
package utils

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"testing"
	"time"
)

// Benchmark PerformanceMetrics operations
func BenchmarkPerformanceMetrics_RecordOperation(b *testing.B) {
	pm := NewPerformanceMetrics()
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			pm.RecordOperation(time.Millisecond*50, true)
		}
	})
}

func BenchmarkPerformanceMetrics_GetSnapshot(b *testing.B) {
	pm := NewPerformanceMetrics()
	// Pre-populate with some data
	for i := 0; i < 1000; i++ {
		pm.RecordOperation(time.Millisecond*time.Duration(i%100), i%2 == 0)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = pm.GetSnapshot()
	}
}

// Benchmark LatencyHistogram operations
func BenchmarkLatencyHistogram_Record(b *testing.B) {
	lh := NewLatencyHistogram(1000)
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			duration := time.Duration(i%1000) * time.Millisecond
			lh.Record(duration)
			i++
		}
	})
}

func BenchmarkLatencyHistogram_GetPercentile(b *testing.B) {
	lh := NewLatencyHistogram(1000)
	
	// Pre-populate with data
	for i := 0; i < 1000; i++ {
		lh.Record(time.Duration(i) * time.Millisecond)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = lh.GetPercentile(95.0)
	}
}

// Benchmark Timer operations
func BenchmarkTimer_NewAndStop(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		timer := NewTimer("benchmark")
		_ = timer.Stop()
	}
}

func BenchmarkTimer_Elapsed(b *testing.B) {
	timer := NewTimer("benchmark")
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = timer.Elapsed()
	}
}

// Benchmark Pool operations
func BenchmarkPool_GetPut(b *testing.B) {
	pool := NewPool(
		func() []byte { return make([]byte, 1024) },
		func(buf []byte) { 
			// Clear buffer
			for i := range buf {
				buf[i] = 0
			}
		},
	)
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			buf := pool.Get()
			pool.Put(buf)
		}
	})
}

// Benchmark HighThroughputMemoryPool vs regular Pool
func BenchmarkHighThroughputPool_GetPut(b *testing.B) {
	htp := NewHighThroughputMemoryPool(runtime.GOMAXPROCS(0),
		func() []byte { return make([]byte, 1024) },
		func(buf []byte) {
			for i := range buf {
				buf[i] = 0
			}
		},
	)
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			buf := htp.Get()
			htp.Put(buf)
		}
	})
}

// Compare regular Pool vs HighThroughputPool under contention
func BenchmarkPoolContention_Regular(b *testing.B) {
	pool := NewPool(
		func() []byte { return make([]byte, 1024) },
		func(buf []byte) {},
	)
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			buf := pool.Get()
			// Simulate some work
			runtime.Gosched()
			pool.Put(buf)
		}
	})
}

func BenchmarkPoolContention_HighThroughput(b *testing.B) {
	htp := NewHighThroughputMemoryPool(runtime.GOMAXPROCS(0),
		func() []byte { return make([]byte, 1024) },
		func(buf []byte) {},
	)
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			buf := htp.Get()
			// Simulate some work
			runtime.Gosched()
			htp.Put(buf)
		}
	})
}

// Benchmark WorkerPool operations
func BenchmarkWorkerPool(b *testing.B) {
	workerFunc := func(input int) (interface{}, error) {
		// Simulate work
		result := input * 2
		return result, nil
	}
	
	wp := NewWorkerPool(4, 100, workerFunc)
	wp.Start()
	defer wp.Close()
	
	b.ResetTimer()
	
	go func() {
		for i := 0; i < b.N; i++ {
			wp.Submit(i)
		}
	}()
	
	for i := 0; i < b.N; i++ {
		select {
		case <-wp.Results():
		case <-wp.Errors():
			b.Fatal("Unexpected error")
		}
	}
}

// Benchmark TokenBucketRateLimiter
func BenchmarkTokenBucketRateLimiter_Allow(b *testing.B) {
	rl := NewTokenBucketRateLimiter(1000, time.Microsecond*100)
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			rl.Allow()
		}
	})
}

func BenchmarkTokenBucketRateLimiter_Wait(b *testing.B) {
	rl := NewTokenBucketRateLimiter(1000, time.Microsecond*10)
	ctx := context.Background()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rl.Wait(ctx)
	}
}

// Benchmark CircuitBreaker
func BenchmarkCircuitBreaker_Execute_Success(b *testing.B) {
	cb := NewCircuitBreaker(5, time.Second)
	successFunc := func() error { return nil }
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cb.Execute(successFunc)
	}
}

func BenchmarkCircuitBreaker_Execute_Failure(b *testing.B) {
	cb := NewCircuitBreaker(5, time.Second)
	failFunc := func() error { return fmt.Errorf("test error") }
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cb.Execute(failFunc)
	}
}

// Benchmark MemoryManager
func BenchmarkMemoryManager_CheckMemoryUsage(b *testing.B) {
	mm := NewMemoryManager(1024*1024*100, time.Second) // 100MB limit
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mm.CheckMemoryUsage()
	}
}

func BenchmarkMemoryManager_GetMemoryStats(b *testing.B) {
	mm := NewMemoryManager(1024*1024*100, time.Second)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = mm.GetMemoryStats()
	}
}

// Benchmark BatchProcessor
func BenchmarkBatchProcessor_Add(b *testing.B) {
	processFunc := func(items []int) error {
		// Simulate processing
		total := 0
		for _, item := range items {
			total += item
		}
		return nil
	}
	
	bp := NewBatchProcessor(100, time.Second, processFunc)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bp.Add(i)
	}
	bp.Flush() // Ensure final batch is processed
}

func BenchmarkAdaptiveBatchProcessor_Add(b *testing.B) {
	processFunc := func(items []int) error {
		// Simulate variable processing time
		total := 0
		for _, item := range items {
			total += item
		}
		// Simulate some processing delay
		time.Sleep(time.Microsecond * time.Duration(len(items)))
		return nil
	}
	
	bp := NewAdaptiveBatchProcessor(50, 10, 200, time.Second, 10*time.Millisecond, processFunc)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bp.Add(i)
	}
	bp.Flush()
}

// Benchmark BufferPool operations
func BenchmarkBufferPool_GetPut(b *testing.B) {
	bp := NewBufferPool()
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			buf := bp.GetBuffer(1024)
			bp.PutBuffer(buf)
		}
	})
}

func BenchmarkBufferPool_GetPut_LargeBuffer(b *testing.B) {
	bp := NewBufferPool()
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			buf := bp.GetBuffer(65536) // 64KB
			bp.PutBuffer(buf)
		}
	})
}

// Benchmark StringPool operations
func BenchmarkStringPool_BuildString(b *testing.B) {
	sp := NewStringPool()
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			result := sp.BuildString(func(sb *strings.Builder) {
				sb.WriteString("Hello")
				sb.WriteString(" ")
				sb.WriteString("World")
				sb.WriteString(" ")
				fmt.Fprintf(sb, "%d", i)
			})
			_ = result
			i++
		}
	})
}

// Compare StringPool vs direct string concatenation
func BenchmarkStringConcatenation_Direct(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := "Hello" + " " + "World" + " " + fmt.Sprintf("%d", i)
		_ = result
	}
}

func BenchmarkStringConcatenation_Pool(b *testing.B) {
	sp := NewStringPool()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := sp.BuildString(func(sb *strings.Builder) {
			sb.WriteString("Hello")
			sb.WriteString(" ")
			sb.WriteString("World")
			sb.WriteString(" ")
			fmt.Fprintf(sb, "%d", i)
		})
		_ = result
	}
}

// Benchmark MeasureOperation utility
func BenchmarkMeasureOperation(b *testing.B) {
	operation := func() error {
		// Simulate some work
		time.Sleep(time.Microsecond)
		return nil
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = MeasureOperation("test", operation)
	}
}

// Memory allocation benchmarks
func BenchmarkMemoryAllocations_WithPool(b *testing.B) {
	pool := NewPool(
		func() []int { return make([]int, 100) },
		func(slice []int) {
			for i := range slice {
				slice[i] = 0
			}
		},
	)
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			slice := pool.Get()
			// Use the slice
			for i := range slice {
				slice[i] = i
			}
			pool.Put(slice)
		}
	})
}

func BenchmarkMemoryAllocations_WithoutPool(b *testing.B) {
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			slice := make([]int, 100)
			// Use the slice
			for i := range slice {
				slice[i] = i
			}
			// Let GC handle cleanup
			_ = slice
		}
	})
}

// Concurrent access benchmarks
func BenchmarkConcurrentAccess_PerformanceMetrics(b *testing.B) {
	pm := NewPerformanceMetrics()
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			go pm.RecordOperation(time.Millisecond, true)
			go pm.GetSnapshot()
		}
	})
}

func BenchmarkConcurrentAccess_LatencyHistogram(b *testing.B) {
	lh := NewLatencyHistogram(1000)
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			go lh.Record(time.Millisecond * 50)
			go lh.GetPercentile(95)
		}
	})
}