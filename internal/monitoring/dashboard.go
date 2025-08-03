// internal/monitoring/dashboard.go
package monitoring

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

// Dashboard provides a web-based monitoring interface
type Dashboard struct {
	metricsManager *MetricsManager
	healthManager  *HealthManager
	config         DashboardConfig
	templates      *template.Template
	alertManager   *AlertManager
	jobTracker     *JobTracker
	mu             sync.RWMutex
}

// DashboardConfig configuration for the dashboard
type DashboardConfig struct {
	Port            string        `json:"port"`
	Path            string        `json:"path"`
	Title           string        `json:"title"`
	RefreshInterval time.Duration `json:"refresh_interval"`
	Theme           string        `json:"theme"`
	EnableAlerts    bool          `json:"enable_alerts"`
	EnableJobs      bool          `json:"enable_jobs"`
	TimeZone        string        `json:"timezone"`
}

// DashboardData represents data for dashboard rendering
type DashboardData struct {
	Title           string                 `json:"title"`
	Timestamp       time.Time              `json:"timestamp"`
	RefreshInterval int                    `json:"refresh_interval"`
	Health          SystemHealth           `json:"health"`
	Metrics         map[string]interface{} `json:"metrics"`
	Jobs            []JobStatus            `json:"jobs,omitempty"`
	Alerts          []Alert                `json:"alerts,omitempty"`
	Charts          []ChartData            `json:"charts"`
	Summary         DashboardSummary       `json:"summary"`
}

// DashboardSummary provides summary statistics
type DashboardSummary struct {
	TotalRequests   int64         `json:"total_requests"`
	SuccessfulPages int64         `json:"successful_pages"`
	FailedPages     int64         `json:"failed_pages"`
	ActiveJobs      int           `json:"active_jobs"`
	QueuedJobs      int           `json:"queued_jobs"`
	Uptime          time.Duration `json:"uptime"`
	MemoryUsage     float64       `json:"memory_usage_mb"`
	CPUUsage        float64       `json:"cpu_usage_percent"`
}

// ChartData represents data for charts
type ChartData struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Title     string                 `json:"title"`
	Labels    []string               `json:"labels"`
	Datasets  []ChartDataset         `json:"datasets"`
	Options   map[string]interface{} `json:"options"`
	UpdateURL string                 `json:"update_url,omitempty"`
}

// ChartDataset represents a dataset for charts
type ChartDataset struct {
	Label           string    `json:"label"`
	Data            []float64 `json:"data"`
	BackgroundColor string    `json:"backgroundColor,omitempty"`
	BorderColor     string    `json:"borderColor,omitempty"`
	Fill            bool      `json:"fill"`
}

// JobTracker tracks scraping jobs
type JobTracker struct {
	jobs   map[string]*JobStatus
	mu     sync.RWMutex
	config JobTrackerConfig
}

// JobTrackerConfig configuration for job tracking
type JobTrackerConfig struct {
	MaxJobs         int           `json:"max_jobs"`
	RetentionPeriod time.Duration `json:"retention_period"`
	UpdateInterval  time.Duration `json:"update_interval"`
}

// JobStatus represents the status of a scraping job
type JobStatus struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Status       string                 `json:"status"`
	StartTime    time.Time              `json:"start_time"`
	EndTime      *time.Time             `json:"end_time,omitempty"`
	Duration     time.Duration          `json:"duration"`
	Progress     float64                `json:"progress"`
	PagesScraped int64                  `json:"pages_scraped"`
	RecordsFound int64                  `json:"records_found"`
	Errors       int64                  `json:"errors"`
	CurrentURL   string                 `json:"current_url,omitempty"`
	Config       map[string]interface{} `json:"config,omitempty"`
	Metrics      JobMetrics             `json:"metrics"`
}

// JobMetrics detailed metrics for a job
type JobMetrics struct {
	RequestsPerSecond   float64          `json:"requests_per_second"`
	SuccessRate         float64          `json:"success_rate"`
	AverageResponseTime time.Duration    `json:"average_response_time"`
	ErrorsByType        map[string]int64 `json:"errors_by_type"`
	ProxyUsage          map[string]int64 `json:"proxy_usage"`
}

// AlertManager manages alerts and notifications
type AlertManager struct {
	alerts []Alert
	rules  []AlertRule
	mu     sync.RWMutex
	config AlertConfig
}

// AlertConfig configuration for alerts
type AlertConfig struct {
	EnableEmail     bool          `json:"enable_email"`
	EnableSlack     bool          `json:"enable_slack"`
	EnableWebhook   bool          `json:"enable_webhook"`
	CheckInterval   time.Duration `json:"check_interval"`
	RetentionPeriod time.Duration `json:"retention_period"`
}

// Alert represents an alert
type Alert struct {
	ID             string                 `json:"id"`
	Level          AlertLevel             `json:"level"`
	Title          string                 `json:"title"`
	Message        string                 `json:"message"`
	Timestamp      time.Time              `json:"timestamp"`
	Source         string                 `json:"source"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	Acknowledged   bool                   `json:"acknowledged"`
	AcknowledgedBy string                 `json:"acknowledged_by,omitempty"`
	AcknowledgedAt *time.Time             `json:"acknowledged_at,omitempty"`
}

// AlertLevel represents alert severity levels
type AlertLevel string

const (
	AlertLevelInfo     AlertLevel = "info"
	AlertLevelWarning  AlertLevel = "warning"
	AlertLevelError    AlertLevel = "error"
	AlertLevelCritical AlertLevel = "critical"
)

// AlertRule defines conditions for triggering alerts
type AlertRule struct {
	Name          string                                    `json:"name"`
	Condition     func(metrics map[string]interface{}) bool `json:"-"`
	Level         AlertLevel                                `json:"level"`
	Message       string                                    `json:"message"`
	Cooldown      time.Duration                             `json:"cooldown"`
	LastTriggered *time.Time                                `json:"last_triggered,omitempty"`
}

// NewDashboard creates a new monitoring dashboard
func NewDashboard(metrics *MetricsManager, health *HealthManager, config DashboardConfig) *Dashboard {
	if config.Port == "" {
		config.Port = ":8080"
	}
	if config.Path == "" {
		config.Path = "/dashboard"
	}
	if config.Title == "" {
		config.Title = "DataScrapexter Monitoring"
	}
	if config.RefreshInterval == 0 {
		config.RefreshInterval = 30 * time.Second
	}
	if config.Theme == "" {
		config.Theme = "dark"
	}
	if config.TimeZone == "" {
		config.TimeZone = "UTC"
	}

	dashboard := &Dashboard{
		metricsManager: metrics,
		healthManager:  health,
		config:         config,
		jobTracker:     NewJobTracker(JobTrackerConfig{}),
		alertManager:   NewAlertManager(AlertConfig{}),
	}

	dashboard.initializeTemplates()
	dashboard.setupAlertRules()

	return dashboard
}

// NewJobTracker creates a new job tracker
func NewJobTracker(config JobTrackerConfig) *JobTracker {
	if config.MaxJobs == 0 {
		config.MaxJobs = 1000
	}
	if config.RetentionPeriod == 0 {
		config.RetentionPeriod = 24 * time.Hour
	}
	if config.UpdateInterval == 0 {
		config.UpdateInterval = 5 * time.Second
	}

	return &JobTracker{
		jobs:   make(map[string]*JobStatus),
		config: config,
	}
}

// NewAlertManager creates a new alert manager
func NewAlertManager(config AlertConfig) *AlertManager {
	if config.CheckInterval == 0 {
		config.CheckInterval = 1 * time.Minute
	}
	if config.RetentionPeriod == 0 {
		config.RetentionPeriod = 7 * 24 * time.Hour
	}

	return &AlertManager{
		alerts: make([]Alert, 0),
		rules:  make([]AlertRule, 0),
		config: config,
	}
}

// Start starts the dashboard server
func (d *Dashboard) Start(ctx context.Context) error {
	mux := http.NewServeMux()

	// Dashboard routes
	mux.HandleFunc(d.config.Path, d.dashboardHandler)
	mux.HandleFunc(d.config.Path+"/api/data", d.apiDataHandler)
	mux.HandleFunc(d.config.Path+"/api/jobs", d.apiJobsHandler)
	mux.HandleFunc(d.config.Path+"/api/alerts", d.apiAlertsHandler)
	mux.HandleFunc(d.config.Path+"/api/charts/", d.apiChartsHandler)
	mux.HandleFunc(d.config.Path+"/static/", d.staticHandler)

	// Health endpoints
	mux.HandleFunc("/health", d.healthManager.HealthHandler())
	mux.HandleFunc("/ready", d.healthManager.ReadinessHandler())
	mux.HandleFunc("/live", d.healthManager.LivenessHandler())

	// Metrics endpoint
	mux.Handle("/metrics", d.metricsManager.MetricsHandler())

	server := &http.Server{
		Addr:    d.config.Port,
		Handler: mux,
	}

	go func() {
		<-ctx.Done()
		server.Shutdown(context.Background())
	}()

	return server.ListenAndServe()
}

// dashboardHandler serves the main dashboard page
func (d *Dashboard) dashboardHandler(w http.ResponseWriter, r *http.Request) {
	data := d.getDashboardData()

	w.Header().Set("Content-Type", "text/html")
	err := d.templates.ExecuteTemplate(w, "dashboard.html", data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// apiDataHandler serves dashboard data as JSON
func (d *Dashboard) apiDataHandler(w http.ResponseWriter, r *http.Request) {
	data := d.getDashboardData()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

// apiJobsHandler serves job data as JSON
func (d *Dashboard) apiJobsHandler(w http.ResponseWriter, r *http.Request) {
	jobs := d.jobTracker.GetAllJobs()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(jobs)
}

// apiAlertsHandler serves alert data as JSON
func (d *Dashboard) apiAlertsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		// Acknowledge alert
		var req struct {
			AlertID string `json:"alert_id"`
			User    string `json:"user"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
			d.alertManager.AcknowledgeAlert(req.AlertID, req.User)
		}
	}

	alerts := d.alertManager.GetActiveAlerts()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(alerts)
}

// apiChartsHandler serves chart data as JSON
func (d *Dashboard) apiChartsHandler(w http.ResponseWriter, r *http.Request) {
	chartType := r.URL.Path[len(d.config.Path+"/api/charts/"):]

	var chartData ChartData

	switch chartType {
	case "requests":
		chartData = d.getRequestsChart()
	case "errors":
		chartData = d.getErrorsChart()
	case "performance":
		chartData = d.getPerformanceChart()
	case "resources":
		chartData = d.getResourcesChart()
	default:
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(chartData)
}

// staticHandler serves static files for the dashboard.
//
// SECURITY: This implementation uses a strict allowlist for static files.
// Only files explicitly listed in serveStaticFileSafely() can be served.
// This is intentional to prevent path traversal, unauthorized file access,
// and other attacks. Do NOT relax this allowlist without careful validation.
//
// SAFE EXTENSION: To add more static files, add them to the allowlist in
// serveStaticFileSafely(), ensuring the file name is hardcoded and the content
// is trusted. Never use user input to construct file paths.
//
// PRODUCTION NOTE: For production deployments, consider using go:embed for
// static assets, or serve files via a CDN or reverse proxy (e.g., nginx).
// This handler is intentionally restrictive and not suitable for serving
// arbitrary files.
func (d *Dashboard) staticHandler(w http.ResponseWriter, r *http.Request) {
	// SECURITY: Implement secure static file serving with generic error messages
	if err := d.serveStaticFileSafely(w, r); err != nil {
		// Structured logging for security monitoring (internal diagnostics)
		d.logSecurityEvent("static_file_access_denied", map[string]interface{}{
			"remote_addr":     r.RemoteAddr,
			"user_agent":      r.Header.Get("User-Agent"),
			"requested_file":  d.sanitizeRequestedPath(r.URL.Path),
			"method":          r.Method,
			"error_type":      "file_access_denied",
			"timestamp":       time.Now().UTC(),
		})

		// Generic error message that doesn't expose security mechanism details
		http.Error(w, "Resource not found", http.StatusNotFound)
		return
	}
}

// serveStaticFileSafely implements secure static file serving with strict validation
func (d *Dashboard) serveStaticFileSafely(w http.ResponseWriter, r *http.Request) error {
	// Extract the requested file path
	requestedPath := r.URL.Path[len(d.config.Path+"/static/"):]

	// SECURITY: Strict allowlist - only allow predefined safe files
	safeFiles := map[string]struct {
		contentType string
		content     string
	}{
		"dashboard.css": {
			contentType: "text/css",
			content:     d.getDefaultCSS(),
		},
		"dashboard.js": {
			contentType: "application/javascript",
			content:     d.getDefaultJS(),
		},
	}

	// Check if requested file is in our safe allowlist
	file, exists := safeFiles[requestedPath]
	if !exists {
		return fmt.Errorf("resource not available")
	}

	// Serve the safe file with security headers
	w.Header().Set("Content-Type", file.contentType)
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", "default-src 'self'")
	w.Write([]byte(file.content))
	return nil
}

// sanitizeRequestedPath sanitizes the requested path for secure logging
// This prevents information disclosure while maintaining useful audit trails
func (d *Dashboard) sanitizeRequestedPath(path string) string {
	// Extract just the requested filename from the static path
	staticPrefix := d.config.Path + "/static/"
	if strings.HasPrefix(path, staticPrefix) {
		requestedFile := path[len(staticPrefix):]
		
		// Only log if it's a reasonable file request
		if len(requestedFile) > 0 && len(requestedFile) < 100 {
			// Allow only alphanumeric, dots, dashes, underscores
			sanitized := strings.Map(func(r rune) rune {
				if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
				   (r >= '0' && r <= '9') || r == '.' || r == '-' || r == '_' {
					return r
				}
				return '_'
			}, requestedFile)
			
			// Truncate if too long
			if len(sanitized) > 50 {
				sanitized = sanitized[:50] + "..."
			}
			return sanitized
		}
	}
	
	// For any other case, return generic identifier
	return "unknown_file"
}

// logSecurityEvent logs security-related events for monitoring and analysis
// This provides detailed internal logging while maintaining generic user-facing errors
func (d *Dashboard) logSecurityEvent(eventType string, details map[string]interface{}) {
	// Add common fields to the security event
	securityEvent := map[string]interface{}{
		"event_type": eventType,
		"component":  "dashboard",
		"severity":   "security",
		"service":    "datascrapexter",
	}

	// Merge in the specific event details
	for k, v := range details {
		securityEvent[k] = v
	}

	// Log as structured JSON for security monitoring tools
	// In production, this should be sent to a security monitoring system
	// such as SIEM, security analytics platform, or dedicated log aggregator

	// Use structured logging for security events
	// In production, configure slog to send to security monitoring systems (SIEM, etc.)
	slog.Error("SECURITY EVENT",
		slog.String("event_type", eventType),
		slog.Any("details", details),
		slog.String("component", "dashboard"),
		slog.String("severity", "security"),
		slog.String("service", "datascrapexter"),
		slog.Time("timestamp", time.Now().UTC()),
	)

	// Future: Integrate with security monitoring systems
	// Example integrations:
	// - Send to SIEM (Splunk, ELK, etc.)
	// - Trigger security alerts
	// - Emit metrics for dashboards
}

// getDashboardData collects all data for the dashboard
func (d *Dashboard) getDashboardData() DashboardData {
	health := d.healthManager.GetHealth()
	metrics := d.metricsManager.GetMetrics()
	jobs := d.jobTracker.GetActiveJobs()
	alerts := d.alertManager.GetActiveAlerts()

	return DashboardData{
		Title:           d.config.Title,
		Timestamp:       time.Now(),
		RefreshInterval: int(d.config.RefreshInterval.Seconds()),
		Health:          health,
		Metrics:         metrics,
		Jobs:            jobs,
		Alerts:          alerts,
		Charts:          d.getCharts(),
		Summary:         d.getSummary(),
	}
}

// getSummary generates dashboard summary statistics
func (d *Dashboard) getSummary() DashboardSummary {
	// This would collect real metrics from the metrics manager
	return DashboardSummary{
		TotalRequests:   12500,
		SuccessfulPages: 11800,
		FailedPages:     700,
		ActiveJobs:      3,
		QueuedJobs:      7,
		Uptime:          time.Since(startTime),
		MemoryUsage:     245.7,
		CPUUsage:        23.4,
	}
}

// isValidStaticFilePath validates file paths against allowed patterns
// This provides positive allowlist validation beyond extension checking
func (d *Dashboard) isValidStaticFilePath(path string) bool {
	// Only allow simple alphanumeric paths with safe separators
	// This prevents various encoding and traversal attacks
	for _, char := range path {
		if !((char >= 'a' && char <= 'z') ||
			(char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') ||
			char == '/' || char == '-' || char == '_' || char == '.') {
			return false
		}
	}

	// Additional pattern validation - prevent suspicious patterns
	suspiciousPatterns := []string{
		"../", "..\\",
		"/..", "\\..",
		"//", "\\\\",
		".", "..",
	}

	for _, pattern := range suspiciousPatterns {
		if strings.Contains(path, pattern) {
			return false
		}
	}

	return true
}

// getDefaultCSS returns secure default CSS content
func (d *Dashboard) getDefaultCSS() string {
	return `
/* DataScrapexter Dashboard Styles */
* { box-sizing: border-box; }
body { 
	font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
	margin: 0; padding: 20px; background: #f5f5f5; line-height: 1.6;
}
.dashboard-container { max-width: 1200px; margin: 0 auto; }
.header { background: #fff; padding: 20px; border-radius: 8px; margin-bottom: 20px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
.metric-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(250px, 1fr)); gap: 20px; margin-bottom: 20px; }
.metric-card { 
	background: #fff; padding: 20px; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1);
	border-left: 4px solid #007bff;
}
.metric-value { font-size: 2em; font-weight: bold; margin-bottom: 8px; }
.metric-label { color: #666; font-size: 0.9em; }
.chart-container { background: #fff; padding: 20px; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); height: 400px; }
.status-healthy { color: #28a745; }
.status-warning { color: #ffc107; }
.status-error { color: #dc3545; }
.status-degraded { color: #fd7e14; }
.jobs-table { width: 100%; border-collapse: collapse; background: #fff; }
.jobs-table th, .jobs-table td { padding: 12px; text-align: left; border-bottom: 1px solid #ddd; }
.jobs-table th { background: #f8f9fa; font-weight: 600; }
.progress-bar { width: 100%; height: 20px; background: #e9ecef; border-radius: 10px; overflow: hidden; }
.progress-fill { height: 100%; background: #28a745; transition: width 0.3s ease; }
.alert { padding: 12px 16px; border-radius: 4px; margin-bottom: 16px; }
.alert-error { background: #f8d7da; color: #721c24; border: 1px solid #f5c6cb; }
.alert-warning { background: #fff3cd; color: #856404; border: 1px solid #ffeaa7; }
.alert-info { background: #d1ecf1; color: #0c5460; border: 1px solid #bee5eb; }
@media (max-width: 768px) {
	.metric-grid { grid-template-columns: 1fr; }
	.dashboard-container { padding: 10px; }
}
`
}

// getDefaultJS returns secure default JavaScript content
func (d *Dashboard) getDefaultJS() string {
	return `
// DataScrapexter Dashboard JavaScript
(function() {
	'use strict';
	
	// Auto-refresh functionality
	let refreshInterval = 30000; // 30 seconds
	let refreshTimer;
	
	function refreshDashboard() {
		fetch(window.location.pathname + '/api/data')
			.then(response => response.json())
			.then(data => updateDashboard(data))
			.catch(error => console.error('Failed to refresh dashboard:', error));
	}
	
	function updateDashboard(data) {
		// Update timestamp
		const timestampEl = document.getElementById('last-updated');
		if (timestampEl) {
			timestampEl.textContent = new Date(data.timestamp).toLocaleString();
		}
		
		// Update metrics if elements exist
		updateMetric('total-requests', data.summary?.total_requests);
		updateMetric('success-rate', data.summary?.successful_pages + '%');
		updateMetric('active-jobs', data.summary?.active_jobs);
		updateMetric('memory-usage', data.summary?.memory_usage_mb + ' MB');
	}
	
	function updateMetric(id, value) {
		const el = document.getElementById(id);
		if (el && value !== undefined) {
			el.textContent = value;
		}
	}
	
	// Initialize auto-refresh
	function startAutoRefresh() {
		refreshTimer = setInterval(refreshDashboard, refreshInterval);
	}
	
	function stopAutoRefresh() {
		if (refreshTimer) {
			clearInterval(refreshTimer);
		}
	}
	
	// Start when page loads
	document.addEventListener('DOMContentLoaded', function() {
		startAutoRefresh();
		
		// Stop refresh when page is hidden to save resources
		document.addEventListener('visibilitychange', function() {
			if (document.hidden) {
				stopAutoRefresh();
			} else {
				startAutoRefresh();
			}
		});
	});
})();
`
}

// getAllowedExtensions returns the allowed file extensions for static files
// This can be made configurable based on deployment requirements
func (d *Dashboard) getAllowedExtensions() map[string]bool {
	// Default safe extensions - customize based on your needs
	// SECURITY: Only include extensions that are safe to serve
	return map[string]bool{
		".css":   true, // Stylesheets
		".js":    true, // JavaScript (ensure Content-Security-Policy is set)
		".png":   true, // Images
		".jpg":   true,
		".jpeg":  true,
		".gif":   true,
		".svg":   true, // SVG (consider sanitization)
		".ico":   true, // Favicons
		".woff":  true, // Fonts
		".woff2": true,
		".ttf":   true,
		".eot":   true,
		// Add more as needed, but be security-conscious
		// ".pdf": false, // Example: PDFs might need special handling
		// ".zip": false, // Example: Archives should be restricted
	}
}

// getCharts generates chart data for the dashboard
func (d *Dashboard) getCharts() []ChartData {
	return []ChartData{
		d.getRequestsChart(),
		d.getErrorsChart(),
		d.getPerformanceChart(),
		d.getResourcesChart(),
	}
}

// getRequestsChart generates requests over time chart
func (d *Dashboard) getRequestsChart() ChartData {
	// Generate mock data - in practice, this would query actual metrics
	labels := []string{}
	data := []float64{}

	now := time.Now()
	for i := 23; i >= 0; i-- {
		labels = append(labels, now.Add(-time.Duration(i)*time.Hour).Format("15:04"))
		data = append(data, float64(100+i*5))
	}

	return ChartData{
		ID:     "requests-chart",
		Type:   "line",
		Title:  "Requests per Hour",
		Labels: labels,
		Datasets: []ChartDataset{
			{
				Label:       "Requests",
				Data:        data,
				BorderColor: "#4CAF50",
				Fill:        false,
			},
		},
		UpdateURL: d.config.Path + "/api/charts/requests",
	}
}

// getErrorsChart generates error rate chart
func (d *Dashboard) getErrorsChart() ChartData {
	return ChartData{
		ID:     "errors-chart",
		Type:   "doughnut",
		Title:  "Error Distribution",
		Labels: []string{"Success", "4xx Errors", "5xx Errors", "Network Errors"},
		Datasets: []ChartDataset{
			{
				Data:            []float64{85, 8, 4, 3},
				BackgroundColor: "#4CAF50,#FF9800,#F44336,#9C27B0",
			},
		},
	}
}

// getPerformanceChart generates performance metrics chart
func (d *Dashboard) getPerformanceChart() ChartData {
	// Generate response time data
	labels := []string{}
	data := []float64{}

	for i := 0; i < 24; i++ {
		labels = append(labels, fmt.Sprintf("%02d:00", i))
		data = append(data, 500+float64(i*10))
	}

	return ChartData{
		ID:     "performance-chart",
		Type:   "bar",
		Title:  "Average Response Time (ms)",
		Labels: labels,
		Datasets: []ChartDataset{
			{
				Label:           "Response Time",
				Data:            data,
				BackgroundColor: "#2196F3",
			},
		},
	}
}

// getResourcesChart generates system resources chart
func (d *Dashboard) getResourcesChart() ChartData {
	return ChartData{
		ID:     "resources-chart",
		Type:   "line",
		Title:  "System Resources",
		Labels: []string{"00:00", "04:00", "08:00", "12:00", "16:00", "20:00"},
		Datasets: []ChartDataset{
			{
				Label:       "CPU %",
				Data:        []float64{15, 25, 45, 60, 40, 30},
				BorderColor: "#FF5722",
				Fill:        false,
			},
			{
				Label:       "Memory %",
				Data:        []float64{40, 42, 48, 55, 50, 45},
				BorderColor: "#4CAF50",
				Fill:        false,
			},
		},
	}
}

// Job tracking methods
func (jt *JobTracker) StartJob(job *JobStatus) {
	jt.mu.Lock()
	defer jt.mu.Unlock()

	job.StartTime = time.Now()
	job.Status = "running"
	jt.jobs[job.ID] = job

	// Clean up old jobs if necessary
	if len(jt.jobs) > jt.config.MaxJobs {
		jt.cleanupOldJobs()
	}
}

func (jt *JobTracker) UpdateJob(jobID string, updates map[string]interface{}) {
	jt.mu.Lock()
	defer jt.mu.Unlock()

	if job, exists := jt.jobs[jobID]; exists {
		// Update job fields based on the updates map
		if progress, ok := updates["progress"].(float64); ok {
			job.Progress = progress
		}
		if pages, ok := updates["pages_scraped"].(int64); ok {
			job.PagesScraped = pages
		}
		if records, ok := updates["records_found"].(int64); ok {
			job.RecordsFound = records
		}
		if errors, ok := updates["errors"].(int64); ok {
			job.Errors = errors
		}
		if url, ok := updates["current_url"].(string); ok {
			job.CurrentURL = url
		}

		job.Duration = time.Since(job.StartTime)
	}
}

func (jt *JobTracker) CompleteJob(jobID string, success bool) {
	jt.mu.Lock()
	defer jt.mu.Unlock()

	if job, exists := jt.jobs[jobID]; exists {
		now := time.Now()
		job.EndTime = &now
		job.Duration = now.Sub(job.StartTime)

		if success {
			job.Status = "completed"
		} else {
			job.Status = "failed"
		}

		job.Progress = 100.0
	}
}

func (jt *JobTracker) GetAllJobs() []JobStatus {
	jt.mu.RLock()
	defer jt.mu.RUnlock()

	jobs := make([]JobStatus, 0, len(jt.jobs))
	for _, job := range jt.jobs {
		jobs = append(jobs, *job)
	}

	// Sort by start time (newest first)
	sort.Slice(jobs, func(i, j int) bool {
		return jobs[i].StartTime.After(jobs[j].StartTime)
	})

	return jobs
}

func (jt *JobTracker) GetActiveJobs() []JobStatus {
	all := jt.GetAllJobs()
	active := make([]JobStatus, 0)

	for _, job := range all {
		if job.Status == "running" || job.Status == "queued" {
			active = append(active, job)
		}
	}

	return active
}

func (jt *JobTracker) cleanupOldJobs() {
	cutoff := time.Now().Add(-jt.config.RetentionPeriod)

	for id, job := range jt.jobs {
		if job.StartTime.Before(cutoff) && job.Status != "running" {
			delete(jt.jobs, id)
		}
	}
}

// Alert management methods
func (am *AlertManager) AddAlert(alert Alert) {
	am.mu.Lock()
	defer am.mu.Unlock()

	alert.Timestamp = time.Now()
	if alert.ID == "" {
		alert.ID = fmt.Sprintf("alert_%d", time.Now().UnixNano())
	}

	am.alerts = append(am.alerts, alert)

	// Clean up old alerts
	am.cleanupOldAlerts()
}

func (am *AlertManager) AcknowledgeAlert(alertID, user string) {
	am.mu.Lock()
	defer am.mu.Unlock()

	for i := range am.alerts {
		if am.alerts[i].ID == alertID {
			now := time.Now()
			am.alerts[i].Acknowledged = true
			am.alerts[i].AcknowledgedBy = user
			am.alerts[i].AcknowledgedAt = &now
			break
		}
	}
}

func (am *AlertManager) GetActiveAlerts() []Alert {
	am.mu.RLock()
	defer am.mu.RUnlock()

	active := make([]Alert, 0)
	for _, alert := range am.alerts {
		if !alert.Acknowledged {
			active = append(active, alert)
		}
	}

	return active
}

func (am *AlertManager) cleanupOldAlerts() {
	cutoff := time.Now().Add(-am.config.RetentionPeriod)

	filtered := make([]Alert, 0)
	for _, alert := range am.alerts {
		if alert.Timestamp.After(cutoff) || !alert.Acknowledged {
			filtered = append(filtered, alert)
		}
	}

	am.alerts = filtered
}

// initializeTemplates sets up HTML templates
func (d *Dashboard) initializeTemplates() {
	// This would load actual template files in practice
	d.templates = template.New("dashboard")
}

// setupAlertRules configures default alert rules
func (d *Dashboard) setupAlertRules() {
	// Add default alert rules
	d.alertManager.rules = []AlertRule{
		{
			Name:     "High Error Rate",
			Level:    AlertLevelWarning,
			Message:  "Error rate has exceeded 10%",
			Cooldown: 5 * time.Minute,
		},
		{
			Name:     "High Memory Usage",
			Level:    AlertLevelWarning,
			Message:  "Memory usage has exceeded 80%",
			Cooldown: 10 * time.Minute,
		},
		{
			Name:     "Job Failure",
			Level:    AlertLevelError,
			Message:  "Scraping job has failed",
			Cooldown: 1 * time.Minute,
		},
	}
}
