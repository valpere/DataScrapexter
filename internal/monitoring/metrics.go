// internal/monitoring/metrics.go
package monitoring

import (
	"context"
	"net/http"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// MetricsManager manages Prometheus metrics for DataScrapexter
type MetricsManager struct {
	// Request metrics
	requestsTotal    *prometheus.CounterVec
	requestDuration  *prometheus.HistogramVec
	requestsInFlight *prometheus.GaugeVec
	requestErrors    *prometheus.CounterVec
	requestRetries   *prometheus.CounterVec

	// Scraping metrics
	pagesScraped      *prometheus.CounterVec
	extractionSuccess *prometheus.CounterVec
	extractionErrors  *prometheus.CounterVec
	recordsExtracted  *prometheus.CounterVec
	extractionTime    *prometheus.HistogramVec

	// Anti-detection metrics
	proxyUsage        *prometheus.CounterVec
	captchaSolved     *prometheus.CounterVec
	captchaFailed     *prometheus.CounterVec
	captchaSolveTime  *prometheus.HistogramVec
	userAgentRotation *prometheus.CounterVec

	// Output metrics
	outputSuccess  *prometheus.CounterVec
	outputErrors   *prometheus.CounterVec
	outputTime     *prometheus.HistogramVec
	outputSize     *prometheus.HistogramVec
	recordsWritten *prometheus.CounterVec

	// System metrics
	memoryUsage    prometheus.Gauge
	cpuUsage       prometheus.Gauge
	goroutineCount prometheus.Gauge

	// Job metrics
	jobsTotal   *prometheus.CounterVec
	jobDuration *prometheus.HistogramVec
	jobsActive  prometheus.Gauge
	jobsQueued  prometheus.Gauge

	// Rate limiting metrics
	rateLimitHits  *prometheus.CounterVec
	rateLimitWaits *prometheus.HistogramVec

	// Custom metrics
	customMetrics map[string]prometheus.Collector
	customMutex   sync.RWMutex

	// Configuration
	namespace string
	subsystem string
	labels    map[string]string
}

// MetricsConfig configuration for metrics
type MetricsConfig struct {
	Namespace            string            `json:"namespace"`
	Subsystem            string            `json:"subsystem"`
	Labels               map[string]string `json:"labels"`
	EnableGoMetrics      bool              `json:"enable_go_metrics"`
	EnableProcessMetrics bool              `json:"enable_process_metrics"`
	MetricsPath          string            `json:"metrics_path"`
	ListenAddress        string            `json:"listen_address"`
}

// NewMetricsManager creates a new metrics manager
func NewMetricsManager(config MetricsConfig) *MetricsManager {
	if config.Namespace == "" {
		config.Namespace = "datascrapexter"
	}
	if config.Subsystem == "" {
		config.Subsystem = "scraper"
	}
	if config.MetricsPath == "" {
		config.MetricsPath = "/metrics"
	}
	if config.ListenAddress == "" {
		config.ListenAddress = ":9090"
	}

	mm := &MetricsManager{
		namespace:     config.Namespace,
		subsystem:     config.Subsystem,
		labels:        config.Labels,
		customMetrics: make(map[string]prometheus.Collector),
	}

	mm.initializeMetrics()

	return mm
}

// initializeMetrics initializes all Prometheus metrics
func (mm *MetricsManager) initializeMetrics() {
	// Request metrics
	mm.requestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: mm.namespace,
			Subsystem: mm.subsystem,
			Name:      "requests_total",
			Help:      "Total number of HTTP requests made",
		},
		[]string{"method", "status_code", "host", "job_id"},
	)

	mm.requestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: mm.namespace,
			Subsystem: mm.subsystem,
			Name:      "request_duration_seconds",
			Help:      "HTTP request duration in seconds",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"method", "host", "job_id"},
	)

	mm.requestsInFlight = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: mm.namespace,
			Subsystem: mm.subsystem,
			Name:      "requests_in_flight",
			Help:      "Number of HTTP requests currently in flight",
		},
		[]string{"host", "job_id"},
	)

	mm.requestErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: mm.namespace,
			Subsystem: mm.subsystem,
			Name:      "request_errors_total",
			Help:      "Total number of HTTP request errors",
		},
		[]string{"error_type", "host", "job_id"},
	)

	mm.requestRetries = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: mm.namespace,
			Subsystem: mm.subsystem,
			Name:      "request_retries_total",
			Help:      "Total number of HTTP request retries",
		},
		[]string{"reason", "host", "job_id"},
	)

	// Scraping metrics
	mm.pagesScraped = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: mm.namespace,
			Subsystem: mm.subsystem,
			Name:      "pages_scraped_total",
			Help:      "Total number of pages scraped",
		},
		[]string{"host", "job_id", "status"},
	)

	mm.extractionSuccess = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: mm.namespace,
			Subsystem: mm.subsystem,
			Name:      "extraction_success_total",
			Help:      "Total number of successful data extractions",
		},
		[]string{"field", "job_id"},
	)

	mm.extractionErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: mm.namespace,
			Subsystem: mm.subsystem,
			Name:      "extraction_errors_total",
			Help:      "Total number of data extraction errors",
		},
		[]string{"field", "error_type", "job_id"},
	)

	mm.recordsExtracted = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: mm.namespace,
			Subsystem: mm.subsystem,
			Name:      "records_extracted_total",
			Help:      "Total number of records extracted",
		},
		[]string{"job_id"},
	)

	mm.extractionTime = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: mm.namespace,
			Subsystem: mm.subsystem,
			Name:      "extraction_duration_seconds",
			Help:      "Data extraction duration in seconds",
			Buckets:   []float64{0.1, 0.5, 1.0, 2.5, 5.0, 10.0, 25.0, 50.0, 100.0},
		},
		[]string{"job_id"},
	)

	// Anti-detection metrics
	mm.proxyUsage = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: mm.namespace,
			Subsystem: mm.subsystem,
			Name:      "proxy_usage_total",
			Help:      "Total number of proxy requests",
		},
		[]string{"proxy_host", "status", "job_id"},
	)

	mm.captchaSolved = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: mm.namespace,
			Subsystem: mm.subsystem,
			Name:      "captcha_solved_total",
			Help:      "Total number of CAPTCHAs solved",
		},
		[]string{"captcha_type", "solver", "job_id"},
	)

	mm.captchaFailed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: mm.namespace,
			Subsystem: mm.subsystem,
			Name:      "captcha_failed_total",
			Help:      "Total number of CAPTCHA solving failures",
		},
		[]string{"captcha_type", "solver", "error_type", "job_id"},
	)

	mm.captchaSolveTime = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: mm.namespace,
			Subsystem: mm.subsystem,
			Name:      "captcha_solve_duration_seconds",
			Help:      "CAPTCHA solving duration in seconds",
			Buckets:   []float64{5, 10, 30, 60, 120, 300, 600},
		},
		[]string{"captcha_type", "solver", "job_id"},
	)

	mm.userAgentRotation = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: mm.namespace,
			Subsystem: mm.subsystem,
			Name:      "user_agent_rotation_total",
			Help:      "Total number of user agent rotations",
		},
		[]string{"user_agent_type", "job_id"},
	)

	// Output metrics
	mm.outputSuccess = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: mm.namespace,
			Subsystem: mm.subsystem,
			Name:      "output_success_total",
			Help:      "Total number of successful output operations",
		},
		[]string{"format", "job_id"},
	)

	mm.outputErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: mm.namespace,
			Subsystem: mm.subsystem,
			Name:      "output_errors_total",
			Help:      "Total number of output errors",
		},
		[]string{"format", "error_type", "job_id"},
	)

	mm.outputTime = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: mm.namespace,
			Subsystem: mm.subsystem,
			Name:      "output_duration_seconds",
			Help:      "Output operation duration in seconds",
			Buckets:   []float64{0.1, 0.5, 1.0, 2.5, 5.0, 10.0, 25.0},
		},
		[]string{"format", "job_id"},
	)

	mm.outputSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: mm.namespace,
			Subsystem: mm.subsystem,
			Name:      "output_size_bytes",
			Help:      "Output file size in bytes",
			Buckets:   prometheus.ExponentialBuckets(1024, 2, 20), // 1KB to 512MB
		},
		[]string{"format", "job_id"},
	)

	mm.recordsWritten = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: mm.namespace,
			Subsystem: mm.subsystem,
			Name:      "records_written_total",
			Help:      "Total number of records written to output",
		},
		[]string{"format", "job_id"},
	)

	// System metrics
	mm.memoryUsage = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: mm.namespace,
			Subsystem: mm.subsystem,
			Name:      "memory_usage_bytes",
			Help:      "Current memory usage in bytes",
		},
	)

	mm.cpuUsage = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: mm.namespace,
			Subsystem: mm.subsystem,
			Name:      "cpu_usage_percent",
			Help:      "Current CPU usage percentage",
		},
	)

	mm.goroutineCount = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: mm.namespace,
			Subsystem: mm.subsystem,
			Name:      "goroutines_count",
			Help:      "Current number of goroutines",
		},
	)

	// Job metrics
	mm.jobsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: mm.namespace,
			Subsystem: mm.subsystem,
			Name:      "jobs_total",
			Help:      "Total number of scraping jobs",
		},
		[]string{"status", "job_type"},
	)

	mm.jobDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: mm.namespace,
			Subsystem: mm.subsystem,
			Name:      "job_duration_seconds",
			Help:      "Job execution duration in seconds",
			Buckets:   []float64{1, 5, 10, 30, 60, 300, 600, 1800, 3600},
		},
		[]string{"job_id", "job_type"},
	)

	mm.jobsActive = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: mm.namespace,
			Subsystem: mm.subsystem,
			Name:      "jobs_active",
			Help:      "Number of currently active jobs",
		},
	)

	mm.jobsQueued = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: mm.namespace,
			Subsystem: mm.subsystem,
			Name:      "jobs_queued",
			Help:      "Number of jobs in queue",
		},
	)

	// Rate limiting metrics
	mm.rateLimitHits = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: mm.namespace,
			Subsystem: mm.subsystem,
			Name:      "rate_limit_hits_total",
			Help:      "Total number of rate limit hits",
		},
		[]string{"host", "job_id"},
	)

	mm.rateLimitWaits = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: mm.namespace,
			Subsystem: mm.subsystem,
			Name:      "rate_limit_wait_duration_seconds",
			Help:      "Rate limit wait duration in seconds",
			Buckets:   []float64{0.1, 0.5, 1.0, 2.0, 5.0, 10.0, 30.0},
		},
		[]string{"host", "job_id"},
	)
}

// HTTP Request metrics
func (mm *MetricsManager) RecordRequest(method, host, jobID string, statusCode int, duration time.Duration) {
	mm.requestsTotal.WithLabelValues(method, strconv.Itoa(statusCode), host, jobID).Inc()
	mm.requestDuration.WithLabelValues(method, host, jobID).Observe(duration.Seconds())
}

func (mm *MetricsManager) IncRequestsInFlight(host, jobID string) {
	mm.requestsInFlight.WithLabelValues(host, jobID).Inc()
}

func (mm *MetricsManager) DecRequestsInFlight(host, jobID string) {
	mm.requestsInFlight.WithLabelValues(host, jobID).Dec()
}

func (mm *MetricsManager) RecordRequestError(errorType, host, jobID string) {
	mm.requestErrors.WithLabelValues(errorType, host, jobID).Inc()
}

func (mm *MetricsManager) RecordRequestRetry(reason, host, jobID string) {
	mm.requestRetries.WithLabelValues(reason, host, jobID).Inc()
}

// Scraping metrics
func (mm *MetricsManager) RecordPageScraped(host, jobID, status string) {
	mm.pagesScraped.WithLabelValues(host, jobID, status).Inc()
}

func (mm *MetricsManager) RecordExtractionSuccess(field, jobID string) {
	mm.extractionSuccess.WithLabelValues(field, jobID).Inc()
}

func (mm *MetricsManager) RecordExtractionError(field, errorType, jobID string) {
	mm.extractionErrors.WithLabelValues(field, errorType, jobID).Inc()
}

func (mm *MetricsManager) RecordRecordsExtracted(jobID string, count int) {
	mm.recordsExtracted.WithLabelValues(jobID).Add(float64(count))
}

func (mm *MetricsManager) RecordExtractionTime(jobID string, duration time.Duration) {
	mm.extractionTime.WithLabelValues(jobID).Observe(duration.Seconds())
}

// Anti-detection metrics
func (mm *MetricsManager) RecordProxyUsage(proxyHost, status, jobID string) {
	mm.proxyUsage.WithLabelValues(proxyHost, status, jobID).Inc()
}

func (mm *MetricsManager) RecordCaptchaSolved(captchaType, solver, jobID string, duration time.Duration) {
	mm.captchaSolved.WithLabelValues(captchaType, solver, jobID).Inc()
	mm.captchaSolveTime.WithLabelValues(captchaType, solver, jobID).Observe(duration.Seconds())
}

func (mm *MetricsManager) RecordCaptchaFailed(captchaType, solver, errorType, jobID string) {
	mm.captchaFailed.WithLabelValues(captchaType, solver, errorType, jobID).Inc()
}

func (mm *MetricsManager) RecordUserAgentRotation(userAgentType, jobID string) {
	mm.userAgentRotation.WithLabelValues(userAgentType, jobID).Inc()
}

// Output metrics
func (mm *MetricsManager) RecordOutputSuccess(format, jobID string, duration time.Duration, size int64, records int) {
	mm.outputSuccess.WithLabelValues(format, jobID).Inc()
	mm.outputTime.WithLabelValues(format, jobID).Observe(duration.Seconds())
	mm.outputSize.WithLabelValues(format, jobID).Observe(float64(size))
	mm.recordsWritten.WithLabelValues(format, jobID).Add(float64(records))
}

func (mm *MetricsManager) RecordOutputError(format, errorType, jobID string) {
	mm.outputErrors.WithLabelValues(format, errorType, jobID).Inc()
}

// System metrics
func (mm *MetricsManager) UpdateMemoryUsage(bytes int64) {
	mm.memoryUsage.Set(float64(bytes))
}

func (mm *MetricsManager) UpdateCPUUsage(percent float64) {
	mm.cpuUsage.Set(percent)
}

func (mm *MetricsManager) UpdateGoroutineCount(count int) {
	mm.goroutineCount.Set(float64(count))
}

// Job metrics
func (mm *MetricsManager) RecordJobStart(jobID, jobType string) {
	mm.jobsTotal.WithLabelValues("started", jobType).Inc()
	mm.jobsActive.Inc()
}

func (mm *MetricsManager) RecordJobComplete(jobID, jobType string, duration time.Duration) {
	mm.jobsTotal.WithLabelValues("completed", jobType).Inc()
	mm.jobDuration.WithLabelValues(jobID, jobType).Observe(duration.Seconds())
	mm.jobsActive.Dec()
}

func (mm *MetricsManager) RecordJobFailed(jobID, jobType string, duration time.Duration) {
	mm.jobsTotal.WithLabelValues("failed", jobType).Inc()
	mm.jobDuration.WithLabelValues(jobID, jobType).Observe(duration.Seconds())
	mm.jobsActive.Dec()
}

func (mm *MetricsManager) UpdateJobsQueued(count int) {
	mm.jobsQueued.Set(float64(count))
}

// Rate limiting metrics
func (mm *MetricsManager) RecordRateLimitHit(host, jobID string, waitDuration time.Duration) {
	mm.rateLimitHits.WithLabelValues(host, jobID).Inc()
	mm.rateLimitWaits.WithLabelValues(host, jobID).Observe(waitDuration.Seconds())
}

// Custom metrics
func (mm *MetricsManager) RegisterCustomCounter(name, help string, labels []string) *prometheus.CounterVec {
	counter := promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: mm.namespace,
			Subsystem: mm.subsystem,
			Name:      name,
			Help:      help,
		},
		labels,
	)

	mm.customMutex.Lock()
	mm.customMetrics[name] = counter
	mm.customMutex.Unlock()

	return counter
}

func (mm *MetricsManager) RegisterCustomGauge(name, help string, labels []string) *prometheus.GaugeVec {
	gauge := promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: mm.namespace,
			Subsystem: mm.subsystem,
			Name:      name,
			Help:      help,
		},
		labels,
	)

	mm.customMutex.Lock()
	mm.customMetrics[name] = gauge
	mm.customMutex.Unlock()

	return gauge
}

func (mm *MetricsManager) RegisterCustomHistogram(name, help string, labels []string, buckets []float64) *prometheus.HistogramVec {
	histogram := promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: mm.namespace,
			Subsystem: mm.subsystem,
			Name:      name,
			Help:      help,
			Buckets:   buckets,
		},
		labels,
	)

	mm.customMutex.Lock()
	mm.customMetrics[name] = histogram
	mm.customMutex.Unlock()

	return histogram
}

// GetCustomMetric retrieves a custom metric by name
func (mm *MetricsManager) GetCustomMetric(name string) (prometheus.Collector, bool) {
	mm.customMutex.RLock()
	defer mm.customMutex.RUnlock()
	metric, exists := mm.customMetrics[name]
	return metric, exists
}

// MetricsHandler returns an HTTP handler for metrics endpoint
func (mm *MetricsManager) MetricsHandler() http.Handler {
	return promhttp.Handler()
}

// StartMetricsServer starts the metrics HTTP server
func (mm *MetricsManager) StartMetricsServer(ctx context.Context, address, path string) error {
	mux := http.NewServeMux()
	mux.Handle(path, mm.MetricsHandler())

	server := &http.Server{
		Addr:    address,
		Handler: mux,
	}

	go func() {
		<-ctx.Done()
		server.Shutdown(context.Background())
	}()

	return server.ListenAndServe()
}

// GetMetrics returns current metric values as a map
// Note: This is a simplified implementation that returns basic metric metadata.
// For full metric values, use the Prometheus /metrics endpoint directly.
func (mm *MetricsManager) GetMetrics() map[string]interface{} {
	metrics := make(map[string]interface{})

	// Get current system metrics (these have actual values)
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	metrics["system"] = map[string]interface{}{
		"memory_alloc_bytes": m.Alloc,
		"memory_sys_bytes":   m.Sys,
		"goroutines_count":   runtime.NumGoroutine(),
		"gc_cycles":          m.NumGC,
	}

	// Metric registry information
	metrics["metric_families"] = map[string]interface{}{
		"requests_total":     "Counter - Total HTTP requests made",
		"jobs_active":        "Gauge - Currently active scraping jobs",
		"memory_usage_bytes": "Gauge - Current memory usage",
		"extraction_success": "Counter - Successful data extractions",
		"captcha_solved":     "Counter - CAPTCHAs successfully solved",
	}

	metrics["note"] = "For current metric values, query the metrics endpoint at the configured address and path (e.g., http://localhost:9090/metrics)"

	return metrics
}
