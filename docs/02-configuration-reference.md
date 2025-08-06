# DataScrapexter Configuration Reference

## Overview

This comprehensive reference covers all configuration options available in DataScrapexter. Configuration files use YAML format and control every aspect of scraping behavior, from basic settings to advanced anti-detection mechanisms.

## Table of Contents

1. [Configuration Structure](#configuration-structure)
2. [Basic Settings](#basic-settings)
3. [Request Configuration](#request-configuration)
4. [Data Extraction](#data-extraction)
5. [Pagination](#pagination)
6. [Output Configuration](#output-configuration)
7. [Anti-Detection](#anti-detection)
8. [Monitoring](#monitoring)
9. [Environment Variables](#environment-variables)
10. [Configuration Templates](#configuration-templates)

## Configuration Structure

### Basic File Structure

```yaml
# Basic identification
name: "scraper_name"                    # Required: Unique identifier
base_url: "https://example.com"         # Required: Starting URL

# Request settings
rate_limit: "2s"                        # Delay between requests
timeout: "30s"                          # Request timeout
max_retries: 3                          # Retry attempts
headers: {}                             # Custom headers
user_agents: []                         # User agent rotation

# Proxy configuration
proxy: {}                               # Proxy settings

# Data extraction
fields: []                              # Field definitions

# Pagination
pagination: {}                          # Multi-page handling

# Output
output: {}                              # Output configuration

# Advanced features
anti_detection: {}                      # Anti-detection mechanisms
monitoring: {}                          # Metrics and health checks
browser: {}                             # Browser automation
```

## Basic Settings

### name (required)

Unique identifier for the scraper configuration.

```yaml
name: "product_price_monitor"
```

### base_url (required)

The starting URL for scraping operations.

```yaml
base_url: "https://example.com/products"

# With environment variable
base_url: "${TARGET_URL}"

# With query parameters
base_url: "https://api.example.com/search?q=products&limit=100"
```

## Request Configuration

### rate_limit

Controls the minimum time between requests.

```yaml
# Simple delay
rate_limit: "2s"

# With random variation
rate_limit: "2s"
random_delay: "1s"  # Add 0-1s random delay

# Advanced rate limiting
anti_detection:
  rate_limiting:
    base_delay: "2s"
    random_delay: "1s"
    adaptive: true
    min_delay: "1s"
    max_delay: "10s"
```

### timeout

Maximum time to wait for server response.

```yaml
timeout: "30s"

# Different timeouts for different operations
browser:
  timeouts:
    page_load: "30s"
    javascript: "10s"
    element_wait: "10s"
```

### max_retries

Number of retry attempts for failed requests.

```yaml
max_retries: 3

# With exponential backoff
max_retries: 5
retry_backoff: "exponential"  # Options: fixed, exponential, linear
```

### headers

Custom HTTP headers for requests.

```yaml
headers:
  Accept: "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8"
  Accept-Language: "en-US,en;q=0.9"
  Accept-Encoding: "gzip, deflate, br"
  Referer: "https://google.com"
  DNT: "1"
  Connection: "keep-alive"
```

### user_agents

User agent strings for rotation.

```yaml
user_agents:
  - "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"
  - "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36"
  - "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36"
```

## Data Extraction

### fields

Data extraction field definitions.

```yaml
fields:
  - name: "title"                       # Required: Output field name
    selector: "h1.product-title"        # Required: CSS selector
    type: "text"                        # Required: Extraction type
    required: true                      # Optional: Field is mandatory
    attribute: "href"                   # Required for type: "attr"
    transform: []                       # Optional: Data transformations
```

### Field Types

#### text

Extract text content from elements.

```yaml
- name: "product_title"
  selector: "h1"
  type: "text"
```

#### attr

Extract attribute values from elements.

```yaml
- name: "product_url"
  selector: "a.product-link"
  type: "attr"
  attribute: "href"
```

#### html

Extract raw HTML content.

```yaml
- name: "product_description"
  selector: ".description"
  type: "html"
```

#### list

Extract from all matching elements as array.

```yaml
- name: "product_images"
  selector: ".gallery img"
  type: "list"
  attribute: "src"
```

### Data Transformations

```yaml
transform:
  # Text cleaning
  - type: "trim"                        # Remove whitespace
  - type: "lowercase"                   # Convert to lowercase
  - type: "uppercase"                   # Convert to uppercase
  - type: "normalize_spaces"            # Replace multiple spaces
  
  # Pattern matching
  - type: "regex"
    pattern: "\\$([0-9,]+\\.?[0-9]*)"
    replacement: "$1"
  
  # Type conversion
  - type: "parse_float"                 # Convert to float
  - type: "parse_int"                   # Convert to integer
  
  # Specialized cleaning
  - type: "clean_price"                 # Extract price from text
  - type: "extract_numbers"             # Extract all numbers
  - type: "remove_html"                 # Strip HTML tags
```

## Pagination

### next_button

Follow "next page" links.

```yaml
pagination:
  type: "next_button"
  selector: ".pagination .next"
  max_pages: 10
```

### page_numbers

Navigate numbered pagination.

```yaml
pagination:
  type: "page_numbers"
  selector: ".pagination a"
  max_pages: 20
```

### url_pattern

Generate URLs from pattern.

```yaml
pagination:
  type: "url_pattern"
  url_pattern: "https://example.com/products?page={page}"
  start_page: 1
  max_pages: 100
```

## Output Configuration

### JSON Output

```yaml
output:
  format: "json"
  file: "output.json"
  json:
    pretty_print: true
    indent: 2
    include_metadata: true
    sort_keys: true
```

### CSV Output

```yaml
output:
  format: "csv"
  file: "output.csv"
  csv:
    include_headers: true
    delimiter: ","
    quote_character: "\""
    flatten_nested: true
```

### Excel Output

```yaml
output:
  format: "excel"
  file: "output.xlsx"
  excel:
    sheet_name: "Data"
    include_headers: true
    auto_filter: true
    freeze_pane: true
    header_style:
      font:
        bold: true
        color: "#FFFFFF"
      fill:
        color: "#2F5597"
    column_widths:
      title: 30
      price: 15
```

### Database Output

```yaml
output:
  format: "database"
  database:
    driver: "postgresql"
    host: "localhost"
    port: 5432
    database: "scraping_data"
    username: "scraper"
    password: "${DB_PASSWORD}"
    table: "products"
    batch_size: 1000
    auto_create_table: true
```

### Multiple Outputs

```yaml
output:
  multiple: true
  outputs:
    - format: "json"
      file: "detailed.json"
    - format: "csv"
      file: "summary.csv"
    - format: "database"
      database:
        driver: "postgresql"
        table: "live_data"
```

## Anti-Detection

### Browser Fingerprinting

```yaml
anti_detection:
  fingerprinting:
    enabled: true
    canvas_spoofing: true
    webgl_spoofing: true
    audio_spoofing: true
    screen_spoofing: true
    font_spoofing: true
    randomize_viewport: true
```

### CAPTCHA Solving

```yaml
anti_detection:
  captcha:
    enabled: true
    service: "2captcha"               # Options: 2captcha, anti-captcha, capmonster
    api_key: "${CAPTCHA_API_KEY}"
    timeout: "120s"
    max_attempts: 3
```

### TLS Fingerprinting

```yaml
anti_detection:
  tls:
    randomize_ja3: true
    randomize_ja4: true
    min_tls_version: "1.2"
    cipher_suites:
      - "TLS_AES_128_GCM_SHA256"
      - "TLS_AES_256_GCM_SHA384"
```

### Proxy Configuration

```yaml
proxy:
  enabled: true
  rotation: "random"                  # Options: random, round_robin, weighted
  health_check: true
  providers:
    - url: "http://proxy1.example.com:8080"
      username: "user1"
      password: "${PROXY1_PASSWORD}"
      weight: 1
      max_concurrent: 10
```

## Monitoring

### Metrics

```yaml
monitoring:
  metrics:
    enabled: true
    namespace: "datascrapexter"
    subsystem: "scraper"
    listen_address: ":9090"
    metrics_path: "/metrics"
```

### Health Checks

```yaml
monitoring:
  health:
    check_interval: "30s"
    health_endpoint: "/health"
    readiness_endpoint: "/ready"
    liveness_endpoint: "/live"
```

### Dashboard

```yaml
monitoring:
  dashboard:
    enabled: true
    port: ":8080"
    title: "Scraper Monitor"
    refresh_interval: "5s"
```

## Environment Variables

### Using Environment Variables

```yaml
# Basic usage
name: "${SCRAPER_NAME}"
base_url: "${TARGET_URL}"

# With defaults
name: "${SCRAPER_NAME:-default_scraper}"
base_url: "${TARGET_URL:-https://example.com}"

# In nested configurations
proxy:
  enabled: true
  url: "${PROXY_URL}"
  username: "${PROXY_USER}"
  password: "${PROXY_PASSWORD}"

output:
  file: "${OUTPUT_DIR}/results_${TIMESTAMP}.json"
```

### Common Environment Variables

```bash
# URLs and endpoints
export TARGET_URL="https://example.com"
export OUTPUT_DIR="/data/scraping"

# Authentication
export PROXY_USER="username"
export PROXY_PASSWORD="password"
export CAPTCHA_API_KEY="your-api-key"
export DB_PASSWORD="secure-password"

# Timing and identification
export TIMESTAMP=$(date +%Y%m%d_%H%M%S)
export DATE=$(date +%Y-%m-%d)
export SCRAPER_NAME="production_scraper"
```

## Configuration Templates

### E-commerce Template

```yaml
name: "ecommerce_scraper"
base_url: "${TARGET_URL}"
rate_limit: "3s"
timeout: "30s"
max_retries: 3

headers:
  Accept: "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8"
  Accept-Language: "en-US,en;q=0.9"

anti_detection:
  fingerprinting:
    enabled: true
  captcha:
    enabled: true
    service: "2captcha"
    api_key: "${CAPTCHA_API_KEY}"

fields:
  - name: "title"
    selector: "h1, .product-title"
    type: "text"
    required: true
    transform:
      - type: "trim"
      - type: "normalize_spaces"
  - name: "price"
    selector: ".price, .sale-price"
    type: "text"
    required: true
    transform:
      - type: "clean_price"
      - type: "parse_float"

output:
  format: "excel"
  file: "products_${DATE}.xlsx"
  excel:
    include_headers: true
    auto_filter: true
```

### News Article Template

```yaml
name: "news_scraper"
base_url: "${NEWS_URL}"
rate_limit: "2s"

browser:
  enabled: true
  headless: true
  wait_for_element: "article"

fields:
  - name: "headline"
    selector: "h1, .headline"
    type: "text"
    required: true
  - name: "content"
    selector: ".article-content, .story-body"
    type: "text"
    transform:
      - type: "remove_html"
      - type: "normalize_spaces"
  - name: "published_date"
    selector: "time, .publish-date"
    type: "attr"
    attribute: "datetime"

output:
  format: "database"
  database:
    driver: "postgresql"
    table: "articles"
    auto_create_table: true
```

### API-like Data Template

```yaml
name: "api_scraper"
base_url: "${API_URL}"
rate_limit: "1s"

headers:
  Accept: "application/json"
  Content-Type: "application/json"

fields:
  - name: "data"
    selector: "body"
    type: "text"
    transform:
      - type: "parse_json"

output:
  format: "json"
  file: "api_data_${TIMESTAMP}.json"
```

## Best Practices

### Configuration Organization

1. **Use Environment Variables**: Store sensitive data in environment variables
2. **Template Reuse**: Create reusable templates for similar sites
3. **Version Control**: Keep configurations in version control
4. **Documentation**: Add comments to explain complex selectors
5. **Validation**: Always validate configurations before deployment

### Performance Optimization

1. **Rate Limiting**: Start conservative, adjust based on server responses
2. **Field Selection**: Only extract needed fields
3. **Transformation Efficiency**: Minimize transformation steps
4. **Output Format**: Choose appropriate format for your use case
5. **Monitoring**: Track performance metrics

### Error Handling

1. **Required Fields**: Mark critical fields as required
2. **Fallback Selectors**: Use multiple selectors for reliability
3. **Retry Logic**: Configure appropriate retry settings
4. **Graceful Degradation**: Handle missing optional fields

### Security and Compliance

1. **Respect robots.txt**: Check website policies
2. **Rate Limiting**: Avoid overwhelming servers
3. **User Identification**: Use appropriate user agents
4. **Data Privacy**: Handle personal data responsibly
5. **Legal Compliance**: Follow applicable laws and regulations

This configuration reference provides comprehensive coverage of all DataScrapexter options. Start with basic configurations and gradually add complexity as needed.
