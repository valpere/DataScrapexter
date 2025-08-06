# DataScrapexter Troubleshooting Guide

## Overview

This guide provides solutions to common issues encountered when using DataScrapexter, including configuration problems, scraping failures, performance issues, and deployment challenges.

## Table of Contents

1. [Quick Diagnostics](#quick-diagnostics)
2. [Configuration Issues](#configuration-issues)
3. [Scraping Failures](#scraping-failures)
4. [Anti-Detection Problems](#anti-detection-problems)
5. [Performance Issues](#performance-issues)
6. [Output Problems](#output-problems)
7. [Monitoring and Health Issues](#monitoring-and-health-issues)
8. [Deployment Issues](#deployment-issues)
9. [Best Practices for Prevention](#best-practices-for-prevention)

## Quick Diagnostics

### Health Check Commands

```bash
# Basic health check
datascrapexter health

# Detailed system status
datascrapexter health --verbose

# Test specific configuration
datascrapexter validate config.yaml

# Test connectivity
datascrapexter test-connectivity --url "https://example.com"

# Check proxy configuration
datascrapexter test-proxy --config config.yaml

# Test CAPTCHA service
datascrapexter test-captcha --service 2captcha --api-key $API_KEY
```

### Log Analysis

```bash
# Enable verbose logging
datascrapexter run config.yaml --log-level debug

# Output logs to file
datascrapexter run config.yaml --log-file scraper.log

# Filter specific log types
grep "ERROR" scraper.log
grep "CAPTCHA" scraper.log
grep "PROXY" scraper.log
```

### Common Error Patterns

```bash
# Check for common issues
datascrapexter diagnose config.yaml

# System resource check
datascrapexter system-info

# Network connectivity test
datascrapexter network-test --target example.com
```

## Configuration Issues

### Invalid YAML Syntax

**Problem**: Configuration file has YAML syntax errors.

**Symptoms**:

```plaintext
Error: yaml: line 15: mapping values are not allowed in this context
```

**Solutions**:

```bash
# Validate YAML syntax
yamllint config.yaml

# Use datascrapexter validation
datascrapexter validate config.yaml

# Common fixes:
# 1. Check indentation (use spaces, not tabs)
# 2. Quote strings with special characters
# 3. Ensure proper nesting
```

**Example Fix**:

```yaml
# Incorrect
fields:
- name: title
selector: h1
type: text

# Correct
fields:
  - name: "title"
    selector: "h1"
    type: "text"
```

### Selector Not Found

**Problem**: CSS selectors don't match any elements.

**Symptoms**:

```plaintext
WARN: Field 'title' extraction failed: no elements found for selector 'h1.title'
```

**Solutions**:

```bash
# Test selectors against live page
datascrapexter test-selectors config.yaml --url "https://example.com"

# Use browser developer tools to verify selectors
# Try more specific or alternative selectors
```

**Configuration Fix**:

```yaml
# Use multiple fallback selectors
fields:
  - name: "title"
    selector: "h1.product-title, h1.title, h1, .main-title"
    type: "text"
    fallback_selector: ".page-title, .header h1"
```

### Environment Variable Issues

**Problem**: Environment variables not being substituted.

**Symptoms**:

```plaintext
Error: invalid URL: ${TARGET_URL}
```

**Solutions**:

```bash
# Check environment variables are set
echo $TARGET_URL
env | grep TARGET

# Test with explicit values first
# Use default values in configuration
```

**Configuration Fix**:

```yaml
# Use defaults for missing variables
name: "${SCRAPER_NAME:-default_scraper}"
base_url: "${TARGET_URL:-https://example.com}"

# Validate all required variables are set
proxy:
  url: "${PROXY_URL:?PROXY_URL environment variable is required}"
```

### Rate Limiting Configuration

**Problem**: Rate limiting causing timeouts or blocks.

**Symptoms**:

```plaintext
Error: request timeout after 30s
HTTP 429: Too Many Requests
```

**Solutions**:

```yaml
# Increase delays
rate_limit: "5s"
random_delay: "2s"

# Add adaptive rate limiting
anti_detection:
  rate_limiting:
    adaptive: true
    min_delay: "2s"
    max_delay: "15s"
    
# Monitor server response times
monitoring:
  metrics:
    enabled: true
```

## Scraping Failures

### Network Connectivity Issues

**Problem**: Cannot connect to target websites.

**Symptoms**:

```plaintext
Error: Get "https://example.com": dial tcp: no such host
Error: context deadline exceeded
```

**Solutions**:

```bash
# Test basic connectivity
curl -I https://example.com
nslookup example.com

# Check firewall/proxy settings
# Verify DNS resolution
# Test with different networks
```

**Configuration Fixes**:

```yaml
# Increase timeouts
timeout: "60s"
max_retries: 5

# Add custom DNS
headers:
  Host: "example.com"
  
# Use proxy if needed
proxy:
  enabled: true
  url: "http://proxy.example.com:8080"
```

### SSL/TLS Certificate Issues

**Problem**: SSL certificate verification failures.

**Symptoms**:

```plaintext
Error: x509: certificate signed by unknown authority
Error: tls: handshake failure
```

**Solutions**:

```yaml
# Disable certificate verification (not recommended for production)
browser:
  ignore_certificate_errors: true

# Or add custom certificates
browser:
  extra_certificates:
    - "/path/to/custom-cert.pem"

# Use different TLS configuration
anti_detection:
  tls:
    min_tls_version: "1.2"
    verify_certificates: false  # Only for testing
```

### JavaScript-Heavy Sites

**Problem**: Content not loading without JavaScript.

**Symptoms**:

```plaintext
WARN: No data extracted, page might require JavaScript
Empty results from dynamic content
```

**Solutions**:

```yaml
# Enable browser automation
browser:
  enabled: true
  headless: true
  wait_for_element: ".content-loaded"
  wait_timeout: "30s"
  
  # Wait for specific conditions
  wait_strategies:
    page_load: "networkidle2"
    
  # Execute custom JavaScript
  custom_scripts:
    post_navigation:
      - "window.scrollTo(0, document.body.scrollHeight)"
      - "document.querySelector('.load-more')?.click()"
```

### Authentication Required

**Problem**: Sites requiring login or authentication.

**Symptoms**:

```plaintext
HTTP 401: Unauthorized
HTTP 403: Forbidden
Redirected to login page
```

**Solutions**:

```yaml
# Add authentication headers
headers:
  Authorization: "Bearer ${API_TOKEN}"
  Cookie: "session_id=${SESSION_ID}"

# Or use browser automation for login
browser:
  enabled: true
  custom_scripts:
    pre_navigation:
      - "localStorage.setItem('token', '${AUTH_TOKEN}')"
    post_navigation:
      - |
        if (document.querySelector('.login-form')) {
          document.querySelector('[name="username"]').value = '${USERNAME}';
          document.querySelector('[name="password"]').value = '${PASSWORD}';
          document.querySelector('.login-form').submit();
        }
```

## Anti-Detection Problems

### High Detection Rate

**Problem**: Frequently blocked or challenged with CAPTCHAs.

**Symptoms**:

```plaintext
HTTP 403: Forbidden
CAPTCHA challenges appearing frequently
Proxy IPs getting blocked
```

**Solutions**:

```yaml
# Enable comprehensive anti-detection
anti_detection:
  fingerprinting:
    enabled: true
    canvas_spoofing: true
    webgl_spoofing: true
    audio_spoofing: true
    hardware_spoofing: true
    
  # Increase delays and add randomization
  rate_limiting:
    base_delay: "8s"
    random_delay: "5s"
    human_simulation:
      enabled: true
      daily_patterns:
        enabled: true
        
  # Use residential proxies
proxy:
  enabled: true
  rotation: "random"
  providers:
    - url: "${RESIDENTIAL_PROXY}"
      type: "residential"
      country: "US"
```

### CAPTCHA Solving Failures

**Problem**: CAPTCHAs not being solved correctly.

**Symptoms**:

```plaintext
Error: CAPTCHA solving failed after 3 attempts
Error: Invalid CAPTCHA solution
Timeout waiting for CAPTCHA solution
```

**Solutions**:

```yaml
anti_detection:
  captcha:
    # Try different service
    service: "anti-captcha"  # Switch from 2captcha
    
    # Increase timeout and attempts
    timeout: "180s"
    max_attempts: 5
    retry_delay: "15s"
    
    # Enable automatic retry
    auto_retry_on_failure: true
    
    # Improve detection
    detection:
      enabled: true
      selectors:
        recaptcha_v2: ".g-recaptcha, [data-sitekey], iframe[src*='recaptcha']"
        hcaptcha: ".h-captcha, [data-sitekey*='hcaptcha']"
```

**Service-Specific Debugging**:

```bash
# Test 2Captcha service
curl -X POST "http://2captcha.com/in.php" \
  -d "key=$API_KEY&method=userrecaptcha&googlekey=SITE_KEY&pageurl=PAGE_URL"

# Check account balance
curl "http://2captcha.com/res.php?key=$API_KEY&action=getbalance"

# Test with different CAPTCHA service
datascrapexter test-captcha --service anti-captcha --api-key $API_KEY
```

### Proxy Issues

**Problem**: Proxies failing or being detected.

**Symptoms**:

```plaintext
Error: proxy connection failed
HTTP 407: Proxy Authentication Required
All proxies marked as unhealthy
```

**Solutions**:

```yaml
proxy:
  # Enable health checking
  health_check: true
  health_check_interval: "30s"
  health_check_timeout: "10s"
  
  # Reduce concurrent usage
  providers:
    - url: "${PROXY_URL}"
      max_concurrent: 1  # Reduce from higher number
      
  # Use session management
  session_management:
    enabled: true
    session_duration: "1800s"
    
  # Add authentication
  authentication:
    username_password: true
    ip_authentication: false
```

**Debugging Steps**:

```bash
# Test proxy directly
curl --proxy http://user:pass@proxy.com:8080 http://httpbin.org/ip

# Check proxy rotation
datascrapexter test-proxy-rotation --config config.yaml --requests 10

# Monitor proxy health
datascrapexter proxy-health --config config.yaml
```

### TLS Fingerprinting Detection

**Problem**: Requests blocked due to TLS fingerprinting.

**Symptoms**:

```plaintext
Connection reset by peer
Unusual TLS handshake failures
Blocking despite using different IPs
```

**Solutions**:

```yaml
anti_detection:
  tls:
    randomize_ja3: true
    randomize_ja4: true
    
    # Use browser profiles
    profile_mode: "browser_simulation"
    random_selection: true
    
    # Randomize cipher suites
    cipher_suites:
      - "TLS_AES_128_GCM_SHA256"
      - "TLS_AES_256_GCM_SHA384"
      - "TLS_CHACHA20_POLY1305_SHA256"
      
    # Randomize extensions
    extensions:
      randomize_order: true
      include_padding: true
```

## Performance Issues

### Slow Scraping Speed

**Problem**: Scraping taking too long to complete.

**Symptoms**:

```plaintext
Very low pages per second
High response times
Timeouts on slow pages
```

**Solutions**:

```yaml
# Increase concurrency
performance:
  concurrency:
    max_concurrent_requests: 20
    max_concurrent_extractions: 10
    
# Optimize browser settings
browser:
  disable_images: true
  disable_css: true
  disable_javascript: false  # Only if needed
  
# Enable resource blocking
browser:
  resource_interception:
    enabled: true
    block_types: ["image", "stylesheet", "font", "media"]
    
# Reduce timeouts for faster failures
timeout: "15s"
rate_limit: "1s"
```

### High Memory Usage

**Problem**: Memory consumption growing over time.

**Symptoms**:

```plaintext
Out of memory errors
Increasing RSS memory usage
System becoming unresponsive
```

**Solutions**:

```yaml
# Enable streaming for large datasets
output:
  json:
    streaming: true
    buffer_size: 1000
    
performance:
  memory_management:
    max_memory_usage: "2GB"
    gc_target_percentage: 70
    
# Process in smaller batches
pagination:
  max_pages: 50  # Process in smaller chunks

# Monitor memory usage
monitoring:
  metrics:
    enabled: true
    track_memory: true
```

**System-Level Solutions**:

```bash
# Monitor memory usage
htop
ps aux | grep datascrapexter

# Increase system limits
ulimit -m 2097152  # 2GB memory limit

# Use swap if needed
sudo swapon -a
```

### Network Performance Issues

**Problem**: Network bottlenecks affecting performance.

**Symptoms**:

```plaintext
High response times
Connection timeouts
DNS resolution delays
```

**Solutions**:

```yaml
# Optimize connection pooling
performance:
  connection_pooling:
    max_idle_connections: 200
    max_connections_per_host: 50
    idle_connection_timeout: "90s"
    tcp_keep_alive: "30s"
    
# Use connection reuse
performance:
  request_optimization:
    keep_alive: true
    connection_reuse: true
    
# Configure DNS
headers:
  Connection: "keep-alive"
```

## Output Problems

### File Permission Issues

**Problem**: Cannot write output files.

**Symptoms**:

```plaintext
Error: permission denied: output.json
Error: no such file or directory
```

**Solutions**:

```bash
# Check directory permissions
ls -la /path/to/output/
mkdir -p /path/to/output/
chmod 755 /path/to/output/

# Use absolute paths
# Ensure directory exists before running
```

**Configuration Fixes**:

```yaml
# Use environment variables for paths
output:
  file: "${OUTPUT_DIR}/results_${TIMESTAMP}.json"
  
# Ensure directory creation
output:
  create_directories: true
  file_permissions: "0644"
  directory_permissions: "0755"
```

### Database Connection Issues

**Problem**: Cannot connect to database.

**Symptoms**:

```plaintext
Error: failed to connect to database
Error: authentication failed
Connection timeout
```

**Solutions**:

```yaml
output:
  database:
    # Increase timeouts
    connection_timeout: "60s"
    
    # Add retry logic
    max_retries: 3
    retry_delay: "5s"
    
    # Use connection pooling
    max_connections: 20
    max_idle_connections: 5
    
    # SSL configuration
    ssl_mode: "require"
    ssl_cert: "${DB_SSL_CERT}"
```

**Debugging Steps**:

```bash
# Test database connection
psql -h localhost -U scraper -d scraping_data -c "SELECT 1;"

# Check connection string
datascrapexter test-database --config config.yaml

# Monitor connection pool
datascrapexter database-stats --config config.yaml
```

### Cloud Storage Upload Failures

**Problem**: Failed to upload files to cloud storage.

**Symptoms**:

```plaintext
Error: AWS credentials not found
Error: access denied to S3 bucket
Upload timeout
```

**Solutions**:

```yaml
output:
  cloud_storage:
    # Increase timeouts
    upload_timeout: "300s"
    
    # Configure retry logic
    max_retries: 3
    retry_delay: "10s"
    
    # Use multipart upload for large files
    multipart_upload: true
    chunk_size: "10MB"
    
    # Authentication
    credentials_file: "${AWS_CREDENTIALS_FILE}"
    use_iam_role: true
```

**AWS Debugging**:

```bash
# Test AWS credentials
aws sts get-caller-identity

# Test S3 access
aws s3 ls s3://your-bucket/

# Check IAM permissions
aws iam get-user
```

## Monitoring and Health Issues

### Metrics Not Available

**Problem**: Prometheus metrics not being exported.

**Symptoms**:

```plaintext
404 Not Found on /metrics endpoint
Empty metrics response
Prometheus not scraping
```

**Solutions**:

```yaml
monitoring:
  metrics:
    enabled: true
    listen_address: ":9090"
    metrics_path: "/metrics"
    
    # Enable specific metric types
    enable_go_metrics: true
    enable_process_metrics: true
```

**Debugging Steps**:

```bash
# Test metrics endpoint
curl http://localhost:9090/metrics

# Check if port is open
netstat -tlnp | grep 9090

# Test Prometheus configuration
promtool check config prometheus.yml
```

### Health Checks Failing

**Problem**: Health check endpoints returning unhealthy status.

**Symptoms**:

```plaintext
/health returns 503 Service Unavailable
Kubernetes pods being restarted
Load balancer removing instances
```

**Solutions**:

```yaml
monitoring:
  health:
    # Adjust check intervals
    check_interval: "60s"
    
    # Configure timeouts
    default_timeout: "30s"
    
    # Enable detailed responses
    detailed_response: true
    
    # Configure thresholds
    checks:
      memory:
        threshold: 90.0  # Increase from 80%
      disk_space:
        threshold: 95.0  # Increase threshold
```

### Dashboard Not Loading

**Problem**: Monitoring dashboard not accessible.

**Symptoms**:

```plaintext
Connection refused on dashboard port
Dashboard shows no data
404 errors on dashboard resources
```

**Solutions**:

```yaml
monitoring:
  dashboard:
    enabled: true
    port: ":8080"
    
    # Add authentication if needed
    authentication:
      enabled: false
      
    # Configure CORS
    cors:
      enabled: true
      allowed_origins: ["*"]
```

## Deployment Issues

### Docker Container Problems

**Problem**: Container failing to start or run.

**Symptoms**:

```plaintext
Container exits immediately
Permission denied in container
Volume mount issues
```

**Solutions**:

```dockerfile
# Use proper user permissions
FROM golang:1.24-alpine AS builder
RUN adduser -D -s /bin/sh scraper

# Set proper working directory
WORKDIR /app
USER scraper

# Copy with proper permissions
COPY --chown=scraper:scraper . .
```

**Docker Compose Fixes**:

```yaml
version: '3.8'
services:
  datascrapexter:
    image: valpere/datascrapexter:latest
    volumes:
      - ./configs:/app/configs:ro
      - ./output:/app/output:rw
    environment:
      - CONFIG_PATH=/app/configs/scraper.yaml
    user: "1000:1000"
    restart: unless-stopped
```

### Kubernetes Deployment Issues

**Problem**: Pods not starting or being killed.

**Symptoms**:

```plaintext
ImagePullBackOff
CrashLoopBackOff
Pod being OOMKilled
```

**Solutions**:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: datascrapexter
spec:
  template:
    spec:
      containers:
      - name: scraper
        image: valpere/datascrapexter:latest
        resources:
          limits:
            memory: "2Gi"
            cpu: "1000m"
          requests:
            memory: "512Mi"
            cpu: "250m"
        
        # Proper health checks
        livenessProbe:
          httpGet:
            path: /live
            port: 8080
          initialDelaySeconds: 60
          periodSeconds: 30
          
        readinessProbe:
          httpGet:
            path: /ready
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 5
```

### Environment Variable Issues

**Problem**: Environment variables not being passed correctly.

**Symptoms**:

```plaintext
Configuration using literal ${VAR} instead of values
Missing required environment variables
Default values not working
```

**Solutions**:

```bash
# Check environment variables in container
kubectl exec -it pod-name -- env | grep SCRAPER

# Use ConfigMaps for non-sensitive data
kubectl create configmap scraper-config --from-env-file=config.env

# Use Secrets for sensitive data
kubectl create secret generic scraper-secrets \
  --from-literal=api-key=your-secret-key
```

**Kubernetes ConfigMap**:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: scraper-config
data:
  SCRAPER_NAME: "production_scraper"
  RATE_LIMIT: "2s"
  OUTPUT_FORMAT: "json"
---
apiVersion: v1
kind: Secret
metadata:
  name: scraper-secrets
type: Opaque
data:
  CAPTCHA_API_KEY: base64-encoded-key
  DB_PASSWORD: base64-encoded-password
```

## Best Practices for Prevention

### Configuration Management

1. **Version Control**: Keep configurations in version control
2. **Environment Separation**: Use different configs for dev/staging/prod
3. **Validation**: Always validate configurations before deployment
4. **Documentation**: Document custom selectors and transformations

### Monitoring and Alerting

1. **Health Checks**: Implement comprehensive health monitoring
2. **Metrics**: Track key performance indicators
3. **Logging**: Use structured logging with appropriate levels
4. **Alerting**: Set up alerts for critical failures

### Testing

1. **Configuration Testing**: Test configurations against live sites
2. **Load Testing**: Test performance under expected load
3. **Failover Testing**: Test failure scenarios and recovery
4. **Regression Testing**: Test after site structure changes

### Security

1. **Credential Management**: Use secure credential storage
2. **Network Security**: Implement proper network controls
3. **Access Control**: Limit access to scraping systems
4. **Audit Logging**: Log all scraping activities

### Performance

1. **Resource Monitoring**: Monitor CPU, memory, and network usage
2. **Capacity Planning**: Plan for growth and peak loads
3. **Optimization**: Regularly review and optimize configurations
4. **Scaling**: Implement horizontal scaling when needed

### Compliance

1. **Legal Review**: Ensure compliance with applicable laws
2. **Rate Limiting**: Implement respectful rate limiting
3. **robots.txt**: Respect robots.txt files
4. **Data Privacy**: Handle personal data appropriately

This troubleshooting guide should help resolve most issues encountered with DataScrapexter. For additional support, check the GitHub issues or community forums.
