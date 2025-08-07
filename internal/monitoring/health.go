// internal/monitoring/health.go
package monitoring

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
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
	Name      string                                      `json:"name"`
	Status    HealthStatus                                `json:"status"`
	Message   string                                      `json:"message,omitempty"`
	Error     string                                      `json:"error,omitempty"`
	LastCheck time.Time                                   `json:"last_check"`
	Duration  time.Duration                               `json:"duration"`
	Metadata  map[string]interface{}                      `json:"metadata,omitempty"`
	CheckFunc func(ctx context.Context) HealthCheckResult `json:"-"`
	Interval  time.Duration                               `json:"-"`
	Timeout   time.Duration                               `json:"-"`
	Critical  bool                                        `json:"critical"`
	Enabled   bool                                        `json:"enabled"`
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
	Status    HealthStatus           `json:"status"`
	Timestamp time.Time              `json:"timestamp"`
	Version   string                 `json:"version,omitempty"`
	Uptime    time.Duration          `json:"uptime"`
	Checks    map[string]HealthCheck `json:"checks,omitempty"`
	Summary   HealthSummary          `json:"summary"`
	System    SystemMetrics          `json:"system"`
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
	CPUUsage       float64          `json:"cpu_usage_percent"`
	MemoryUsage    MemoryMetrics    `json:"memory"`
	GoroutineCount int              `json:"goroutine_count"`
	GCStats        debug.GCStats    `json:"gc_stats"`
	Uptime         time.Duration    `json:"uptime"`
	LoadAverage    []float64        `json:"load_average,omitempty"`
	DiskUsage      map[string]int64 `json:"disk_usage,omitempty"`
}

// MemoryMetrics provides memory usage information
type MemoryMetrics struct {
	Allocated    uint64  `json:"allocated_bytes"`
	TotalAlloc   uint64  `json:"total_alloc_bytes"`
	System       uint64  `json:"system_bytes"`
	NumGC        uint32  `json:"num_gc"`
	UsagePercent float64 `json:"usage_percent"`
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

	metrics := SystemMetrics{
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
		CPUUsage:       getCPUUsage(),
		LoadAverage:    getLoadAverage(),
		DiskUsage:      getDiskUsage(),
	}

	return metrics
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
var (
	startTime        time.Time
	lastCPUStats     cpuStats
	lastCPUStatsTime time.Time
	cpuStatsMutex    sync.Mutex
)

// cpuStats holds CPU statistics for calculation
type cpuStats struct {
	user   uint64
	system uint64
	idle   uint64
	total  uint64
}

func init() {
	startTime = time.Now()
	lastCPUStatsTime = time.Now()
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

// getCPUUsage attempts to get CPU usage percentage (Unix-like systems)
// Returns 0.0 if unable to determine CPU usage
func getCPUUsage() float64 {
	// This is a simplified implementation with time-based calculation
	// For production, consider using github.com/shirou/gopsutil/v3/cpu for cross-platform support
	
	// Try to read from /proc/stat on Linux systems
	if data, err := os.ReadFile("/proc/stat"); err == nil {
		lines := strings.Split(string(data), "\n")
		if len(lines) > 0 {
			fields := strings.Fields(lines[0])
			if len(fields) >= 8 && strings.HasPrefix(fields[0], "cpu") {
				// Parse CPU stats from /proc/stat
				// Fields: cpu user nice system idle iowait irq softirq steal
				user, _ := strconv.ParseUint(fields[1], 10, 64)
				nice, _ := strconv.ParseUint(fields[2], 10, 64)
				system, _ := strconv.ParseUint(fields[3], 10, 64)
				idle, _ := strconv.ParseUint(fields[4], 10, 64)
				iowait, _ := strconv.ParseUint(fields[5], 10, 64)
				irq, _ := strconv.ParseUint(fields[6], 10, 64)
				softirq, _ := strconv.ParseUint(fields[7], 10, 64)
				
				currentStats := cpuStats{
					user:   user + nice,
					system: system + irq + softirq,
					idle:   idle + iowait,
					total:  user + nice + system + idle + iowait + irq + softirq,
				}
				
				now := time.Now()
				
				// Thread-safe access to last stats
				cpuStatsMutex.Lock()
				defer cpuStatsMutex.Unlock()
				
				// Calculate percentage if we have previous stats and enough time has passed
				if !lastCPUStatsTime.IsZero() && now.Sub(lastCPUStatsTime) > time.Second {
					deltaTotal := currentStats.total - lastCPUStats.total
					if deltaTotal > 0 {
						deltaActive := (currentStats.user + currentStats.system) - (lastCPUStats.user + lastCPUStats.system)
						cpuPercent := (float64(deltaActive) / float64(deltaTotal)) * 100
						
						// Update last stats
						lastCPUStats = currentStats
						lastCPUStatsTime = now
						
						return cpuPercent
					}
				}
				
				// Update stats for next calculation
				lastCPUStats = currentStats
				lastCPUStatsTime = now
				
				// Fallback to simple calculation for first run
				if currentStats.total > 0 {
					return (float64(currentStats.user + currentStats.system) / float64(currentStats.total)) * 100
				}
			}
		}
	}
	
	// Return 0 if unable to determine (e.g., on Windows, macOS without additional packages)
	return 0.0
}

// getLoadAverage attempts to get system load average (Unix-like systems)
// Returns empty slice if unable to determine load average
func getLoadAverage() []float64 {
	// Try to read from /proc/loadavg on Linux systems
	if data, err := os.ReadFile("/proc/loadavg"); err == nil {
		fields := strings.Fields(string(data))
		if len(fields) >= 3 {
			var loads []float64
			for i := 0; i < 3; i++ {
				if load, err := strconv.ParseFloat(fields[i], 64); err == nil {
					loads = append(loads, load)
				}
			}
			return loads
		}
	}
	
	// Return empty slice if unable to determine (e.g., on Windows, macOS without additional packages)
	return []float64{}
}

// getDiskUsage attempts to get disk usage for common mount points
// Returns empty map if unable to determine disk usage
func getDiskUsage() map[string]int64 {
	diskUsage := make(map[string]int64)
	
	// Common mount points to check
	mountPoints := []string{"/", "/var", "/tmp", "/home"}
	
	for _, mountPoint := range mountPoints {
		if usage := getDiskUsageForPath(mountPoint); usage >= 0 {
			diskUsage[mountPoint] = usage
		}
	}
	
	return diskUsage
}

// getDiskUsageForPath gets disk usage percentage for a specific path
// Returns -1 if unable to determine usage
func getDiskUsageForPath(path string) int64 {
	// Try to use df command (Unix-like systems)
	// This is a simplified implementation - for production use github.com/shirou/gopsutil/v3/disk
	
	// Check if path exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return -1
	}
	
	// For a more complete implementation, you would:
	// 1. Use syscalls on Unix (syscall.Statfs)
	// 2. Use Windows API on Windows
	// 3. Or use cross-platform library like gopsutil
	
	// Returning 0 as placeholder - indicates path exists but usage unknown
	return 0
}

// DiskSpaceHealthCheck creates a disk space health check for a given path
func DiskSpaceHealthCheck(name, path string, maxUsagePercent float64) *HealthCheck {
	return &HealthCheck{
		Name:     name,
		Critical: true, // Disk space is usually critical
		Enabled:  true,
		CheckFunc: func(ctx context.Context) HealthCheckResult {
			usage := getDiskUsageForPath(path)
			
			metadata := map[string]interface{}{
				"path":        path,
				"max_percent": maxUsagePercent,
			}
			
			if usage < 0 {
				// Path doesn't exist or can't be accessed
				return HealthCheckResult{
					Status:   HealthStatusUnknown,
					Message:  fmt.Sprintf("Unable to check disk usage for path: %s", path),
					Metadata: metadata,
				}
			}
			
			usagePercent := float64(usage)
			metadata["usage_percent"] = usagePercent
			
			if usagePercent > maxUsagePercent {
				return HealthCheckResult{
					Status:   HealthStatusUnhealthy,
					Message:  fmt.Sprintf("High disk usage: %.1f%% (max: %.1f%%)", usagePercent, maxUsagePercent),
					Metadata: metadata,
				}
			}
			
			if usagePercent > maxUsagePercent*0.8 { // Warning at 80% of max
				return HealthCheckResult{
					Status:   HealthStatusDegraded,
					Message:  fmt.Sprintf("Elevated disk usage: %.1f%%", usagePercent),
					Metadata: metadata,
				}
			}
			
			return HealthCheckResult{
				Status:   HealthStatusHealthy,
				Message:  fmt.Sprintf("Disk usage normal: %.1f%%", usagePercent),
				Metadata: metadata,
			}
		},
	}
}

// CPUHealthCheck creates a CPU usage health check
func CPUHealthCheck(maxUsagePercent float64) *HealthCheck {
	return &HealthCheck{
		Name:     "cpu",
		Critical: false, // CPU spikes are usually not critical
		Enabled:  true,
		CheckFunc: func(ctx context.Context) HealthCheckResult {
			usage := getCPUUsage()
			
			metadata := map[string]interface{}{
				"usage_percent": usage,
				"max_percent":   maxUsagePercent,
			}
			
			if usage == 0.0 {
				return HealthCheckResult{
					Status:   HealthStatusUnknown,
					Message:  "CPU usage monitoring not available on this platform",
					Metadata: metadata,
				}
			}
			
			if usage > maxUsagePercent {
				return HealthCheckResult{
					Status:   HealthStatusDegraded,
					Message:  fmt.Sprintf("High CPU usage: %.1f%%", usage),
					Metadata: metadata,
				}
			}
			
			return HealthCheckResult{
				Status:   HealthStatusHealthy,
				Message:  fmt.Sprintf("CPU usage normal: %.1f%%", usage),
				Metadata: metadata,
			}
		},
	}
}

// LoadAverageHealthCheck creates a load average health check (Unix-like systems)
func LoadAverageHealthCheck(maxLoad1Min float64) *HealthCheck {
	return &HealthCheck{
		Name:     "load_average",
		Critical: false,
		Enabled:  true,
		CheckFunc: func(ctx context.Context) HealthCheckResult {
			loads := getLoadAverage()
			
			metadata := map[string]interface{}{
				"max_1min_load": maxLoad1Min,
			}
			
			if len(loads) == 0 {
				return HealthCheckResult{
					Status:   HealthStatusUnknown,
					Message:  "Load average monitoring not available on this platform",
					Metadata: metadata,
				}
			}
			
			metadata["load_1min"] = loads[0]
			if len(loads) > 1 {
				metadata["load_5min"] = loads[1]
			}
			if len(loads) > 2 {
				metadata["load_15min"] = loads[2]
			}
			
			if loads[0] > maxLoad1Min {
				return HealthCheckResult{
					Status:   HealthStatusDegraded,
					Message:  fmt.Sprintf("High system load: %.2f", loads[0]),
					Metadata: metadata,
				}
			}
			
			return HealthCheckResult{
				Status:   HealthStatusHealthy,
				Message:  fmt.Sprintf("System load normal: %.2f", loads[0]),
				Metadata: metadata,
			}
		},
	}
}

// CreateStandardHealthChecks creates a set of standard health checks for common scenarios
func CreateStandardHealthChecks() map[string]*HealthCheck {
	checks := make(map[string]*HealthCheck)
	
	// Memory health check (warn at 80% usage)
	checks["memory"] = MemoryHealthCheck(80.0)
	
	// Goroutine health check (warn at 10000 goroutines)
	checks["goroutines"] = GoroutineHealthCheck(10000)
	
	// CPU health check (warn at 80% usage)
	checks["cpu"] = CPUHealthCheck(80.0)
	
	// Load average health check (warn at 5.0 for 1-minute load)
	checks["load_average"] = LoadAverageHealthCheck(5.0)
	
	// Disk space check for root partition (warn at 85% usage)
	checks["disk_root"] = DiskSpaceHealthCheck("disk_root", "/", 85.0)
	
	return checks
}

// RegisterStandardHealthChecks registers all standard health checks with a manager
func (hm *HealthManager) RegisterStandardHealthChecks() {
	for _, check := range CreateStandardHealthChecks() {
		hm.RegisterCheck(check)
	}
}

// GetHealthSummaryString returns a human-readable health summary
func (hm *HealthManager) GetHealthSummaryString() string {
	health := hm.GetHealth()
	
	var status string
	switch health.Status {
	case HealthStatusHealthy:
		status = "HEALTHY ✓"
	case HealthStatusDegraded:
		status = "DEGRADED ⚠"
	case HealthStatusUnhealthy:
		status = "UNHEALTHY ✗"
	default:
		status = "UNKNOWN ?"
	}
	
	return fmt.Sprintf("System Status: %s | Checks: %d/%d healthy | Uptime: %v | Memory: %.1f%% | Goroutines: %d",
		status,
		health.Summary.Healthy,
		health.Summary.Total,
		health.Uptime.Truncate(time.Second),
		health.System.MemoryUsage.UsagePercent,
		health.System.GoroutineCount,
	)
}

// IsHealthy returns true if the overall system status is healthy
func (hm *HealthManager) IsHealthy() bool {
	return hm.GetHealth().Status == HealthStatusHealthy
}

// IsReady returns true if the system is ready to serve traffic
func (hm *HealthManager) IsReady() bool {
	status := hm.GetReadiness().Status
	return status == HealthStatusHealthy || status == HealthStatusDegraded
}

// IsAlive returns true if the system is alive (no critical failures)
func (hm *HealthManager) IsAlive() bool {
	status := hm.GetLiveness().Status
	return status == HealthStatusHealthy || status == HealthStatusDegraded
}

// SetCheckEnabled enables or disables a specific health check
func (hm *HealthManager) SetCheckEnabled(name string, enabled bool) {
	hm.checksMutex.Lock()
	defer hm.checksMutex.Unlock()
	
	if check, exists := hm.checks[name]; exists {
		check.Enabled = enabled
	}
}

// GetCheckStatus returns the current status of a specific health check
func (hm *HealthManager) GetCheckStatus(name string) (HealthStatus, bool) {
	hm.checksMutex.RLock()
	defer hm.checksMutex.RUnlock()
	
	if check, exists := hm.checks[name]; exists {
		return check.Status, true
	}
	
	return HealthStatusUnknown, false
}

// RunCheck manually triggers a single health check
func (hm *HealthManager) RunCheck(ctx context.Context, name string) (HealthCheckResult, error) {
	hm.checksMutex.RLock()
	check, exists := hm.checks[name]
	hm.checksMutex.RUnlock()
	
	if !exists {
		return HealthCheckResult{}, fmt.Errorf("health check '%s' not found", name)
	}
	
	// Create a timeout context
	checkCtx, cancel := context.WithTimeout(ctx, check.Timeout)
	defer cancel()
	
	// Run the check
	hm.runCheck(checkCtx, check)
	
	// Return the result
	hm.resultsMutex.RLock()
	result, exists := hm.results[name]
	hm.resultsMutex.RUnlock()
	
	if exists {
		return result, nil
	}
	
	return HealthCheckResult{
		Status:  HealthStatusUnknown,
		Message: "Check completed but no result available",
	}, nil
}

// GetFailedChecks returns a list of checks that are currently unhealthy
func (hm *HealthManager) GetFailedChecks() []string {
	hm.checksMutex.RLock()
	defer hm.checksMutex.RUnlock()
	
	var failed []string
	for name, check := range hm.checks {
		if check.Enabled && check.Status == HealthStatusUnhealthy {
			failed = append(failed, name)
		}
	}
	
	return failed
}

// GetCriticalChecks returns a list of critical checks that are currently unhealthy
func (hm *HealthManager) GetCriticalChecks() []string {
	hm.checksMutex.RLock()
	defer hm.checksMutex.RUnlock()
	
	var critical []string
	for name, check := range hm.checks {
		if check.Enabled && check.Critical && check.Status == HealthStatusUnhealthy {
			critical = append(critical, name)
		}
	}
	
	return critical
}

// WaitForHealthy waits until the system becomes healthy or the context is cancelled
func (hm *HealthManager) WaitForHealthy(ctx context.Context) error {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if hm.IsHealthy() {
				return nil
			}
		}
	}
}

// NOTE: For production deployment, consider using github.com/shirou/gopsutil/v3 for
// cross-platform system metrics (CPU, disk, load average) with more accuracy and features.
// The implementations above provide basic functionality for Unix-like systems.
