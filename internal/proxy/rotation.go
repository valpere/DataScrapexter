// internal/proxy/rotation.go - Advanced proxy rotation strategies
package proxy

import (
	"crypto/rand"
	"fmt"
	"math"
	mathrand "math/rand"
	"net"
	"net/url"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
	"unsafe"

	"github.com/valpere/DataScrapexter/internal/utils"
)

// Initialize secure random seed on package load
func init() {
	// Seed math/rand with cryptographically secure random data for better randomness
	var seed int64
	seedBytes := make([]byte, 8)
	_, err := rand.Read(seedBytes)
	if err != nil {
		// Check if we're in security-sensitive mode
		if os.Getenv("DATASCRAPEXTER_SECURITY_STRICT") == "true" || os.Getenv("DATASCRAPEXTER_FAIL_ON_WEAK_RANDOM") == "true" {
			// SECURITY: Fail fast in security-sensitive environments
			rotationLogger.Error(fmt.Sprintf("FATAL: Cryptographically secure randomization failed in strict security mode: %v", err))
			panic(fmt.Sprintf("SECURITY REQUIREMENT VIOLATION: crypto/rand unavailable in strict mode - proxy rotation requires cryptographically secure randomization: %v", err))
		}
		
		// SECURITY: Log critical warning about reduced security
		rotationLogger.Error(fmt.Sprintf("CRITICAL SECURITY WARNING: Cryptographically secure randomization failed, using enhanced time-based seeding with reduced security: %v", err))
		rotationLogger.Warn("Proxy rotation patterns may be predictable - consider fixing the crypto/rand issue or set DATASCRAPEXTER_SECURITY_STRICT=true for fail-fast behavior")
		
		// Enhanced time-based seeding with multiple entropy sources to improve unpredictability
		now := time.Now()
		seed = now.UnixNano()
		// Add additional entropy from multiple time sources
		seed ^= now.Unix() << 32
		seed ^= int64(now.Nanosecond()) << 16
		// Add memory address entropy (varies per process startup)
		seed ^= int64(uintptr(unsafe.Pointer(&seed)))
	} else {
		// Convert cryptographically secure bytes to int64 for seeding
		for i, b := range seedBytes {
			seed |= int64(b) << (8 * i)
		}
	}
	mathrand.Seed(seed)
}

var rotationLogger = utils.NewComponentLogger("proxy-rotation")

// AdvancedRotationStrategy extends the basic rotation strategies
type AdvancedRotationStrategy string

const (
	// Basic strategies (inherited)
	AdvancedRotationRoundRobin AdvancedRotationStrategy = "round_robin"
	AdvancedRotationRandom     AdvancedRotationStrategy = "random"
	AdvancedRotationWeighted   AdvancedRotationStrategy = "weighted"
	AdvancedRotationHealthy    AdvancedRotationStrategy = "healthy"

	// Advanced strategies
	AdvancedRotationGeographic    AdvancedRotationStrategy = "geographic"
	AdvancedRotationPerformance   AdvancedRotationStrategy = "performance"
	AdvancedRotationLatencyBased  AdvancedRotationStrategy = "latency_based"
	AdvancedRotationLoadBalanced  AdvancedRotationStrategy = "load_balanced"
	AdvancedRotationFailoverGroup AdvancedRotationStrategy = "failover_group"
	AdvancedRotationTimeZoneBased AdvancedRotationStrategy = "timezone_based"
	AdvancedRotationCostOptimized AdvancedRotationStrategy = "cost_optimized"
	AdvancedRotationMLPredictive  AdvancedRotationStrategy = "ml_predictive"
)

// GeographicLocation represents a geographic location for proxy routing
type GeographicLocation struct {
	Country     string  `yaml:"country" json:"country"`
	City        string  `yaml:"city,omitempty" json:"city,omitempty"`
	Region      string  `yaml:"region,omitempty" json:"region,omitempty"`
	Continent   string  `yaml:"continent,omitempty" json:"continent,omitempty"`
	Latitude    float64 `yaml:"latitude,omitempty" json:"latitude,omitempty"`
	Longitude   float64 `yaml:"longitude,omitempty" json:"longitude,omitempty"`
	TimeZone    string  `yaml:"timezone,omitempty" json:"timezone,omitempty"`
	CountryCode string  `yaml:"country_code,omitempty" json:"country_code,omitempty"`
}

// PerformanceMetrics represents performance characteristics of a proxy
type PerformanceMetrics struct {
	AverageLatency    time.Duration `json:"average_latency"`
	SuccessRate       float64       `json:"success_rate"`
	Throughput        float64       `json:"throughput"` // requests per second
	Bandwidth         float64       `json:"bandwidth"` // MB/s
	ConcurrentLimit   int           `json:"concurrent_limit"`
	Cost              float64       `json:"cost"` // cost per request
	ReliabilityScore  float64       `json:"reliability_score"`
	QualityScore      float64       `json:"quality_score"`
	LastMeasured      time.Time     `json:"last_measured"`
	SampleSize        int           `json:"sample_size"`
	ErrorRate         float64       `json:"error_rate"`
	TimeoutRate       float64       `json:"timeout_rate"`
	RetryRate         float64       `json:"retry_rate"`
	DataQuality       float64       `json:"data_quality"` // success rate of data extraction
}

// ProxyGroup represents a group of proxies for failover scenarios
type ProxyGroup struct {
	Name        string           `yaml:"name" json:"name"`
	Priority    int              `yaml:"priority" json:"priority"`
	Proxies     []*ProxyInstance `yaml:"-" json:"proxies"`
	ProxyNames  []string         `yaml:"proxy_names" json:"proxy_names"`
	GroupType   string           `yaml:"group_type" json:"group_type"` // primary, secondary, emergency
	MaxFailures int              `yaml:"max_failures" json:"max_failures"`
	Enabled     bool             `yaml:"enabled" json:"enabled"`
}

// AdvancedProxyProvider extends ProxyProvider with advanced features
type AdvancedProxyProvider struct {
	ProxyProvider                  // Embed basic provider
	Location         *GeographicLocation `yaml:"location,omitempty" json:"location,omitempty"`
	Performance      *PerformanceMetrics `yaml:"performance,omitempty" json:"performance,omitempty"`
	Groups           []string            `yaml:"groups,omitempty" json:"groups,omitempty"`
	Tags             []string            `yaml:"tags,omitempty" json:"tags,omitempty"`
	MaxConcurrent    int                 `yaml:"max_concurrent,omitempty" json:"max_concurrent,omitempty"`
	BandwidthLimit   float64             `yaml:"bandwidth_limit,omitempty" json:"bandwidth_limit,omitempty"` // MB/s
	CostTier         string              `yaml:"cost_tier,omitempty" json:"cost_tier,omitempty"`             // free, premium, enterprise
	Provider         string              `yaml:"provider,omitempty" json:"provider,omitempty"`               // brightdata, oxylabs, etc.
	Residential      bool                `yaml:"residential,omitempty" json:"residential,omitempty"`
	StickySession    bool                `yaml:"sticky_session,omitempty" json:"sticky_session,omitempty"`
	RotationTime     time.Duration       `yaml:"rotation_time,omitempty" json:"rotation_time,omitempty"`
}

// AdvancedProxyConfig extends ProxyConfig with advanced rotation features
type AdvancedProxyConfig struct {
	ProxyConfig                           // Embed basic config
	AdvancedStrategy      AdvancedRotationStrategy `yaml:"advanced_strategy" json:"advanced_strategy"`
	GeographicPreference  []string                 `yaml:"geographic_preference,omitempty" json:"geographic_preference,omitempty"`
	PerformanceThresholds *PerformanceThresholds   `yaml:"performance_thresholds,omitempty" json:"performance_thresholds,omitempty"`
	Groups                []ProxyGroup             `yaml:"groups,omitempty" json:"groups,omitempty"`
	LoadBalancing         *LoadBalancingConfig     `yaml:"load_balancing,omitempty" json:"load_balancing,omitempty"`
	MLConfig              *MLPredictionConfig      `yaml:"ml_config,omitempty" json:"ml_config,omitempty"`
	CostOptimization      *CostOptimizationConfig  `yaml:"cost_optimization,omitempty" json:"cost_optimization,omitempty"`
	AdvancedProviders     []AdvancedProxyProvider  `yaml:"advanced_providers,omitempty" json:"advanced_providers,omitempty"`
}

// PerformanceThresholds defines minimum performance requirements
type PerformanceThresholds struct {
	MaxLatency       time.Duration `yaml:"max_latency" json:"max_latency"`
	MinSuccessRate   float64       `yaml:"min_success_rate" json:"min_success_rate"`
	MinThroughput    float64       `yaml:"min_throughput" json:"min_throughput"`
	MinQualityScore  float64       `yaml:"min_quality_score" json:"min_quality_score"`
	MaxErrorRate     float64       `yaml:"max_error_rate" json:"max_error_rate"`
	MaxTimeoutRate   float64       `yaml:"max_timeout_rate" json:"max_timeout_rate"`
	MinDataQuality   float64       `yaml:"min_data_quality" json:"min_data_quality"`
	MinSampleSize    int           `yaml:"min_sample_size" json:"min_sample_size"`
}

// LoadBalancingConfig defines load balancing parameters
type LoadBalancingConfig struct {
	Algorithm           string        `yaml:"algorithm" json:"algorithm"` // round_robin, least_connections, weighted_round_robin
	MaxConcurrentPerProxy int         `yaml:"max_concurrent_per_proxy" json:"max_concurrent_per_proxy"`
	HealthCheckInterval time.Duration `yaml:"health_check_interval" json:"health_check_interval"`
	CircuitBreakerEnabled bool        `yaml:"circuit_breaker_enabled" json:"circuit_breaker_enabled"`
	CircuitBreakerThreshold int       `yaml:"circuit_breaker_threshold" json:"circuit_breaker_threshold"`
	CircuitBreakerTimeout time.Duration `yaml:"circuit_breaker_timeout" json:"circuit_breaker_timeout"`
}

// MLPredictionConfig defines performance-based prediction parameters
// Note: Currently implements heuristic selection rather than machine learning
type MLPredictionConfig struct {
	Enabled                bool          `yaml:"enabled" json:"enabled"`
	ModelPath              string        `yaml:"model_path,omitempty" json:"model_path,omitempty"`
	PredictionWindow       time.Duration `yaml:"prediction_window" json:"prediction_window"`
	LearningRate           float64       `yaml:"learning_rate" json:"learning_rate"`
	Features               []string      `yaml:"features" json:"features"`
	TrainingDataRetention  time.Duration `yaml:"training_data_retention" json:"training_data_retention"`
	MinTrainingDataPoints  int           `yaml:"min_training_data_points" json:"min_training_data_points"`
	RetrainInterval        time.Duration `yaml:"retrain_interval" json:"retrain_interval"`
}

// CostOptimizationConfig defines cost optimization parameters
type CostOptimizationConfig struct {
	Enabled           bool    `yaml:"enabled" json:"enabled"`
	MaxCostPerRequest float64 `yaml:"max_cost_per_request" json:"max_cost_per_request"`
	BudgetLimit       float64 `yaml:"budget_limit" json:"budget_limit"` // per hour/day
	CostPriority      float64 `yaml:"cost_priority" json:"cost_priority"` // 0-1, weight vs performance
	PreferFreeProxies bool    `yaml:"prefer_free_proxies" json:"prefer_free_proxies"`
	CostTracking      bool    `yaml:"cost_tracking" json:"cost_tracking"`
}

// AdvancedProxyManager extends ProxyManager with advanced rotation strategies
type AdvancedProxyManager struct {
	*ProxyManager
	advancedConfig    *AdvancedProxyConfig
	advancedProviders []*AdvancedProxyInstance
	groups            map[string]*ProxyGroup
	performanceTracker *PerformanceTracker
	geoResolver       *GeographicResolver
	mlPredictor       *MLPredictor
	costTracker       *CostTracker
	loadBalancer      *LoadBalancer
	mu                sync.RWMutex
}

// AdvancedProxyInstance extends ProxyInstance with advanced features
type AdvancedProxyInstance struct {
	*ProxyInstance
	Advanced          *AdvancedProxyProvider
	CurrentConnections int
	Performance       *PerformanceMetrics
	GroupMembership   []string
	LastUsed          time.Time
	StickySessionID   string
	circuitBreaker    *CircuitBreaker
}

// CircuitBreaker implements circuit breaker pattern for proxy failures
type CircuitBreaker struct {
	failureCount    int
	threshold       int
	timeout         time.Duration
	lastFailure     time.Time
	state           string // closed, open, half_open
	mu              sync.RWMutex
}

// PerformanceTracker tracks proxy performance metrics
type PerformanceTracker struct {
	metrics map[string]*PerformanceMetrics
	mu      sync.RWMutex
}

// GeographicResolver resolves geographic information for target URLs
type GeographicResolver struct {
	cache    map[string]*GeographicLocation
	cacheMu  sync.RWMutex
	resolver func(string) (*GeographicLocation, error)
}

// MLPredictor implements performance-based heuristic proxy selection
// Note: Despite the name, this currently uses performance metrics rather than machine learning
// TODO: Replace with actual ML implementation or rename to PerformanceBasedSelector
type MLPredictor struct {
	enabled    bool
	model      interface{} // Placeholder for ML model - currently unused
	features   []string
	history    []PredictionDataPoint
	historyMu  sync.RWMutex
}

// PredictionDataPoint represents a data point for ML training
type PredictionDataPoint struct {
	ProxyName     string
	Features      map[string]float64
	ActualLatency time.Duration
	Success       bool
	Timestamp     time.Time
}

// CostTracker tracks proxy usage costs
type CostTracker struct {
	usage       map[string]*CostUsage
	usageMu     sync.RWMutex
	budget      float64
	currentCost float64
}

// CostUsage represents cost usage for a proxy
type CostUsage struct {
	ProxyName    string
	RequestCount int64
	TotalCost    float64
	LastReset    time.Time
}

// LoadBalancer handles load balancing across proxies
type LoadBalancer struct {
	algorithm     string
	connections   map[string]int
	connectionsMu sync.RWMutex
}

// NewAdvancedProxyManager creates a new advanced proxy manager
func NewAdvancedProxyManager(config *AdvancedProxyConfig) *AdvancedProxyManager {
	// Create base proxy manager
	baseManager := NewProxyManager(&config.ProxyConfig)

	apm := &AdvancedProxyManager{
		ProxyManager:       baseManager,
		advancedConfig:     config,
		groups:             make(map[string]*ProxyGroup),
		performanceTracker: NewPerformanceTracker(),
		geoResolver:        NewGeographicResolver(),
		mlPredictor:        NewMLPredictor(config.MLConfig),
		costTracker:        NewCostTracker(config.CostOptimization),
		loadBalancer:       NewLoadBalancer(config.LoadBalancing),
	}

	// Initialize advanced proxies
	apm.initializeAdvancedProxies()

	// Initialize groups
	apm.initializeGroups()

	return apm
}

// GetAdvancedProxy returns a proxy using advanced rotation strategies
func (apm *AdvancedProxyManager) GetAdvancedProxy(targetURL string) (*AdvancedProxyInstance, error) {
	if !apm.advancedConfig.Enabled || len(apm.advancedProviders) == 0 {
		return nil, nil
	}

	apm.mu.Lock()
	defer apm.mu.Unlock()

	switch apm.advancedConfig.AdvancedStrategy {
	case AdvancedRotationGeographic:
		return apm.getGeographicProxy(targetURL)
	case AdvancedRotationPerformance:
		return apm.getPerformanceProxy()
	case AdvancedRotationLatencyBased:
		return apm.getLatencyBasedProxy()
	case AdvancedRotationLoadBalanced:
		return apm.getLoadBalancedProxy()
	case AdvancedRotationFailoverGroup:
		return apm.getFailoverGroupProxy()
	case AdvancedRotationTimeZoneBased:
		return apm.getTimeZoneBasedProxy(targetURL)
	case AdvancedRotationCostOptimized:
		return apm.getCostOptimizedProxy()
	case AdvancedRotationMLPredictive:
		return apm.getMLPredictiveProxy(targetURL)
	default:
		return apm.getPerformanceProxy()
	}
}

// getGeographicProxy selects proxy based on geographic proximity to target
func (apm *AdvancedProxyManager) getGeographicProxy(targetURL string) (*AdvancedProxyInstance, error) {
	// Resolve target location
	targetLocation, err := apm.geoResolver.ResolveLocation(targetURL)
	if err != nil {
		rotationLogger.Debug(fmt.Sprintf("Failed to resolve target location for %s: %v", targetURL, err))
		// Fall back to performance-based selection
		return apm.getPerformanceProxy()
	}

	// Find proxies in preferred geographic regions
	var candidates []*AdvancedProxyInstance
	
	// First priority: same country
	for _, proxy := range apm.advancedProviders {
		if !apm.isProxyAvailable(proxy) {
			continue
		}
		if proxy.Advanced.Location != nil && 
		   proxy.Advanced.Location.Country == targetLocation.Country {
			candidates = append(candidates, proxy)
		}
	}

	// Second priority: same continent
	if len(candidates) == 0 {
		for _, proxy := range apm.advancedProviders {
			if !apm.isProxyAvailable(proxy) {
				continue
			}
			if proxy.Advanced.Location != nil && 
			   proxy.Advanced.Location.Continent == targetLocation.Continent {
				candidates = append(candidates, proxy)
			}
		}
	}

	// Third priority: geographic preferences from config
	if len(candidates) == 0 {
		for _, preferredCountry := range apm.advancedConfig.GeographicPreference {
			for _, proxy := range apm.advancedProviders {
				if !apm.isProxyAvailable(proxy) {
					continue
				}
				if proxy.Advanced.Location != nil && 
				   proxy.Advanced.Location.Country == preferredCountry {
					candidates = append(candidates, proxy)
				}
			}
			if len(candidates) > 0 {
				break
			}
		}
	}

	// Fall back to all available proxies
	if len(candidates) == 0 {
		candidates = apm.getAvailableAdvancedProxies()
	}

	if len(candidates) == 0 {
		return nil, fmt.Errorf("no available proxies for geographic routing")
	}

	// Sort by geographic distance if coordinates are available
	if targetLocation.Latitude != 0 && targetLocation.Longitude != 0 {
		sort.Slice(candidates, func(i, j int) bool {
			distI := apm.calculateDistance(targetLocation, candidates[i].Advanced.Location)
			distJ := apm.calculateDistance(targetLocation, candidates[j].Advanced.Location)
			return distI < distJ
		})
	}

	// Select best candidate based on performance among geographically close proxies
	return apm.selectBestCandidate(candidates), nil
}

// getPerformanceProxy selects proxy based on performance metrics
func (apm *AdvancedProxyManager) getPerformanceProxy() (*AdvancedProxyInstance, error) {
	candidates := apm.getAvailableAdvancedProxies()
	if len(candidates) == 0 {
		return nil, fmt.Errorf("no available proxies for performance-based routing")
	}

	// Filter by performance thresholds
	if apm.advancedConfig.PerformanceThresholds != nil {
		candidates = apm.filterByPerformanceThresholds(candidates)
	}

	if len(candidates) == 0 {
		rotationLogger.Warn("No proxies meet performance thresholds, using all available")
		candidates = apm.getAvailableAdvancedProxies()
	}

	// Sort by composite performance score
	sort.Slice(candidates, func(i, j int) bool {
		scoreI := apm.calculatePerformanceScore(candidates[i])
		scoreJ := apm.calculatePerformanceScore(candidates[j])
		return scoreI > scoreJ // Higher score is better
	})

	return candidates[0], nil
}

// getLatencyBasedProxy selects proxy with lowest latency
func (apm *AdvancedProxyManager) getLatencyBasedProxy() (*AdvancedProxyInstance, error) {
	candidates := apm.getAvailableAdvancedProxies()
	if len(candidates) == 0 {
		return nil, fmt.Errorf("no available proxies for latency-based routing")
	}

	// Sort by average latency (ascending)
	sort.Slice(candidates, func(i, j int) bool {
		latencyI := candidates[i].Performance.AverageLatency
		latencyJ := candidates[j].Performance.AverageLatency
		if latencyI == 0 {
			latencyI = time.Hour // Treat unknown latency as very high
		}
		if latencyJ == 0 {
			latencyJ = time.Hour
		}
		return latencyI < latencyJ
	})

	return candidates[0], nil
}

// getLoadBalancedProxy selects proxy based on current load
func (apm *AdvancedProxyManager) getLoadBalancedProxy() (*AdvancedProxyInstance, error) {
	return apm.loadBalancer.SelectProxy(apm.getAvailableAdvancedProxies())
}

// getFailoverGroupProxy selects proxy from failover groups
func (apm *AdvancedProxyManager) getFailoverGroupProxy() (*AdvancedProxyInstance, error) {
	// Sort groups by priority
	var sortedGroups []*ProxyGroup
	for _, group := range apm.groups {
		if group.Enabled {
			sortedGroups = append(sortedGroups, group)
		}
	}

	sort.Slice(sortedGroups, func(i, j int) bool {
		return sortedGroups[i].Priority < sortedGroups[j].Priority
	})

	// Try each group in priority order
	for _, group := range sortedGroups {
		availableInGroup := apm.getAvailableProxiesInGroup(group)
		if len(availableInGroup) > 0 {
			// Use performance-based selection within the group
			return apm.selectBestCandidate(availableInGroup), nil
		}
	}

	return nil, fmt.Errorf("no available proxies in any failover group")
}

// getTimeZoneBasedProxy selects proxy based on target timezone
func (apm *AdvancedProxyManager) getTimeZoneBasedProxy(targetURL string) (*AdvancedProxyInstance, error) {
	// Resolve target timezone
	targetLocation, err := apm.geoResolver.ResolveLocation(targetURL)
	if err != nil || targetLocation.TimeZone == "" {
		// Fall back to performance-based selection
		return apm.getPerformanceProxy()
	}

	// Find proxies in compatible timezones (business hours overlap)
	candidates := apm.getProxiesInCompatibleTimezones(targetLocation.TimeZone)
	
	if len(candidates) == 0 {
		// Fall back to all available proxies
		candidates = apm.getAvailableAdvancedProxies()
	}

	if len(candidates) == 0 {
		return nil, fmt.Errorf("no available proxies for timezone-based routing")
	}

	return apm.selectBestCandidate(candidates), nil
}

// getCostOptimizedProxy selects proxy based on cost optimization
func (apm *AdvancedProxyManager) getCostOptimizedProxy() (*AdvancedProxyInstance, error) {
	candidates := apm.getAvailableAdvancedProxies()
	if len(candidates) == 0 {
		return nil, fmt.Errorf("no available proxies for cost-optimized routing")
	}

	// Check budget constraints
	if apm.advancedConfig.CostOptimization != nil && apm.advancedConfig.CostOptimization.Enabled {
		if apm.costTracker.currentCost >= apm.costTracker.budget {
			// Only use free proxies if budget exceeded
			candidates = apm.filterFreeProxies(candidates)
		}
	}

	if len(candidates) == 0 {
		return nil, fmt.Errorf("budget exceeded, no free proxies available")
	}

	// Sort by cost-performance ratio
	sort.Slice(candidates, func(i, j int) bool {
		ratioI := apm.calculateCostPerformanceRatio(candidates[i])
		ratioJ := apm.calculateCostPerformanceRatio(candidates[j])
		return ratioI < ratioJ // Lower ratio is better (cheaper for same performance)
	})

	return candidates[0], nil
}

// getMLPredictiveProxy uses performance-based heuristics to select optimal proxy
// Note: Despite the name, this uses performance metrics rather than machine learning
func (apm *AdvancedProxyManager) getMLPredictiveProxy(targetURL string) (*AdvancedProxyInstance, error) {
	if !apm.mlPredictor.enabled {
		// Fall back to performance-based selection
		return apm.getPerformanceProxy()
	}

	candidates := apm.getAvailableAdvancedProxies()
	if len(candidates) == 0 {
		return nil, fmt.Errorf("no available proxies for ML predictive routing")
	}

	// Get ML predictions for each candidate
	bestProxy, err := apm.mlPredictor.PredictBestProxy(candidates, targetURL)
	if err != nil {
		rotationLogger.Warn(fmt.Sprintf("ML prediction failed: %v, falling back to performance-based selection", err))
		return apm.getPerformanceProxy()
	}

	return bestProxy, nil
}

// Helper methods

func (apm *AdvancedProxyManager) isProxyAvailable(proxy *AdvancedProxyInstance) bool {
	proxy.mu.RLock()
	defer proxy.mu.RUnlock()
	
	available := proxy.Status.Available && 
		proxy.Status.FailureCount < apm.advancedConfig.FailureThreshold
	
	// Check circuit breaker
	if proxy.circuitBreaker != nil {
		available = available && proxy.circuitBreaker.CanExecute()
	}
	
	// Check concurrent connection limit
	if proxy.Advanced.MaxConcurrent > 0 {
		available = available && proxy.CurrentConnections < proxy.Advanced.MaxConcurrent
	}
	
	return available
}

func (apm *AdvancedProxyManager) getAvailableAdvancedProxies() []*AdvancedProxyInstance {
	var available []*AdvancedProxyInstance
	for _, proxy := range apm.advancedProviders {
		if apm.isProxyAvailable(proxy) {
			available = append(available, proxy)
		}
	}
	return available
}

func (apm *AdvancedProxyManager) calculateDistance(loc1, loc2 *GeographicLocation) float64 {
	if loc1 == nil || loc2 == nil || loc1.Latitude == 0 || loc2.Latitude == 0 {
		return math.MaxFloat64
	}

	// Haversine formula for calculating distance between two points on Earth
	const earthRadius = 6371 // km

	lat1Rad := loc1.Latitude * math.Pi / 180
	lat2Rad := loc2.Latitude * math.Pi / 180
	deltaLat := (loc2.Latitude - loc1.Latitude) * math.Pi / 180
	deltaLng := (loc2.Longitude - loc1.Longitude) * math.Pi / 180

	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(deltaLng/2)*math.Sin(deltaLng/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadius * c
}

func (apm *AdvancedProxyManager) calculatePerformanceScore(proxy *AdvancedProxyInstance) float64 {
	if proxy.Performance == nil {
		return 0
	}

	// Weighted performance score (0-100)
	latencyScore := math.Max(0, 100-(float64(proxy.Performance.AverageLatency.Milliseconds())/10))
	successScore := proxy.Performance.SuccessRate
	throughputScore := math.Min(100, proxy.Performance.Throughput*10)
	qualityScore := proxy.Performance.QualityScore
	reliabilityScore := proxy.Performance.ReliabilityScore

	// Weighted average
	weights := map[string]float64{
		"latency":     0.25,
		"success":     0.25,
		"throughput":  0.15,
		"quality":     0.20,
		"reliability": 0.15,
	}

	return latencyScore*weights["latency"] +
		successScore*weights["success"] +
		throughputScore*weights["throughput"] +
		qualityScore*weights["quality"] +
		reliabilityScore*weights["reliability"]
}

func (apm *AdvancedProxyManager) calculateCostPerformanceRatio(proxy *AdvancedProxyInstance) float64 {
	if proxy.Performance == nil || proxy.Performance.Cost == 0 {
		return 0
	}

	performanceScore := apm.calculatePerformanceScore(proxy)
	if performanceScore == 0 {
		return math.MaxFloat64
	}

	return proxy.Performance.Cost / performanceScore
}

func (apm *AdvancedProxyManager) selectBestCandidate(candidates []*AdvancedProxyInstance) *AdvancedProxyInstance {
	if len(candidates) == 0 {
		return nil
	}
	if len(candidates) == 1 {
		return candidates[0]
	}

	// Sort by performance score
	sort.Slice(candidates, func(i, j int) bool {
		scoreI := apm.calculatePerformanceScore(candidates[i])
		scoreJ := apm.calculatePerformanceScore(candidates[j])
		return scoreI > scoreJ
	})

	return candidates[0]
}

func (apm *AdvancedProxyManager) filterByPerformanceThresholds(candidates []*AdvancedProxyInstance) []*AdvancedProxyInstance {
	thresholds := apm.advancedConfig.PerformanceThresholds
	if thresholds == nil {
		return candidates
	}

	var filtered []*AdvancedProxyInstance
	for _, proxy := range candidates {
		if proxy.Performance == nil {
			continue
		}

		meets := true
		if thresholds.MaxLatency > 0 && proxy.Performance.AverageLatency > thresholds.MaxLatency {
			meets = false
		}
		if thresholds.MinSuccessRate > 0 && proxy.Performance.SuccessRate < thresholds.MinSuccessRate {
			meets = false
		}
		if thresholds.MinThroughput > 0 && proxy.Performance.Throughput < thresholds.MinThroughput {
			meets = false
		}
		if thresholds.MinQualityScore > 0 && proxy.Performance.QualityScore < thresholds.MinQualityScore {
			meets = false
		}
		if thresholds.MaxErrorRate > 0 && proxy.Performance.ErrorRate > thresholds.MaxErrorRate {
			meets = false
		}
		if thresholds.MaxTimeoutRate > 0 && proxy.Performance.TimeoutRate > thresholds.MaxTimeoutRate {
			meets = false
		}
		if thresholds.MinDataQuality > 0 && proxy.Performance.DataQuality < thresholds.MinDataQuality {
			meets = false
		}

		if meets {
			filtered = append(filtered, proxy)
		}
	}

	return filtered
}

func (apm *AdvancedProxyManager) getAvailableProxiesInGroup(group *ProxyGroup) []*AdvancedProxyInstance {
	var available []*AdvancedProxyInstance
	for _, proxyName := range group.ProxyNames {
		for _, proxy := range apm.advancedProviders {
			if proxy.Provider.Name == proxyName && apm.isProxyAvailable(proxy) {
				available = append(available, proxy)
				break
			}
		}
	}
	return available
}

func (apm *AdvancedProxyManager) getProxiesInCompatibleTimezones(targetTimezone string) []*AdvancedProxyInstance {
	var compatible []*AdvancedProxyInstance
	
	// Simple timezone compatibility: same timezone or ±3 hours
	for _, proxy := range apm.advancedProviders {
		if !apm.isProxyAvailable(proxy) {
			continue
		}
		
		if proxy.Advanced.Location == nil || proxy.Advanced.Location.TimeZone == "" {
			continue
		}
		
		// For simplicity, consider timezones compatible if they're the same
		// In a real implementation, you'd parse timezone offsets and check overlap
		if proxy.Advanced.Location.TimeZone == targetTimezone {
			compatible = append(compatible, proxy)
		}
	}
	
	return compatible
}

func (apm *AdvancedProxyManager) filterFreeProxies(candidates []*AdvancedProxyInstance) []*AdvancedProxyInstance {
	var free []*AdvancedProxyInstance
	for _, proxy := range candidates {
		if proxy.Performance != nil && proxy.Performance.Cost == 0 {
			free = append(free, proxy)
		}
	}
	return free
}

func (apm *AdvancedProxyManager) initializeAdvancedProxies() {
	// Convert basic proxies to advanced proxies
	for _, basicProxy := range apm.ProxyManager.proxies {
		advancedProxy := &AdvancedProxyInstance{
			ProxyInstance: basicProxy,
			Advanced: &AdvancedProxyProvider{
				ProxyProvider: basicProxy.Provider,
				Performance: &PerformanceMetrics{
					LastMeasured: time.Now(),
				},
			},
			Performance: &PerformanceMetrics{
				LastMeasured: time.Now(),
			},
			circuitBreaker: &CircuitBreaker{
				threshold: 5,
				timeout:   time.Minute * 5,
				state:     "closed",
			},
		}
		apm.advancedProviders = append(apm.advancedProviders, advancedProxy)
	}

	// Add advanced providers from config
	for _, advProvider := range apm.advancedConfig.AdvancedProviders {
		// Convert to basic proxy instance first
		basicProxy := &ProxyInstance{
			Provider: advProvider.ProxyProvider,
			Status: ProxyStatus{
				Available:   true,
				LastChecked: time.Now(),
			},
		}

		advancedProxy := &AdvancedProxyInstance{
			ProxyInstance: basicProxy,
			Advanced:      &advProvider,
			Performance: &PerformanceMetrics{
				LastMeasured: time.Now(),
			},
			circuitBreaker: &CircuitBreaker{
				threshold: 5,
				timeout:   time.Minute * 5,
				state:     "closed",
			},
		}
		apm.advancedProviders = append(apm.advancedProviders, advancedProxy)
	}
}

func (apm *AdvancedProxyManager) initializeGroups() {
	for _, group := range apm.advancedConfig.Groups {
		apm.groups[group.Name] = &group
		
		// Assign proxies to group
		for _, proxyName := range group.ProxyNames {
			for _, proxy := range apm.advancedProviders {
				if proxy.Provider.Name == proxyName {
					proxy.GroupMembership = append(proxy.GroupMembership, group.Name)
					group.Proxies = append(group.Proxies, proxy.ProxyInstance)
					break
				}
			}
		}
	}
}

// ReportAdvancedSuccess reports successful usage with performance metrics
func (apm *AdvancedProxyManager) ReportAdvancedSuccess(proxy *AdvancedProxyInstance, latency time.Duration, dataQuality float64) {
	if proxy == nil {
		return
	}

	// Update basic success
	apm.ProxyManager.ReportSuccess(proxy.ProxyInstance)
	
	// Update advanced metrics
	proxy.mu.Lock()
	if proxy.Performance != nil {
		proxy.Performance.SuccessRate = apm.updateSuccessRate(proxy.Performance.SuccessRate, true)
		proxy.Performance.AverageLatency = apm.updateAverageLatency(proxy.Performance.AverageLatency, latency)
		proxy.Performance.DataQuality = apm.updateDataQuality(proxy.Performance.DataQuality, dataQuality)
		proxy.Performance.LastMeasured = time.Now()
		proxy.Performance.SampleSize++
	}
	proxy.CurrentConnections--
	proxy.LastUsed = time.Now()
	proxy.mu.Unlock()

	// Update performance tracker
	apm.performanceTracker.UpdateMetrics(proxy.Provider.Name, latency, true, dataQuality)
	
	// Update cost tracker
	if proxy.Performance != nil && proxy.Performance.Cost > 0 {
		apm.costTracker.RecordUsage(proxy.Provider.Name, proxy.Performance.Cost)
	}
	
	// Update circuit breaker
	if proxy.circuitBreaker != nil {
		proxy.circuitBreaker.OnSuccess()
	}
}

// ReportAdvancedFailure reports failed usage with detailed error information
func (apm *AdvancedProxyManager) ReportAdvancedFailure(proxy *AdvancedProxyInstance, err error, errorType string) {
	if proxy == nil {
		return
	}

	// Update basic failure
	apm.ProxyManager.ReportFailure(proxy.ProxyInstance, err)
	
	// Update advanced metrics
	proxy.mu.Lock()
	if proxy.Performance != nil {
		proxy.Performance.SuccessRate = apm.updateSuccessRate(proxy.Performance.SuccessRate, false)
		
		switch errorType {
		case "timeout":
			proxy.Performance.TimeoutRate = apm.updateErrorRate(proxy.Performance.TimeoutRate, true)
		default:
			proxy.Performance.ErrorRate = apm.updateErrorRate(proxy.Performance.ErrorRate, true)
		}
		
		proxy.Performance.LastMeasured = time.Now()
		proxy.Performance.SampleSize++
	}
	proxy.CurrentConnections--
	proxy.mu.Unlock()

	// Update performance tracker
	apm.performanceTracker.UpdateMetrics(proxy.Provider.Name, 0, false, 0)
	
	// Update circuit breaker
	if proxy.circuitBreaker != nil {
		proxy.circuitBreaker.OnFailure()
	}
}

// Helper methods for metric updates
func (apm *AdvancedProxyManager) updateSuccessRate(currentRate float64, success bool) float64 {
	// Exponential moving average: new_value = alpha * new_data + (1 - alpha) * old_value
	// alpha = 0.1 means 10% weight to new data, 90% weight to historical data
	alpha := 0.1
	var newDataPoint float64
	if success {
		newDataPoint = 100.0 // 100% success for this data point
	} else {
		newDataPoint = 0.0 // 0% success for this data point
	}
	return alpha*newDataPoint + (1-alpha)*currentRate
}

func (apm *AdvancedProxyManager) updateAverageLatency(currentLatency, newLatency time.Duration) time.Duration {
	// Exponential moving average: new_value = alpha * new_data + (1 - alpha) * old_value
	// alpha = 0.1 means 10% weight to new data, 90% weight to historical data
	alpha := 0.1
	return time.Duration(alpha*float64(newLatency) + (1-alpha)*float64(currentLatency))
}

func (apm *AdvancedProxyManager) updateDataQuality(currentQuality, newQuality float64) float64 {
	// Exponential moving average: new_value = alpha * new_data + (1 - alpha) * old_value
	// alpha = 0.1 means 10% weight to new data, 90% weight to historical data
	alpha := 0.1
	return alpha*newQuality + (1-alpha)*currentQuality
}

func (apm *AdvancedProxyManager) updateErrorRate(currentRate float64, isError bool) float64 {
	// Exponential moving average: new_value = alpha * new_data + (1 - alpha) * old_value
	// alpha = 0.1 means 10% weight to new data, 90% weight to historical data
	alpha := 0.1
	var newDataPoint float64
	if isError {
		newDataPoint = 100.0 // 100% error for this data point
	} else {
		newDataPoint = 0.0 // 0% error for this data point
	}
	return alpha*newDataPoint + (1-alpha)*currentRate
}

// Circuit Breaker implementation
func (cb *CircuitBreaker) CanExecute() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	
	switch cb.state {
	case "closed":
		return true
	case "open":
		if time.Since(cb.lastFailure) > cb.timeout {
			cb.mu.RUnlock()
			cb.mu.Lock()
			cb.state = "half_open"
			cb.mu.Unlock()
			cb.mu.RLock()
			return true
		}
		return false
	case "half_open":
		return true
	default:
		return false
	}
}

func (cb *CircuitBreaker) OnSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	
	cb.failureCount = 0
	cb.state = "closed"
}

func (cb *CircuitBreaker) OnFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	
	cb.failureCount++
	cb.lastFailure = time.Now()
	
	if cb.failureCount >= cb.threshold {
		cb.state = "open"
	}
}

// Placeholder implementations for components
func NewPerformanceTracker() *PerformanceTracker {
	return &PerformanceTracker{
		metrics: make(map[string]*PerformanceMetrics),
	}
}

func (pt *PerformanceTracker) UpdateMetrics(proxyName string, latency time.Duration, success bool, dataQuality float64) {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	
	if _, exists := pt.metrics[proxyName]; !exists {
		pt.metrics[proxyName] = &PerformanceMetrics{
			LastMeasured: time.Now(),
		}
	}
	
	// Update metrics (simplified implementation)
	metrics := pt.metrics[proxyName]
	if success {
		metrics.SuccessRate = (metrics.SuccessRate + 100) / 2
	} else {
		metrics.SuccessRate = metrics.SuccessRate / 2
	}
	metrics.AverageLatency = latency
	metrics.DataQuality = dataQuality
	metrics.LastMeasured = time.Now()
}

func NewGeographicResolver() *GeographicResolver {
	return &GeographicResolver{
		cache: make(map[string]*GeographicLocation),
	}
}

func (gr *GeographicResolver) ResolveLocation(urlStr string) (*GeographicLocation, error) {
	// Parse URL using standard library for proper hostname extraction
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL %s: %w", urlStr, err)
	}
	
	hostname := parsedURL.Hostname()
	if hostname == "" {
		return nil, fmt.Errorf("could not extract hostname from URL: %s", urlStr)
	}
	
	gr.cacheMu.RLock()
	if location, exists := gr.cache[hostname]; exists {
		gr.cacheMu.RUnlock()
		return location, nil
	}
	gr.cacheMu.RUnlock()
	
	// Basic geolocation implementation using hostname patterns and public IP ranges
	// For production use, consider integrating MaxMind GeoIP or similar service
	location := gr.resolveLocationBasic(hostname)
	
	// If hostname-based detection fails, try IP-based detection
	if location.Country == "Unknown" {
		ips, err := net.LookupIP(hostname)
		if err != nil {
			rotationLogger.Debug(fmt.Sprintf("Failed to resolve IP for %s: %v", hostname, err))
		} else if len(ips) > 0 {
			for _, ip := range ips {
				if ipLocation := gr.resolveIPLocation(ip); ipLocation.Country != "Unknown" {
					location = ipLocation
					break
				}
			}
		}
	}
	
	gr.cacheMu.Lock()
	gr.cache[hostname] = location
	gr.cacheMu.Unlock()
	
	return location, nil
}

// resolveLocationBasic performs basic hostname-based geolocation
func (gr *GeographicResolver) resolveLocationBasic(hostname string) *GeographicLocation {
	// Default unknown location
	location := &GeographicLocation{
		Country:   "Unknown",
		Continent: "Unknown",  
		TimeZone:  "UTC",
	}
	
	// Extract domain patterns for common geographical indicators
	hostname = strings.ToLower(hostname)
	
	// Check for country-specific TLDs and patterns
	if strings.HasSuffix(hostname, ".uk") || strings.Contains(hostname, ".co.uk") {
		location.Country = "United Kingdom"
		location.Continent = "Europe"
		location.TimeZone = "Europe/London"
	} else if strings.HasSuffix(hostname, ".de") || strings.Contains(hostname, "germany") {
		location.Country = "Germany" 
		location.Continent = "Europe"
		location.TimeZone = "Europe/Berlin"
	} else if strings.HasSuffix(hostname, ".fr") || strings.Contains(hostname, "france") {
		location.Country = "France"
		location.Continent = "Europe" 
		location.TimeZone = "Europe/Paris"
	} else if strings.HasSuffix(hostname, ".jp") || strings.Contains(hostname, "japan") {
		location.Country = "Japan"
		location.Continent = "Asia"
		location.TimeZone = "Asia/Tokyo"
	} else if strings.HasSuffix(hostname, ".cn") || strings.Contains(hostname, "china") {
		location.Country = "China"
		location.Continent = "Asia"
		location.TimeZone = "Asia/Shanghai"
	} else if strings.HasSuffix(hostname, ".au") || strings.Contains(hostname, "australia") {
		location.Country = "Australia"
		location.Continent = "Oceania"
		location.TimeZone = "Australia/Sydney"
	} else if strings.HasSuffix(hostname, ".ca") || strings.Contains(hostname, "canada") {
		location.Country = "Canada"
		location.Continent = "North America"
		location.TimeZone = "America/Toronto"
	} else if strings.HasSuffix(hostname, ".br") || strings.Contains(hostname, "brazil") {
		location.Country = "Brazil"
		location.Continent = "South America"
		location.TimeZone = "America/Sao_Paulo"
	} else if strings.HasSuffix(hostname, ".com") || strings.HasSuffix(hostname, ".net") || strings.HasSuffix(hostname, ".org") {
		// For generic TLDs, assume US if no other indicators
		location.Country = "United States"
		location.Continent = "North America"
		location.TimeZone = "America/New_York"
	}
	
	return location
}

// resolveIPLocation performs basic IP-based geolocation using public IP ranges
func (gr *GeographicResolver) resolveIPLocation(ip net.IP) *GeographicLocation {
	location := &GeographicLocation{
		Country:   "Unknown",
		Continent: "Unknown",
		TimeZone:  "UTC",
	}
	
	// Skip private/local IPs (IsPrivate() requires Go 1.17+, project uses Go 1.24)
	if isPrivateIP(ip) || ip.IsLoopback() {
		location.Country = "Local"
		location.Continent = "Local"
		return location
	}
	
	// Basic geographic IP range detection (simplified)
	// In production, use MaxMind GeoLite2 or similar database
	ipStr := ip.String()
	
	// Some known public IP ranges for basic detection
	// This is a very limited implementation - real geolocation requires proper databases
	if strings.HasPrefix(ipStr, "8.8.") || strings.HasPrefix(ipStr, "74.125.") {
		// Google's public ranges (mostly US-based)
		location.Country = "United States"
		location.Continent = "North America"
		location.TimeZone = "America/New_York"
	} else if strings.HasPrefix(ipStr, "208.67.") {
		// OpenDNS (US-based)
		location.Country = "United States"
		location.Continent = "North America"
		location.TimeZone = "America/Los_Angeles"
	} else {
		// Default to global for unknown public IPs
		location.Country = "Global"
		location.Continent = "Unknown"
	}
	
	return location
}

// isPrivateIP checks if an IP address is in a private range
func isPrivateIP(ip net.IP) bool {
	return ip.IsPrivate()
}

func NewMLPredictor(config *MLPredictionConfig) *MLPredictor {
	if config == nil {
		return &MLPredictor{enabled: false}
	}
	
	return &MLPredictor{
		enabled:  config.Enabled,
		features: config.Features,
		history:  make([]PredictionDataPoint, 0),
	}
}

func (mlp *MLPredictor) PredictBestProxy(candidates []*AdvancedProxyInstance, targetURL string) (*AdvancedProxyInstance, error) {
	if !mlp.enabled || len(candidates) == 0 {
		return nil, fmt.Errorf("performance predictor not enabled or no candidates")
	}
	
	// Performance-based selection using historical metrics
	// This is a heuristic-based approach, not machine learning
	// TODO: To implement actual ML prediction:
	// 1. Collect training data (proxy performance features vs. success outcomes)
	// 2. Train a regression or classification model (e.g., using scikit-learn via Python integration)
	// 3. Load the trained model and use it for predictions
	// 4. Features could include: latency history, success rate trends, geographic proximity,
	//    time of day patterns, target domain characteristics, etc.
	
	// Current implementation: Select proxy with best performance score
	best := candidates[0]
	bestScore := 0.0
	
	for _, candidate := range candidates {
		if candidate.Performance != nil {
			// Weighted score: success rate (0-100) + latency penalty (lower is better)
			latencyPenalty := float64(candidate.Performance.AverageLatency.Milliseconds()) / 10.0
			if latencyPenalty > 100 {
				latencyPenalty = 100 // Cap penalty at 100 points
			}
			score := candidate.Performance.SuccessRate + (100 - latencyPenalty)
			if score > bestScore {
				bestScore = score
				best = candidate
			}
		}
	}
	
	return best, nil
}

func NewCostTracker(config *CostOptimizationConfig) *CostTracker {
	budget := 0.0
	if config != nil {
		budget = config.BudgetLimit
	}
	
	return &CostTracker{
		usage:  make(map[string]*CostUsage),
		budget: budget,
	}
}

func (ct *CostTracker) RecordUsage(proxyName string, cost float64) {
	ct.usageMu.Lock()
	defer ct.usageMu.Unlock()
	
	if _, exists := ct.usage[proxyName]; !exists {
		ct.usage[proxyName] = &CostUsage{
			ProxyName: proxyName,
			LastReset: time.Now(),
		}
	}
	
	ct.usage[proxyName].RequestCount++
	ct.usage[proxyName].TotalCost += cost
	ct.currentCost += cost
}

func NewLoadBalancer(config *LoadBalancingConfig) *LoadBalancer {
	algorithm := "round_robin"
	if config != nil && config.Algorithm != "" {
		algorithm = config.Algorithm
	}
	
	return &LoadBalancer{
		algorithm:   algorithm,
		connections: make(map[string]int),
	}
}

func (lb *LoadBalancer) SelectProxy(candidates []*AdvancedProxyInstance) (*AdvancedProxyInstance, error) {
	if len(candidates) == 0 {
		return nil, fmt.Errorf("no candidates for load balancing")
	}
	
	switch lb.algorithm {
	case "least_connections":
		return lb.selectLeastConnections(candidates), nil
	case "weighted_round_robin":
		return lb.selectWeightedRoundRobin(candidates), nil
	default: // round_robin
		return candidates[mathrand.Intn(len(candidates))], nil
	}
}

func (lb *LoadBalancer) selectLeastConnections(candidates []*AdvancedProxyInstance) *AdvancedProxyInstance {
	lb.connectionsMu.RLock()
	defer lb.connectionsMu.RUnlock()
	
	best := candidates[0]
	minConnections := lb.connections[best.Provider.Name]
	
	for _, candidate := range candidates[1:] {
		connections := lb.connections[candidate.Provider.Name]
		if connections < minConnections {
			minConnections = connections
			best = candidate
		}
	}
	
	return best
}

func (lb *LoadBalancer) selectWeightedRoundRobin(candidates []*AdvancedProxyInstance) *AdvancedProxyInstance {
	// Simple implementation - in practice you'd use a more sophisticated algorithm
	totalWeight := 0
	for _, candidate := range candidates {
		weight := candidate.Provider.Weight
		if weight <= 0 {
			weight = 1
		}
		totalWeight += weight
	}
	
	if totalWeight == 0 {
		return candidates[0]
	}
	
	random := mathrand.Intn(totalWeight)
	currentWeight := 0
	
	for _, candidate := range candidates {
		weight := candidate.Provider.Weight
		if weight <= 0 {
			weight = 1
		}
		currentWeight += weight
		if random < currentWeight {
			return candidate
		}
	}
	
	return candidates[0]
}