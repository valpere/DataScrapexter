// internal/monitoring/health.go
package monitoring

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"runtime/debug"
	"sync"
	"time"
)

// HealthStatus represents the health status of a component
type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
	HealthStatusDegraded  HealthStatus = "degraded"
	HealthStatusUnknown   HealthStatus = "unknown"
)

// HealthCheck represents a single health check
type HealthCheck struct {
	Name        string                 `json:"name"`
	Status      HealthStatus           `json:"status"`
	Message     string                 `json:"message,omitempty"`
	Error       string                 `json:"error,omitempty"`
	LastCheck   time.Time              `json:"last_check"`
	Duration    time.Duration          `json:"duration"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	CheckFunc   func(ctx context.Context) HealthCheckResult `json:"-"`
	Interval    time.Duration          `json:"-"`
	Timeout     time.Duration          `json:"-"`
	Critical    bool                   `json:"critical"`
	Enabled     bool                   `json:"enabled"`
}

// HealthCheckResult represents the result of a health check
type HealthCheckResult struct {
	Status   HealthStatus           `json:"status"`
	Message  string                 `json:"message,omitempty"`
	Error    error                  `json:"-"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// HealthManager manages health checks and monitoring
type HealthManager struct {
	checks       map[string]*HealthCheck
	checksMutex  sync.RWMutex
	results      map[string]HealthCheckResult
	resultsMutex sync.RWMutex
	ticker       *time.Ticker
	stopCh       chan struct{}
	config       HealthConfig
}

// HealthConfig configuration for health monitoring
type HealthConfig struct {
	CheckInterval     time.Duration `json:"check_interval"`
	DefaultTimeout    time.Duration `json:"default_timeout"`
	HealthEndpoint    string        `json:"health_endpoint"`
	ReadinessEndpoint string        `json:"readiness_endpoint"`
	LivenessEndpoint  string        `json:"liveness_endpoint"`
	DetailedResponse  bool          `json:"detailed_response"`
	EnableCaching     bool          `json:"enable_caching"`
	CacheTTL          time.Duration `json:"cache_ttl"`
}

// SystemHealth represents overall system health information
type SystemHealth struct {
	Status     HealthStatus           `json:"status"`
	Timestamp  time.Time              `json:"timestamp"`
	Version    string                 `json:"version,omitempty"`
	Uptime     time.Duration          `json:"uptime"`
	Checks     map[string]HealthCheck `json:"checks,omitempty"`
	Summary    HealthSummary          `json:"summary"`
	System     SystemMetrics          `json:"system"`
}

// HealthSummary provides a summary of health checks
type HealthSummary struct {
	Total     int `json:"total"`
	Healthy   int `json:"healthy"`
	Unhealthy int `json:"unhealthy"`
	Degraded  int `json:"degraded"`
	Unknown   int `json:"unknown"`
	Critical  int `json:"critical"`
}

// SystemMetrics provides system-level metrics
type SystemMetrics struct {
	CPUUsage       float64           `json:"cpu_usage_percent"`
	MemoryUsage    MemoryMetrics     `json:"memory"`
	GoroutineCount int               `json:"goroutine_count"`
	GCStats        debug.GCStats     `json:"gc_stats"`
	Uptime         time.Duration     `json:"uptime"`
	LoadAverage    []float64         `json:"load_average,omitempty"`
	DiskUsage      map[string]int64  `json:"disk_usage,omitempty"`
}

// MemoryMetrics provides memory usage information
type MemoryMetrics struct {
	Allocated     uint64  `json:"allocated_bytes"`
	TotalAlloc    uint64  `json:"total_alloc_bytes"`
	System        uint64  `json:"system_bytes"`
	NumGC         uint32  `json:"num_gc"`
	UsagePercent  float64 `json:"usage_percent"`
}

// NewHealthManager creates a new health manager
func NewHealthManager(config HealthConfig) *HealthManager {
	if config.CheckInterval == 0 {
		config.CheckInterval = 30 * time.Second
	}
	if config.DefaultTimeout == 0 {
		config.DefaultTimeout = 10 * time.Second
	}
	if config.HealthEndpoint == "" {
		config.HealthEndpoint = "/health"
	}
	if config.ReadinessEndpoint == "" {
		config.ReadinessEndpoint = "/ready"
	}
	if config.LivenessEndpoint == "" {
		config.LivenessEndpoint = "/live"
	}
	if config.CacheTTL == 0 {
		config.CacheTTL = 5 * time.Second
	}

	return &HealthManager{
		checks:  make(map[string]*HealthCheck),
		results: make(map[string]HealthCheckResult),
		stopCh:  make(chan struct{}),
		config:  config,
	}
}

// RegisterCheck registers a new health check
func (hm *HealthManager) RegisterCheck(check *HealthCheck) {
	if check.Timeout == 0 {
		check.Timeout = hm.config.DefaultTimeout
	}
	if check.Interval == 0 {
		check.Interval = hm.config.CheckInterval
	}
	if !check.Enabled {
		check.Enabled = true
	}

	hm.checksMutex.Lock()
	hm.checks[check.Name] = check
	hm.checksMutex.Unlock()
}

// RemoveCheck removes a health check
func (hm *HealthManager) RemoveCheck(name string) {
	hm.checksMutex.Lock()
	delete(hm.checks, name)
	hm.checksMutex.Unlock()

	hm.resultsMutex.Lock()
	delete(hm.results, name)
	hm.resultsMutex.Unlock()
}

// Start starts the health monitoring
func (hm *HealthManager) Start(ctx context.Context) {
	hm.ticker = time.NewTicker(hm.config.CheckInterval)
	
	go func() {
		// Run initial checks
		hm.runAllChecks(ctx)
		
		for {
			select {
			case <-hm.ticker.C:
				hm.runAllChecks(ctx)
			case <-hm.stopCh:
				return
			case <-ctx.Done():
				return
			}
		}
	}()
}

// Stop stops the health monitoring
func (hm *HealthManager) Stop() {
	if hm.ticker != nil {
		hm.ticker.Stop()
	}
	close(hm.stopCh)
}

// runAllChecks runs all registered health checks
func (hm *HealthManager) runAllChecks(ctx context.Context) {
	hm.checksMutex.RLock()
	checks := make([]*HealthCheck, 0, len(hm.checks))
	for _, check := range hm.checks {
		if check.Enabled {
			checks = append(checks, check)
		}
	}
	hm.checksMutex.RUnlock()

	// Run checks concurrently
	var wg sync.WaitGroup
	for _, check := range checks {
		wg.Add(1)
		go func(c *HealthCheck) {
			defer wg.Done()
			hm.runCheck(ctx, c)
		}(check)
	}
	wg.Wait()
}

// runCheck runs a single health check
func (hm *HealthManager) runCheck(ctx context.Context, check *HealthCheck) {
	start := time.Now()
	
	// Create timeout context
	checkCtx, cancel := context.WithTimeout(ctx, check.Timeout)
	defer cancel()
	
	var result HealthCheckResult
	
	if check.CheckFunc != nil {
		result = check.CheckFunc(checkCtx)
	} else {
		result = HealthCheckResult{
			Status:  HealthStatusUnknown,
			Message: "No check function defined",
		}
	}
	
	duration := time.Since(start)
	
	// Update check metadata
	check.LastCheck = start
	check.Duration = duration
	check.Status = result.Status
	check.Message = result.Message
	if result.Error != nil {
		check.Error = result.Error.Error()
	} else {
		check.Error = ""
	}
	if result.Metadata != nil {
		check.Metadata = result.Metadata
	}
	
	// Store result
	hm.resultsMutex.Lock()
	hm.results[check.Name] = result
	hm.resultsMutex.Unlock()
}

// GetHealth returns the overall health status
func (hm *HealthManager) GetHealth() SystemHealth {
	hm.checksMutex.RLock()
	hm.resultsMutex.RLock()
	defer hm.checksMutex.RUnlock()
	defer hm.resultsMutex.RUnlock()

	health := SystemHealth{
		Timestamp: time.Now(),
		Uptime:    time.Since(startTime),
		System:    hm.getSystemMetrics(),
	}

	if hm.config.DetailedResponse {
		health.Checks = make(map[string]HealthCheck)
		for name, check := range hm.checks {
			health.Checks[name] = *check
		}
	}

	// Calculate overall status and summary
	summary := HealthSummary{}
	overallStatus := HealthStatusHealthy
	
	for _, check := range hm.checks {
		if !check.Enabled {
			continue
		}
		
		summary.Total++
		
		switch check.Status {
		case HealthStatusHealthy:
			summary.Healthy++
		case HealthStatusUnhealthy:
			summary.Unhealthy++
			if check.Critical {
				overallStatus = HealthStatusUnhealthy
			} else if overallStatus == HealthStatusHealthy {
				overallStatus = HealthStatusDegraded
			}
		case HealthStatusDegraded:
			summary.Degraded++
			if overallStatus == HealthStatusHealthy {
				overallStatus = HealthStatusDegraded
			}
		case HealthStatusUnknown:
			summary.Unknown++
			if overallStatus == HealthStatusHealthy {
				overallStatus = HealthStatusDegraded
			}
		}
		
		if check.Critical {
			summary.Critical++
		}
	}

	health.Status = overallStatus
	health.Summary = summary

	return health
}

// GetReadiness returns readiness status (for Kubernetes readiness probes)
func (hm *HealthManager) GetReadiness() SystemHealth {
	health := hm.GetHealth()
	
	// Readiness focuses on whether the service can serve traffic
	// We consider degraded as ready (but log it), but unhealthy as not ready
	if health.Status == HealthStatusUnhealthy {
		health.Status = HealthStatusUnhealthy
	} else {
		health.Status = HealthStatusHealthy
	}
	
	return health
}

// GetLiveness returns liveness status (for Kubernetes liveness probes)  
func (hm *HealthManager) GetLiveness() SystemHealth {
	health := hm.GetHealth()
	
	// Liveness is about whether the service is alive and should be restarted
	// Only critical failures should affect liveness
	criticalFailures := false
	
	hm.checksMutex.RLock()
	for _, check := range hm.checks {
		if check.Critical && check.Status == HealthStatusUnhealthy {
			criticalFailures = true
			break
		}
	}
	hm.checksMutex.RUnlock()
	
	if criticalFailures {
		health.Status = HealthStatusUnhealthy
	} else {
		health.Status = HealthStatusHealthy
	}
	
	return health
}

// getSystemMetrics collects system-level metrics
func (hm *HealthManager) getSystemMetrics() SystemMetrics {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	var gcStats debug.GCStats
	debug.ReadGCStats(&gcStats)

	return SystemMetrics{
		MemoryUsage: MemoryMetrics{
			Allocated:    m.Alloc,
			TotalAlloc:   m.TotalAlloc,
			System:       m.Sys,
			NumGC:        m.NumGC,
			UsagePercent: float64(m.Alloc) / float64(m.Sys) * 100,
		},
		GoroutineCount: runtime.NumGoroutine(),
		GCStats:        gcStats,
		Uptime:         time.Since(startTime),
	}
}

// HealthHandler returns HTTP handlers for health endpoints
func (hm *HealthManager) HealthHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		health := hm.GetHealth()
		
		w.Header().Set("Content-Type", "application/json")
		
		if health.Status == HealthStatusUnhealthy {
			w.WriteHeader(http.StatusServiceUnavailable)
		} else if health.Status == HealthStatusDegraded {
			w.WriteHeader(http.StatusOK) // Still serve traffic but log warnings
		} else {
			w.WriteHeader(http.StatusOK)
		}
		
		json.NewEncoder(w).Encode(health)
	}
}

// ReadinessHandler returns HTTP handler for readiness endpoint
func (hm *HealthManager) ReadinessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		health := hm.GetReadiness()
		
		w.Header().Set("Content-Type", "application/json")
		
		if health.Status == HealthStatusUnhealthy {
			w.WriteHeader(http.StatusServiceUnavailable)
		} else {
			w.WriteHeader(http.StatusOK)
		}
		
		json.NewEncoder(w).Encode(health)
	}
}

// LivenessHandler returns HTTP handler for liveness endpoint
func (hm *HealthManager) LivenessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		health := hm.GetLiveness()
		
		w.Header().Set("Content-Type", "application/json")
		
		if health.Status == HealthStatusUnhealthy {
			w.WriteHeader(http.StatusServiceUnavailable)
		} else {
			w.WriteHeader(http.StatusOK)
		}
		
		json.NewEncoder(w).Encode(health)
	}
}

// Package-level variables initialized safely
var startTime time.Time

func init() {
	startTime = time.Now()
}

// DatabaseHealthCheck creates a database connectivity health check
func DatabaseHealthCheck(name string, checkFunc func(ctx context.Context) error) *HealthCheck {
	return &HealthCheck{
		Name:     name,
		Critical: true,
		Enabled:  true,
		CheckFunc: func(ctx context.Context) HealthCheckResult {
			err := checkFunc(ctx)
			if err != nil {
				return HealthCheckResult{
					Status:  HealthStatusUnhealthy,
					Message: "Database connection failed",
					Error:   err,
				}
			}
			return HealthCheckResult{
				Status:  HealthStatusHealthy,
				Message: "Database connection successful",
			}
		},
	}
}

// MemoryHealthCheck creates a memory usage health check
func MemoryHealthCheck(maxUsagePercent float64) *HealthCheck {
	return &HealthCheck{
		Name:     "memory",
		Critical: false,
		Enabled:  true,
		CheckFunc: func(ctx context.Context) HealthCheckResult {
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			
			usagePercent := float64(m.Alloc) / float64(m.Sys) * 100
			
			metadata := map[string]interface{}{
				"allocated_bytes": m.Alloc,
				"system_bytes":    m.Sys,
				"usage_percent":   usagePercent,
			}
			
			if usagePercent > maxUsagePercent {
				return HealthCheckResult{
					Status:   HealthStatusDegraded,
					Message:  fmt.Sprintf("High memory usage: %.2f%%", usagePercent),
					Metadata: metadata,
				}
			}
			
			return HealthCheckResult{
				Status:   HealthStatusHealthy,
				Message:  fmt.Sprintf("Memory usage normal: %.2f%%", usagePercent),
				Metadata: metadata,
			}
		},
	}
}

// GoroutineHealthCheck creates a goroutine count health check
func GoroutineHealthCheck(maxGoroutines int) *HealthCheck {
	return &HealthCheck{
		Name:     "goroutines",
		Critical: false,
		Enabled:  true,
		CheckFunc: func(ctx context.Context) HealthCheckResult {
			count := runtime.NumGoroutine()
			
			metadata := map[string]interface{}{
				"goroutine_count": count,
				"max_allowed":     maxGoroutines,
			}
			
			if count > maxGoroutines {
				return HealthCheckResult{
					Status:   HealthStatusDegraded,
					Message:  fmt.Sprintf("High goroutine count: %d", count),
					Metadata: metadata,
				}
			}
			
			return HealthCheckResult{
				Status:   HealthStatusHealthy,
				Message:  fmt.Sprintf("Goroutine count normal: %d", count),
				Metadata: metadata,
			}
		},
	}
}

// HTTPHealthCheck creates an HTTP endpoint health check
func HTTPHealthCheck(name, url string, timeout time.Duration) *HealthCheck {
	return &HealthCheck{
		Name:     name,
		Critical: false,
		Enabled:  true,
		Timeout:  timeout,
		CheckFunc: func(ctx context.Context) HealthCheckResult {
			client := &http.Client{Timeout: timeout}
			
			req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
			if err != nil {
				return HealthCheckResult{
					Status:  HealthStatusUnhealthy,
					Message: "Failed to create request",
					Error:   err,
				}
			}
			
			start := time.Now()
			resp, err := client.Do(req)
			duration := time.Since(start)
			
			metadata := map[string]interface{}{
				"url":              url,
				"response_time_ms": duration.Milliseconds(),
			}
			
			if err != nil {
				return HealthCheckResult{
					Status:   HealthStatusUnhealthy,
					Message:  "HTTP request failed",
					Error:    err,
					Metadata: metadata,
				}
			}
			defer resp.Body.Close()
			
			metadata["status_code"] = resp.StatusCode
			
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				return HealthCheckResult{
					Status:   HealthStatusHealthy,
					Message:  fmt.Sprintf("HTTP check passed (%d)", resp.StatusCode),
					Metadata: metadata,
				}
			}
			
			return HealthCheckResult{
				Status:   HealthStatusUnhealthy,
				Message:  fmt.Sprintf("HTTP check failed (%d)", resp.StatusCode),
				Metadata: metadata,
			}
		},
	}
}

// DiskSpaceHealthCheck creates a disk space health check
// DEPRECATED: This is a stub implementation that is not functional.
// Consider removing this function or implementing platform-specific disk space checking.
// For production use, implement using:
// - Unix/Linux: syscall.Statfs() or golang.org/x/sys/unix.Statfs()
// - Windows: golang.org/x/sys/windows GetDiskFreeSpaceEx()
func DiskSpaceHealthCheck(path string, minFreePercent float64) *HealthCheck {
	return &HealthCheck{
		Name:     "disk_space_" + path,
		Critical: false,
		Enabled:  false, // Disabled by default since it's not implemented
		CheckFunc: func(ctx context.Context) HealthCheckResult {
			// TODO: Implement actual disk space checking
			// This requires platform-specific code:
			// - Unix/Linux: syscall.Statfs()
			// - Windows: GetDiskFreeSpaceEx()
			return HealthCheckResult{
				Status:  HealthStatusUnknown,
				Message: fmt.Sprintf("Disk space check not implemented for path: %s", path),
				Metadata: map[string]interface{}{
					"path":               path,
					"min_free_percent":   minFreePercent,
					"implementation":     "stub",
					"requires":          "platform-specific implementation",
				},
			}
		},
	}
}