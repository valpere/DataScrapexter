# DataScrapexter User Guide

## Table of Contents

1. [Introduction](#introduction)
2. [Installation](#installation)
3. [Getting Started](#getting-started)
4. [Configuration Guide](#configuration-guide)
5. [Advanced Features](#advanced-features)
6. [Troubleshooting](#troubleshooting)
7. [Best Practices](#best-practices)
8. [Legal and Ethical Considerations](#legal-and-ethical-considerations)

## Introduction

DataScrapexter is a high-performance web scraping tool built with Go that combines powerful data extraction capabilities with sophisticated anti-detection mechanisms. This guide will walk you through everything you need to know to effectively use DataScrapexter for your web scraping needs.

### Key Features

DataScrapexter offers configuration-driven operation, eliminating the need for programming knowledge for basic scraping tasks. The tool includes built-in anti-detection measures such as user agent rotation and rate limiting to ensure reliable operation. It supports multiple output formats including JSON and CSV, making it easy to integrate scraped data into your workflows. The pagination support enables scraping of multi-page websites automatically, while the data transformation pipeline allows you to clean and format extracted data according to your needs.

### Use Cases

DataScrapexter excels in various scenarios including e-commerce price monitoring, news article aggregation, job listing collection, real estate data gathering, and research data collection. The tool is designed to handle both simple static websites and complex dynamic sites with equal efficiency.

## Installation

### Prerequisites

Before installing DataScrapexter, ensure you have Go 1.24 or later installed on your system. You can verify your Go installation by running `go version` in your terminal. Additionally, Git is required for cloning the repository and managing version control.

### Installation Methods

#### From Source

To install DataScrapexter from source, first clone the repository:

```bash
git clone https://github.com/valpere/DataScrapexter.git
cd DataScrapexter
```

Then install the required dependencies and build the application:

```bash
make deps
make build
```

The binary will be created in the `./bin` directory. You can install it system-wide using:

```bash
make install
```

#### Using Go Install

For a quick installation using Go's package manager:

```bash
go install github.com/valpere/DataScrapexter/cmd/datascrapexter@latest
```

#### Docker Installation

If you prefer using Docker, you can pull the pre-built image:

```bash
docker pull ghcr.io/valpere/datascrapexter:latest
```

### Verification

After installation, verify that DataScrapexter is properly installed by checking its version:

```bash
datascrapexter version
```

This should display the version information along with build details.

## Getting Started

### Your First Scraping Task

Let's begin with a simple example to understand how DataScrapexter works. We'll scrape quotes from a test website.

#### Step 1: Create a Configuration File

Create a new file named `quotes.yaml` with the following content:

```yaml
name: "quotes_scraper"
base_url: "http://quotes.toscrape.com/"

# Rate limiting to be respectful
rate_limit: "1s"
timeout: "30s"

# Define what data to extract
fields:
  - name: "quotes"
    selector: ".quote .text"
    type: "list"
    required: true

  - name: "authors"
    selector: ".quote .author"
    type: "list"

  - name: "tags"
    selector: ".quote .tags .tag"
    type: "list"

# Output configuration
output:
  format: "json"
  file: "quotes.json"
```

#### Step 2: Validate the Configuration

Before running the scraper, validate your configuration:

```bash
datascrapexter validate quotes.yaml
```

You should see a success message if the configuration is valid.

#### Step 3: Run the Scraper

Execute the scraping operation:

```bash
datascrapexter run quotes.yaml
```

DataScrapexter will process the website and save the results to `quotes.json`.

#### Step 4: Review the Results

Open the `quotes.json` file to see the extracted data. The output will be formatted as a JSON array containing the scraped information from each page.

### Understanding the Output

The output file contains structured data with each scraped page represented as an object. The structure includes the URL of the scraped page, the HTTP status code, extracted data fields as configured, and a timestamp indicating when the scraping occurred.

## Configuration Guide

### Configuration Structure

DataScrapexter uses YAML configuration files to define scraping behavior. A configuration file consists of several main sections that control different aspects of the scraping process.

### Basic Configuration Options

The foundation of any configuration includes the scraper name and base URL. The name serves as an identifier for your scraping job, while the base URL specifies where the scraping begins. Rate limiting controls the frequency of requests to avoid overwhelming the target server, with values specified as duration strings like "2s" for two seconds or "500ms" for half a second.

Request timeout defines how long to wait for a server response before giving up. A typical value is "30s" for thirty seconds, but this can be adjusted based on the target website's response times. The maximum number of retry attempts for failed requests can be configured, with a default of 3 retries using exponential backoff.

### Field Extraction

Field extraction forms the core of DataScrapexter's functionality. Each field definition specifies what data to extract and how to process it. The name identifies the field in the output, while the selector uses CSS syntax to target specific HTML elements. The extraction type determines how data is retrieved from the selected elements.

For text extraction, the "text" type retrieves the text content of elements. The "html" type captures the raw HTML, useful when you need to preserve formatting. The "attr" type extracts specific attribute values, requiring an additional attribute specification. The "list" type collects data from all matching elements as an array.

### Data Transformation

DataScrapexter includes a powerful transformation pipeline that processes extracted data before output. Transformations are applied in the order specified, allowing you to chain multiple operations.

Text transformations include trimming whitespace, converting case, and normalizing spaces. Numeric transformations can parse strings into numbers, with special handling for currency and formatted numbers. Regular expression transformations provide powerful pattern matching and replacement capabilities.

### Pagination Handling

For websites with multiple pages, DataScrapexter offers several pagination strategies. Next button pagination follows links marked as "next" or similar, making it ideal for blogs and article listings. Page number pagination works with numbered page links, common in e-commerce sites. URL pattern pagination generates URLs based on templates, useful when page URLs follow predictable patterns.

### Advanced Configuration

#### Custom Headers

Custom HTTP headers can be specified to mimic browser behavior more closely:

```yaml
headers:
  Accept-Language: "en-US,en;q=0.9"
  Accept-Encoding: "gzip, deflate, br"
  Cache-Control: "no-cache"
  Referer: "https://www.google.com/"
```

#### Proxy Configuration

For scenarios requiring proxy usage:

```yaml
proxy:
  enabled: true
  url: "http://proxy.example.com:8080"
  # Or use environment variable
  # url: "${PROXY_URL}"
```

#### User Agent Rotation

Specify custom user agents for rotation:

```yaml
user_agents:
  - "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"
  - "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36"
  - "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36"
```

## Advanced Features

### Complex Selectors

DataScrapexter supports the full range of CSS selectors, enabling precise element targeting. You can use descendant selectors to navigate nested structures, such as `div.product > h2` to select h2 elements that are direct children of product divs. Attribute selectors like `[data-price]` target elements with specific attributes, while pseudo-selectors such as `:nth-child(2)` or `:last-of-type` help select elements based on their position.

### Conditional Extraction

When dealing with varying page structures, you can define multiple extraction attempts with fallbacks. This is particularly useful for websites that may have different layouts for different types of content.

### Environment Variables

Sensitive information such as API keys and proxy credentials should not be stored directly in configuration files. DataScrapexter supports environment variable expansion:

```yaml
proxy:
  url: "${PROXY_URL}"

output:
  database:
    url: "${DATABASE_URL}"
```

### Output Formats

#### JSON Output

JSON output provides a structured format ideal for further processing:

```yaml
output:
  format: "json"
  file: "data.json"
  # Use "-" for stdout
  # file: "-"
```

#### CSV Output

CSV output automatically flattens nested data and generates headers:

```yaml
output:
  format: "csv"
  file: "data.csv"
```

### Concurrent Scraping

While DataScrapexter processes pages sequentially by default to respect rate limits, you can control concurrency for better performance when appropriate:

```bash
datascrapexter run config.yaml --concurrency 5
```

## Troubleshooting

### Common Issues and Solutions

#### No Data Extracted

When no data is extracted, first verify that your CSS selectors are correct by testing them in the browser's developer console. Check if the website uses JavaScript to load content dynamically, which would require browser automation features coming in version 0.5. Ensure the website structure hasn't changed since you created your configuration.

#### Rate Limiting Errors

If you encounter rate limiting errors (HTTP 429), increase the rate limit delay in your configuration. Consider adding random delays between requests to appear more human-like. Using proxy rotation can also help distribute requests across different IP addresses.

#### Connection Timeouts

For connection timeouts, increase the timeout value in your configuration. Check your internet connection stability and verify that the target website is accessible. If using a proxy, ensure it's functioning correctly.

#### Invalid Selector Errors

Invalid selector errors typically indicate syntax issues in your CSS selectors. Verify selector syntax using online CSS selector testers. Remember that class names with spaces should use dots between words (e.g., `.product.item` for `class="product item"`).

### Debug Mode

Enable detailed logging for troubleshooting:

```bash
datascrapexter run config.yaml --log-level debug
```

This provides detailed information about request/response cycles, selector matching results, and transformation pipeline execution.

### Performance Optimization

For large-scale scraping operations, consider these optimization strategies. Adjust rate limiting based on server capacity and response times. Use appropriate timeout values to avoid waiting too long for slow responses. Implement pagination limits to prevent infinite crawling. Monitor memory usage and adjust concurrency accordingly.

## Best Practices

### Configuration Management

Maintain your configuration files in version control to track changes over time. Use descriptive names for your scrapers and fields to make configurations self-documenting. Comment complex selectors or transformations to explain their purpose. Separate environment-specific settings using environment variables.

### Respectful Scraping

Always check and respect robots.txt files before scraping. Implement reasonable rate limiting to avoid overwhelming servers. Include appropriate user agent strings that identify your scraper. Monitor for signs that your scraping is causing issues and adjust accordingly.

### Data Quality

Use the required field attribute to catch missing critical data early. Implement appropriate transformations to clean and normalize data. Validate extracted data matches expected formats. Regular test your configurations to ensure they still work as websites evolve.

### Error Handling

Plan for failures by setting appropriate retry counts and delays. Log errors comprehensively for debugging. Implement monitoring to detect configuration breakage. Have fallback strategies for critical data extraction.

### Scalability

Start with conservative rate limits and gradually increase if needed. Monitor resource usage as you scale up operations. Consider distributed scraping for very large projects. Implement data deduplication for efficiency.

## Legal and Ethical Considerations

### Legal Compliance

Before scraping any website, review and understand its terms of service. Many websites explicitly prohibit automated data collection. Respect intellectual property rights and copyright laws. Be aware of data protection regulations like GDPR when handling personal information.

### Ethical Guidelines

Only collect publicly available data. Avoid scraping personal information without consent. Don't use scraped data for harmful purposes. Consider the impact of your scraping on website operations. Be transparent about your data collection when appropriate.

### Technical Compliance

DataScrapexter includes features to help with compliance. The robots.txt parser (coming in v0.5) automatically respects crawling rules. Rate limiting prevents server overload. User agent identification allows websites to identify your scraper.

### Data Handling

Implement appropriate data retention policies. Secure any sensitive data you collect. Anonymize personal information when possible. Be prepared to delete data upon request. Document your data collection and usage practices.

## Conclusion

DataScrapexter provides a powerful yet accessible solution for web scraping needs. By following this guide and adhering to best practices, you can effectively extract data while respecting technical and legal boundaries. As you become more familiar with the tool, explore advanced features to handle increasingly complex scraping scenarios.

For additional support, consult the API documentation for programmatic usage, refer to the examples directory for more configuration templates, and engage with the community through GitHub discussions. Regular updates and new features are released, so keep your installation current to benefit from improvements and bug fixes.

Remember that successful web scraping balances technical capability with responsible usage. DataScrapexter provides the tools; how you use them determines the value and ethics of your data collection efforts.
