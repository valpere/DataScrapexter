# DataScrapexter Troubleshooting Guide

## Overview

This troubleshooting guide addresses common issues encountered when using DataScrapexter and provides systematic approaches to diagnose and resolve problems. Whether you're dealing with configuration errors, extraction failures, or performance issues, this guide offers practical solutions and debugging strategies to get your scrapers running smoothly.

Web scraping inherently involves interacting with dynamic, changing systems, making troubleshooting an essential skill. This guide is organized by problem categories, with each section providing symptoms, causes, and step-by-step solutions. Understanding these patterns helps you quickly identify and resolve issues, minimizing downtime and maintaining reliable data collection.

## Configuration Issues

### Symptom: "Configuration validation failed"

Configuration validation failures are among the most common issues, typically occurring when YAML syntax is incorrect or required fields are missing. These errors prevent DataScrapexter from starting the scraping process.

**Common Causes:**
- Incorrect YAML indentation (YAML requires consistent spacing)
- Missing required fields like name or base_url
- Typographical errors in field names
- Invalid selector syntax
- Mismatched quotes or special characters

**Solution Steps:**

First, validate the YAML syntax using an online YAML validator or command-line tool:

```bash
# Using Python's yaml module
python -c "import yaml; yaml.safe_load(open('config.yaml'))"

# Using DataScrapexter's validation
datascrapexter validate --strict config.yaml
```

Check for common YAML pitfalls:
- Ensure consistent indentation (use spaces, not tabs)
- Verify all strings with special characters are properly quoted
- Confirm lists are properly formatted with hyphens
- Check that colons are followed by spaces

Example of correct formatting:

```yaml
# Correct
fields:
  - name: "price"
    selector: ".product-price"
    type: "text"

# Incorrect - missing space after colon
fields:
  - name:"price"  # Error!
    selector:".product-price"  # Error!
```

### Symptom: "Selector not finding any elements"

When selectors fail to match any elements, scraped data fields return empty or null values. This often indicates that the website structure differs from your configuration expectations.

**Common Causes:**
- Website structure has changed since configuration was created
- Dynamic content loaded by JavaScript not present in initial HTML
- Incorrect selector syntax or typos
- Case-sensitive class names not matched correctly
- Content inside iframes or shadow DOM

**Solution Steps:**

Use browser developer tools to verify selectors:

1. Open the target website in Chrome/Firefox
2. Right-click on the element you want to extract
3. Select "Inspect" to open developer tools
4. Right-click the element in the HTML view
5. Choose "Copy > Copy selector" for a precise selector

Test selectors in the browser console:

```javascript
// Test CSS selector
document.querySelectorAll('.your-selector').length

// View matched elements
document.querySelectorAll('.your-selector')
```

Improve selector robustness:

```yaml
# Too specific - fragile
selector: "body > div:nth-child(2) > div > div.content > h1"

# Better - more resilient to structure changes
selector: ".content h1, article h1, [role='heading']"
```

Consider multiple fallback selectors:

```yaml
fields:
  - name: "title"
    selector: "h1.product-title, h1.item-name, .product-header h1, h1"
    type: "text"
```

## Network and Connection Issues

### Symptom: "Connection timeout" or "Network error"

Network errors prevent DataScrapexter from reaching the target website, resulting in failed scraping attempts. These issues can be temporary or indicate persistent connectivity problems.

**Common Causes:**
- Target website is temporarily down or experiencing high load
- Network connectivity issues on your system
- Firewall or security software blocking connections
- Proxy configuration errors
- DNS resolution failures

**Solution Steps:**

Verify basic connectivity:

```bash
# Test DNS resolution
nslookup example.com
dig example.com

# Test HTTP connectivity
curl -I https://example.com
wget --spider https://example.com

# Test with DataScrapexter's user agent
curl -H "User-Agent: Mozilla/5.0" https://example.com
```

Increase timeout values in configuration:

```yaml
timeout: "60s"  # Increase from default 30s
max_retries: 5   # Increase retry attempts
```

Check proxy settings if applicable:

```bash
# Test proxy connectivity
curl -x http://proxy:8080 https://example.com

# Set proxy environment variables
export HTTP_PROXY=http://proxy:8080
export HTTPS_PROXY=http://proxy:8080
```

Debug DNS issues:

```bash
# Use specific DNS servers
echo "nameserver 8.8.8.8" | sudo tee /etc/resolv.conf
echo "nameserver 1.1.1.1" | sudo tee -a /etc/resolv.conf
```

### Symptom: "429 Too Many Requests" or rate limiting errors

Rate limiting errors occur when scraping too aggressively, triggering the website's protective mechanisms. These errors are increasingly common as websites implement sophisticated anti-bot measures.

**Common Causes:**
- Request rate exceeds website limits
- Multiple scrapers running simultaneously
- Insufficient delay between requests
- IP address flagged for excessive requests

**Solution Steps:**

Adjust rate limiting configuration:

```yaml
# Conservative rate limiting
rate_limit: "5s"        # 5 seconds between requests
max_retries: 3          # Retry with backoff
timeout: "45s"          # Allow for slow responses
```

Implement progressive delays:

```yaml
# Start slow and monitor response
rate_limit: "10s"       # Very conservative start

# Gradually decrease if successful
rate_limit: "5s"        # After testing
rate_limit: "3s"        # Final optimized value
```

Use proxy rotation for distributed requests:

```yaml
proxy:
  enabled: true
  rotation: "random"
  list:
    - "http://proxy1:8080"
    - "http://proxy2:8080"
    - "http://proxy3:8080"
```

Monitor rate limit headers:

```bash
# Check response headers for rate limit information
datascrapexter --log-level debug run config.yaml 2>&1 | grep -i "rate"
```

## Data Extraction Problems

### Symptom: "Extracted data is empty or incorrect"

When data extraction succeeds but returns unexpected values, the issue typically lies in selector precision or transformation logic. This is particularly frustrating as the scraper appears to work but produces unusable data.

**Common Causes:**
- Selector matching wrong elements
- Multiple elements matching selector
- Hidden or duplicate content in HTML
- Incorrect transformation rules
- Character encoding issues

**Solution Steps:**

Debug selector matching:

```yaml
# Add debug logging
datascrapexter --log-level debug run config.yaml 2>&1 | grep "selector"
```

Test with simplified extraction:

```yaml
# Temporarily remove transformations
fields:
  - name: "raw_price"
    selector: ".price"
    type: "text"
    # transform: commented out for testing
```

Handle multiple matches explicitly:

```yaml
# For first match only
fields:
  - name: "first_price"
    selector: ".price:first-of-type"
    type: "text"

# For all matches
fields:
  - name: "all_prices"
    selector: ".price"
    type: "list"
```

Fix character encoding issues:

```yaml
# Ensure proper encoding handling
headers:
  Accept-Charset: "utf-8"
  Accept: "text/html; charset=utf-8"
```

### Symptom: "Transformation errors"

Transformation failures occur when extracted data doesn't match expected formats, causing parsing or conversion errors. These issues often appear in logs as "transformation failed" messages.

**Common Causes:**
- Unexpected data format
- Regex patterns not matching
- Type conversion failures
- Null or empty values
- Special characters breaking patterns

**Solution Steps:**

Test transformations incrementally:

```yaml
# Add transformations one at a time
transform:
  - type: "trim"
  # - type: "clean_price"    # Add after trim works
  # - type: "parse_float"    # Add after clean_price works
```

Debug regex patterns:

```yaml
# Use online regex testers with sample data
transform:
  - type: "regex"
    pattern: "\\$([0-9,]+\\.?[0-9]*)"  # Test this pattern
    replacement: "$1"
```

Handle edge cases:

```yaml
# Add defensive transformations
transform:
  - type: "trim"
  - type: "regex"
    pattern: "^$"              # Check for empty
    replacement: "0"           # Default value
  - type: "clean_price"
  - type: "parse_float"
```

## Performance Issues

### Symptom: "Scraping is very slow"

Slow scraping performance impacts productivity and may indicate inefficient configuration or resource constraints. Performance issues compound when scraping large websites.

**Common Causes:**
- Conservative rate limiting
- Single-threaded execution
- Large page sizes
- Inefficient selectors
- Network latency

**Solution Steps:**

Profile scraping performance:

```bash
# Time individual operations
time datascrapexter run config.yaml

# Monitor detailed timing
datascrapexter --log-level debug run config.yaml 2>&1 | grep -i "time"
```

Optimize rate limiting:

```yaml
# Balance speed with reliability
rate_limit: "500ms"     # Faster but still respectful
timeout: "15s"          # Reduce if pages load quickly
max_retries: 2          # Fewer retries for speed
```

Enable concurrent processing:

```bash
# Increase worker threads
datascrapexter run --concurrency 5 config.yaml
```

Optimize selectors for performance:

```yaml
# Avoid universal selectors
# Bad: * .class
# Good: div.specific-class

# Use ID selectors when possible
selector: "#product-price"  # Fastest

# Limit scope
selector: ".product-list .price"  # Better than just .price
```

### Symptom: "High memory usage"

Excessive memory consumption can cause system slowdowns or crashes, particularly when scraping large datasets or running multiple scrapers simultaneously.

**Common Causes:**
- Large result sets held in memory
- Memory leaks in long-running scrapers
- Inefficient data structures
- Concurrent scraper accumulation

**Solution Steps:**

Monitor memory usage:

```bash
# Track memory consumption
ps aux | grep datascrapexter

# Use system monitoring
top -p $(pgrep datascrapexter)

# Detailed memory profiling
datascrapexter run --pprof-port 6060 config.yaml &
go tool pprof -http=:8080 http://localhost:6060/debug/pprof/heap
```

Implement streaming output:

```yaml
# Output to file immediately instead of accumulating
output:
  format: "csv"
  file: "streaming-output.csv"
  streaming: true  # Future feature
```

Limit result set sizes:

```yaml
# Process in smaller batches
pagination:
  max_pages: 10  # Process 10 pages at a time
```

## Anti-Bot Detection Issues

### Symptom: "Access denied" or "Bot detection triggered"

Modern websites employ sophisticated bot detection systems that can identify and block automated scrapers. These blocks manifest as access denied pages, CAPTCHAs, or connection refusals.

**Common Causes:**
- Browser fingerprint detection
- Behavioral pattern recognition
- IP reputation issues
- Missing browser indicators
- JavaScript challenge failures

**Solution Steps:**

Enhance browser simulation:

```yaml
# Rotate user agents
user_agents:
  - "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"
  - "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36"
  - "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36"

# Add realistic headers
headers:
  Accept: "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8"
  Accept-Language: "en-US,en;q=0.9"
  Accept-Encoding: "gzip, deflate, br"
  DNT: "1"
  Upgrade-Insecure-Requests: "1"
```

Implement human-like behavior:

```yaml
# Variable delays
rate_limit: "3s"  # Base delay

# Add randomization (future feature)
rate_limit:
  min: "2s"
  max: "5s"
  distribution: "normal"
```

Use residential proxies for difficult sites:

```yaml
proxy:
  enabled: true
  type: "residential"  # Future feature
  provider: "brightdata"
  rotation: "per-request"
```

## Output and Storage Issues

### Symptom: "Failed to write output file"

Output failures prevent scraped data from being saved, potentially losing valuable information after successful extraction.

**Common Causes:**
- Insufficient disk space
- Permission denied on output directory
- Invalid file paths
- File system limitations
- Concurrent write conflicts

**Solution Steps:**

Check disk space and permissions:

```bash
# Verify disk space
df -h .

# Check directory permissions
ls -la outputs/

# Create directory with proper permissions
mkdir -p outputs
chmod 755 outputs
```

Use absolute paths for reliability:

```yaml
output:
  format: "json"
  file: "/home/user/datascrapexter/outputs/data.json"
```

Handle special characters in filenames:

```bash
# Clean filename generation
DATE=$(date +%Y%m%d_%H%M%S)
SAFE_NAME=$(echo "$SCRAPER_NAME" | tr ' /' '_')
OUTPUT_FILE="outputs/${SAFE_NAME}_${DATE}.json"
```

Implement output rotation:

```bash
# Prevent single file from growing too large
datascrapexter run config.yaml -o "outputs/data_$(date +%Y%m%d_%H).json"
```

## Debugging Strategies

### Enable Comprehensive Logging

Detailed logging is essential for understanding scraper behavior and diagnosing issues:

```bash
# Maximum verbosity
datascrapexter -vv run config.yaml 2>&1 | tee debug.log

# Filter specific components
datascrapexter -vv run config.yaml 2>&1 | grep -E "(ERROR|WARN|selector)"

# JSON logging for structured analysis
datascrapexter --log-format json -v run config.yaml > debug.json
```

### Use Debug Proxy

Inspect actual HTTP traffic using debugging proxies:

```bash
# Using mitmproxy
mitmdump -p 8080

# Configure DataScrapexter to use proxy
HTTP_PROXY=http://localhost:8080 datascrapexter run config.yaml
```

### Test in Isolation

Isolate problems by testing components individually:

```bash
# Test network connectivity
curl -v $(grep base_url config.yaml | cut -d'"' -f2)

# Test selector in browser
# Copy this to browser console
document.querySelector('.your-selector')

# Test with minimal config
cat > test.yaml << EOF
name: "minimal_test"
base_url: "https://example.com"
fields:
  - name: "title"
    selector: "h1"
    type: "text"
output:
  format: "json"
  file: "-"
EOF
datascrapexter run test.yaml
```

### Progressive Enhancement

Build configurations gradually to identify issues:

```bash
# Start with basic extraction
# Add fields one at a time
# Add transformations incrementally
# Add pagination last
```

## Preventive Measures

### Configuration Testing

Implement systematic testing for configurations:

```bash
#!/bin/bash
# test_config.sh

CONFIG=$1
echo "Testing configuration: $CONFIG"

# Validate syntax
if ! datascrapexter validate --strict "$CONFIG"; then
    echo "Validation failed"
    exit 1
fi

# Dry run test
if ! datascrapexter run --dry-run "$CONFIG"; then
    echo "Dry run failed"
    exit 1
fi

# Limited real test
datascrapexter run --max-pages 1 "$CONFIG"
```

### Monitoring and Alerts

Set up monitoring for production scrapers:

```bash
# Health check script
#!/bin/bash

LAST_RUN=$(find outputs -name "*.json" -mmin -60 | wc -l)
if [ $LAST_RUN -eq 0 ]; then
    echo "ALERT: No output in last hour"
    # Send notification
fi

# Check error rates
ERROR_COUNT=$(grep -c ERROR logs/scraper.log)
if [ $ERROR_COUNT -gt 10 ]; then
    echo "ALERT: High error rate"
fi
```

### Regular Maintenance

Schedule regular maintenance tasks:

```bash
# Weekly configuration validation
0 0 * * 0 find configs -name "*.yaml" -exec datascrapexter validate {} \;

# Monthly performance review
0 0 1 * * /opt/datascrapexter/scripts/performance_review.sh

# Cleanup old outputs
0 2 * * * find outputs -name "*.json" -mtime +30 -delete
```

## Common Error Messages

### "No such file or directory"

This error typically indicates a missing configuration file or incorrect path specification.

**Quick Fix:**
```bash
# Verify file exists
ls -la config.yaml

# Use absolute path
datascrapexter run $(pwd)/config.yaml

# Check working directory
pwd
```

### "Permission denied"

Permission errors occur when DataScrapexter lacks necessary access rights.

**Quick Fix:**
```bash
# Check file permissions
ls -la config.yaml

# Fix permissions
chmod 644 config.yaml
chmod 755 outputs/

# Run with appropriate user
sudo -u scraper datascrapexter run config.yaml
```

### "Invalid memory address or nil pointer dereference"

This Go runtime error indicates a bug in the application, often triggered by unexpected input.

**Quick Fix:**
```bash
# Update to latest version
go install github.com/valpere/DataScrapexter/cmd/datascrapexter@latest

# Report issue with details
datascrapexter version
# Include configuration and error message in bug report
```

### "Context deadline exceeded"

This error occurs when operations exceed configured timeouts.

**Quick Fix:**
```yaml
# Increase timeouts
timeout: "60s"        # Increase from default
rate_limit: "5s"      # Allow more time between requests
```

## Platform-Specific Issues

### Linux-Specific Problems

**DNS Resolution Issues:**
```bash
# Check DNS configuration
cat /etc/resolv.conf

# Test with specific DNS
echo "nameserver 8.8.8.8" | sudo tee /etc/resolv.conf
```

**Certificate Verification Errors:**
```bash
# Update CA certificates
sudo apt-get update && sudo apt-get install ca-certificates
# or
sudo yum install ca-certificates
```

### macOS-Specific Problems

**Security Restrictions:**
```bash
# Allow unsigned binary
xattr -d com.apple.quarantine datascrapexter

# Grant network permissions
# System Preferences > Security & Privacy > Privacy > Full Disk Access
```

**SSL Certificate Issues:**
```bash
# Update certificates via Homebrew
brew install ca-certificates
```

### Windows-Specific Problems

**Path Issues:**
```powershell
# Use proper path separators
datascrapexter run configs\example.yaml

# Handle spaces in paths
datascrapexter run "C:\Program Files\DataScrapexter\configs\example.yaml"
```

**Antivirus Interference:**
- Add DataScrapexter to antivirus exceptions
- Temporarily disable real-time scanning for testing
- Check quarantine for removed files

## Advanced Debugging Techniques

### HTTP Traffic Analysis

Capture and analyze HTTP traffic for deep debugging:

```bash
# Using tcpdump
sudo tcpdump -i any -w capture.pcap host example.com

# Using Wireshark
# Filter: http.host == "example.com"
```

### Systematic Problem Isolation

Follow this systematic approach for complex issues:

1. **Verify Basic Functionality:**
   ```bash
   datascrapexter version
   datascrapexter template > test.yaml
   datascrapexter validate test.yaml
   ```

2. **Test Network Connectivity:**
   ```bash
   ping example.com
   curl -I https://example.com
   ```

3. **Validate Configuration:**
   ```bash
   datascrapexter validate --strict config.yaml
   python -m json.tool < config.json  # For JSON configs
   ```

4. **Test Minimal Extraction:**
   ```bash
   # Create minimal config
   echo 'name: test
   base_url: "https://example.com"
   fields:
     - name: "title"
       selector: "title"
       type: "text"
   output:
     format: "json"
     file: "-"' | datascrapexter run -
   ```

5. **Incremental Complexity:**
   - Add fields one by one
   - Add transformations individually
   - Enable pagination last

### Performance Profiling

For performance-related issues, use Go's built-in profiling:

```bash
# CPU profiling
datascrapexter run --pprof-port 6060 config.yaml &
go tool pprof http://localhost:6060/debug/pprof/profile

# Memory profiling
go tool pprof http://localhost:6060/debug/pprof/heap

# Execution trace
wget http://localhost:6060/debug/pprof/trace?seconds=10
go tool trace trace
```

## Getting Help

### Before Asking for Help

Gather essential information for effective troubleshooting:

```bash
# System information
uname -a                              # OS details
go version                            # Go version
datascrapexter version               # DataScrapexter version

# Configuration and error
cat config.yaml                       # Configuration file
datascrapexter validate config.yaml  # Validation output
datascrapexter -vv run config.yaml 2>&1 | tee error.log  # Full error log
```

### Where to Get Help

1. **GitHub Issues**: Search existing issues or create new ones
   - Include all system information
   - Provide minimal reproducible example
   - Attach relevant logs and configurations

2. **Community Discord**: Real-time help from community members
   - Use appropriate channels (#help, #bugs)
   - Be respectful of volunteers' time

3. **Stack Overflow**: For programming-related questions
   - Tag with 'datascrapexter' and 'web-scraping'
   - Follow Stack Overflow guidelines

4. **Commercial Support**: For business-critical issues
   - Email: support@datascrapexter.com
   - Include license information

### Creating Effective Bug Reports

Good bug reports include:

1. **Clear Title**: Summarize the issue concisely
2. **Environment Details**: OS, versions, configuration
3. **Steps to Reproduce**: Exact commands and configurations
4. **Expected Behavior**: What should happen
5. **Actual Behavior**: What actually happens
6. **Error Messages**: Complete error output
7. **Minimal Example**: Smallest config that reproduces issue

Example bug report template:

```markdown
## Environment
- OS: Ubuntu 20.04
- DataScrapexter: v0.1.0
- Go: 1.24.0

## Configuration
```yaml
name: "bug_example"
base_url: "https://example.com"
fields:
  - name: "title"
    selector: ".missing-class"
    type: "text"
    required: true
```

## Steps to Reproduce
1. Save above configuration as bug.yaml
2. Run: datascrapexter run bug.yaml

## Expected Behavior
Should extract title text from page

## Actual Behavior
Error: no elements found for selector: .missing-class

## Logs
[Attach full debug log]
```

## Best Practices for Reliable Scraping

### Defensive Configuration

Build resilient configurations that handle edge cases:

```yaml
# Use multiple selectors
fields:
  - name: "price"
    selector: ".price-now, .current-price, .product-price, [itemprop='price']"
    type: "text"
    
# Set reasonable defaults
    transform:
      - type: "trim"
      - type: "regex"
        pattern: "^$"
        replacement: "0.00"  # Default for empty
      - type: "clean_price"
      - type: "parse_float"
```

### Error Recovery

Implement comprehensive error handling:

```bash
#!/bin/bash
# Scraper with recovery

MAX_ATTEMPTS=3
ATTEMPT=0

while [ $ATTEMPT -lt $MAX_ATTEMPTS ]; do
    if datascrapexter run config.yaml; then
        echo "Scraping successful"
        break
    else
        ATTEMPT=$((ATTEMPT + 1))
        echo "Attempt $ATTEMPT failed, retrying..."
        sleep 60  # Wait before retry
    fi
done

if [ $ATTEMPT -eq $MAX_ATTEMPTS ]; then
    echo "All attempts failed"
    # Send alert
    exit 1
fi
```

### Monitoring Strategy

Implement comprehensive monitoring:

```bash
# Monitor multiple aspects
#!/bin/bash

# Check process
if ! pgrep -f datascrapexter > /dev/null; then
    echo "DataScrapexter not running"
fi

# Check output freshness
LATEST=$(find outputs -type f -mmin -60 | wc -l)
if [ $LATEST -eq 0 ]; then
    echo "No recent output files"
fi

# Check error rates
ERRORS=$(tail -n 1000 logs/scraper.log | grep -c ERROR)
if [ $ERRORS -gt 50 ]; then
    echo "High error rate: $ERRORS errors in last 1000 lines"
fi

# Check disk space
DISK_USAGE=$(df -h outputs | tail -1 | awk '{print $5}' | sed 's/%//')
if [ $DISK_USAGE -gt 90 ]; then
    echo "Low disk space: ${DISK_USAGE}% used"
fi
```

## Conclusion

Effective troubleshooting requires systematic approaches, patience, and good documentation. Most issues fall into common categories with established solutions. By following this guide's recommendations and maintaining good practices, you can minimize problems and quickly resolve those that do occur.

Remember that web scraping involves interacting with constantly changing systems. Regular maintenance, monitoring, and updates are essential for long-term reliability. When encountering new issues, document your solutions to build institutional knowledge and help the community.

Keep this guide handy as a reference, and contribute your own solutions back to the DataScrapexter community. Together, we can build more reliable and efficient web scraping systems.
