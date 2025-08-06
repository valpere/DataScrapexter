# DataScrapexter Examples and Tutorials

## Overview

This guide provides practical, real-world examples and step-by-step tutorials for using DataScrapexter across various scraping scenarios. Each example includes complete configurations, explanations, and expected outputs.

## Table of Contents

1. [Basic Examples](#basic-examples)
2. [E-commerce Scraping](#e-commerce-scraping)
3. [News and Content](#news-and-content)
4. [Real Estate](#real-estate)
5. [Job Boards](#job-boards)
6. [Advanced Scenarios](#advanced-scenarios)
7. [Anti-Detection Examples](#anti-detection-examples)
8. [Monitoring and Production](#monitoring-and-production)

## Basic Examples

### 1. Simple Quote Scraper

**Scenario**: Extract quotes from a simple quotes website.

**Configuration** (`quotes.yaml`):

```yaml
name: "quotes_scraper"
base_url: "http://quotes.toscrape.com/"
rate_limit: "1s"
timeout: "10s"

fields:
  - name: "quote"
    selector: ".quote .text"
    type: "text"
    required: true
    transform:
      - type: "trim"
  - name: "author"
    selector: ".quote .author"
    type: "text"
    required: true
  - name: "tags"
    selector: ".quote .tags a"
    type: "list"

output:
  format: "json"
  file: "quotes.json"
```

**Usage**:

```bash
datascrapexter run quotes.yaml
```

**Expected Output**:

```json
{
  "metadata": {
    "scraper_name": "quotes_scraper",
    "scraped_at": "2024-01-15T10:30:00Z",
    "total_records": 10
  },
  "data": [
    {
      "quote": "\"The world as we have created it is a process of our thinking.\"",
      "author": "Albert Einstein",
      "tags": ["change", "deep-thoughts", "thinking", "world"]
    }
  ]
}
```

### 2. Book Catalog Scraper

**Scenario**: Extract book information with pagination.

**Configuration** (`books.yaml`):

```yaml
name: "book_catalog"
base_url: "http://books.toscrape.com/"
rate_limit: "2s"

fields:
  - name: "title"
    selector: "h3 a"
    type: "attr"
    attribute: "title"
    required: true
  - name: "price"
    selector: ".price_color"
    type: "text"
    transform:
      - type: "clean_price"
      - type: "parse_float"
  - name: "availability"
    selector: ".availability"
    type: "text"
    transform:
      - type: "trim"
      - type: "lowercase"
  - name: "rating"
    selector: ".star-rating"
    type: "attr"
    attribute: "class"
    transform:
      - type: "regex"
        pattern: "star-rating (\\w+)"
        replacement: "$1"

pagination:
  type: "next_button"
  selector: ".next a"
  max_pages: 5

output:
  format: "csv"
  file: "books.csv"
  csv:
    include_headers: true
```

## E-commerce Scraping

### 1. Product Price Monitor

**Scenario**: Monitor product prices with anti-detection.

**Configuration** (`ecommerce-monitor.yaml`):

```yaml
name: "product_price_monitor"
base_url: "https://example-store.com/products"
rate_limit: "5s"
timeout: "45s"
max_retries: 5

# Anti-detection configuration
anti_detection:
  fingerprinting:
    enabled: true
    canvas_spoofing: true
    webgl_spoofing: true
  captcha:
    enabled: true
    service: "2captcha"
    api_key: "${CAPTCHA_API_KEY}"

# Realistic headers
headers:
  Accept: "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8"
  Accept-Language: "en-US,en;q=0.5"
  Accept-Encoding: "gzip, deflate, br"
  DNT: "1"
  Connection: "keep-alive"

# User agent rotation
user_agents:
  - "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"
  - "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36"

# Proxy configuration
proxy:
  enabled: true
  rotation: "random"
  providers:
    - url: "${PROXY_URL_1}"
      username: "${PROXY_USER}"
      password: "${PROXY_PASS}"

fields:
  - name: "product_id"
    selector: "[data-product-id]"
    type: "attr"
    attribute: "data-product-id"
    required: true
  - name: "title"
    selector: "h1.product-title, .product-name h1"
    type: "text"
    required: true
    transform:
      - type: "trim"
      - type: "normalize_spaces"
  - name: "current_price"
    selector: ".price-current, .sale-price"
    type: "text"
    required: true
    transform:
      - type: "clean_price"
      - type: "parse_float"
  - name: "original_price"
    selector: ".price-original, .regular-price"
    type: "text"
    transform:
      - type: "clean_price"
      - type: "parse_float"
  - name: "discount_percent"
    selector: ".discount-percent"
    type: "text"
    transform:
      - type: "extract_numbers"
      - type: "parse_float"
  - name: "availability"
    selector: ".stock-status"
    type: "text"
    transform:
      - type: "lowercase"
      - type: "trim"
  - name: "rating"
    selector: ".rating-value"
    type: "text"
    transform:
      - type: "parse_float"
  - name: "review_count"
    selector: ".review-count"
    type: "text"
    transform:
      - type: "extract_numbers"
      - type: "parse_int"

pagination:
  type: "url_pattern"
  url_pattern: "https://example-store.com/products?page={page}"
  start_page: 1
  max_pages: 10

output:
  format: "excel"
  file: "products_${TIMESTAMP}.xlsx"
  excel:
    sheet_name: "Product Monitoring"
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
      title: 40
      current_price: 15
      original_price: 15
      availability: 20
```

**Environment Setup**:

```bash
export CAPTCHA_API_KEY="your-2captcha-key"
export PROXY_URL_1="http://proxy.example.com:8080"
export PROXY_USER="username"
export PROXY_PASS="password"
export TIMESTAMP=$(date +%Y%m%d_%H%M%S)
```

### 2. Product Comparison Scraper

**Scenario**: Compare products across multiple categories.

**Configuration** (`product-comparison.yaml`):

```yaml
name: "product_comparison"
base_url: "https://comparison-site.com/category/electronics"
rate_limit: "3s"

fields:
  - name: "product_name"
    selector: ".product-title"
    type: "text"
    required: true
  - name: "brand"
    selector: ".brand-name"
    type: "text"
  - name: "model"
    selector: ".model-number"
    type: "text"
  - name: "specs"
    selector: ".specifications li"
    type: "list"
  - name: "pros"
    selector: ".pros li"
    type: "list"
  - name: "cons"
    selector: ".cons li"
    type: "list"
  - name: "expert_score"
    selector: ".expert-rating .score"
    type: "text"
    transform:
      - type: "parse_float"
  - name: "user_score"
    selector: ".user-rating .score"
    type: "text"
    transform:
      - type: "parse_float"

output:
  multiple: true
  outputs:
    - format: "json"
      file: "detailed_comparison.json"
    - format: "csv"
      file: "comparison_summary.csv"
      field_selection: ["product_name", "brand", "expert_score", "user_score"]
```

## News and Content

### 1. News Article Collector

**Scenario**: Collect news articles with full content.

**Configuration** (`news-collector.yaml`):

```yaml
name: "news_article_collector"
base_url: "https://news-site.com/latest"
rate_limit: "2s"
timeout: "20s"

# Browser automation for JavaScript-heavy sites
browser:
  enabled: true
  headless: true
  wait_for_element: "article.loaded"
  wait_timeout: "15s"
  viewport:
    width: 1920
    height: 1080

fields:
  - name: "headline"
    selector: "h1.headline, h1.article-title"
    type: "text"
    required: true
    transform:
      - type: "trim"
      - type: "normalize_spaces"
  - name: "subheadline"
    selector: ".article-summary, .excerpt"
    type: "text"
    transform:
      - type: "trim"
  - name: "content"
    selector: ".article-content, .story-body"
    type: "text"
    required: true
    transform:
      - type: "remove_html"
      - type: "normalize_spaces"
  - name: "author"
    selector: ".author-name, .byline .author"
    type: "text"
  - name: "published_date"
    selector: "time[datetime]"
    type: "attr"
    attribute: "datetime"
    required: true
  - name: "category"
    selector: ".category, .section"
    type: "text"
    transform:
      - type: "lowercase"
  - name: "tags"
    selector: ".tags a"
    type: "list"
  - name: "image_url"
    selector: ".featured-image img"
    type: "attr"
    attribute: "src"
  - name: "word_count"
    selector: ".word-count"
    type: "text"
    transform:
      - type: "extract_numbers"
      - type: "parse_int"

pagination:
  type: "next_button"
  selector: ".pagination .next"
  max_pages: 20

output:
  format: "database"
  database:
    driver: "postgresql"
    host: "${DB_HOST}"
    port: 5432
    database: "news_db"
    username: "${DB_USER}"
    password: "${DB_PASSWORD}"
    table: "articles"
    batch_size: 100
    auto_create_table: true
    table_schema:
      headline: "VARCHAR(500)"
      subheadline: "TEXT"
      content: "TEXT"
      author: "VARCHAR(255)"
      published_date: "TIMESTAMP"
      category: "VARCHAR(100)"
      tags: "TEXT[]"
      image_url: "TEXT"
      word_count: "INTEGER"
      scraped_at: "TIMESTAMP DEFAULT CURRENT_TIMESTAMP"
```

### 2. Blog Post Aggregator

**Scenario**: Aggregate blog posts from multiple sources.

**Configuration** (`blog-aggregator.yaml`):

```yaml
name: "blog_aggregator"
base_url: "${BLOG_URL}"
rate_limit: "1s"

fields:
  - name: "title"
    selector: "h1, .post-title, .entry-title"
    type: "text"
    required: true
  - name: "excerpt"
    selector: ".excerpt, .post-excerpt, .entry-summary"
    type: "text"
    transform:
      - type: "remove_html"
      - type: "trim"
  - name: "read_time"
    selector: ".read-time, .reading-time"
    type: "text"
    transform:
      - type: "extract_numbers"
      - type: "parse_int"
  - name: "post_url"
    selector: ".post-title a, .read-more"
    type: "attr"
    attribute: "href"
    required: true

output:
  format: "yaml"
  file: "blog_posts_${DATE}.yaml"
  yaml:
    include_metadata: true
    sort_keys: true
```

## Real Estate

### 1. Property Listings Scraper

**Scenario**: Extract real estate listings with detailed information.

**Configuration** (`real-estate.yaml`):

```yaml
name: "real_estate_scraper"
base_url: "https://realestate-site.com/listings"
rate_limit: "4s"
timeout: "25s"

# Realistic browser headers
headers:
  Accept: "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8"
  Accept-Language: "en-US,en;q=0.5"
  Accept-Encoding: "gzip, deflate"
  DNT: "1"
  Connection: "keep-alive"

fields:
  - name: "listing_id"
    selector: "[data-listing-id]"
    type: "attr"
    attribute: "data-listing-id"
    required: true
  - name: "address"
    selector: ".property-address, .listing-address h1"
    type: "text"
    required: true
    transform:
      - type: "trim"
      - type: "normalize_spaces"
  - name: "price"
    selector: ".price, .listing-price"
    type: "text"
    required: true
    transform:
      - type: "clean_price"
      - type: "parse_float"
  - name: "bedrooms"
    selector: ".bedrooms, .bed-count"
    type: "text"
    transform:
      - type: "extract_numbers"
      - type: "parse_int"
  - name: "bathrooms"
    selector: ".bathrooms, .bath-count"
    type: "text"
    transform:
      - type: "extract_numbers"
      - type: "parse_float"
  - name: "square_feet"
    selector: ".sqft, .square-feet"
    type: "text"
    transform:
      - type: "extract_numbers"
      - type: "parse_int"
  - name: "lot_size"
    selector: ".lot-size"
    type: "text"
    transform:
      - type: "extract_numbers"
      - type: "parse_float"
  - name: "property_type"
    selector: ".property-type"
    type: "text"
    transform:
      - type: "lowercase"
  - name: "year_built"
    selector: ".year-built"
    type: "text"
    transform:
      - type: "extract_numbers"
      - type: "parse_int"
  - name: "description"
    selector: ".property-description"
    type: "text"
    transform:
      - type: "remove_html"
      - type: "trim"
  - name: "agent_name"
    selector: ".agent-name"
    type: "text"
  - name: "agent_phone"
    selector: ".agent-phone"
    type: "text"
  - name: "images"
    selector: ".property-images img"
    type: "list"
    attribute: "src"

pagination:
  type: "page_numbers"
  selector: ".pagination a"
  max_pages: 50

output:
  multiple: true
  outputs:
    - format: "csv"
      file: "listings_${DATE}.csv"
    - format: "excel"
      file: "listings_detailed_${DATE}.xlsx"
      excel:
        sheet_name: "Property Listings"
        auto_filter: true
        column_widths:
          address: 40
          description: 50
          price: 15
    - format: "json"
      file: "listings_full_${DATE}.json"
```

## Job Boards

### 1. Job Listings Scraper

**Scenario**: Extract job postings for market analysis.

**Configuration** (`job-board.yaml`):

```yaml
name: "job_board_scraper"
base_url: "https://jobboard.com/search?q=software+engineer"
rate_limit: "3s"
timeout: "20s"

# Anti-detection for professional sites
anti_detection:
  fingerprinting:
    enabled: true
    randomize_viewport: true
  tls:
    randomize_ja3: true

user_agents:
  - "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"
  - "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15"

fields:
  - name: "job_id"
    selector: "[data-job-id]"
    type: "attr"
    attribute: "data-job-id"
    required: true
  - name: "title"
    selector: ".job-title, h2.title"
    type: "text"
    required: true
    transform:
      - type: "trim"
  - name: "company"
    selector: ".company-name, .employer"
    type: "text"
    required: true
  - name: "location"
    selector: ".job-location, .location"
    type: "text"
  - name: "salary_range"
    selector: ".salary, .pay-range"
    type: "text"
    transform:
      - type: "trim"
  - name: "job_type"
    selector: ".job-type, .employment-type"
    type: "text"
    transform:
      - type: "lowercase"
  - name: "experience_level"
    selector: ".experience, .seniority-level"
    type: "text"
    transform:
      - type: "lowercase"
  - name: "description"
    selector: ".job-description"
    type: "text"
    transform:
      - type: "remove_html"
      - type: "trim"
  - name: "requirements"
    selector: ".requirements"
    type: "text"
    transform:
      - type: "remove_html"
  - name: "posted_date"
    selector: ".posted-date, time"
    type: "text"
  - name: "application_url"
    selector: ".apply-link"
    type: "attr"
    attribute: "href"
  - name: "skills"
    selector: ".skills .skill"
    type: "list"

pagination:
  type: "url_pattern"
  url_pattern: "https://jobboard.com/search?q=software+engineer&page={page}"
  start_page: 1
  max_pages: 20

output:
  format: "database"
  database:
    driver: "postgresql"
    table: "job_listings"
    batch_size: 500
    auto_create_table: true
```

## Advanced Scenarios

### 1. Multi-Site Comparison

**Scenario**: Compare data across multiple similar websites.

**Configuration Template** (`multi-site-template.yaml`):

```yaml
name: "${SITE_NAME}_scraper"
base_url: "${SITE_URL}"
rate_limit: "3s"

fields:
  - name: "title"
    selector: "${TITLE_SELECTOR}"
    type: "text"
    required: true
  - name: "price"
    selector: "${PRICE_SELECTOR}"
    type: "text"
    transform:
      - type: "clean_price"
      - type: "parse_float"
  - name: "source_site"
    value: "${SITE_NAME}"
    type: "static"

output:
  format: "json"
  file: "${SITE_NAME}_data.json"
```

**Usage Script**:

```bash
#!/bin/bash

# Site configurations
declare -A sites=(
    ["site1"]="https://site1.com|.product-title|.price"
    ["site2"]="https://site2.com|h1.title|.cost"
    ["site3"]="https://site3.com|.name|.amount"
)

# Run scrapers for each site
for site in "${!sites[@]}"; do
    IFS='|' read -r url title_sel price_sel <<< "${sites[$site]}"
    
    export SITE_NAME="$site"
    export SITE_URL="$url"
    export TITLE_SELECTOR="$title_sel"
    export PRICE_SELECTOR="$price_sel"
    
    datascrapexter run multi-site-template.yaml
done

# Combine results
jq -s 'map(.data) | add' *_data.json > combined_results.json
```

### 2. Dynamic Content Scraping

**Scenario**: Scrape JavaScript-heavy single-page applications.

**Configuration** (`spa-scraper.yaml`):

```yaml
name: "spa_scraper"
base_url: "https://spa-app.com"
rate_limit: "5s"

browser:
  enabled: true
  headless: true
  wait_for_element: ".content-loaded"
  wait_timeout: "30s"
  javascript_timeout: "10s"
  
  # Custom JavaScript execution
  custom_scripts:
    pre_navigation:
      - "window.localStorage.setItem('consent', 'accepted')"
    post_navigation:
      - "document.querySelector('.load-more')?.click()"
      - "window.scrollTo(0, document.body.scrollHeight)"
  
  # Wait after script execution
  wait_after_script: "3s"

fields:
  - name: "dynamic_content"
    selector: ".dynamic-data"
    type: "text"
    required: true
  - name: "ajax_loaded_data"
    selector: ".ajax-content"
    type: "list"

output:
  format: "json"
  file: "spa_data.json"
```

## Anti-Detection Examples

### 1. High-Security Site Scraping

**Scenario**: Scrape heavily protected sites with comprehensive anti-detection.

**Configuration** (`stealth-scraper.yaml`):

```yaml
name: "stealth_scraper"
base_url: "https://protected-site.com"
rate_limit: "8s"
timeout: "60s"
max_retries: 3

# Comprehensive anti-detection
anti_detection:
  fingerprinting:
    enabled: true
    canvas_spoofing: true
    webgl_spoofing: true
    audio_spoofing: true
    screen_spoofing: true
    font_spoofing: true
    randomize_viewport: true
    hardware_spoofing: true
    
  captcha:
    enabled: true
    service: "2captcha"
    api_key: "${CAPTCHA_API_KEY}"
    timeout: "180s"
    max_attempts: 5
    
  tls:
    randomize_ja3: true
    randomize_ja4: true
    profile_mode: "browser_simulation"

# Browser automation with stealth
browser:
  enabled: true
  headless: true
  stealth_mode: true
  user_data_dir: "/tmp/stealth_browser"
  disable_images: true
  extra_headers:
    Accept-Language: "en-US,en;q=0.9"

# Residential proxy rotation
proxy:
  enabled: true
  rotation: "random"
  health_check: true
  providers:
    - url: "${RESIDENTIAL_PROXY_1}"
      username: "${PROXY_USER}"
      password: "${PROXY_PASS}"
      type: "residential"
      max_concurrent: 1

# Conservative extraction
fields:
  - name: "protected_content"
    selector: ".main-content"
    type: "text"
    required: true

output:
  format: "json"
  file: "stealth_results.json"
```

## Monitoring and Production

### 1. Production Scraper with Full Monitoring

**Scenario**: Enterprise-grade scraper with comprehensive monitoring.

**Configuration** (`production-scraper.yaml`):

```yaml
name: "production_scraper"
base_url: "${TARGET_URL}"
rate_limit: "2s"

# Comprehensive monitoring
monitoring:
  metrics:
    enabled: true
    namespace: "datascrapexter"
    subsystem: "production"
    listen_address: ":9090"
    enable_go_metrics: true
    
  health:
    check_interval: "30s"
    health_endpoint: "/health"
    readiness_endpoint: "/ready"
    detailed_response: true
    
  dashboard:
    enabled: true
    port: ":8080"
    title: "Production Scraper Monitor"
    refresh_interval: "5s"

# Anti-detection
anti_detection:
  fingerprinting:
    enabled: true
  captcha:
    enabled: true
    service: "2captcha"
    api_key: "${CAPTCHA_API_KEY}"

# Robust configuration
max_retries: 5
retry_backoff: "exponential"

fields:
  - name: "data"
    selector: ".data-item"
    type: "text"
    required: true

# Multiple outputs for different consumers
output:
  multiple: true
  outputs:
    - format: "database"
      database:
        driver: "postgresql"
        table: "production_data"
        batch_size: 1000
        on_conflict: "update"
    - format: "json"
      file: "/data/backups/data_${TIMESTAMP}.json"
      json:
        compress: true
```

**Docker Compose for Production**:

```yaml
version: '3.8'
services:
  datascrapexter:
    image: valpere/datascrapexter:latest
    volumes:
      - ./configs:/configs
      - ./data:/data
    environment:
      - TARGET_URL=${TARGET_URL}
      - CAPTCHA_API_KEY=${CAPTCHA_API_KEY}
      - DB_PASSWORD=${DB_PASSWORD}
    ports:
      - "8080:8080"  # Dashboard
      - "9090:9090"  # Metrics
    command: run /configs/production-scraper.yaml
    
  prometheus:
    image: prom/prometheus:latest
    ports:
      - "9091:9090"
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml
      
  grafana:
    image: grafana/grafana:latest
    ports:
      - "3000:3000"
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=admin
```

## Usage Patterns

### Environment-Specific Configurations

**Development** (`config-dev.yaml`):

```yaml
name: "dev_scraper"
rate_limit: "5s"
max_retries: 1
output:
  format: "json"
  file: "dev_output.json"
```

**Staging** (`config-staging.yaml`):

```yaml
name: "staging_scraper"
rate_limit: "3s"
max_retries: 3
monitoring:
  metrics:
    enabled: true
output:
  format: "database"
  database:
    driver: "postgresql"
    table: "staging_data"
```

**Production** (`config-prod.yaml`):

```yaml
name: "prod_scraper"
rate_limit: "2s"
max_retries: 5
anti_detection:
  fingerprinting:
    enabled: true
monitoring:
  metrics:
    enabled: true
  health:
    enabled: true
  dashboard:
    enabled: true
output:
  multiple: true
  outputs:
    - format: "database"
    - format: "json"
```

These examples provide a comprehensive foundation for using DataScrapexter across various scenarios. Each example can be customized further based on specific requirements and target websites.
