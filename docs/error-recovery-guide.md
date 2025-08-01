# Comprehensive Error Recovery Guide

DataScrapexter provides sophisticated error recovery mechanisms including circuit breakers, fallback strategies, intelligent retry logic, and memory-efficient health tracking. This guide explains how to configure and use these features effectively.

## Table of Contents
1. [Overview](#overview)
2. [Circuit Breakers](#circuit-breakers)
3. [Fallback Strategies](#fallback-strategies)
4. [Intelligent Retry Logic](#intelligent-retry-logic)
5. [Memory-Efficient Health Tracking](#memory-efficient-health-tracking)
6. [Configuration Reference](#configuration-reference)
7. [Best Practices](#best-practices)
8. [Troubleshooting](#troubleshooting)

## Overview

The error recovery system provides multiple layers of protection:

- **Circuit Breakers**: Prevent cascading failures by temporarily stopping requests to failing services
- **Fallback Strategies**: Provide alternative responses when operations fail
- **Intelligent Retry Logic**: Automatically retry failed operations with exponential backoff
- **Health Tracking**: Monitor operation health with memory-efficient sliding windows
- **Adaptive Rate Limiting**: Automatically adjust request rates based on error patterns

## Circuit Breakers

Circuit breakers implement the circuit breaker pattern to prevent cascading failures. They have three states:

### States
- **Closed**: Normal operation, requests pass through
- **Open**: Failures exceeded threshold, requests are blocked
- **Half-Open**: Testing if service has recovered

### Configuration

```yaml
error_recovery:
  enabled: true
  circuit_breakers:
    fetch_document:
      max_failures: 5                 # Open after 5 consecutive failures
      reset_timeout: "60s"            # Try again after 60 seconds

    extract_field:
      max_failures: 3                 # More sensitive for critical operations
      reset_timeout: "30s"
```

### Operation-Specific Circuit Breakers

Different operations can have different circuit breaker configurations:

- `fetch_document`: For HTTP document fetching
- `extract_field`: For individual field extraction
- `parse_response`: For response parsing
- Custom operation names for specific use cases

## Fallback Strategies

When operations fail, fallback strategies provide alternative responses:

### Strategy Types

#### 1. Cached Fallback
Returns previously cached successful results:

```yaml
fallbacks:
  fetch_document:
    strategy: "cached"
    cache_timeout: "5m"               # Cache valid for 5 minutes
```

#### 2. Default Value Fallback
Returns a configured default value:

```yaml
fallbacks:
  product_price:
    strategy: "default"
    default_value: 0.0
```

#### 3. Alternative Fallback
Uses an alternative method or endpoint:

```yaml
fallbacks:
  fetch_document:
    strategy: "alternative"
    alternative: "mobile_version"
```

#### 4. Degraded Service Fallback
Returns a degraded response indicating partial failure:

```yaml
fallbacks:
  product_reviews:
    strategy: "degrade"
    degraded:
      message: "Reviews temporarily unavailable"
      count: 0
      available: false
```

## Intelligent Retry Logic

The system automatically retries failed operations with intelligent backoff:

### Retryable Errors
- Network timeouts
- Connection refused
- DNS resolution failures
- HTTP 5xx errors
- HTTP 429 (rate limit) errors
- Temporary service unavailable
- Custom error patterns

### Exponential Backoff

```go
// Default retry configuration
RetryConfig{
    MaxRetries:    3,
    BaseDelay:     2 * time.Second,
    BackoffFactor: 2.0,
    MaxDelay:      5 * time.Minute,
}
```

Retry delays: 2s → 4s → 8s → stop

### Context Cancellation

All retry operations respect context cancellation for graceful shutdowns.

## Memory-Efficient Health Tracking

The system tracks operation health with optimized memory usage:

### Features
- **Sliding Window**: Tracks errors within a configurable time window
- **Memory Protection**: Limits maximum tracked errors to prevent memory exhaustion
- **Periodic Cleanup**: Efficient cleanup every N operations instead of every error
- **In-Place Filtering**: Avoids allocations during cleanup

### Memory Protection Constants

```go
const (
    MaxHealthErrors       = 1000  // Maximum errors to track
    HealthCleanupInterval = 100   // Cleanup frequency
)
```

### Performance Characteristics
- **Memory Usage**: Bounded by MaxHealthErrors
- **Cleanup Efficiency**: O(n) in-place filtering
- **CPU Impact**: Minimal overhead with periodic cleanup

## Configuration Reference

### Error Recovery Configuration

```yaml
error_recovery:
  enabled: true                       # Enable error recovery system

  circuit_breakers:
    operation_name:
      max_failures: 5                 # Failures before opening circuit
      reset_timeout: "60s"            # Time before trying again

  fallbacks:
    operation_name:
      strategy: "cached"              # Strategy: cached, default, alternative, degrade
      cache_timeout: "5m"             # Cache validity (for cached strategy)
      default_value: null             # Default return value
      alternative: "alt_method"       # Alternative method name
      degraded:                       # Degraded response object
        status: "degraded"
        message: "Service unavailable"
```

### Integration with Rate Limiting

Error recovery works seamlessly with adaptive rate limiting:

```yaml
rate_limiter:
  base_interval: "1s"
  strategy: 3                         # Hybrid strategy
  adaptation_threshold: "1s"          # Adaptation frequency
  error_rate_threshold: 0.1           # Error rate triggering adaptation
  consecutive_err_limit: 5            # Consecutive errors threshold
  min_change_threshold: 0.1           # Minimum change percentage
```

## Best Practices

### 1. Configure Circuit Breakers by Operation Criticality

```yaml
circuit_breakers:
  fetch_document:
    max_failures: 2                   # Critical operation - fail fast
    reset_timeout: "60s"

  extract_reviews:
    max_failures: 10                  # Optional data - more tolerant
    reset_timeout: "5m"
```

### 2. Use Appropriate Fallback Strategies

- **Critical Data**: Use cached fallbacks with reasonable timeouts
- **Optional Data**: Use degraded service responses
- **Numeric Values**: Use default values (0, null, etc.)
- **Lists/Arrays**: Use empty arrays with status messages

### 3. Monitor Circuit Breaker States

```go
stats := engine.GetErrorRecoveryStats()
circuitBreakers := stats["circuit_breakers"]
```

### 4. Cache Management

- Clear cache periodically for data freshness
- Use appropriate cache timeouts for different data types
- Monitor cache hit rates

### 5. Balance Retry and Circuit Breaker Settings

- Short retry intervals with circuit breakers for fast failure detection
- Longer circuit breaker timeouts for service recovery
- Consider upstream service characteristics

## Troubleshooting

### Common Issues

#### 1. Circuit Breaker Opens Too Frequently

**Symptoms**: Operations fail with "circuit breaker is open" errors
**Solutions**:
- Increase `max_failures` threshold
- Increase `reset_timeout` for more recovery time
- Check if upstream service is actually failing
- Review rate limiting configuration

#### 2. Fallbacks Not Working

**Symptoms**: Operations fail without using configured fallbacks
**Solutions**:
- Verify `error_recovery.enabled: true`
- Check operation names match between circuit breakers and fallbacks
- Ensure fallback strategy is correctly configured
- Check cache timeout settings for cached strategy

#### 3. Memory Usage Growing

**Symptoms**: Memory usage increases over time
**Solutions**:
- Verify health error cleanup is working
- Check `MaxHealthErrors` constant
- Monitor health tracking statistics
- Consider reducing health window duration

#### 4. Poor Performance

**Symptoms**: Slow response times, high CPU usage
**Solutions**:
- Adjust `HealthCleanupInterval` for less frequent cleanup
- Reduce health window duration
- Check if circuit breaker timeouts are appropriate
- Monitor retry attempt counts

### Debugging Commands

#### View Circuit Breaker Statistics

```go
engine := scraper.NewEngine(config)
stats := engine.GetErrorRecoveryStats()
fmt.Printf("Circuit Breakers: %+v\n", stats["circuit_breakers"])
```

#### View Cache Statistics

```go
cacheStats := stats["cache"]
fmt.Printf("Cache Entries: %d\n", cacheStats["total_entries"])
```

#### Reset Error Recovery

```go
engine.ResetErrorRecovery()  // Clear all circuit breakers and cache
```

### Monitoring Metrics

Track these metrics in production:

- Circuit breaker state changes
- Fallback usage rates
- Cache hit/miss rates
- Error recovery success rates
- Operation attempt counts
- Memory usage of health tracking

## Advanced Use Cases

### 1. API Rate Limit Handling

```yaml
error_recovery:
  enabled: true
  circuit_breakers:
    api_request:
      max_failures: 1                 # Fail fast on rate limits
      reset_timeout: "300s"           # 5 minute cool-down
  fallbacks:
    api_request:
      strategy: "cached"
      cache_timeout: "10m"            # Use cached data during rate limits
```

### 2. Multi-Source Data Aggregation

```yaml
error_recovery:
  enabled: true
  fallbacks:
    primary_source:
      strategy: "alternative"
      alternative: "secondary_source"

    secondary_source:
      strategy: "cached"
      cache_timeout: "1h"

    optional_data:
      strategy: "degrade"
      degraded:
        available: false
        message: "Data source unavailable"
```

### 3. Progressive Degradation

```yaml
error_recovery:
  enabled: true
  fallbacks:
    high_res_images:
      strategy: "alternative"
      alternative: "low_res_images"

    low_res_images:
      strategy: "default"
      default_value: "/images/placeholder.jpg"

    detailed_specs:
      strategy: "degrade"
      degraded:
        specs: {}
        message: "Detailed specifications unavailable"
```

## Performance Optimization

### Memory Usage

The error recovery system is designed for efficiency:

- **Bounded Memory**: Health tracking uses fixed-size limits
- **Periodic Cleanup**: Reduces CPU overhead
- **In-Place Operations**: Minimizes allocations
- **Smart Caching**: Configurable timeouts prevent unbounded growth

### CPU Usage

- Circuit breaker state checks are O(1)
- Health cleanup is O(n) but infrequent
- Fallback strategies have minimal overhead
- Cache lookups are O(1) hash map operations

### Network Efficiency

- Circuit breakers prevent unnecessary network calls
- Cached fallbacks reduce upstream load
- Intelligent retry logic respects service capacity
- Integration with adaptive rate limiting

## Integration Examples

### With Browser Automation

```yaml
browser:
  enabled: true
  timeout: "30s"

error_recovery:
  enabled: true
  circuit_breakers:
    browser_fetch:
      max_failures: 2
      reset_timeout: "120s"           # Browser recovery takes longer
  fallbacks:
    browser_fetch:
      strategy: "alternative"
      alternative: "http_fallback"    # Fall back to HTTP-only scraping
```

### With Proxy Rotation

```yaml
proxy:
  enabled: true
  rotation: "random"

error_recovery:
  enabled: true
  circuit_breakers:
    proxy_request:
      max_failures: 3
      reset_timeout: "60s"
  fallbacks:
    proxy_request:
      strategy: "alternative"
      alternative: "direct_connection"
```

This comprehensive error recovery system ensures your scraping operations are resilient, efficient, and maintainable.