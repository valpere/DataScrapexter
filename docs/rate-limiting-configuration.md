# Rate Limiting Configuration Guide

DataScrapexter provides sophisticated rate limiting capabilities with configurable adaptation sensitivity. This document explains the various configuration options and their production-appropriate defaults.

## Configuration Parameters

### Basic Rate Limiting
```yaml
rate_limiter:
  base_interval: "1s"           # Starting request interval
  burst_size: 5                 # Number of burst tokens
  max_interval: "30s"           # Maximum backoff interval
  strategy: 3                   # 0=Fixed, 1=Adaptive, 2=Burst, 3=Hybrid
```

### Adaptation Sensitivity Controls

These parameters control how aggressively the rate limiter adapts to errors:

#### `adaptation_threshold`
- **Default**: `1s` (production)
- **Purpose**: Minimum time between rate adjustments
- **Impact**: Lower values = more frequent adaptations, higher values = more stable rates
- **Recommendation**: Use `1s` for production, `100ms` for testing

```yaml
adaptation_threshold: "1s"    # Don't adapt more than once per second
```

#### `error_rate_threshold`
- **Default**: `0.1` (10% error rate)
- **Purpose**: Error rate that triggers adaptation
- **Impact**: Lower values = more sensitive to errors, higher values = more tolerant
- **Recommendation**: `0.1` for most sites, `0.05` for fragile sites, `0.2` for robust APIs

```yaml
error_rate_threshold: 0.1     # Adapt when >10% of requests fail
```

#### `consecutive_err_limit`
- **Default**: `5`
- **Purpose**: Number of consecutive errors before aggressive backoff
- **Impact**: Lower values = quicker backoff, higher values = more tolerance
- **Recommendation**: `5` for production, `3` for fragile sites, `10` for robust APIs

```yaml
consecutive_err_limit: 5      # Backoff after 5 consecutive errors
```

#### `min_change_threshold`
- **Default**: `0.1` (10% change)
- **Purpose**: Minimum rate change percentage to apply
- **Impact**: Lower values = more frequent small adjustments, higher values = fewer large adjustments
- **Recommendation**: `0.1` for stable operation, `0.05` for fine-tuning

```yaml
min_change_threshold: 0.1     # Only apply changes >10%
```

## Strategy Comparison

### Fixed Strategy (0)
- **Use Case**: Stable, well-known APIs
- **Behavior**: Constant rate, no adaptation
- **Configuration**: Only `base_interval` and `burst_size` used

### Adaptive Strategy (1)
- **Use Case**: Unknown or variable server performance
- **Behavior**: Adjusts rate based on error patterns
- **Configuration**: Uses all adaptation sensitivity controls

### Burst Strategy (2)
- **Use Case**: APIs that handle traffic spikes well
- **Behavior**: Token bucket with refill, no error adaptation
- **Configuration**: Focus on `burst_size` and `burst_refill_rate`

### Hybrid Strategy (3) - Recommended
- **Use Case**: General-purpose scraping
- **Behavior**: Combines burst handling with error adaptation
- **Configuration**: Uses all parameters for optimal balance

## Common Configuration Patterns

### High-Performance APIs
```yaml
rate_limiter:
  base_interval: "100ms"        # Fast base rate
  burst_size: 20                # Large bursts
  strategy: 3                   # Hybrid
  adaptation_threshold: "500ms" # Quick adaptation
  error_rate_threshold: 0.2     # Tolerant of occasional errors
  consecutive_err_limit: 10     # Allow more consecutive errors
  min_change_threshold: 0.15    # Stable rate changes
```

### Fragile/Rate-Limited Sites
```yaml
rate_limiter:
  base_interval: "5s"           # Very conservative
  burst_size: 2                 # Minimal bursts
  strategy: 1                   # Pure adaptive
  adaptation_threshold: "2s"    # Don't overreact
  error_rate_threshold: 0.05    # Very sensitive to errors
  consecutive_err_limit: 3      # Quick backoff
  min_change_threshold: 0.05    # React to small changes
```

### E-commerce Sites
```yaml
rate_limiter:
  base_interval: "2s"           # Moderate pace
  burst_size: 5                 # Reasonable bursts
  strategy: 3                   # Hybrid approach
  adaptation_threshold: "1s"    # Standard adaptation
  error_rate_threshold: 0.1     # Standard sensitivity
  consecutive_err_limit: 5      # Standard tolerance
  min_change_threshold: 0.1     # Standard stability
```

## Production vs Testing Values

The rate limiter automatically applies production-safe defaults when parameters are not specified:

| Parameter | Production Default | Testing Override |
|-----------|-------------------|------------------|
| `adaptation_threshold` | `1s` | `10ms` - `100ms` |
| `error_rate_threshold` | `0.1` (10%) | `0.0` (any error) |
| `consecutive_err_limit` | `5` | `2` - `3` |
| `min_change_threshold` | `0.1` (10%) | `0.0` (any change) |

## Backward Compatibility

Legacy rate limiting configuration is automatically converted:

```yaml
# Legacy format (still supported)
rate_limit: "2s"
burst_size: 5

# Converted to modern format internally with production defaults
rate_limiter:
  base_interval: "2s"
  burst_size: 5
  strategy: 0                   # Fixed strategy
  adaptation_threshold: "1s"    # Production default
  error_rate_threshold: 0.1     # Production default
  consecutive_err_limit: 5      # Production default
  min_change_threshold: 0.1     # Production default
```

## Monitoring and Debugging

Access rate limiter statistics in your application:

```go
engine := NewEngine(config)
stats := engine.GetRateLimiterStats()

fmt.Printf("Current interval: %v\n", stats.CurrentInterval)
fmt.Printf("Error rate: %.2f%%\n", stats.ErrorRate * 100)
fmt.Printf("Consecutive errors: %d\n", stats.ConsecutiveErrs)
fmt.Printf("Burst tokens: %d\n", stats.BurstTokens)
```

## Best Practices

1. **Start Conservative**: Begin with default settings and monitor performance
2. **Monitor Error Rates**: Watch for excessive adaptations that indicate problems
3. **Test Thoroughly**: Use testing overrides to verify adaptation behavior
4. **Document Custom Settings**: Always comment why you deviated from defaults
5. **Regular Review**: Periodically review and adjust based on actual performance

## Troubleshooting

### Too Aggressive Adaptation
**Symptoms**: Frequent rate changes, slow scraping
**Solutions**: 
- Increase `error_rate_threshold` (e.g., 0.15)
- Increase `adaptation_threshold` (e.g., 2s)
- Increase `min_change_threshold` (e.g., 0.2)

### Too Slow Recovery
**Symptoms**: Rate stays slow after errors resolve
**Solutions**:
- Decrease `adaptation_threshold` (e.g., 500ms)
- Decrease `min_change_threshold` (e.g., 0.05)
- Use Hybrid strategy for better recovery

### Inconsistent Performance
**Symptoms**: Erratic scraping speeds
**Solutions**:
- Increase `min_change_threshold` for stability
- Use Fixed strategy for predictable rates
- Review server-side rate limiting policies