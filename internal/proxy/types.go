// internal/proxy/types.go
package proxy

import (
	"net/url"
	"sync"
	"time"
)

// ProxyType represents the type of proxy
type ProxyType string

const (
	ProxyTypeHTTP   ProxyType = "http"
	ProxyTypeHTTPS  ProxyType = "https"
	ProxyTypeSOCKS5 ProxyType = "socks5"
)

// RotationStrategy defines how proxies are rotated
type RotationStrategy string

const (
	RotationRoundRobin RotationStrategy = "round_robin"
	RotationRandom     RotationStrategy = "random"
	RotationWeighted   RotationStrategy = "weighted"
	RotationHealthy    RotationStrategy = "healthy"
)

// ProxyConfig defines proxy configuration
type ProxyConfig struct {
	Enabled          bool             `yaml:"enabled" json:"enabled"`
	Rotation         RotationStrategy `yaml:"rotation" json:"rotation"`
	HealthCheck      bool             `yaml:"health_check" json:"health_check"`
	HealthCheckURL   string           `yaml:"health_check_url,omitempty" json:"health_check_url,omitempty"`
	HealthCheckRate  time.Duration    `yaml:"health_check_rate" json:"health_check_rate"`
	Timeout          time.Duration    `yaml:"timeout" json:"timeout"`
	MaxRetries       int              `yaml:"max_retries" json:"max_retries"`
	RetryDelay       time.Duration    `yaml:"retry_delay" json:"retry_delay"`
	Providers        []ProxyProvider  `yaml:"providers" json:"providers"`
	Authentication   *ProxyAuth       `yaml:"authentication,omitempty" json:"authentication,omitempty"`
	FailureThreshold int              `yaml:"failure_threshold" json:"failure_threshold"`
	RecoveryTime     time.Duration    `yaml:"recovery_time" json:"recovery_time"`
}

// ProxyProvider represents a proxy provider configuration
type ProxyProvider struct {
	Name      string    `yaml:"name" json:"name"`
	Type      ProxyType `yaml:"type" json:"type"`
	Host      string    `yaml:"host" json:"host"`
	Port      int       `yaml:"port" json:"port"`
	Username  string    `yaml:"username,omitempty" json:"username,omitempty"`
	Password  string    `yaml:"password,omitempty" json:"password,omitempty"`
	Weight    int       `yaml:"weight,omitempty" json:"weight,omitempty"`
	Enabled   bool      `yaml:"enabled" json:"enabled"`
	Whitelist []string  `yaml:"whitelist,omitempty" json:"whitelist,omitempty"`
	Blacklist []string  `yaml:"blacklist,omitempty" json:"blacklist,omitempty"`
}

// ProxyAuth represents proxy authentication configuration
type ProxyAuth struct {
	Username string `yaml:"username" json:"username"`
	Password string `yaml:"password" json:"password"`
}

// ProxyStatus represents the current status of a proxy
type ProxyStatus struct {
	Available    bool          `json:"available"`
	ResponseTime time.Duration `json:"response_time"`
	LastChecked  time.Time     `json:"last_checked"`
	FailureCount int           `json:"failure_count"`
	LastFailure  time.Time     `json:"last_failure,omitempty"`
	LastSuccess  time.Time     `json:"last_success,omitempty"`
	UseCount     int64         `json:"use_count"`
}

// ProxyInstance represents a runtime proxy instance
type ProxyInstance struct {
	Provider ProxyProvider `json:"provider"`
	URL      *url.URL      `json:"url"`
	Status   ProxyStatus   `json:"status"`
	mu       sync.RWMutex  `json:"-"`
}

// Manager defines the proxy management interface
type Manager interface {
	// GetProxy returns the next proxy according to rotation strategy
	GetProxy() (*ProxyInstance, error)
	
	// ReportSuccess reports successful usage of a proxy
	ReportSuccess(proxy *ProxyInstance)
	
	// ReportFailure reports failed usage of a proxy
	ReportFailure(proxy *ProxyInstance, err error)
	
	// GetStats returns proxy usage statistics
	GetStats() ManagerStats
	
	// HealthCheck performs health checks on all proxies
	HealthCheck() error
	
	// Start starts the proxy manager
	Start() error
	
	// Stop stops the proxy manager
	Stop() error
	
	// IsEnabled returns whether proxy rotation is enabled
	IsEnabled() bool
	
	// GetHealthyProxies returns list of healthy proxies
	GetHealthyProxies() []*ProxyInstance
	
	// RefreshProxies refreshes the proxy list
	RefreshProxies() error
}

// ManagerStats represents proxy manager statistics
type ManagerStats struct {
	TotalProxies    int                           `json:"total_proxies"`
	HealthyProxies  int                           `json:"healthy_proxies"`
	FailedProxies   int                           `json:"failed_proxies"`
	TotalRequests   int64                         `json:"total_requests"`
	SuccessRate     float64                       `json:"success_rate"`
	AverageResponse time.Duration                 `json:"average_response"`
	ProxyStats      map[string]*ProxyInstanceStat `json:"proxy_stats"`
	LastHealthCheck time.Time                     `json:"last_health_check"`
}

// ProxyInstanceStat represents statistics for a single proxy instance
type ProxyInstanceStat struct {
	Name         string        `json:"name"`
	URL          string        `json:"url"`
	Healthy      bool          `json:"healthy"`
	UseCount     int64         `json:"use_count"`
	SuccessCount int64         `json:"success_count"`
	FailureCount int64         `json:"failure_count"`
	SuccessRate  float64       `json:"success_rate"`
	ResponseTime time.Duration `json:"response_time"`
	LastUsed     time.Time     `json:"last_used"`
}

// HealthChecker defines interface for proxy health checking
type HealthChecker interface {
	Check(proxy *ProxyInstance) error
	GetHealthCheckURL() string
	SetHealthCheckURL(url string)
}