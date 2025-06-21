# DataScrapexter Configuration Templates Guide

## Overview

DataScrapexter provides sophisticated configuration templates for common web scraping scenarios across various industries. These templates serve as comprehensive starting points that can be customized for specific needs while demonstrating best practices and advanced features.

## Available Templates

### 1. E-commerce Product Monitoring (`ecommerce`)

The e-commerce template provides extensive configuration for monitoring online retail platforms, including:

- **Product catalog scraping** with price tracking and availability monitoring
- **Multi-variant product handling** for sizes, colors, and other options
- **Inventory tracking** with low-stock alerts
- **Price change detection** with configurable thresholds
- **Review and rating extraction**
- **Seller information tracking** for marketplace platforms
- **Promotional offer detection**

**Key Features:**
- Intelligent price normalization across different currency formats
- Automatic detection of product variants and options
- Image URL extraction with high-resolution preference
- Shipping information parsing
- Dynamic pricing detection (member prices, bulk discounts)

**Use Cases:**
- Competitive price monitoring
- Product availability tracking
- Market research and analysis
- Inventory management
- Price optimization strategies

### 2. Real Estate Listings (`real-estate`)

The real estate template specializes in property listing extraction with market analysis capabilities:

- **Property details extraction** including MLS numbers, addresses, and specifications
- **Pricing analytics** with per-square-foot calculations
- **Agent and brokerage information**
- **School district and neighborhood data**
- **Property feature extraction** from descriptions
- **Historical price tracking**
- **Map-based property data extraction**

**Key Features:**
- Address parsing and standardization
- Automated geocoding support
- Multiple property type handling
- HOA and tax information extraction
- Virtual tour URL detection
- Market trend analysis

**Use Cases:**
- Real estate market analysis
- Property investment research
- Competitive market analysis (CMA)
- Lead generation for agents
- Housing market trend tracking

### 3. Job Listings and Recruitment (`job-listings`)

The job listings template provides comprehensive job market monitoring:

- **Job posting extraction** with structured data parsing
- **Salary range detection** and normalization
- **Skills extraction** and categorization
- **Company information enrichment**
- **Remote work classification**
- **Application tracking**
- **Benefits parsing and categorization**

**Key Features:**
- Intelligent salary parsing (hourly, annual, ranges)
- Skills taxonomy mapping
- Experience level classification
- Location parsing with remote work detection
- Industry and job function categorization
- Security clearance requirement detection

**Use Cases:**
- Talent acquisition and recruiting
- Salary benchmarking
- Skills demand analysis
- Job market trend analysis
- Competitive intelligence for HR

### 4. News and Media Monitoring (`news-media`)

The news monitoring template handles multi-source news aggregation:

- **Article extraction** from news websites and RSS feeds
- **Author and publication metadata**
- **Content categorization** and tagging
- **Sentiment analysis** integration
- **Entity extraction** (people, organizations, locations)
- **Related article detection**
- **Multimedia content handling**

**Key Features:**
- Multi-language support
- Publication date parsing and normalization
- Paywall detection
- Quote extraction
- Source attribution tracking
- Real-time monitoring capabilities
- Duplicate article detection

**Use Cases:**
- Brand monitoring and PR
- Competitive intelligence
- Market research
- Crisis management
- Trend analysis

### 5. Social Media Monitoring (`social-media`)

The social media template provides multi-platform social monitoring:

- **Cross-platform post extraction** (Twitter, Instagram, LinkedIn, Reddit, YouTube)
- **Engagement metrics tracking**
- **Influencer identification**
- **Hashtag and mention monitoring**
- **Sentiment and emotion analysis**
- **Crisis detection alerts**
- **Competitive tracking**

**Key Features:**
- Platform-specific API and web scraping
- Real-time streaming support
- Influencer scoring algorithms
- Viral content detection
- Multi-language sentiment analysis
- Media content analysis (images, videos)

**Use Cases:**
- Brand reputation monitoring
- Influencer marketing
- Social media analytics
- Crisis management
- Campaign performance tracking

## Using Templates

### Basic Usage

Generate a template using the DataScrapexter CLI:

```bash
# List available templates
datascrapexter template --list

# Generate a specific template
datascrapexter template --type ecommerce > my-ecommerce-config.yaml

# Generate template with output file
datascrapexter template --type real-estate -o property-monitor.yaml
```

### Customizing Templates

Templates use environment variables for easy customization:

```bash
# Set environment variables
export PRODUCT_LISTING_URL="https://example.com/products"
export BRAND_NAME="MyBrand"
export NEWS_SITE_URL="https://news.example.com"

# Run with customized template
datascrapexter run my-config.yaml
```

### Template Structure

All templates follow a consistent structure:

1. **Target Configuration**: URL patterns and request settings
2. **Authentication**: Login credentials and API keys
3. **Anti-detection**: Browser automation and request randomization
4. **Extraction Rules**: Field definitions and transformations
5. **Data Processing**: Validation and enrichment
6. **Output Configuration**: Format and storage options
7. **Monitoring**: Alerts and notifications
8. **Compliance**: Legal and ethical settings

## Advanced Features

### Dynamic Field Extraction

Templates support complex field extraction patterns:

```yaml
fields:
  - name: "price"
    selector: ".price-now, .sale-price"
    transform:
      - type: "extract_number"
      - type: "parse_currency"
    validation:
      min: 0
      max: 999999
```

### Conditional Logic

Templates can include conditional extraction:

```yaml
fields:
  - name: "availability"
    selector: ".stock-status"
    conditions:
      - if: "contains('in stock')"
        value: "available"
      - if: "contains('out of')"
        value: "unavailable"
      - else: "unknown"
```

### Data Enrichment

Templates support post-processing enrichment:

```yaml
enrichment:
  - type: "geocoding"
    fields: ["address"]
  - type: "sentiment_analysis"
    fields: ["description", "reviews"]
```

### Multi-Stage Extraction

Templates can define multi-stage scraping workflows:

```yaml
extraction:
  - name: "listing_page"
    type: "listing"
    fields: [...]
    
  - name: "detail_page"
    type: "detail"
    follow_field: "product_url"
    fields: [...]
```

## Best Practices

### 1. Start with the Right Template

Choose the template that most closely matches your use case. It's easier to modify an existing template than to start from scratch.

### 2. Use Environment Variables

Keep sensitive information like API keys and URLs in environment variables:

```yaml
auth:
  api_key: "${API_KEY}"
target:
  url: "${TARGET_URL}"
```

### 3. Enable Monitoring

Always configure monitoring for production deployments:

```yaml
monitoring:
  alerts:
    - type: "data_quality"
      threshold: 0.95
    - type: "extraction_failure"
      max_failures: 5
```

### 4. Implement Rate Limiting

Respect target websites by implementing appropriate rate limiting:

```yaml
request:
  rate_limit:
    requests_per_second: 2
    respect_robots_txt: true
```

### 5. Handle Errors Gracefully

Configure retry logic and error handling:

```yaml
request:
  retry:
    attempts: 3
    delay: 5
    backoff: "exponential"
```

## Extending Templates

### Creating Custom Templates

To create your own template:

1. Start with an existing template
2. Modify extraction rules for your target site
3. Add custom transformations
4. Configure appropriate output format
5. Save as a reusable configuration

### Sharing Templates

Templates can be shared within teams:

```bash
# Save to shared location
datascrapexter template --type ecommerce > configs/shared/ecommerce-base.yaml

# Version control templates
git add configs/
git commit -m "Add customized e-commerce template"
```

## Troubleshooting

### Common Issues

1. **Missing Required Fields**
   - Check selector accuracy
   - Verify page structure hasn't changed
   - Enable debug logging

2. **Rate Limiting**
   - Reduce requests per second
   - Add delays between requests
   - Use proxy rotation

3. **Dynamic Content**
   - Enable browser mode
   - Add wait conditions
   - Use JavaScript execution

### Debug Mode

Run templates in debug mode for detailed logging:

```bash
datascrapexter run config.yaml --debug --verbose
```

## Template Maintenance

### Updating Selectors

Regularly review and update CSS selectors:

```bash
# Validate configuration
datascrapexter validate config.yaml --strict

# Test specific selectors
datascrapexter test-selector ".product-price" --url "https://example.com"
```

### Version Control

Track template changes:

```yaml
# Add version information
name: "ecommerce-monitor"
version: "1.2.0"
changelog:
  - version: "1.2.0"
    changes: "Updated price selector for new site design"
```

## Performance Optimization

### Caching

Enable caching for better performance:

```yaml
cache:
  enabled: true
  ttl: 3600
  storage: "redis"
```

### Parallel Processing

Configure parallel extraction:

```yaml
performance:
  parallel_requests: 5
  batch_size: 100
```

## Integration Examples

### With Monitoring Scripts

Templates work seamlessly with DataScrapexter scripts:

```bash
# Daily product monitoring
./scripts/daily_scrape.sh ecommerce-config.yaml

# Price change analysis
./scripts/analyze_price_changes.pl --config ecommerce-config.yaml
```

### With Data Processing

Combine templates with data processing:

```bash
# Extract data
datascrapexter run job-listings-config.yaml

# Process results
./scripts/combine_data.pl --sources "outputs/jobs/*.json"
```

## Conclusion

DataScrapexter's configuration templates provide powerful, industry-specific starting points for web scraping projects. By understanding and customizing these templates, you can quickly deploy sophisticated scraping solutions while following best practices for performance, reliability, and compliance.
