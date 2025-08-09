// Test for the configurable max query period feature
package proxy

import (
	"testing"
	"time"
)

func TestMonitoringConfig_MaxQueryPeriod_Default(t *testing.T) {
	// Test that default MaxQueryPeriod is set when not provided
	monitor := NewProxyMonitor(nil, nil)
	
	expected := 7 * 24 * time.Hour // 7 days
	if monitor.config.MaxQueryPeriod != expected {
		t.Errorf("Expected default MaxQueryPeriod to be %v, got %v", 
			expected, monitor.config.MaxQueryPeriod)
	}
}

func TestMonitoringConfig_MaxQueryPeriod_Custom(t *testing.T) {
	// Test custom MaxQueryPeriod configuration
	customPeriod := 3 * 24 * time.Hour // 3 days
	config := &MonitoringConfig{
		Enabled:          true,
		MaxQueryPeriod:   customPeriod,
		HistoryRetention: 24 * time.Hour,
	}
	
	monitor := NewProxyMonitor(config, nil)
	
	if monitor.config.MaxQueryPeriod != customPeriod {
		t.Errorf("Expected custom MaxQueryPeriod to be %v, got %v", 
			customPeriod, monitor.config.MaxQueryPeriod)
	}
}

func TestMonitoringConfig_MaxQueryPeriod_ZeroValue(t *testing.T) {
	// Test that zero MaxQueryPeriod gets set to default
	config := &MonitoringConfig{
		Enabled:          true,
		MaxQueryPeriod:   0, // Should be replaced with default
		HistoryRetention: 24 * time.Hour,
	}
	
	monitor := NewProxyMonitor(config, nil)
	
	expected := 7 * 24 * time.Hour // Should default to 7 days
	if monitor.config.MaxQueryPeriod != expected {
		t.Errorf("Expected zero MaxQueryPeriod to default to %v, got %v", 
			expected, monitor.config.MaxQueryPeriod)
	}
}

func TestMonitoringConfig_MaxQueryPeriod_NegativeValue(t *testing.T) {
	// Test that negative MaxQueryPeriod gets set to default
	config := &MonitoringConfig{
		Enabled:          true,
		MaxQueryPeriod:   -time.Hour, // Should be replaced with default
		HistoryRetention: 24 * time.Hour,
	}
	
	monitor := NewProxyMonitor(config, nil)
	
	expected := 7 * 24 * time.Hour // Should default to 7 days
	if monitor.config.MaxQueryPeriod != expected {
		t.Errorf("Expected negative MaxQueryPeriod to default to %v, got %v", 
			expected, monitor.config.MaxQueryPeriod)
	}
}

func TestMonitoringConfig_MaxQueryPeriod_Validation(t *testing.T) {
	// Test various MaxQueryPeriod configurations
	testCases := []struct {
		name           string
		maxQuery       time.Duration
		historyReten   time.Duration
		expectedQuery  time.Duration
		expectWarning  bool // We can't easily test for warnings, but document expectation
	}{
		{
			name:          "Normal configuration",
			maxQuery:      24 * time.Hour,
			historyReten:  48 * time.Hour,
			expectedQuery: 24 * time.Hour,
			expectWarning: false,
		},
		{
			name:          "MaxQuery equals retention",
			maxQuery:      24 * time.Hour,
			historyReten:  24 * time.Hour,
			expectedQuery: 24 * time.Hour,
			expectWarning: false,
		},
		{
			name:          "MaxQuery slightly larger than retention",
			maxQuery:      36 * time.Hour,
			historyReten:  24 * time.Hour,
			expectedQuery: 36 * time.Hour,
			expectWarning: false,
		},
		{
			name:          "MaxQuery much larger than retention (should warn)",
			maxQuery:      7 * 24 * time.Hour,
			historyReten:  24 * time.Hour,
			expectedQuery: 7 * 24 * time.Hour,
			expectWarning: true, // MaxQuery > HistoryRetention * 2
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := &MonitoringConfig{
				Enabled:          true,
				MaxQueryPeriod:   tc.maxQuery,
				HistoryRetention: tc.historyReten,
			}
			
			monitor := NewProxyMonitor(config, nil)
			
			if monitor.config.MaxQueryPeriod != tc.expectedQuery {
				t.Errorf("Expected MaxQueryPeriod to be %v, got %v", 
					tc.expectedQuery, monitor.config.MaxQueryPeriod)
			}
		})
	}
}

// Test showing configuration usage patterns
func TestMaxQueryPeriodConfigurations(t *testing.T) {
	// Development environment - short retention and query period
	devConfig := &MonitoringConfig{
		Enabled:          true,
		HistoryRetention: 2 * time.Hour,
		MaxQueryPeriod:   6 * time.Hour,
	}
	devMonitor := NewProxyMonitor(devConfig, nil)
	
	if devMonitor.config.MaxQueryPeriod != 6*time.Hour {
		t.Error("Development config not set correctly")
	}
	
	// Production environment - longer retention and query period
	prodConfig := &MonitoringConfig{
		Enabled:          true,
		HistoryRetention: 7 * 24 * time.Hour, // 1 week
		MaxQueryPeriod:   30 * 24 * time.Hour, // 1 month
	}
	prodMonitor := NewProxyMonitor(prodConfig, nil)
	
	if prodMonitor.config.MaxQueryPeriod != 30*24*time.Hour {
		t.Error("Production config not set correctly")
	}
	
	// High-volume production - balanced configuration
	balancedConfig := &MonitoringConfig{
		Enabled:          true,
		HistoryRetention: 3 * 24 * time.Hour, // 3 days
		MaxQueryPeriod:   7 * 24 * time.Hour, // 1 week
	}
	balancedMonitor := NewProxyMonitor(balancedConfig, nil)
	
	if balancedMonitor.config.MaxQueryPeriod != 7*24*time.Hour {
		t.Error("Balanced config not set correctly")
	}
}