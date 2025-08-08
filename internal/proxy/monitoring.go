// internal/proxy/monitoring.go - Advanced proxy monitoring and analytics
package proxy

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/valpere/DataScrapexter/internal/utils"
)

var monitoringLogger = utils.NewComponentLogger("proxy-monitoring")

// MonitoringConfig defines monitoring configuration
type MonitoringConfig struct {
	Enabled              bool             `yaml:"enabled" json:"enabled"`
	MetricsPort          int              `yaml:"metrics_port" json:"metrics_port"`
	DetailedMetrics      bool             `yaml:"detailed_metrics" json:"detailed_metrics"`
	HistoryRetention     time.Duration    `yaml:"history_retention" json:"history_retention"`
	AlertingEnabled      bool             `yaml:"alerting_enabled" json:"alerting_enabled"`
	AlertThresholds      *AlertThresholds `yaml:"alert_thresholds,omitempty" json:"alert_thresholds,omitempty"`
	BudgetConfig         *BudgetConfig    `yaml:"budget_config,omitempty" json:"budget_config,omitempty"`
	RealTimeUpdates      bool             `yaml:"realtime_updates" json:"realtime_updates"`
	ExportPrometheus     bool             `yaml:"export_prometheus" json:"export_prometheus"`
	ExportInfluxDB       bool             `yaml:"export_influxdb" json:"export_influxdb"`
	ExportInterval       time.Duration    `yaml:"export_interval" json:"export_interval"`
	HealthCheckEndpoints []string         `yaml:"health_check_endpoints,omitempty" json:"health_check_endpoints,omitempty"`
}

// BudgetConfig defines budget monitoring configuration
type BudgetConfig struct {
	// DailyBudget is the maximum amount to spend per day
	DailyBudget float64 `yaml:"daily_budget" json:"daily_budget"`
	
	// HourlyBudget is the maximum amount to spend per hour
	HourlyBudget float64 `yaml:"hourly_budget" json:"hourly_budget"`
	
	// MonthlyBudget is the maximum amount to spend per month
	MonthlyBudget float64 `yaml:"monthly_budget" json:"monthly_budget"`
	
	// Currency specifies the currency for budget amounts (e.g., "USD", "EUR")
	Currency string `yaml:"currency" json:"currency"`
	
	// AlertThresholds for budget usage (percentages)
	WarnThreshold  float64 `yaml:"warn_threshold" json:"warn_threshold"`   // Default: 80%
	CritThreshold  float64 `yaml:"crit_threshold" json:"crit_threshold"`   // Default: 90%
	
	// BudgetPeriod specifies the budget tracking period ("hourly", "daily", "monthly")
	BudgetPeriod string `yaml:"budget_period" json:"budget_period"`
}

// AlertThresholds defines alerting thresholds
type AlertThresholds struct {
	MinSuccessRate     float64       `yaml:"min_success_rate" json:"min_success_rate"`
	MaxLatency         time.Duration `yaml:"max_latency" json:"max_latency"`
	MaxErrorRate       float64       `yaml:"max_error_rate" json:"max_error_rate"`
	MinHealthyProxies  int           `yaml:"min_healthy_proxies" json:"min_healthy_proxies"`
	MaxFailureCount    int           `yaml:"max_failure_count" json:"max_failure_count"`
	BudgetThreshold    float64       `yaml:"budget_threshold" json:"budget_threshold"` // Percentage of budget used
}

// ProxyMonitor handles monitoring and analytics for proxies
type ProxyMonitor struct {
	config          *MonitoringConfig
	manager         *AdvancedProxyManager
	metrics         *MetricsCollector
	alerts          *AlertManager
	history         *HistoryManager
	server          *http.Server
	stopChan        chan struct{}
	mu              sync.RWMutex
}

// MetricsCollector collects and aggregates proxy metrics
type MetricsCollector struct {
	current    *CurrentMetrics
	historical map[string]*HistoricalMetrics
	mu         sync.RWMutex
}

// CurrentMetrics represents current real-time metrics
type CurrentMetrics struct {
	Timestamp            time.Time                        `json:"timestamp"`
	TotalRequests        int64                            `json:"total_requests"`
	SuccessfulRequests   int64                            `json:"successful_requests"`
	FailedRequests       int64                            `json:"failed_requests"`
	AverageLatency       time.Duration                    `json:"average_latency"`
	TotalProxies         int                              `json:"total_proxies"`
	HealthyProxies       int                              `json:"healthy_proxies"`
	UnhealthyProxies     int                              `json:"unhealthy_proxies"`
	TotalCost            float64                          `json:"total_cost"`
	CostPerRequest       float64                          `json:"cost_per_request"`
	ProxyMetrics         map[string]*ProxyMetricsSummary  `json:"proxy_metrics"`
	GeographicDistribution map[string]int                `json:"geographic_distribution"`
	PerformanceTiers     map[string]int                   `json:"performance_tiers"`
	ErrorBreakdown       map[string]int                   `json:"error_breakdown"`
}

// HistoricalMetrics represents historical time-series metrics
type HistoricalMetrics struct {
	ProxyName    string                `json:"proxy_name"`
	DataPoints   []MetricDataPoint     `json:"data_points"`
	Aggregations *MetricAggregations   `json:"aggregations"`
	LastUpdated  time.Time             `json:"last_updated"`
}

// MetricDataPoint represents a single metric measurement
type MetricDataPoint struct {
	Timestamp    time.Time     `json:"timestamp"`
	Latency      time.Duration `json:"latency"`
	Success      bool          `json:"success"`
	Cost         float64       `json:"cost"`
	DataQuality  float64       `json:"data_quality"`
	ErrorType    string        `json:"error_type,omitempty"`
	RequestSize  int64         `json:"request_size,omitempty"`
	ResponseSize int64         `json:"response_size,omitempty"`
	TargetURL    string        `json:"target_url"`
	UserAgent    string        `json:"user_agent,omitempty"`
}

// MetricAggregations represents aggregated metrics over time periods
type MetricAggregations struct {
	Hourly  map[string]*AggregatedMetrics `json:"hourly"`
	Daily   map[string]*AggregatedMetrics `json:"daily"`
	Weekly  map[string]*AggregatedMetrics `json:"weekly"`
	Monthly map[string]*AggregatedMetrics `json:"monthly"`
}

// AggregatedMetrics represents metrics aggregated over a time period
type AggregatedMetrics struct {
	Period              string        `json:"period"`
	TotalRequests       int64         `json:"total_requests"`
	SuccessfulRequests  int64         `json:"successful_requests"`
	FailedRequests      int64         `json:"failed_requests"`
	SuccessRate         float64       `json:"success_rate"`
	AverageLatency      time.Duration `json:"average_latency"`
	MinLatency          time.Duration `json:"min_latency"`
	MaxLatency          time.Duration `json:"max_latency"`
	P50Latency          time.Duration `json:"p50_latency"`
	P95Latency          time.Duration `json:"p95_latency"`
	P99Latency          time.Duration `json:"p99_latency"`
	TotalCost           float64       `json:"total_cost"`
	AverageDataQuality  float64       `json:"average_data_quality"`
	ErrorBreakdown      map[string]int `json:"error_breakdown"`
}

// ProxyMetricsSummary represents summarized metrics for a single proxy
type ProxyMetricsSummary struct {
	ProxyName         string        `json:"proxy_name"`
	Status            string        `json:"status"` // healthy, unhealthy, unknown
	LastSeen          time.Time     `json:"last_seen"`
	RequestsLast1h    int64         `json:"requests_last_1h"`
	RequestsLast24h   int64         `json:"requests_last_24h"`
	SuccessRate1h     float64       `json:"success_rate_1h"`
	SuccessRate24h    float64       `json:"success_rate_24h"`
	AverageLatency1h  time.Duration `json:"average_latency_1h"`
	AverageLatency24h time.Duration `json:"average_latency_24h"`
	CurrentLoad       int           `json:"current_load"` // Current concurrent connections
	MaxLoad           int           `json:"max_load"`     // Maximum concurrent connections
	CostLast1h        float64       `json:"cost_last_1h"`
	CostLast24h       float64       `json:"cost_last_24h"`
	DataQuality1h     float64       `json:"data_quality_1h"`
	DataQuality24h    float64       `json:"data_quality_24h"`
	Geographic        *GeographicLocation `json:"geographic,omitempty"`
	Tags              []string      `json:"tags,omitempty"`
}

// AlertManager handles alerting based on thresholds
type AlertManager struct {
	config     *AlertThresholds
	alerts     []Alert
	handlers   []AlertHandler
	mu         sync.RWMutex
}

// Alert represents an active alert
type Alert struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Severity    string                 `json:"severity"` // low, medium, high, critical
	Message     string                 `json:"message"`
	ProxyName   string                 `json:"proxy_name,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
	Resolved    bool                   `json:"resolved"`
	ResolvedAt  *time.Time             `json:"resolved_at,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// AlertHandler defines interface for alert handlers
type AlertHandler interface {
	HandleAlert(alert Alert) error
	GetHandlerType() string
}

// HistoryManager manages historical data retention and cleanup
type HistoryManager struct {
	retention   time.Duration
	storage     map[string][]MetricDataPoint
	storageMu   sync.RWMutex
	cleanupTicker *time.Ticker
	stopChan    chan struct{}
}

// NewProxyMonitor creates a new proxy monitor
func NewProxyMonitor(config *MonitoringConfig, manager *AdvancedProxyManager) *ProxyMonitor {
	if config == nil {
		config = &MonitoringConfig{
			Enabled:         false,
			MetricsPort:     9090,
			DetailedMetrics: true,
			HistoryRetention: 24 * time.Hour,
			AlertingEnabled: false,
			RealTimeUpdates: true,
			ExportInterval:  time.Minute,
		}
	}

	monitor := &ProxyMonitor{
		config:   config,
		manager:  manager,
		stopChan: make(chan struct{}),
	}

	if config.Enabled {
		monitor.metrics = NewMetricsCollector()
		monitor.alerts = NewAlertManager(config.AlertThresholds)
		monitor.history = NewHistoryManager(config.HistoryRetention)
		
		if config.MetricsPort > 0 {
			monitor.setupHTTPServer()
		}
	}

	return monitor
}

// Start starts the proxy monitor
func (pm *ProxyMonitor) Start(ctx context.Context) error {
	if !pm.config.Enabled {
		return nil
	}

	monitoringLogger.Info("Starting proxy monitor")

	// Start metrics collection
	go pm.metricsCollectionLoop(ctx)

	// Start alerting
	if pm.config.AlertingEnabled {
		go pm.alertingLoop(ctx)
	}

	// Start history cleanup
	go pm.history.Start(ctx)

	// Start HTTP server
	if pm.server != nil {
		go func() {
			if err := pm.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				monitoringLogger.Error(fmt.Sprintf("HTTP server error: %v", err))
			}
		}()
	}

	return nil
}

// Stop stops the proxy monitor
func (pm *ProxyMonitor) Stop() error {
	if !pm.config.Enabled {
		return nil
	}

	monitoringLogger.Info("Stopping proxy monitor")
	
	close(pm.stopChan)

	if pm.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return pm.server.Shutdown(ctx)
	}

	return nil
}

// RecordRequest records a proxy request with detailed metrics
func (pm *ProxyMonitor) RecordRequest(proxyName string, dataPoint MetricDataPoint) {
	if !pm.config.Enabled {
		return
	}

	pm.metrics.RecordDataPoint(proxyName, dataPoint)
	pm.history.AddDataPoint(proxyName, dataPoint)
}

// GetCurrentMetrics returns current real-time metrics
func (pm *ProxyMonitor) GetCurrentMetrics() *CurrentMetrics {
	if !pm.config.Enabled {
		return nil
	}

	return pm.metrics.GetCurrentMetrics(pm.manager)
}

// GetProxyMetrics returns detailed metrics for a specific proxy
func (pm *ProxyMonitor) GetProxyMetrics(proxyName string) *ProxyMetricsSummary {
	if !pm.config.Enabled {
		return nil
	}

	return pm.metrics.GetProxyMetrics(proxyName)
}

// GetHistoricalMetrics returns historical metrics for a proxy
func (pm *ProxyMonitor) GetHistoricalMetrics(proxyName string, period time.Duration) *HistoricalMetrics {
	if !pm.config.Enabled {
		return nil
	}

	return pm.history.GetHistoricalMetrics(proxyName, period)
}

// GetActiveAlerts returns currently active alerts
func (pm *ProxyMonitor) GetActiveAlerts() []Alert {
	if !pm.config.Enabled || !pm.config.AlertingEnabled {
		return nil
	}

	return pm.alerts.GetActiveAlerts()
}

// GetPerformanceReport generates a performance report
func (pm *ProxyMonitor) GetPerformanceReport(period time.Duration) *PerformanceReport {
	if !pm.config.Enabled {
		return nil
	}

	return pm.generatePerformanceReport(period)
}

// GetCostReport generates a cost analysis report
func (pm *ProxyMonitor) GetCostReport(period time.Duration) *CostReport {
	if !pm.config.Enabled {
		return nil
	}

	return pm.generateCostReport(period)
}

// PerformanceReport represents a detailed performance analysis
type PerformanceReport struct {
	Period          string                           `json:"period"`
	GeneratedAt     time.Time                        `json:"generated_at"`
	Summary         *PerformanceSummary              `json:"summary"`
	TopPerformers   []ProxyPerformanceRanking        `json:"top_performers"`
	WorstPerformers []ProxyPerformanceRanking        `json:"worst_performers"`
	Trends          *PerformanceTrends               `json:"trends"`
	Recommendations []PerformanceRecommendation      `json:"recommendations"`
	GeographicAnalysis *GeographicPerformanceAnalysis `json:"geographic_analysis"`
}

// CostReport represents a detailed cost analysis
type CostReport struct {
	Period              string                    `json:"period"`
	GeneratedAt         time.Time                 `json:"generated_at"`
	TotalCost           float64                   `json:"total_cost"`
	CostByProxy         map[string]float64        `json:"cost_by_proxy"`
	CostByGeography     map[string]float64        `json:"cost_by_geography"`
	CostTrends          *CostTrends              `json:"cost_trends"`
	BudgetAnalysis      *BudgetAnalysis          `json:"budget_analysis"`
	CostOptimizations   []CostOptimization       `json:"cost_optimizations"`
}

// Implementation of core monitoring methods

func (pm *ProxyMonitor) metricsCollectionLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-pm.stopChan:
			return
		case <-ticker.C:
			pm.collectMetrics()
		}
	}
}

func (pm *ProxyMonitor) alertingLoop(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-pm.stopChan:
			return
		case <-ticker.C:
			pm.checkAlerts()
		}
	}
}

func (pm *ProxyMonitor) collectMetrics() {
	// This would be called periodically to update current metrics
	current := pm.metrics.GetCurrentMetrics(pm.manager)
	monitoringLogger.Debug(fmt.Sprintf("Collected metrics: %d total proxies, %d healthy, avg latency: %v", 
		current.TotalProxies, current.HealthyProxies, current.AverageLatency))
}

func (pm *ProxyMonitor) checkAlerts() {
	if pm.config.AlertThresholds == nil {
		return
	}

	current := pm.metrics.GetCurrentMetrics(pm.manager)
	
	// Check various alert conditions
	pm.checkSuccessRateAlert(current)
	pm.checkLatencyAlert(current)
	pm.checkHealthyProxiesAlert(current)
	pm.checkBudgetAlert(current)
}

func (pm *ProxyMonitor) checkSuccessRateAlert(metrics *CurrentMetrics) {
	if metrics.TotalRequests == 0 {
		return
	}

	successRate := float64(metrics.SuccessfulRequests) / float64(metrics.TotalRequests) * 100
	if successRate < pm.config.AlertThresholds.MinSuccessRate {
		alert := Alert{
			ID:        fmt.Sprintf("success_rate_%d", time.Now().Unix()),
			Type:      "success_rate",
			Severity:  "high",
			Message:   fmt.Sprintf("Success rate %.2f%% is below threshold %.2f%%", successRate, pm.config.AlertThresholds.MinSuccessRate),
			Timestamp: time.Now(),
		}
		pm.alerts.TriggerAlert(alert)
	}
}

func (pm *ProxyMonitor) checkLatencyAlert(metrics *CurrentMetrics) {
	if metrics.AverageLatency > pm.config.AlertThresholds.MaxLatency {
		alert := Alert{
			ID:        fmt.Sprintf("latency_%d", time.Now().Unix()),
			Type:      "latency",
			Severity:  "medium",
			Message:   fmt.Sprintf("Average latency %v exceeds threshold %v", metrics.AverageLatency, pm.config.AlertThresholds.MaxLatency),
			Timestamp: time.Now(),
		}
		pm.alerts.TriggerAlert(alert)
	}
}

func (pm *ProxyMonitor) checkHealthyProxiesAlert(metrics *CurrentMetrics) {
	if metrics.HealthyProxies < pm.config.AlertThresholds.MinHealthyProxies {
		alert := Alert{
			ID:        fmt.Sprintf("healthy_proxies_%d", time.Now().Unix()),
			Type:      "healthy_proxies",
			Severity:  "critical",
			Message:   fmt.Sprintf("Only %d healthy proxies available, below threshold %d", metrics.HealthyProxies, pm.config.AlertThresholds.MinHealthyProxies),
			Timestamp: time.Now(),
		}
		pm.alerts.TriggerAlert(alert)
	}
}

func (pm *ProxyMonitor) checkBudgetAlert(metrics *CurrentMetrics) {
	if pm.config.AlertThresholds == nil || pm.config.AlertThresholds.BudgetThreshold == 0 || pm.config.BudgetConfig == nil {
		return
	}

	// Get the appropriate budget based on the configured period
	budget := pm.getEffectiveBudget()
	if budget <= 0 {
		monitoringLogger.Warn("Budget alert check skipped: no valid budget configured")
		return
	}

	// Calculate budget usage percentage
	budgetUsagePercent := (metrics.TotalCost / budget) * 100

	if budgetUsagePercent >= pm.config.AlertThresholds.BudgetThreshold {
		severity := "medium"
		if budgetUsagePercent >= 90 {
			severity = "high"
		}
		if budgetUsagePercent >= 95 {
			severity = "critical"
		}

		alert := Alert{
			ID:        fmt.Sprintf("budget_%d", time.Now().Unix()),
			Type:      "budget_threshold",
			Severity:  severity,
			Message:   fmt.Sprintf("Budget usage at %.2f%% (threshold: %.2f%%, cost: $%.2f)", budgetUsagePercent, pm.config.AlertThresholds.BudgetThreshold, metrics.TotalCost),
			Timestamp: time.Now(),
			Metadata: map[string]interface{}{
				"budget_usage_percent": budgetUsagePercent,
				"current_cost":         metrics.TotalCost,
				"configured_budget":    budget,
				"threshold":            pm.config.AlertThresholds.BudgetThreshold,
			},
		}
		pm.alerts.TriggerAlert(alert)
	}
}

func (pm *ProxyMonitor) setupHTTPServer() {
	mux := http.NewServeMux()
	
	// Metrics endpoints
	mux.HandleFunc("/metrics", pm.handleMetrics)
	mux.HandleFunc("/metrics/current", pm.handleCurrentMetrics)
	mux.HandleFunc("/metrics/proxy/", pm.handleProxyMetrics)
	mux.HandleFunc("/metrics/historical/", pm.handleHistoricalMetrics)
	
	// Alert endpoints
	mux.HandleFunc("/alerts", pm.handleAlerts)
	mux.HandleFunc("/alerts/active", pm.handleActiveAlerts)
	
	// Report endpoints
	mux.HandleFunc("/reports/performance", pm.handlePerformanceReport)
	mux.HandleFunc("/reports/cost", pm.handleCostReport)
	
	// Health endpoint
	mux.HandleFunc("/health", pm.handleHealth)

	pm.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", pm.config.MetricsPort),
		Handler: mux,
	}
}

// HTTP handlers

func (pm *ProxyMonitor) handleMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	metrics := pm.GetCurrentMetrics()
	if metrics == nil {
		http.Error(w, "Monitoring not enabled", http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

func (pm *ProxyMonitor) handleCurrentMetrics(w http.ResponseWriter, r *http.Request) {
	pm.handleMetrics(w, r)
}

func (pm *ProxyMonitor) handleProxyMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract proxy name from URL path (e.g., /metrics/proxy/proxy_name)
	path := r.URL.Path
	parts := strings.Split(path, "/")
	if len(parts) < 4 {
		http.Error(w, "Proxy name required in path", http.StatusBadRequest)
		return
	}
	
	proxyName := parts[3]
	if proxyName == "" {
		http.Error(w, "Invalid proxy name", http.StatusBadRequest)
		return
	}

	metrics := pm.GetProxyMetrics(proxyName)
	if metrics == nil {
		http.Error(w, "Proxy not found or monitoring disabled", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

func (pm *ProxyMonitor) handleHistoricalMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract proxy name from URL path (e.g., /metrics/historical/proxy_name)
	path := r.URL.Path
	parts := strings.Split(path, "/")
	if len(parts) < 4 {
		http.Error(w, "Proxy name required in path", http.StatusBadRequest)
		return
	}
	
	proxyName := parts[3]
	if proxyName == "" {
		http.Error(w, "Invalid proxy name", http.StatusBadRequest)
		return
	}

	// Parse period parameter from query string (default to 24 hours)
	period := 24 * time.Hour
	if periodStr := r.URL.Query().Get("period"); periodStr != "" {
		if parsedPeriod, err := time.ParseDuration(periodStr); err == nil {
			period = parsedPeriod
		} else {
			http.Error(w, "Invalid period format (use duration format like '24h', '1h30m')", http.StatusBadRequest)
			return
		}
	}

	// Limit maximum period to prevent excessive memory usage
	maxPeriod := 7 * 24 * time.Hour // 7 days
	if period > maxPeriod {
		period = maxPeriod
	}

	metrics := pm.GetHistoricalMetrics(proxyName, period)
	if metrics == nil {
		http.Error(w, "No historical data found for proxy or monitoring disabled", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

func (pm *ProxyMonitor) handleAlerts(w http.ResponseWriter, r *http.Request) {
	alerts := pm.GetActiveAlerts()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(alerts)
}

func (pm *ProxyMonitor) handleActiveAlerts(w http.ResponseWriter, r *http.Request) {
	pm.handleAlerts(w, r)
}

func (pm *ProxyMonitor) handlePerformanceReport(w http.ResponseWriter, r *http.Request) {
	period := 24 * time.Hour // Default to 24 hours
	report := pm.GetPerformanceReport(period)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(report)
}

func (pm *ProxyMonitor) handleCostReport(w http.ResponseWriter, r *http.Request) {
	period := 24 * time.Hour // Default to 24 hours
	report := pm.GetCostReport(period)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(report)
}

func (pm *ProxyMonitor) handleHealth(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now(),
		"version":   "1.0.0",
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// Placeholder implementations for supporting components

func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		current:    &CurrentMetrics{Timestamp: time.Now()},
		historical: make(map[string]*HistoricalMetrics),
	}
}

func (mc *MetricsCollector) RecordDataPoint(proxyName string, dataPoint MetricDataPoint) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	// Update current metrics
	mc.current.TotalRequests++
	if dataPoint.Success {
		mc.current.SuccessfulRequests++
	} else {
		mc.current.FailedRequests++
	}
	
	// Update historical metrics
	if _, exists := mc.historical[proxyName]; !exists {
		mc.historical[proxyName] = &HistoricalMetrics{
			ProxyName:   proxyName,
			DataPoints:  make([]MetricDataPoint, 0),
			LastUpdated: time.Now(),
		}
	}
	
	mc.historical[proxyName].DataPoints = append(mc.historical[proxyName].DataPoints, dataPoint)
	mc.historical[proxyName].LastUpdated = time.Now()
}

func (mc *MetricsCollector) GetCurrentMetrics(manager *AdvancedProxyManager) *CurrentMetrics {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	
	// Update current metrics from manager
	if manager != nil {
		stats := manager.GetStats()
		mc.current.TotalProxies = stats.TotalProxies
		mc.current.HealthyProxies = stats.HealthyProxies
		mc.current.UnhealthyProxies = stats.FailedProxies
		mc.current.AverageLatency = stats.AverageResponse
	}
	
	mc.current.Timestamp = time.Now()
	return mc.current
}

func (mc *MetricsCollector) GetProxyMetrics(proxyName string) *ProxyMetricsSummary {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	
	if historical, exists := mc.historical[proxyName]; exists {
		// Calculate comprehensive summary from historical data
		now := time.Now()
		oneHourAgo := now.Add(-time.Hour)
		oneDayAgo := now.Add(-24 * time.Hour)
		
		var requests1h, requests24h, successful1h, successful24h int64
		var totalLatency1h, totalLatency24h time.Duration
		var cost1h, cost24h, qualitySum1h, qualitySum24h float64
		var latencyCount1h, latencyCount24h, qualityCount1h, qualityCount24h int
		
		// Analyze historical data points
		for _, dp := range historical.DataPoints {
			if dp.Timestamp.After(oneDayAgo) {
				requests24h++
				if dp.Success {
					successful24h++
				}
				totalLatency24h += dp.Latency
				latencyCount24h++
				cost24h += dp.Cost
				if dp.DataQuality > 0 {
					qualitySum24h += dp.DataQuality
					qualityCount24h++
				}
				
				if dp.Timestamp.After(oneHourAgo) {
					requests1h++
					if dp.Success {
						successful1h++
					}
					totalLatency1h += dp.Latency
					latencyCount1h++
					cost1h += dp.Cost
					if dp.DataQuality > 0 {
						qualitySum1h += dp.DataQuality
						qualityCount1h++
					}
				}
			}
		}
		
		// Calculate success rates
		var successRate1h, successRate24h float64
		if requests1h > 0 {
			successRate1h = float64(successful1h) / float64(requests1h) * 100
		}
		if requests24h > 0 {
			successRate24h = float64(successful24h) / float64(requests24h) * 100
		}
		
		// Calculate average latencies
		var avgLatency1h, avgLatency24h time.Duration
		if latencyCount1h > 0 {
			avgLatency1h = totalLatency1h / time.Duration(latencyCount1h)
		}
		if latencyCount24h > 0 {
			avgLatency24h = totalLatency24h / time.Duration(latencyCount24h)
		}
		
		// Calculate average data quality
		var avgQuality1h, avgQuality24h float64
		if qualityCount1h > 0 {
			avgQuality1h = qualitySum1h / float64(qualityCount1h)
		}
		if qualityCount24h > 0 {
			avgQuality24h = qualitySum24h / float64(qualityCount24h)
		}
		
		// Determine status based on recent performance
		status := "healthy"
		if successRate1h < 80 {
			status = "unhealthy"
		} else if successRate1h < 95 {
			status = "degraded"
		}
		if requests1h == 0 && requests24h == 0 {
			status = "unknown"
		}
		
		return &ProxyMetricsSummary{
			ProxyName:         proxyName,
			Status:            status,
			LastSeen:          historical.LastUpdated,
			RequestsLast1h:    requests1h,
			RequestsLast24h:   requests24h,
			SuccessRate1h:     successRate1h,
			SuccessRate24h:    successRate24h,
			AverageLatency1h:  avgLatency1h,
			AverageLatency24h: avgLatency24h,
			CostLast1h:        cost1h,
			CostLast24h:       cost24h,
			DataQuality1h:     avgQuality1h,
			DataQuality24h:    avgQuality24h,
		}
	}
	
	return nil
}

func NewAlertManager(thresholds *AlertThresholds) *AlertManager {
	return &AlertManager{
		config:   thresholds,
		alerts:   make([]Alert, 0),
		handlers: make([]AlertHandler, 0),
	}
}

func (am *AlertManager) TriggerAlert(alert Alert) {
	am.mu.Lock()
	defer am.mu.Unlock()
	
	// Check for duplicate alerts
	for _, existing := range am.alerts {
		if existing.Type == alert.Type && existing.ProxyName == alert.ProxyName && !existing.Resolved {
			return // Alert already exists
		}
	}
	
	am.alerts = append(am.alerts, alert)
	
	// Notify handlers
	for _, handler := range am.handlers {
		go handler.HandleAlert(alert)
	}
	
	monitoringLogger.Warn(fmt.Sprintf("Alert triggered: %s - %s", alert.Type, alert.Message))
}

func (am *AlertManager) GetActiveAlerts() []Alert {
	am.mu.RLock()
	defer am.mu.RUnlock()
	
	var active []Alert
	for _, alert := range am.alerts {
		if !alert.Resolved {
			active = append(active, alert)
		}
	}
	
	return active
}

func NewHistoryManager(retention time.Duration) *HistoryManager {
	return &HistoryManager{
		retention: retention,
		storage:   make(map[string][]MetricDataPoint),
		stopChan:  make(chan struct{}),
	}
}

func (hm *HistoryManager) Start(ctx context.Context) {
	hm.cleanupTicker = time.NewTicker(time.Hour)
	defer hm.cleanupTicker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-hm.stopChan:
			return
		case <-hm.cleanupTicker.C:
			hm.cleanup()
		}
	}
}

func (hm *HistoryManager) cleanup() {
	hm.storageMu.Lock()
	defer hm.storageMu.Unlock()
	
	cutoff := time.Now().Add(-hm.retention)
	
	for proxyName, dataPoints := range hm.storage {
		var filtered []MetricDataPoint
		for _, dp := range dataPoints {
			if dp.Timestamp.After(cutoff) {
				filtered = append(filtered, dp)
			}
		}
		hm.storage[proxyName] = filtered
	}
}

func (hm *HistoryManager) AddDataPoint(proxyName string, dataPoint MetricDataPoint) {
	hm.storageMu.Lock()
	defer hm.storageMu.Unlock()
	
	if _, exists := hm.storage[proxyName]; !exists {
		hm.storage[proxyName] = make([]MetricDataPoint, 0)
	}
	
	hm.storage[proxyName] = append(hm.storage[proxyName], dataPoint)
}

func (hm *HistoryManager) GetHistoricalMetrics(proxyName string, period time.Duration) *HistoricalMetrics {
	hm.storageMu.RLock()
	defer hm.storageMu.RUnlock()
	
	dataPoints, exists := hm.storage[proxyName]
	if !exists {
		return nil
	}
	
	cutoff := time.Now().Add(-period)
	var filtered []MetricDataPoint
	for _, dp := range dataPoints {
		if dp.Timestamp.After(cutoff) {
			filtered = append(filtered, dp)
		}
	}
	
	return &HistoricalMetrics{
		ProxyName:   proxyName,
		DataPoints:  filtered,
		LastUpdated: time.Now(),
	}
}

// Report generation methods (simplified implementations)

func (pm *ProxyMonitor) generatePerformanceReport(period time.Duration) *PerformanceReport {
	current := pm.metrics.current
	
	// Calculate success rate safely
	successRate := 0.0
	if current.TotalRequests > 0 {
		successRate = float64(current.SuccessfulRequests) / float64(current.TotalRequests) * 100
	}
	
	// Generate proxy performance rankings
	var topPerformers, worstPerformers []ProxyPerformanceRanking
	proxyScores := make(map[string]float64)
	
	// Calculate performance scores for each proxy
	for proxyName := range pm.metrics.historical {
		metrics := pm.metrics.GetProxyMetrics(proxyName)
		if metrics != nil {
			// Simple performance score: success rate weighted by request volume
			score := metrics.SuccessRate24h * (float64(metrics.RequestsLast24h) / 100.0)
			proxyScores[proxyName] = score
		}
	}
	
	// Sort and create rankings
	rank := 1
	for proxyName, score := range proxyScores {
		ranking := ProxyPerformanceRanking{
			ProxyName: proxyName,
			Score:     score,
			Rank:      rank,
		}
		
		if len(topPerformers) < 5 {
			topPerformers = append(topPerformers, ranking)
		}
		if len(worstPerformers) < 5 && score < 80 {
			worstPerformers = append(worstPerformers, ranking)
		}
		rank++
	}
	
	// Generate recommendations based on current metrics
	var recommendations []PerformanceRecommendation
	if successRate < 95 {
		recommendations = append(recommendations, PerformanceRecommendation{
			Type:        "success_rate",
			Priority:    "high",
			Description: fmt.Sprintf("Overall success rate is %.2f%%, below optimal threshold", successRate),
			Action:      "Review failing proxies and consider adding more reliable proxy sources",
		})
	}
	
	if current.AverageLatency > 5*time.Second {
		recommendations = append(recommendations, PerformanceRecommendation{
			Type:        "latency",
			Priority:    "medium",
			Description: fmt.Sprintf("Average latency is %v, which may impact performance", current.AverageLatency),
			Action:      "Consider using geographically closer proxies or faster proxy tiers",
		})
	}
	
	return &PerformanceReport{
		Period:      period.String(),
		GeneratedAt: time.Now(),
		Summary: &PerformanceSummary{
			TotalRequests:  current.TotalRequests,
			SuccessRate:    successRate,
			AverageLatency: current.AverageLatency,
		},
		TopPerformers:   topPerformers,
		WorstPerformers: worstPerformers,
		Recommendations: recommendations,
		Trends: &PerformanceTrends{
			LatencyTrend:    "stable", // Would be calculated from historical data
			SuccessTrend:    "stable", // Would be calculated from historical data
			ThroughputTrend: "stable", // Would be calculated from historical data
		},
	}
}

func (pm *ProxyMonitor) generateCostReport(period time.Duration) *CostReport {
	current := pm.metrics.current
	
	// Calculate cost by proxy
	costByProxy := make(map[string]float64)
	costByGeography := make(map[string]float64)
	
	for proxyName := range pm.metrics.historical {
		metrics := pm.metrics.GetProxyMetrics(proxyName)
		if metrics != nil {
			costByProxy[proxyName] = metrics.CostLast24h
			
			// Geographic cost aggregation (simplified)
			if metrics.Geographic != nil && metrics.Geographic.Country != "" {
				costByGeography[metrics.Geographic.Country] += metrics.CostLast24h
			} else {
				costByGeography["Unknown"] += metrics.CostLast24h
			}
		}
	}
	
	// Generate cost optimizations recommendations
	var optimizations []CostOptimization
	
	// Find expensive proxies with low success rates
	for proxyName, cost := range costByProxy {
		if cost > 10.0 { // Threshold for expensive proxies
			metrics := pm.metrics.GetProxyMetrics(proxyName)
			if metrics != nil && metrics.SuccessRate24h < 90 {
				optimizations = append(optimizations, CostOptimization{
					Type:             "expensive_low_performance",
					PotentialSavings: cost * 0.5, // Estimated savings
					Description:      fmt.Sprintf("Proxy %s has high cost ($%.2f) but low success rate (%.1f%%)", proxyName, cost, metrics.SuccessRate24h),
				})
			}
		}
	}
	
	// Budget analysis using configured budget
	configuredBudget := pm.getEffectiveBudget()
	if configuredBudget <= 0 {
		configuredBudget = 1000.0 // fallback budget for reporting
	}
	budgetUsed := current.TotalCost
	budgetRemaining := configuredBudget - budgetUsed
	
	// Simple projection based on current rate
	hoursInPeriod := period.Hours()
	if hoursInPeriod > 0 {
		dailyRate := current.TotalCost / (hoursInPeriod / 24)
		projectedSpend := dailyRate * 30 // 30-day projection
		
		return &CostReport{
			Period:          period.String(),
			GeneratedAt:     time.Now(),
			TotalCost:       current.TotalCost,
			CostByProxy:     costByProxy,
			CostByGeography: costByGeography,
			CostTrends: &CostTrends{
				Trend:         "stable", // Would be calculated from historical data
				ChangePercent: 0,        // Would be calculated from historical data
			},
			BudgetAnalysis: &BudgetAnalysis{
				BudgetUsed:      budgetUsed,
				BudgetRemaining: budgetRemaining,
				ProjectedSpend:  projectedSpend,
			},
			CostOptimizations: optimizations,
		}
	}
	
	return &CostReport{
		Period:          period.String(),
		GeneratedAt:     time.Now(),
		TotalCost:       current.TotalCost,
		CostByProxy:     costByProxy,
		CostByGeography: costByGeography,
		BudgetAnalysis: &BudgetAnalysis{
			BudgetUsed:      budgetUsed,
			BudgetRemaining: budgetRemaining,
		},
		CostOptimizations: optimizations,
	}
}

// Supporting types for reports (placeholder structures)

type PerformanceSummary struct {
	TotalRequests   int64   `json:"total_requests"`
	SuccessRate     float64 `json:"success_rate"`
	AverageLatency  time.Duration `json:"average_latency"`
}

type ProxyPerformanceRanking struct {
	ProxyName string  `json:"proxy_name"`
	Score     float64 `json:"score"`
	Rank      int     `json:"rank"`
}

type PerformanceTrends struct {
	LatencyTrend    string `json:"latency_trend"` // improving, degrading, stable
	SuccessTrend    string `json:"success_trend"`
	ThroughputTrend string `json:"throughput_trend"`
}

type PerformanceRecommendation struct {
	Type        string `json:"type"`
	Priority    string `json:"priority"`
	Description string `json:"description"`
	Action      string `json:"action"`
}

type GeographicPerformanceAnalysis struct {
	BestRegions  []string `json:"best_regions"`
	WorstRegions []string `json:"worst_regions"`
}

type CostTrends struct {
	Trend         string  `json:"trend"` // increasing, decreasing, stable
	ChangePercent float64 `json:"change_percent"`
}

type BudgetAnalysis struct {
	BudgetUsed    float64 `json:"budget_used"`
	BudgetRemaining float64 `json:"budget_remaining"`
	ProjectedSpend float64 `json:"projected_spend"`
}

type CostOptimization struct {
	Type            string  `json:"type"`
	PotentialSavings float64 `json:"potential_savings"`
	Description     string  `json:"description"`
}

// getEffectiveBudget returns the effective budget amount based on the configured period
func (pm *ProxyMonitor) getEffectiveBudget() float64 {
	if pm.config.BudgetConfig == nil {
		return 0
	}

	budgetConfig := pm.config.BudgetConfig
	
	// Determine budget based on configured period
	switch strings.ToLower(budgetConfig.BudgetPeriod) {
	case "hourly":
		return budgetConfig.HourlyBudget
	case "daily":
		return budgetConfig.DailyBudget
	case "monthly":
		return budgetConfig.MonthlyBudget
	default:
		// Default to daily budget if no period specified
		if budgetConfig.DailyBudget > 0 {
			return budgetConfig.DailyBudget
		}
		if budgetConfig.HourlyBudget > 0 {
			return budgetConfig.HourlyBudget
		}
		if budgetConfig.MonthlyBudget > 0 {
			return budgetConfig.MonthlyBudget
		}
	}
	
	return 0
}

// ValidateBudgetConfig validates budget configuration settings
func ValidateBudgetConfig(config *BudgetConfig) error {
	if config == nil {
		return nil // Budget monitoring is optional
	}
	
	// At least one budget must be configured
	if config.DailyBudget <= 0 && config.HourlyBudget <= 0 && config.MonthlyBudget <= 0 {
		return fmt.Errorf("at least one budget amount (daily, hourly, or monthly) must be configured")
	}
	
	// Validate thresholds
	if config.WarnThreshold < 0 || config.WarnThreshold > 100 {
		return fmt.Errorf("warn_threshold must be between 0 and 100")
	}
	if config.CritThreshold < 0 || config.CritThreshold > 100 {
		return fmt.Errorf("crit_threshold must be between 0 and 100")
	}
	if config.WarnThreshold >= config.CritThreshold {
		return fmt.Errorf("warn_threshold must be less than crit_threshold")
	}
	
	// Validate budget period
	validPeriods := []string{"hourly", "daily", "monthly"}
	if config.BudgetPeriod != "" {
		period := strings.ToLower(config.BudgetPeriod)
		valid := false
		for _, validPeriod := range validPeriods {
			if period == validPeriod {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("budget_period must be one of: %v", validPeriods)
		}
	}
	
	return nil
}