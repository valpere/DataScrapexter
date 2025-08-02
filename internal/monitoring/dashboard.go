// internal/monitoring/dashboard.go
package monitoring

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"path/filepath"
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
	TotalRequests     int64         `json:"total_requests"`
	SuccessfulPages   int64         `json:"successful_pages"`
	FailedPages       int64         `json:"failed_pages"`
	ActiveJobs        int           `json:"active_jobs"`
	QueuedJobs        int           `json:"queued_jobs"`
	Uptime           time.Duration `json:"uptime"`
	MemoryUsage      float64       `json:"memory_usage_mb"`
	CPUUsage         float64       `json:"cpu_usage_percent"`
}

// ChartData represents data for charts
type ChartData struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Title       string                 `json:"title"`
	Labels      []string               `json:"labels"`
	Datasets    []ChartDataset         `json:"datasets"`
	Options     map[string]interface{} `json:"options"`
	UpdateURL   string                 `json:"update_url,omitempty"`
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
	ID            string                 `json:"id"`
	Name          string                 `json:"name"`
	Status        string                 `json:"status"`
	StartTime     time.Time              `json:"start_time"`
	EndTime       *time.Time             `json:"end_time,omitempty"`
	Duration      time.Duration          `json:"duration"`
	Progress      float64                `json:"progress"`
	PagesScraped  int64                  `json:"pages_scraped"`
	RecordsFound  int64                  `json:"records_found"`
	Errors        int64                  `json:"errors"`
	CurrentURL    string                 `json:"current_url,omitempty"`
	Config        map[string]interface{} `json:"config,omitempty"`
	Metrics       JobMetrics             `json:"metrics"`
}

// JobMetrics detailed metrics for a job
type JobMetrics struct {
	RequestsPerSecond float64           `json:"requests_per_second"`
	SuccessRate       float64           `json:"success_rate"`
	AverageResponseTime time.Duration  `json:"average_response_time"`
	ErrorsByType      map[string]int64  `json:"errors_by_type"`
	ProxyUsage        map[string]int64  `json:"proxy_usage"`
}

// AlertManager manages alerts and notifications
type AlertManager struct {
	alerts   []Alert
	rules    []AlertRule
	mu       sync.RWMutex
	config   AlertConfig
}

// AlertConfig configuration for alerts
type AlertConfig struct {
	EnableEmail    bool          `json:"enable_email"`
	EnableSlack    bool          `json:"enable_slack"`
	EnableWebhook  bool          `json:"enable_webhook"`
	CheckInterval  time.Duration `json:"check_interval"`
	RetentionPeriod time.Duration `json:"retention_period"`
}

// Alert represents an alert
type Alert struct {
	ID          string                 `json:"id"`
	Level       AlertLevel             `json:"level"`
	Title       string                 `json:"title"`
	Message     string                 `json:"message"`
	Timestamp   time.Time              `json:"timestamp"`
	Source      string                 `json:"source"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Acknowledged bool                  `json:"acknowledged"`
	AcknowledgedBy string              `json:"acknowledged_by,omitempty"`
	AcknowledgedAt *time.Time          `json:"acknowledged_at,omitempty"`
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
	Name        string                                    `json:"name"`
	Condition   func(metrics map[string]interface{}) bool `json:"-"`
	Level       AlertLevel                                `json:"level"`
	Message     string                                    `json:"message"`
	Cooldown    time.Duration                             `json:"cooldown"`
	LastTriggered *time.Time                             `json:"last_triggered,omitempty"`
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

// staticHandler serves static files for the dashboard
// PRODUCTION NOTE: This is a development stub. For production deployment:
//   1. Use go:embed to embed static files: 
//      //go:embed static/*
//      var staticFiles embed.FS
//   2. Use http.FS(staticFiles) with http.FileServer
//   3. Or serve static files through a CDN/reverse proxy like nginx
func (d *Dashboard) staticHandler(w http.ResponseWriter, r *http.Request) {
	// Extract and decode the requested file path
	requestedFile := r.URL.Path[len(d.config.Path+"/static/"):]
	
	// URL decode to handle encoded characters
	decodedFile, err := url.QueryUnescape(requestedFile)
	if err != nil {
		http.Error(w, "Invalid file path encoding", http.StatusBadRequest)
		return
	}
	
	// Clean the path and perform robust security checks
	cleanedPath := filepath.Clean(decodedFile)

	// Prevent directory traversal - ensure cleaned path doesn't escape static directory
	staticDir := filepath.Join(d.config.Path, "static")
	absRequestedPath := filepath.Join(staticDir, cleanedPath)
	rel, err := filepath.Rel(staticDir, absRequestedPath)
	if err != nil || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
		http.Error(w, "Invalid file path", http.StatusForbidden)
		return
	}
	
	// Additional validation - only allow specific file types for security
	allowedExtensions := map[string]bool{
		".css": true, ".js": true, ".png": true, ".jpg": true, ".jpeg": true, 
		".gif": true, ".svg": true, ".ico": true, ".woff": true, ".woff2": true,
	}
	
	ext := filepath.Ext(strings.ToLower(cleanedPath))
	if !allowedExtensions[ext] {
		http.Error(w, "File type not allowed", http.StatusForbidden)
		return
	}
	
	// For now, return a helpful message with basic CSS for dashboard functionality
	// In production, this would serve actual static files from an embedded filesystem
	w.Header().Set("Content-Type", "text/css")
	w.Write([]byte(`
/* Basic Dashboard Styles - Production implementation needed */
body { font-family: Arial, sans-serif; margin: 0; padding: 20px; }
.dashboard-container { max-width: 1200px; margin: 0 auto; }
.metric-card { border: 1px solid #ddd; padding: 15px; margin: 10px 0; border-radius: 5px; }
.chart-container { width: 100%; height: 300px; margin: 20px 0; }
.status-healthy { color: #28a745; }
.status-warning { color: #ffc107; }
.status-error { color: #dc3545; }
	`))
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
		Uptime:         time.Since(startTime),
		MemoryUsage:    245.7,
		CPUUsage:       23.4,
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
		ID:    "requests-chart",
		Type:  "line",
		Title: "Requests per Hour",
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
		ID:    "errors-chart",
		Type:  "doughnut",
		Title: "Error Distribution",
		Labels: []string{"Success", "4xx Errors", "5xx Errors", "Network Errors"},
		Datasets: []ChartDataset{
			{
				Data: []float64{85, 8, 4, 3},
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
		ID:    "performance-chart",
		Type:  "bar",
		Title: "Average Response Time (ms)",
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
		ID:    "resources-chart",
		Type:  "line",
		Title: "System Resources",
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
			Name:    "High Error Rate",
			Level:   AlertLevelWarning,
			Message: "Error rate has exceeded 10%",
			Cooldown: 5 * time.Minute,
		},
		{
			Name:    "High Memory Usage",
			Level:   AlertLevelWarning,
			Message: "Memory usage has exceeded 80%",
			Cooldown: 10 * time.Minute,
		},
		{
			Name:    "Job Failure",
			Level:   AlertLevelError,
			Message: "Scraping job has failed",
			Cooldown: 1 * time.Minute,
		},
	}
}