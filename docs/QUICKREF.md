# DataScrapexter Quick Reference

Quick reference for common DataScrapexter operations and configurations.

## 🚀 Command Line Interface

### Basic Commands
```bash
# Validate configuration file
datascrapexter validate config.yaml

# Run scraper with configuration
datascrapexter run config.yaml

# Generate configuration template
datascrapexter template --type basic > config.yaml
datascrapexter template --type ecommerce > ecommerce.yaml

# Health check
datascrapexter health

# Server mode (API)
datascrapexter serve --port 8080 --metrics-port 9090
```

### Template Types
```bash
--type basic      # Simple scraping setup
--type ecommerce  # E-commerce product scraping
--type news       # News article extraction
--type jobs       # Job listing scraper
--type social     # Social media content (planned)
```

### Debugging & Testing
```bash
# Debug mode with verbose logging
datascrapexter run config.yaml --log-level debug

# Test proxy configuration
datascrapexter proxy-test --config config.yaml

# Test CAPTCHA solver
datascrapexter captcha-test --solver 2captcha --api-key YOUR_KEY

# Profile performance
datascrapexter run config.yaml --pprof-port 6060
```

## ⚙️ Configuration Quick Reference

### Basic Structure
```yaml
name: "scraper_name"           # Required: Scraper identifier
base_url: "https://site.com"   # Required: Target website
rate_limit: "2s"               # Delay between requests

# Data extraction rules
fields:
  - name: "field_name"         # Required: Field identifier
    selector: "h1"             # Required: CSS selector
    type: "text"               # text|attr|html|list|number|boolean
    required: true             # Optional: Fail if missing
    transform:                 # Optional: Data transformations
      - type: "regex"
        pattern: "\\$([0-9.]+)"
        replacement: "$1"

# Output configuration
output:
  format: "json"               # json|csv|xml|yaml|database
  file: "output.json"          # Output file path
```

### Database Output
```yaml
output:
  database:
    type: "postgresql"         # postgresql|sqlite
    connection_string: "${DATABASE_URL}"  # PostgreSQL
    database_path: "data.db"   # SQLite
    table: "scraped_data"
    batch_size: 1000
    create_table: true
    on_conflict: "ignore"      # ignore|error|replace(SQLite)
```

### Anti-Detection
```yaml
anti_detection:
  proxy:
    enabled: true
    rotation: "random"         # round_robin|random
    providers:
      - "http://proxy1:8080"
      - "http://proxy2:8080"

  captcha:
    solver: "2captcha"         # 2captcha|anticaptcha|capmonster
    api_key: "${CAPTCHA_KEY}"

  fingerprinting:
    randomize_viewport: true
    spoof_canvas: true
    rotate_user_agents: true

# User agent rotation
user_agents:
  - "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"
  - "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36"
```

### Rate Limiting
```yaml
# Simple rate limiting
rate_limit: "2s"

# Advanced rate limiting
rate_limit:
  strategy: "adaptive"         # fixed|adaptive|burst|hybrid
  base_delay: "2s"
  max_delay: "30s"
  burst_size: 5
  consecutive_errors_threshold: 3
```

### Browser Automation
```yaml
browser:
  enabled: true
  headless: true
  user_data_dir: "/tmp/chrome"
  wait_for_element: ".content"
  timeout: 30s
  viewport:
    width: 1920
    height: 1080
```

### Pagination
```yaml
pagination:
  type: "next_button"          # next_button|url_pattern|offset|cursor
  selector: ".pagination .next"
  max_pages: 100

  # URL pattern pagination
  # type: "url_pattern"
  # url_template: "https://site.com/page/{page}"
  # start_page: 1
  # max_pages: 50
```

## 🔍 Field Types & Extractors

### Basic Types
```yaml
fields:
  - name: "title"
    selector: "h1"
    type: "text"               # Extract text content

  - name: "link"
    selector: "a"
    type: "attr"               # Extract attribute
    attribute: "href"

  - name: "content"
    selector: ".content"
    type: "html"               # Extract HTML content

  - name: "tags"
    selector: ".tag"
    type: "list"               # Extract multiple elements
```

### Advanced Types
```yaml
fields:
  - name: "price"
    selector: ".price"
    type: "number"             # Extract and parse number

  - name: "in_stock"
    selector: ".stock"
    type: "boolean"            # Extract boolean value

  - name: "published"
    selector: ".date"
    type: "date"               # Extract and parse date

  - name: "url"
    selector: "a"
    type: "url"                # Extract and resolve URL

  - name: "email"
    selector: ".contact"
    type: "email"              # Extract email address

  - name: "phone"
    selector: ".phone"
    type: "phone"              # Extract phone number
```

## 🔄 Transformations

### Common Transformations
```yaml
transform:
  - type: "trim_spaces"        # Remove leading/trailing spaces
  - type: "normalize_spaces"   # Normalize multiple spaces
  - type: "lowercase"          # Convert to lowercase
  - type: "uppercase"          # Convert to uppercase
  - type: "title_case"         # Convert to Title Case

  - type: "regex"              # Regular expression replacement
    pattern: "\\$([0-9,]+)"
    replacement: "$1"

  - type: "substring"          # Extract substring
    start: 0
    length: 100

  - type: "split"              # Split string
    delimiter: ","
    index: 0                   # Take first part

  - type: "parse_int"          # Parse as integer
  - type: "parse_float"        # Parse as float
  - type: "format_currency"    # Format as currency
    currency: "USD"
```

## 🐳 Docker Quick Start

### Basic Usage
```bash
# Pull latest image
docker pull ghcr.io/valpere/datascrapexter:latest

# Run with config file
docker run -v $(pwd)/config.yaml:/app/config.yaml \
           -v $(pwd)/output:/app/output \
           ghcr.io/valpere/datascrapexter:latest run /app/config.yaml
```

### Docker Compose
```yaml
version: '3.8'
services:
  datascrapexter:
    image: ghcr.io/valpere/datascrapexter:latest
    volumes:
      - ./configs:/app/configs
      - ./output:/app/output
    environment:
      - DATABASE_URL=${DATABASE_URL}
      - CAPTCHA_API_KEY=${CAPTCHA_API_KEY}
    command: run /app/configs/production.yaml
```

## 📊 Monitoring & Health Checks

### Health Endpoints
```bash
# Basic health check
curl http://localhost:8080/health

# Readiness check
curl http://localhost:8080/ready

# Prometheus metrics
curl http://localhost:9090/metrics
```

### Key Metrics
- `datascrapexter_requests_total` - Total HTTP requests
- `datascrapexter_requests_duration_seconds` - Request latency
- `datascrapexter_extraction_success_rate` - Success rate
- `datascrapexter_proxy_health` - Proxy status
- `datascrapexter_captcha_solve_rate` - CAPTCHA success rate

## 🔧 Environment Variables

### Common Variables
```bash
# Database connection
DATABASE_URL="postgresql://user:pass@host:5432/db"

# CAPTCHA solver API key
CAPTCHA_API_KEY="your_2captcha_api_key"

# Proxy configuration
PROXY_LIST="http://proxy1:8080,http://proxy2:8080"

# Logging level
LOG_LEVEL="info"  # debug|info|warn|error

# Rate limiting
RATE_LIMIT="2s"
```

## 🚨 Troubleshooting Quick Fixes

### Common Issues
```bash
# Configuration validation errors
datascrapexter validate config.yaml --strict

# Network/proxy issues
datascrapexter proxy-test --config config.yaml

# High memory usage
datascrapexter run config.yaml --pprof-port 6060

# CAPTCHA solving issues
datascrapexter captcha-test --solver 2captcha --api-key YOUR_KEY

# Rate limiting errors
# Increase rate_limit in config: rate_limit: "5s"

# JavaScript rendering issues
# Enable browser mode:
# browser:
#   enabled: true
#   headless: true
```

### Debug Mode
```bash
# Maximum verbosity
datascrapexter run config.yaml \
  --log-level debug \
  --log-format json \
  --pprof-port 6060 \
  --metrics-port 9090
```

## 📝 Example Configurations

### Minimal Configuration
```yaml
name: "basic_scraper"
base_url: "https://example.com"
fields:
  - name: "title"
    selector: "h1"
    type: "text"
output:
  format: "json"
  file: "output.json"
```

### Production Configuration
```yaml
name: "production_scraper"
base_url: "https://ecommerce-site.com"

rate_limit:
  strategy: "adaptive"
  base_delay: "3s"
  max_delay: "30s"

anti_detection:
  proxy:
    enabled: true
    rotation: "random"
    providers: ["${PROXY_LIST}"]
  fingerprinting:
    randomize_viewport: true
    rotate_user_agents: true

browser:
  enabled: true
  headless: true
  timeout: 30s

fields:
  - name: "product_name"
    selector: "h1.product-title"
    type: "text"
    required: true
  - name: "price"
    selector: ".price"
    type: "number"
    transform:
      - type: "regex"
        pattern: "\\$([0-9,]+\\.?[0-9]*)"
        replacement: "$1"

pagination:
  type: "next_button"
  selector: ".pagination-next"
  max_pages: 100

output:
  database:
    type: "postgresql"
    connection_string: "${DATABASE_URL}"
    table: "products"
    batch_size: 1000
    create_table: true
    on_conflict: "ignore"
```

## 🔗 Quick Links

- [Full Documentation](README.md)
- [Configuration Reference](configuration.md)
- [API Documentation](api.md)
- [Troubleshooting Guide](troubleshooting.md)
- [Examples Directory](../examples/)
- [GitHub Repository](https://github.com/valpere/DataScrapexter)

---

*Quick reference last updated: $(date)*