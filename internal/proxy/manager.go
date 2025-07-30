// internal/proxy/manager.go
package proxy

import (
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"sort"
	"sync"
	"time"
)

// ProxyManager implements the Manager interface
type ProxyManager struct {
	config       *ProxyConfig
	proxies      []*ProxyInstance
	currentIndex int
	mu           sync.RWMutex
	stats        ManagerStats
	healthTicker *time.Ticker
	stopChan     chan struct{}
	client       *http.Client
}

// NewProxyManager creates a new proxy manager
func NewProxyManager(config *ProxyConfig) *ProxyManager {
	if config == nil {
		config = &ProxyConfig{
			Enabled:          false,
			Rotation:         RotationRoundRobin,
			HealthCheck:      false,
			HealthCheckRate:  5 * time.Minute,
			Timeout:          30 * time.Second,
			MaxRetries:       3,
			RetryDelay:       1 * time.Second,
			FailureThreshold: 5,
			RecoveryTime:     10 * time.Minute,
		}
	}

	// Create HTTP client with configurable TLS settings
	tlsConfig, err := BuildTLSConfig(config.TLS)
	if err != nil {
		// Fall back to default secure configuration
		fmt.Printf("Warning: Failed to build TLS config, using defaults: %v\n", err)
		tlsConfig = GetDefaultTLSConfig()
	}
	
	transport := &http.Transport{
		TLSClientConfig: tlsConfig,
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   config.Timeout,
	}

	manager := &ProxyManager{
		config:   config,
		proxies:  make([]*ProxyInstance, 0),
		client:   client,
		stopChan: make(chan struct{}),
		stats: ManagerStats{
			ProxyStats: make(map[string]*ProxyInstanceStat),
		},
	}

	// Initialize proxies from configuration
	if err := manager.initializeProxies(); err != nil {
		// Log error but don't fail - manager can still work without proxies
		fmt.Printf("Warning: Failed to initialize proxies: %v\n", err)
	}

	return manager
}

// initializeProxies creates proxy instances from configuration
func (pm *ProxyManager) initializeProxies() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.proxies = make([]*ProxyInstance, 0, len(pm.config.Providers))

	for _, provider := range pm.config.Providers {
		if !provider.Enabled {
			continue
		}

		proxyURL, err := pm.buildProxyURL(&provider)
		if err != nil {
			return fmt.Errorf("failed to build proxy URL for %s: %v", provider.Name, err)
		}

		instance := &ProxyInstance{
			Provider: provider,
			URL:      proxyURL,
			Status: ProxyStatus{
				Available:   true,
				LastChecked: time.Now(),
			},
		}

		pm.proxies = append(pm.proxies, instance)
		pm.stats.ProxyStats[provider.Name] = &ProxyInstanceStat{
			Name:    provider.Name,
			URL:     proxyURL.String(),
			Healthy: true,
		}
	}

	pm.stats.TotalProxies = len(pm.proxies)
	pm.stats.HealthyProxies = len(pm.proxies)

	return nil
}

// buildProxyURL constructs a proxy URL from provider configuration
func (pm *ProxyManager) buildProxyURL(provider *ProxyProvider) (*url.URL, error) {
	var scheme string
	switch provider.Type {
	case ProxyTypeHTTP:
		scheme = "http"
	case ProxyTypeHTTPS:
		scheme = "https"
	case ProxyTypeSOCKS5:
		scheme = "socks5"
	default:
		return nil, fmt.Errorf("unsupported proxy type: %s", provider.Type)
	}

	proxyURL := &url.URL{
		Scheme: scheme,
		Host:   fmt.Sprintf("%s:%d", provider.Host, provider.Port),
	}

	// Add authentication if provided
	if provider.Username != "" && provider.Password != "" {
		proxyURL.User = url.UserPassword(provider.Username, provider.Password)
	} else if pm.config.Authentication != nil {
		proxyURL.User = url.UserPassword(pm.config.Authentication.Username, pm.config.Authentication.Password)
	}

	return proxyURL, nil
}

// GetProxy returns the next proxy according to rotation strategy
func (pm *ProxyManager) GetProxy() (*ProxyInstance, error) {
	if !pm.config.Enabled || len(pm.proxies) == 0 {
		return nil, nil
	}

	pm.mu.Lock()
	defer pm.mu.Unlock()

	var proxy *ProxyInstance
	var err error

	switch pm.config.Rotation {
	case RotationRoundRobin:
		proxy, err = pm.getRoundRobinProxy()
	case RotationRandom:
		proxy, err = pm.getRandomProxy()
	case RotationWeighted:
		proxy, err = pm.getWeightedProxy()
	case RotationHealthy:
		proxy, err = pm.getHealthyProxy()
	default:
		proxy, err = pm.getRoundRobinProxy()
	}

	if err != nil {
		return nil, err
	}

	if proxy != nil {
		proxy.mu.Lock()
		proxy.Status.UseCount++
		pm.stats.ProxyStats[proxy.Provider.Name].UseCount++
		pm.stats.ProxyStats[proxy.Provider.Name].LastUsed = time.Now()
		proxy.mu.Unlock()
		pm.stats.TotalRequests++
	}

	return proxy, nil
}

// getRoundRobinProxy returns the next proxy in round-robin order
func (pm *ProxyManager) getRoundRobinProxy() (*ProxyInstance, error) {
	if len(pm.proxies) == 0 {
		return nil, fmt.Errorf("no proxies available")
	}

	// Find next available proxy
	startIndex := pm.currentIndex
	for i := 0; i < len(pm.proxies); i++ {
		index := (startIndex + i) % len(pm.proxies)
		proxy := pm.proxies[index]
		
		proxy.mu.RLock()
		available := proxy.Status.Available && proxy.Status.FailureCount < pm.config.FailureThreshold
		proxy.mu.RUnlock()

		if available {
			pm.currentIndex = (index + 1) % len(pm.proxies)
			return proxy, nil
		}
	}

	return nil, fmt.Errorf("no healthy proxies available")
}

// getRandomProxy returns a random available proxy
func (pm *ProxyManager) getRandomProxy() (*ProxyInstance, error) {
	availableProxies := pm.getAvailableProxies()
	if len(availableProxies) == 0 {
		return nil, fmt.Errorf("no healthy proxies available")
	}

	index := rand.Intn(len(availableProxies))
	return availableProxies[index], nil
}

// getWeightedProxy returns a proxy based on weighted selection
func (pm *ProxyManager) getWeightedProxy() (*ProxyInstance, error) {
	availableProxies := pm.getAvailableProxies()
	if len(availableProxies) == 0 {
		return nil, fmt.Errorf("no healthy proxies available")
	}

	// Calculate total weight
	totalWeight := 0
	for _, proxy := range availableProxies {
		weight := proxy.Provider.Weight
		if weight <= 0 {
			weight = 1
		}
		totalWeight += weight
	}

	if totalWeight == 0 {
		return availableProxies[0], nil
	}

	// Select proxy based on weight
	random := rand.Intn(totalWeight)
	currentWeight := 0

	for _, proxy := range availableProxies {
		weight := proxy.Provider.Weight
		if weight <= 0 {
			weight = 1
		}
		currentWeight += weight
		if random < currentWeight {
			return proxy, nil
		}
	}

	return availableProxies[0], nil
}

// getHealthyProxy returns the healthiest proxy (lowest response time)
func (pm *ProxyManager) getHealthyProxy() (*ProxyInstance, error) {
	availableProxies := pm.getAvailableProxies()
	if len(availableProxies) == 0 {
		return nil, fmt.Errorf("no healthy proxies available")
	}

	// Sort by response time (ascending)
	sort.Slice(availableProxies, func(i, j int) bool {
		availableProxies[i].mu.RLock()
		availableProxies[j].mu.RLock()
		defer availableProxies[i].mu.RUnlock()
		defer availableProxies[j].mu.RUnlock()
		
		return availableProxies[i].Status.ResponseTime < availableProxies[j].Status.ResponseTime
	})

	return availableProxies[0], nil
}

// getAvailableProxies returns list of available proxies
func (pm *ProxyManager) getAvailableProxies() []*ProxyInstance {
	var available []*ProxyInstance
	
	for _, proxy := range pm.proxies {
		proxy.mu.RLock()
		isAvailable := proxy.Status.Available && proxy.Status.FailureCount < pm.config.FailureThreshold
		proxy.mu.RUnlock()
		
		// Check if proxy is in recovery period
		if !isAvailable && time.Since(proxy.Status.LastFailure) > pm.config.RecoveryTime {
			proxy.mu.Lock()
			proxy.Status.Available = true
			proxy.Status.FailureCount = 0
			proxy.mu.Unlock()
			isAvailable = true
		}

		if isAvailable {
			available = append(available, proxy)
		}
	}

	return available
}

// ReportSuccess reports successful usage of a proxy
func (pm *ProxyManager) ReportSuccess(proxy *ProxyInstance) {
	if proxy == nil {
		return
	}

	proxy.mu.Lock()
	proxy.Status.LastSuccess = time.Now()
	proxy.Status.Available = true
	proxy.mu.Unlock()

	pm.mu.Lock()
	if stat, exists := pm.stats.ProxyStats[proxy.Provider.Name]; exists {
		stat.SuccessCount++
		stat.SuccessRate = float64(stat.SuccessCount) / float64(stat.SuccessCount+stat.FailureCount) * 100
	}
	pm.mu.Unlock()
}

// ReportFailure reports failed usage of a proxy
func (pm *ProxyManager) ReportFailure(proxy *ProxyInstance, err error) {
	if proxy == nil {
		return
	}

	proxy.mu.Lock()
	proxy.Status.FailureCount++
	proxy.Status.LastFailure = time.Now()
	
	// Mark proxy as unavailable if failure threshold exceeded
	if proxy.Status.FailureCount >= pm.config.FailureThreshold {
		proxy.Status.Available = false
	}
	proxy.mu.Unlock()

	pm.mu.Lock()
	if stat, exists := pm.stats.ProxyStats[proxy.Provider.Name]; exists {
		stat.FailureCount++
		stat.SuccessRate = float64(stat.SuccessCount) / float64(stat.SuccessCount+stat.FailureCount) * 100
		stat.Healthy = proxy.Status.Available
	}
	pm.mu.Unlock()
}

// GetStats returns proxy usage statistics
func (pm *ProxyManager) GetStats() ManagerStats {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	// Update healthy proxy count
	healthyCount := 0
	for _, proxy := range pm.proxies {
		proxy.mu.RLock()
		if proxy.Status.Available {
			healthyCount++
		}
		proxy.mu.RUnlock()
	}

	pm.stats.HealthyProxies = healthyCount
	pm.stats.FailedProxies = pm.stats.TotalProxies - healthyCount

	// Calculate overall success rate
	totalSuccess := int64(0)
	totalFailure := int64(0)
	totalResponse := time.Duration(0)
	validResponses := 0

	for _, stat := range pm.stats.ProxyStats {
		totalSuccess += stat.SuccessCount
		totalFailure += stat.FailureCount
		if stat.ResponseTime > 0 {
			totalResponse += stat.ResponseTime
			validResponses++
		}
	}

	if totalSuccess+totalFailure > 0 {
		pm.stats.SuccessRate = float64(totalSuccess) / float64(totalSuccess+totalFailure) * 100
	}

	if validResponses > 0 {
		pm.stats.AverageResponse = totalResponse / time.Duration(validResponses)
	}

	return pm.stats
}

// IsEnabled returns whether proxy rotation is enabled
func (pm *ProxyManager) IsEnabled() bool {
	if !pm.config.Enabled {
		return false
	}

	pm.mu.RLock()
	defer pm.mu.RUnlock()

	for _, proxy := range pm.proxies {
		proxy.mu.RLock()
		if proxy.Status.Available {
			proxy.mu.RUnlock()
			return true
		}
		proxy.mu.RUnlock()
	}

	return false
}

// GetHealthyProxies returns list of healthy proxies
func (pm *ProxyManager) GetHealthyProxies() []*ProxyInstance {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.getAvailableProxies()
}

// Start starts the proxy manager
func (pm *ProxyManager) Start() error {
	if pm.config.HealthCheck && pm.config.HealthCheckRate > 0 {
		pm.healthTicker = time.NewTicker(pm.config.HealthCheckRate)
		go pm.healthCheckLoop()
	}
	return nil
}

// Stop stops the proxy manager
func (pm *ProxyManager) Stop() error {
	if pm.healthTicker != nil {
		pm.healthTicker.Stop()
	}
	close(pm.stopChan)
	return nil
}

// healthCheckLoop runs periodic health checks
func (pm *ProxyManager) healthCheckLoop() {
	for {
		select {
		case <-pm.healthTicker.C:
			pm.HealthCheck()
		case <-pm.stopChan:
			return
		}
	}
}

// HealthCheck performs health checks on all proxies
func (pm *ProxyManager) HealthCheck() error {
	if !pm.config.HealthCheck {
		return nil
	}

	pm.mu.Lock()
	pm.stats.LastHealthCheck = time.Now()
	pm.mu.Unlock()

	checkURL := pm.config.HealthCheckURL
	if checkURL == "" {
		checkURL = "http://httpbin.org/ip"
	}

	var wg sync.WaitGroup
	for _, proxy := range pm.proxies {
		wg.Add(1)
		go func(p *ProxyInstance) {
			defer wg.Done()
			
			start := time.Now()
			err := pm.checkProxyHealth(p, checkURL)
			duration := time.Since(start)

			p.mu.Lock()
			p.Status.LastChecked = time.Now()
			p.Status.ResponseTime = duration
			
			if err != nil {
				p.Status.FailureCount++
				if p.Status.FailureCount >= pm.config.FailureThreshold {
					p.Status.Available = false
				}
			} else {
				p.Status.Available = true
				p.Status.FailureCount = 0
			}
			p.mu.Unlock()

			pm.mu.Lock()
			if stat, exists := pm.stats.ProxyStats[p.Provider.Name]; exists {
				stat.ResponseTime = duration
				stat.Healthy = p.Status.Available
			}
			pm.mu.Unlock()
		}(proxy)
	}

	wg.Wait()
	return nil
}

// checkProxyHealth checks if a single proxy is healthy
func (pm *ProxyManager) checkProxyHealth(proxy *ProxyInstance, url string) error {
	// Build TLS config
	tlsConfig, err := BuildTLSConfig(pm.config.TLS)
	if err != nil {
		tlsConfig = GetDefaultTLSConfig()
	}

	// Create a client with the proxy
	proxyURL := proxy.URL
	transport := &http.Transport{
		Proxy:           http.ProxyURL(proxyURL),
		TLSClientConfig: tlsConfig,
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   pm.config.Timeout,
	}

	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check failed with status %d", resp.StatusCode)
	}

	return nil
}

// RefreshProxies refreshes the proxy list
func (pm *ProxyManager) RefreshProxies() error {
	return pm.initializeProxies()
}