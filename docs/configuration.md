# DataScrapexter Configuration Reference

## Overview

DataScrapexter uses YAML configuration files to define web scraping behavior. This reference document provides comprehensive information about all available configuration options, their syntax, and usage patterns. Understanding these options enables you to create efficient, reliable scrapers tailored to specific websites and data extraction requirements.

The configuration system follows a hierarchical structure where each section controls different aspects of the scraping process. From basic settings like the target URL to advanced features such as pagination and data transformation, every aspect of DataScrapexter's behavior can be customized through configuration files.

## Configuration File Structure

A DataScrapexter configuration file consists of several top-level sections, each serving a specific purpose in the scraping process. The fundamental structure includes metadata about the scraper, connection and request settings, data extraction rules, pagination configuration, and output specifications. Understanding this structure is essential for creating effective configurations.

The YAML format was chosen for its human readability and support for complex data structures. Comments can be added using the hash symbol (#) to document your configuration choices and provide context for future maintenance. While YAML is sensitive to indentation, this requirement ensures clear visual hierarchy in configuration files.

## Root Configuration Options

### name (required)

The name field serves as a unique identifier for your scraper configuration. This name appears in logs, output files, and monitoring dashboards, making it essential for distinguishing between multiple scrapers in your system. Choose descriptive names that clearly indicate the scraper's purpose, such as "amazon_electronics_monitor" or "news_site_daily_scraper".

```yaml
name: "product_price_monitor"
```

### base_url (required)

The base_url specifies the starting point for your scraping operation. This should be a complete, valid URL including the protocol (http or https). DataScrapexter uses this URL as the initial request target and as a reference point for resolving relative URLs found during scraping.

```yaml
base_url: "https://example.com/products/category/electronics"
```

Environment variables can be used for dynamic URL configuration, particularly useful when the same configuration needs to target different URLs based on deployment environment:

```yaml
base_url: "${TARGET_URL}"
```

## Request Configuration

### rate_limit

The rate_limit setting controls the minimum time between consecutive requests to prevent overwhelming target servers. This value accepts duration strings in Go's time format, supporting units from nanoseconds to hours. Common values include "1s" for one second, "500ms" for half a second, or "2s" for two seconds.

```yaml
rate_limit: "2s"  # Wait 2 seconds between requests
```

Appropriate rate limiting demonstrates responsible scraping practices and helps avoid IP blocking or rate limit errors from target websites. Consider the website's size and infrastructure when setting this value, with smaller sites generally requiring more conservative limits.

### timeout

The timeout parameter defines the maximum duration DataScrapexter will wait for a server response before considering the request failed. This prevents the scraper from hanging indefinitely on slow or unresponsive servers. The value uses the same duration format as rate_limit.

```yaml
timeout: "30s"  # Maximum 30 seconds per request
```

Setting appropriate timeouts balances reliability with efficiency. Shorter timeouts may cause false failures on slow servers, while longer timeouts can significantly impact overall scraping performance when encountering multiple slow responses.

### max_retries

This setting determines how many times DataScrapexter will retry a failed request before giving up. Retries use exponential backoff, meaning the delay between attempts increases progressively. This approach helps handle temporary failures while avoiding excessive load on struggling servers.

```yaml
max_retries: 3  # Retry failed requests up to 3 times
```

The retry mechanism handles various failure scenarios including network timeouts, server errors (5xx status codes), and rate limit responses (429 status code). Each retry attempt is logged for debugging purposes.

### headers

Custom HTTP headers can be specified to mimic browser behavior more closely or to meet specific requirements of the target website. Headers are defined as key-value pairs and are included in every request made by the scraper.

```yaml
headers:
  Accept: "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8"
  Accept-Language: "en-US,en;q=0.9"
  Accept-Encoding: "gzip, deflate, br"
  Cache-Control: "no-cache"
  DNT: "1"
  Referer: "https://www.google.com/"
```

Common headers include Accept for content type preferences, Accept-Language for localization, Referer to indicate traffic source, and custom headers required by specific websites. Be cautious not to include sensitive information like authentication tokens directly in configuration files.

### user_agents

User agent rotation helps scrapers appear more like regular browser traffic. When specified, DataScrapexter randomly selects from the provided list for each request, distributing requests across different browser signatures.

```yaml
user_agents:
  - "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"
  - "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36"
  - "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36"
```

If not specified, DataScrapexter uses a built-in list of common user agents. Custom lists allow you to target specific browser versions or platforms if the website responds differently to various user agents.

## Proxy Configuration

### proxy

Proxy configuration enables request routing through intermediate servers, useful for distributing requests across multiple IP addresses or accessing geo-restricted content. The proxy section supports both single proxy and multiple proxy configurations.

```yaml
proxy:
  enabled: true
  url: "http://username:password@proxy.example.com:8080"
```

For multiple proxies with rotation:

```yaml
proxy:
  enabled: true
  rotation: "random"  # or "round-robin"
  list:
    - "http://proxy1.example.com:8080"
    - "http://proxy2.example.com:8080"
    - "http://proxy3.example.com:8080"
```

Proxy URLs should include authentication credentials if required. Environment variables are recommended for sensitive proxy information to avoid storing credentials in configuration files.

## Data Extraction Configuration

### fields

The fields section defines what data to extract from web pages. Each field represents a specific piece of information you want to collect, with its own extraction rules and processing options. This is the heart of your scraper configuration, determining what data will be collected and how it will be processed.

Each field requires several properties that control extraction behavior:

```yaml
fields:
  - name: "product_title"
    selector: "h1.product-name"
    type: "text"
    required: true
    transform:
      - type: "trim"
      - type: "normalize_spaces"
```

### Field Properties

#### name (required)

The name property provides a unique identifier for the extracted data field. This name appears as the key in output files and should be descriptive enough to understand the data's purpose. Use consistent naming conventions across your configurations, such as snake_case or camelCase.

#### selector (required)

The selector property uses CSS selector syntax to identify HTML elements containing the desired data. DataScrapexter supports the full range of CSS selectors including class selectors (.classname), ID selectors (#id), attribute selectors ([attribute="value"]), pseudo-selectors (:first-child, :nth-of-type), and combinators (>, +, ~).

Complex selectors can target specific elements within nested structures:

```yaml
selector: "div.product-details > span.price:first-child"
```

Multiple selectors can be specified as comma-separated values, with DataScrapexter using the first matching element:

```yaml
selector: ".price-now, .current-price, span[itemprop='price']"
```

#### type (required)

The type property determines how data is extracted from selected elements. DataScrapexter supports several extraction types, each suited to different scenarios.

The "text" type extracts the text content of elements, automatically handling nested tags and returning clean text. This is the most common extraction type for content like titles, descriptions, and prices.

The "html" type retrieves the raw HTML content of selected elements, preserving all markup. This is useful when you need to maintain formatting or perform additional processing on structured content.

The "attr" type extracts specific attribute values from elements. This requires the additional "attribute" property to specify which attribute to extract. Common uses include extracting href values from links or src values from images.

The "list" type collects data from all matching elements as an array. This is essential for extracting multiple items like product listings, article summaries, or navigation links.

```yaml
fields:
  - name: "image_urls"
    selector: "img.product-image"
    type: "list"
    attribute: "src"
```

#### attribute (conditional)

Required when type is "attr", this property specifies which HTML attribute to extract. Common attributes include href for links, src for images and scripts, alt for image descriptions, data-* for custom data attributes, and title for tooltip text.

#### required (optional)

When set to true, the field is considered mandatory. If a required field cannot be extracted, the entire scraping operation for that page is marked as failed. This helps ensure data quality by catching structural changes early.

```yaml
required: true  # Fail if this field cannot be extracted
```

Use required fields judiciously for critical data while allowing optional fields for supplementary information that may not always be present.

#### transform (optional)

The transform property defines a sequence of data processing operations applied to extracted values. Transformations execute in order, with each transformation receiving the output of the previous one. This pipeline approach enables complex data cleaning and normalization.

```yaml
transform:
  - type: "trim"
  - type: "regex"
    pattern: "\\$([0-9,]+\\.?[0-9]*)"
    replacement: "$1"
  - type: "parse_float"
```

## Transformation Rules

### Available Transformation Types

DataScrapexter provides a comprehensive set of transformation rules for data processing. Each transformation serves a specific purpose in cleaning, normalizing, or converting extracted data.

Text transformations modify string content without changing its fundamental meaning. The "trim" transformation removes leading and trailing whitespace, essential for cleaning data extracted from HTML. The "lowercase" and "uppercase" transformations standardize text case for consistent storage and comparison. The "normalize_spaces" transformation replaces multiple consecutive spaces with single spaces, cleaning up poorly formatted text.

Pattern matching transformations use regular expressions for complex text processing. The "regex" transformation requires "pattern" and "replacement" parameters, enabling sophisticated find-and-replace operations. Regular expressions follow Go's RE2 syntax, providing powerful pattern matching while maintaining predictable performance.

Numeric transformations convert text to numbers for mathematical operations and proper data typing. The "parse_int" transformation converts strings to integers, automatically removing common formatting like commas. The "parse_float" transformation handles decimal numbers, supporting various international formats. The "clean_price" transformation specifically targets currency values, extracting numeric amounts from formatted price strings.

Data extraction transformations pull specific information from larger text blocks. The "extract_numbers" transformation finds all numeric values in text, useful for parsing specifications or measurements. The "remove_html" transformation strips HTML tags while preserving text content, cleaning up rich text fields.

### Transformation Examples

Price extraction often requires multiple transformations to handle various formats:

```yaml
transform:
  - type: "clean_price"      # Extracts numeric price from text
  - type: "parse_float"      # Converts to floating-point number
```

Text normalization ensures consistent data storage:

```yaml
transform:
  - type: "trim"             # Remove extra whitespace
  - type: "lowercase"        # Standardize case
  - type: "normalize_spaces" # Clean internal spacing
```

Complex pattern extraction using regular expressions:

```yaml
transform:
  - type: "regex"
    pattern: "SKU:\\s*([A-Z0-9]+)"
    replacement: "$1"
  - type: "uppercase"        # Ensure consistent SKU format
```

## Pagination Configuration

### pagination

Pagination configuration enables DataScrapexter to automatically navigate through multiple pages of results. This is essential for collecting complete datasets from sites that split content across pages. The pagination section supports multiple strategies to handle different implementation patterns.

```yaml
pagination:
  type: "next_button"
  selector: ".pagination .next"
  max_pages: 10
```

### Pagination Types

#### next_button

The next_button type follows "next page" links by locating and clicking navigation elements. This approach works well for blogs, news sites, and simple product listings where each page contains a link to the subsequent page.

```yaml
pagination:
  type: "next_button"
  selector: "a.next-page:not(.disabled)"
  max_pages: 50
```

The selector should target the clickable element that navigates to the next page. Include pseudo-selectors like :not(.disabled) to avoid clicking inactive buttons.

#### page_numbers

The page_numbers type handles numbered pagination by identifying and following page number links. This approach suits sites with visible page numbers in their navigation.

```yaml
pagination:
  type: "page_numbers"
  selector: ".pagination a"
  max_pages: 20
```

DataScrapexter automatically detects numeric patterns in pagination links and follows them sequentially.

#### url_pattern

The url_pattern type generates URLs based on a template pattern. This is the most reliable approach when URL structures follow predictable patterns.

```yaml
pagination:
  type: "url_pattern"
  url_pattern: "https://example.com/products?page={page}"
  start_page: 1
  max_pages: 100
```

Supported placeholders include {page} for page numbers and {offset} for result offsets. The start_page parameter defines where pagination begins, typically 1 or 0 depending on the site's implementation.

### max_pages

The max_pages parameter limits the number of pages DataScrapexter will process, preventing infinite crawling and controlling resource usage. Set this value based on expected content volume and practical requirements.

```yaml
max_pages: 25  # Stop after processing 25 pages
```

This limit includes the initial page, so max_pages: 1 processes only the starting URL without pagination.

## Output Configuration

### output

The output section determines how and where scraped data is saved. DataScrapexter supports multiple output formats, each suited to different use cases and downstream processing requirements.

```yaml
output:
  format: "json"
  file: "scraped_data.json"
```

### Output Formats

#### JSON Output

JSON format provides structured data output ideal for programmatic processing. The output includes all extracted fields, metadata about the scraping operation, and maintains data type information.

```yaml
output:
  format: "json"
  file: "products.json"
```

JSON output preserves nested structures and arrays, making it suitable for complex data relationships. The file is formatted with indentation for human readability while remaining valid for machine parsing.

#### CSV Output

CSV format creates tabular data suitable for spreadsheet applications and data analysis tools. DataScrapexter automatically generates headers from field names and flattens nested structures.

```yaml
output:
  format: "csv"
  file: "products.csv"
```

Array fields are joined with semicolons by default, and special characters are properly escaped according to CSV standards. This format works best for flat data structures without deep nesting.

#### Excel Output (Future)

Excel output support is planned for future releases, providing native spreadsheet format with multiple sheets, formatting, and formulas.

### file

The file parameter specifies where output data should be written. Paths can be absolute or relative to the current working directory. Special values include "-" for stdout output and empty string for console display only.

```yaml
file: "outputs/daily_scrape.json"    # Relative path
file: "/var/data/scraper/output.csv" # Absolute path
file: "-"                            # Output to stdout
```

Environment variables can be used for dynamic file naming:

```yaml
file: "${OUTPUT_DIR}/scrape_${DATE}.json"
```

### database (Future)

Database output configuration is planned for future releases, enabling direct writing to PostgreSQL, MySQL, MongoDB, and other database systems.

## Advanced Configuration Patterns

### Multi-Site Configuration

While each configuration file typically targets a single website, you can create template configurations for similar sites using environment variables:

```yaml
name: "multi_site_scraper"
base_url: "${SITE_URL}"

fields:
  - name: "title"
    selector: "${TITLE_SELECTOR}"
    type: "text"
```

This approach enables reusing configurations across similar websites with minor variations.

### Conditional Extraction

For sites with varying structures, use multiple selectors with fallbacks:

```yaml
fields:
  - name: "price"
    selector: ".sale-price, .regular-price, .price, span[itemprop='price']"
    type: "text"
    required: true
```

DataScrapexter tries each selector in order until finding a match.

### Development vs Production Configurations

Maintain separate configurations for different environments:

```yaml
# config-dev.yaml
rate_limit: "5s"      # Slower for development
max_retries: 1        # Fail fast during testing
output:
  format: "json"
  file: "test-output.json"

# config-prod.yaml  
rate_limit: "1s"      # Faster for production
max_retries: 5        # More resilient
output:
  format: "csv"
  file: "${OUTPUT_DIR}/production-${TIMESTAMP}.csv"
```

## Configuration Validation

DataScrapexter performs comprehensive validation before executing scrapers. Validation checks include required fields presence, data type correctness, selector syntax validity, transformation rule parameters, and logical consistency.

Run validation without scraping using:

```bash
datascrapexter validate config.yaml
```

Validation errors provide specific information about configuration problems, including line numbers and suggested fixes.

## Best Practices

Effective configuration requires balancing completeness with maintainability. Start with minimal configurations and add complexity as needed. Document your configurations with comments explaining selector choices and transformation logic. Use descriptive field names that clearly indicate data content.

Test configurations thoroughly on sample pages before full-scale deployment. Monitor scraper output regularly to detect structural changes requiring configuration updates. Version control your configuration files to track changes and enable rollbacks.

Consider creating configuration templates for common website types, reducing duplication and ensuring consistency. Use environment variables for sensitive information and deployment-specific values. Regular review and refactoring of configurations helps maintain efficiency and reliability.

## Troubleshooting Configuration Issues

Common configuration problems often stem from incorrect selector syntax or website structure changes. Use browser developer tools to verify selectors before adding them to configurations. Test individual field extraction before combining into complex configurations.

When fields extract unexpected data, examine the HTML structure carefully. Websites often have multiple elements matching simple selectors, requiring more specific targeting. Use pseudo-selectors and combinators to precisely identify desired elements.

Transformation failures typically indicate data format mismatches. Verify that transformation types match the data being processed. Chain transformations carefully, ensuring each step's output matches the next step's input requirements.

Performance issues may arise from overly broad selectors or excessive retries. Optimize selectors for specificity and adjust rate limits based on server responses. Monitor logs to identify bottlenecks and adjust configurations accordingly.

This configuration reference serves as a comprehensive guide to DataScrapexter's capabilities. Regular consultation during configuration development ensures optimal scraper performance and reliability.
