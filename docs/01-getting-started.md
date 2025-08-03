# DataScrapexter - Getting Started Guide

## Overview

DataScrapexter is a professional web scraping platform built with Go 1.24+ that combines high performance, intelligent anti-detection mechanisms, and configuration-driven operation to enable seamless data extraction from any website structure.

## Table of Contents

1. [Installation](#installation)
2. [Quick Start](#quick-start)
3. [Basic Configuration](#basic-configuration)
4. [Your First Scraper](#your-first-scraper)
5. [Command Line Usage](#command-line-usage)
6. [Next Steps](#next-steps)

## Installation

### Prerequisites

- Go 1.24 or later
- Git
- Chrome/Chromium browser (for JavaScript-heavy sites)

### Binary Installation

Download the latest release from GitHub:

```bash
# Download and install
curl -L https://github.com/valpere/DataScrapexter/releases/latest/download/datascrapexter-linux-amd64 \
  -o /usr/local/bin/datascrapexter
chmod +x /usr/local/bin/datascrapexter

# Verify installation
datascrapexter version
```

### From Source

```bash
# Clone repository
git clone https://github.com/valpere/DataScrapexter.git
cd DataScrapexter

# Build from source
make build

# Install globally
sudo make install
```

### Docker Installation

```bash
# Pull official image
docker pull valpere/datascrapexter:latest

# Run with Docker
docker run --rm -v $(pwd)/config:/config -v $(pwd)/output:/output \
  valpere/datascrapexter:latest run /config/scraper.yaml
```

## Quick Start

### 1. Create Your First Configuration

Create a simple configuration file `quotes.yaml`:

```yaml
name: "quotes_scraper"
base_url: "http://quotes.toscrape.com/"
rate_limit: "2s"
timeout: "10s"

fields:
  - name: "quote"
    selector: ".quote .text"
    type: "text"
    required: true
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

### 2. Run Your First Scraper

```bash
# Validate configuration
datascrapexter validate quotes.yaml

# Run the scraper
datascrapexter run quotes.yaml

# Check the output
cat quotes.json
```

### 3. View Results

Your `quotes.json` file will contain:

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

## Basic Configuration

### Configuration File Structure

DataScrapexter uses YAML configuration files with the following structure:

```yaml
# Required fields
name: "scraper_name"           # Unique identifier
base_url: "https://example.com" # Starting URL

# Optional request settings
rate_limit: "2s"               # Delay between requests
timeout: "30s"                 # Request timeout
max_retries: 3                 # Retry attempts

# Data extraction rules
fields:
  - name: "field_name"         # Output field name
    selector: ".css-selector"   # CSS selector
    type: "text"               # Extraction type
    required: true             # Whether field is mandatory

# Output configuration
output:
  format: "json"               # Output format
  file: "output.json"          # Output file
```

### Common Field Types

- **text**: Extract text content from elements
- **attr**: Extract attribute values (requires `attribute` field)
- **html**: Extract raw HTML content
- **list**: Extract from all matching elements as array

### CSS Selectors

DataScrapexter supports standard CSS selectors:

```yaml
fields:
  # Class selectors
  - name: "title"
    selector: ".product-title"
    type: "text"
    
  # ID selectors
  - name: "price"
    selector: "#price-display"
    type: "text"
    
  # Attribute selectors
  - name: "product_id"
    selector: "[data-product-id]"
    type: "attr"
    attribute: "data-product-id"
    
  # Pseudo-selectors
  - name: "first_review"
    selector: ".review:first-child .text"
    type: "text"
```

## Your First Scraper

Let's build a more comprehensive scraper for an e-commerce site:

### 1. Product Information Scraper

Create `products.yaml`:

```yaml
name: "product_scraper"
base_url: "https://example-store.com/products"
rate_limit: "3s"
timeout: "30s"
max_retries: 3

# Headers to appear more like a browser
headers:
  Accept: "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8"
  Accept-Language: "en-US,en;q=0.9"
  User-Agent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"

fields:
  - name: "title"
    selector: "h1.product-title, .product-name"
    type: "text"
    required: true
    transform:
      - type: "trim"
      - type: "normalize_spaces"
      
  - name: "price"
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
      
  - name: "availability"
    selector: ".stock-status, .availability"
    type: "text"
    transform:
      - type: "lowercase"
      - type: "trim"
      
  - name: "rating"
    selector: ".rating-value, [data-rating]"
    type: "text"
    transform:
      - type: "parse_float"
      
  - name: "image_url"
    selector: ".product-image img"
    type: "attr"
    attribute: "src"

pagination:
  type: "next_button"
  selector: ".pagination .next"
  max_pages: 5

output:
  format: "csv"
  file: "products_${DATE}.csv"
  csv:
    include_headers: true
    delimiter: ","
```

### 2. Run with Environment Variables

```bash
# Set date variable
export DATE=$(date +%Y-%m-%d)

# Run the scraper
datascrapexter run products.yaml
```

### 3. Advanced Features

Add monitoring and error handling:

```yaml
# Add to products.yaml
monitoring:
  metrics:
    enabled: true
    listen_address: ":9090"
  health:
    check_interval: "30s"

# Error handling
max_retries: 5
retry_backoff: "exponential"

# Rate limiting
rate_limit: "3s"
random_delay: "1s"  # Add random delay up to 1s
```

## Command Line Usage

### Basic Commands

```bash
# Validate configuration
datascrapexter validate config.yaml

# Run scraper
datascrapexter run config.yaml

# Generate template
datascrapexter template --type ecommerce > new-config.yaml

# Check version
datascrapexter version

# Get help
datascrapexter help
```

### Advanced Commands

```bash
# Run with custom output
datascrapexter run config.yaml --output custom-output.json

# Run with verbose logging
datascrapexter run config.yaml --verbose

# Run in server mode
datascrapexter serve --port 8080

# Test proxy configuration
datascrapexter test-proxy --config config.yaml

# View metrics
datascrapexter metrics --format json
```

### Configuration Validation

```bash
# Validate syntax
datascrapexter validate config.yaml

# Validate with detailed output
datascrapexter validate config.yaml --verbose

# Test selectors against live page
datascrapexter test-selectors config.yaml --url "https://example.com"
```

## Next Steps

### 1. Learn Advanced Features

- **[Configuration Reference](02-configuration-reference.md)**: Complete configuration options
- **[Output Formats](03-output-formats.md)**: JSON, CSV, Excel, XML, YAML, databases
- **[Anti-Detection](04-anti-detection.md)**: Bypass blocks and CAPTCHAs
- **[Monitoring](05-monitoring.md)**: Metrics, health checks, dashboards

### 2. Explore Examples

- **[Examples Collection](06-examples.md)**: Real-world scraping scenarios
- E-commerce product monitoring
- News article collection
- Real estate listings
- Job board scraping

### 3. Production Deployment

- **[API Reference](07-api-reference.md)**: Go programming interface
- **[Docker & Kubernetes](08-deployment.md)**: Container deployment
- **[Troubleshooting](09-troubleshooting.md)**: Common issues and solutions

### 4. Best Practices

- Always respect robots.txt files
- Implement appropriate rate limiting
- Handle errors gracefully
- Monitor scraper performance
- Keep configurations version controlled

### 5. Community and Support

- GitHub Repository: https://github.com/valpere/DataScrapexter
- Issues and Bug Reports: Use GitHub Issues
- Feature Requests: GitHub Discussions
- Documentation: https://valpere.github.io/datascrapexter/

## Common First Steps

1. **Start Simple**: Begin with basic text extraction
2. **Add Transformations**: Clean and normalize your data
3. **Handle Pagination**: Extract data from multiple pages
4. **Implement Monitoring**: Track scraper performance
5. **Add Anti-Detection**: Avoid blocks and rate limits
6. **Scale Up**: Deploy to production with proper infrastructure

This getting started guide should have you up and running with DataScrapexter quickly. For more advanced topics, continue to the detailed reference documentation.