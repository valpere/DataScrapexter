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
	TLS              *TLSConfig       `yaml:"tls,omitempty" json:"tls,omitempty"`
}

// TLSConfig defines TLS/SSL configuration for proxy connections
type TLSConfig struct {
	// InsecureSkipVerify controls whether the proxy manager skips verification of server certificates.
	// 
	// ⚠️  SECURITY WARNING: Setting this to true is DANGEROUS and makes connections vulnerable to attacks.
	// ⚠️  This bypasses ALL certificate verification including hostname validation.
	// ⚠️  NEVER use this in production environments or with untrusted networks.
	// ⚠️  Only use for testing with self-signed certificates in controlled environments.
	InsecureSkipVerify bool `yaml:"insecure_skip_verify" json:"insecure_skip_verify"`
	// ServerName is used to verify the hostname on the returned certificates unless InsecureSkipVerify is true.
	// It is also included in the client's handshake to support virtual hosting.
	ServerName string `yaml:"server_name,omitempty" json:"server_name,omitempty"`
	
	// RootCAs defines the set of root certificate authorities that clients use when verifying server certificates.
	// If RootCAs is nil, TLS uses the host's root CA set.
	RootCAs []string `yaml:"root_cas,omitempty" json:"root_cas,omitempty"`
	
	// ClientCert and ClientKey define the client certificate and key for mutual TLS authentication.
	ClientCert string `yaml:"client_cert,omitempty" json:"client_cert,omitempty"`
	ClientKey  string `yaml:"client_key,omitempty" json:"client_key,omitempty"`
	
	// SuppressWarnings controls whether security warnings are logged when insecure settings are used.
	// This can be useful in production environments where warnings might clutter logs.
	SuppressWarnings bool `yaml:"suppress_warnings,omitempty" json:"suppress_warnings,omitempty"`
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