# DataScrapexter FAQ (Frequently Asked Questions)

## General Questions

### What is DataScrapexter?

DataScrapexter is a high-performance, configuration-driven web scraping tool built with Go. It enables users to extract structured data from websites without writing code, using YAML configuration files to define what data to collect and how to process it. The tool includes advanced anti-detection features, automatic pagination handling, and comprehensive data transformation capabilities.

### Why should I use DataScrapexter instead of other scraping tools?

DataScrapexter offers several unique advantages:

- **Go Performance**: Native compilation provides exceptional speed and low memory usage compared to interpreted languages
- **Configuration-Driven**: No programming required for most scraping tasks
- **Built-in Anti-Detection**: Advanced features to avoid bot detection without additional setup
- **Type Safety**: Go's type system prevents many common scraping errors
- **Single Binary**: Easy deployment without dependency management
- **Concurrent Processing**: Efficient handling of multiple pages simultaneously

### Is DataScrapexter free to use?

Yes, DataScrapexter is open-source software released under the MIT License. You can use it freely for personal and commercial projects. The core functionality is completely free, with potential premium features planned for enterprise users in future releases.

### What operating systems does DataScrapexter support?

DataScrapexter supports:
- Linux (Ubuntu 20.04+, Debian 10+, CentOS 8+, and other modern distributions)
- macOS (10.15 Catalina and later, including Apple Silicon)
- Windows (Windows 10 version 1909 and later)

Pre-compiled binaries are available for all major platforms and architectures.

### Do I need programming knowledge to use DataScrapexter?

No programming knowledge is required for basic usage. DataScrapexter uses YAML configuration files that are human-readable and easy to understand. However, knowledge of CSS selectors and basic command-line usage is helpful. Advanced features like custom transformations may benefit from technical understanding.

## Installation and Setup

### How do I install DataScrapexter?

The easiest installation methods are:

1. **Download pre-compiled binary**: Visit the releases page and download the appropriate version for your system
2. **Using Go**: `go install github.com/valpere/DataScrapexter/cmd/datascrapexter@latest`
3. **From source**: Clone the repository and run `make build`
4. **Docker**: `docker pull ghcr.io/valpere/datascrapexter:latest`

See the [Installation Guide](installation.md) for detailed instructions.

### Why am I getting "command not found" after installation?

This typically means the DataScrapexter binary is not in your system's PATH. Solutions:

1. Add the installation directory to your PATH:
   ```bash
   export PATH=$PATH:/path/to/datascrapexter
   ```
2. Use the full path to the binary:
   ```bash
   /usr/local/bin/datascrapexter run config.yaml
   ```
3. For Go installations, ensure `$GOPATH/bin` is in your PATH

### Can I install DataScrapexter without admin/root privileges?

Yes, you can install DataScrapexter to any directory you have write access to. Download the binary and place it in a local directory like `~/bin`, then add that directory to your PATH. No system-wide installation is required.

### How do I update DataScrapexter to the latest version?

Update methods depend on your installation:

- **Binary**: Download the new version and replace the old binary
- **Go install**: Run `go install github.com/valpere/DataScrapexter/cmd/datascrapexter@latest`
- **Source**: Pull latest changes and rebuild with `git pull && make build`
- **Docker**: Run `docker pull ghcr.io/valpere/datascrapexter:latest`

## Configuration

### How do I create my first configuration file?

Start with a template:

```bash
datascrapexter template > my-config.yaml
```

Then edit the file to match your target website. Key sections to modify:
- `base_url`: The website you want to scrape
- `fields`: Define what data to extract using CSS selectors
- `output`: Specify where to save the results

### What are CSS selectors and how do I find them?

CSS selectors are patterns used to identify HTML elements. To find selectors:

1. Open the target website in Chrome or Firefox
2. Right-click on the element you want to scrape
3. Select "Inspect" or "Inspect Element"
4. In the developer tools, right-click the highlighted HTML
5. Choose "Copy > Copy selector"

Common selector patterns:
- `.classname` - Elements with specific class
- `#id` - Element with specific ID
- `tag` - All elements of a type (h1, p, div)
- `parent > child` - Direct child elements
- `[attribute="value"]` - Elements with specific attributes

### Can I use XPath instead of CSS selectors?

Currently, DataScrapexter supports CSS selectors only. CSS selectors cover most scraping needs and are generally faster and more readable than XPath. XPath support may be added in future versions based on user demand.

### How do I handle dynamic content loaded by JavaScript?

DataScrapexter v0.1.0 processes static HTML content. For JavaScript-rendered content, you currently need to:

1. Find API endpoints the site uses and scrape those directly
2. Look for mobile or simplified versions of the site
3. Wait for v0.5.0 which will include browser automation support

### Can I scrape multiple pages automatically?

Yes, DataScrapexter supports three pagination methods:

```yaml
# Follow "Next" buttons
pagination:
  type: "next_button"
  selector: ".pagination .next"
  max_pages: 10

# Use URL patterns
pagination:
  type: "url_pattern"
  url_pattern: "https://example.com/page/{page}"
  start_page: 1
  max_pages: 20
```

### How do I extract data from tables?

For table data, target specific cells or rows:

```yaml
fields:
  # Extract all rows
  - name: "table_rows"
    selector: "table.data-table tr"
    type: "list"

  # Extract specific columns
  - name: "product_names"
    selector: "table tr td:nth-child(1)"
    type: "list"

  - name: "prices"
    selector: "table tr td:nth-child(3)"
    type: "list"
```

## Usage and Operation

### How do I run a scraper?

Basic usage:

```bash
datascrapexter run config.yaml
```

With options:

```bash
# Override output file
datascrapexter run -o results.json config.yaml

# Increase concurrency
datascrapexter run --concurrency 5 config.yaml

# Dry run (test without making requests)
datascrapexter run --dry-run config.yaml
```

### What does "rate limiting" mean and why is it important?

Rate limiting controls how frequently DataScrapexter makes requests to a website. It's important because:

1. **Respectful scraping**: Prevents overwhelming target servers
2. **Avoid blocking**: Many sites block IPs making too many requests
3. **Legal compliance**: Some sites specify acceptable request rates
4. **Reliability**: Slower scraping is often more reliable

Configure rate limiting:

```yaml
rate_limit: "2s"  # Wait 2 seconds between requests
```

### Can I run multiple scrapers simultaneously?

Yes, you can run multiple DataScrapexter instances simultaneously. However, consider:

1. Combined request rate to the same website
2. System resource usage (CPU, memory, network)
3. Output file conflicts (use different output files)

Example parallel execution:

```bash
datascrapexter run config1.yaml &
datascrapexter run config2.yaml &
datascrapexter run config3.yaml &
wait  # Wait for all to complete
```

### How do I schedule scrapers to run automatically?

Use your operating system's scheduling tools:

**Linux/macOS (cron):**
```bash
# Add to crontab
0 6 * * * cd /path/to/project && datascrapexter run daily-scraper.yaml
```

**Windows (Task Scheduler):**
Create a scheduled task that runs `datascrapexter.exe run config.yaml`

### What output formats are supported?

Currently supported formats:
- **JSON**: Structured data with nested objects and arrays
- **CSV**: Tabular data for spreadsheets and data analysis

Planned formats:
- **Excel**: Native Excel files with formatting
- **Database**: Direct writing to PostgreSQL, MySQL, MongoDB

## Troubleshooting

### Why is my scraper not finding any data?

Common causes and solutions:

1. **Incorrect selectors**: Verify selectors in browser developer tools
2. **Dynamic content**: Check if content loads via JavaScript
3. **Website changes**: Site structure may have changed
4. **Authentication required**: Some data needs login (not yet supported)

Debug with:

```bash
datascrapexter --log-level debug run config.yaml
```

### How do I handle "429 Too Many Requests" errors?

This means you're scraping too fast. Solutions:

1. Increase rate limit:
   ```yaml
   rate_limit: "5s"  # Wait longer between requests
   ```
2. Add retries with backoff:
   ```yaml
   max_retries: 5
   ```
3. Use proxy rotation (configure proxy settings)
4. Reduce concurrency if using parallel workers

### Why am I getting "connection timeout" errors?

Timeouts occur when the server doesn't respond quickly enough. Try:

1. Increase timeout setting:
   ```yaml
   timeout: "60s"  # Increase from default 30s
   ```
2. Check your internet connection
3. Verify the website is accessible in your browser
4. Try using a proxy if the site blocks your region

### Can DataScrapexter handle CAPTCHAs?

DataScrapexter v0.1.0 cannot automatically solve CAPTCHAs. When encountering CAPTCHAs:

1. Use proxy rotation to avoid triggering them
2. Implement longer delays between requests
3. Look for API alternatives
4. CAPTCHA solving integration is planned for v1.0.0

### How do I debug transformation errors?

Test transformations step by step:

1. Remove all transformations and check raw data
2. Add transformations one at a time
3. Use debug logging to see intermediate values
4. Test regex patterns on regex101.com

Example debugging approach:

```yaml
fields:
  - name: "price_debug"
    selector: ".price"
    type: "text"
    # Start with no transformations

  - name: "price_step1"
    selector: ".price"
    type: "text"
    transform:
      - type: "trim"  # Add one at a time
```

## Data Processing

### How do I clean and format extracted data?

DataScrapexter provides built-in transformations:

```yaml
transform:
  # Text cleaning
  - type: "trim"              # Remove whitespace
  - type: "normalize_spaces"  # Clean internal spaces
  - type: "lowercase"         # Convert to lowercase

  # Data extraction
  - type: "regex"
    pattern: "\\$([0-9,]+)"
    replacement: "$1"

  # Type conversion
  - type: "parse_float"       # Convert to number
```

### Can I extract data from multiple similar elements?

Yes, use the "list" type to extract from all matching elements:

```yaml
fields:
  - name: "all_prices"
    selector: ".product .price"
    type: "list"  # Gets all matching elements
    transform:
      - type: "clean_price"
      - type: "parse_float"
```

### How do I handle missing or optional data?

Mark fields as optional by not setting `required: true`:

```yaml
fields:
  - name: "price"
    selector: ".price"
    type: "text"
    required: true  # Scraping fails if missing

  - name: "discount"
    selector: ".discount"
    type: "text"
    # Optional - scraping continues if missing
```

### Can I combine data from multiple fields?

Currently, field combination must be done in post-processing. However, you can extract related data:

```yaml
fields:
  - name: "currency"
    selector: ".price .currency"
    type: "text"

  - name: "amount"
    selector: ".price .amount"
    type: "text"
    transform:
      - type: "parse_float"
```

Then combine in your application or using command-line tools.

## Performance and Scaling

### How many pages can DataScrapexter handle?

DataScrapexter can handle virtually unlimited pages, constrained only by:

1. Time (rate limiting affects total time)
2. Storage space for results
3. Memory (minimal - uses streaming where possible)
4. Target website limits

Practical limits depend on your configuration and use case.

### How can I make my scraper faster?

Optimization strategies:

1. **Reduce rate limiting** (carefully):
   ```yaml
   rate_limit: "500ms"  # Faster but less polite
   ```

2. **Increase concurrency**:
   ```bash
   datascrapexter run --concurrency 10 config.yaml
   ```

3. **Optimize selectors**: Use specific selectors that match quickly

4. **Limit scope**: Scrape only necessary data

5. **Use appropriate timeouts**: Don't wait longer than necessary

### Does DataScrapexter support distributed scraping?

DataScrapexter v0.1.0 runs on a single machine. Distributed scraping is planned for future releases. Current workarounds:

1. Split configurations across multiple machines
2. Use different proxy endpoints per instance
3. Coordinate through shared storage or message queues

### How much memory does DataScrapexter use?

Memory usage is typically low (50-200MB) due to Go's efficiency. Factors affecting memory:

1. Size of pages being scraped
2. Number of concurrent workers
3. Size of result set before writing

Monitor memory usage:

```bash
# Linux/macOS
ps aux | grep datascrapexter

# Detailed profiling
datascrapexter run --pprof-port 6060 config.yaml
```

## Legal and Ethical

### Is web scraping legal?

Web scraping legality depends on:

1. **Website terms of service**: Many sites prohibit automated access
2. **Copyright laws**: Don't republish copyrighted content
3. **Data protection laws**: GDPR, CCPA affect personal data
4. **Computer fraud laws**: Varies by jurisdiction

Always:
- Review website terms of service
- Respect robots.txt
- Don't scrape personal data without consent
- Consult legal counsel for commercial use

### How do I respect robots.txt?

DataScrapexter v0.1.0 doesn't automatically parse robots.txt. Check manually:

1. Visit `https://example.com/robots.txt`
2. Look for Disallow rules for your target paths
3. Respect crawl-delay directives
4. Implement specified delays in your configuration

Automatic robots.txt support is planned for v0.5.0.

### What are best practices for ethical scraping?

Follow these guidelines:

1. **Identify yourself**: Use descriptive User-Agent
2. **Rate limit appropriately**: Don't overwhelm servers
3. **Scrape during off-peak hours**: Reduce impact
4. **Cache responses**: Don't re-scrape unchanged data
5. **Respect opt-outs**: Honor robots.txt and terms
6. **Use APIs when available**: Prefer official data sources

### Can I use scraped data commercially?

Commercial use depends on:

1. **Data ownership**: Who owns the original data?
2. **Terms of service**: What do they allow?
3. **Copyright**: Is the content protected?
4. **Database rights**: Some regions protect compilations

Generally safer to:
- Use data for analysis rather than republishing
- Transform data significantly
- Attribute sources appropriately
- Seek permission for commercial use

## Advanced Topics

### Can I use DataScrapexter as a Go library?

Yes, DataScrapexter can be imported as a Go package:

```go
import "github.com/valpere/DataScrapexter/pkg/scraper"

// Use in your Go application
engine, err := scraper.NewEngine(config)
result, err := engine.Scrape(ctx, url, extractors)
```

See the [API Documentation](api.md) for details.

### Does DataScrapexter support proxy rotation?

Yes, configure proxy settings:

```yaml
proxy:
  enabled: true
  url: "http://proxy:8080"
  # Or for rotation (planned feature)
  rotation: "round-robin"
  list:
    - "http://proxy1:8080"
    - "http://proxy2:8080"
```

Current version supports single proxy; rotation coming in v0.5.0.

### Can I modify request headers?

Yes, add custom headers in configuration:

```yaml
headers:
  Accept-Language: "en-US,en;q=0.9"
  Referer: "https://google.com"
  X-Custom-Header: "value"
```

User-Agent rotation is configured separately in the `user_agents` section.

### How do I handle authentication?

DataScrapexter v0.1.0 doesn't support login/authentication. Workarounds:

1. Scrape public data only
2. Use authenticated API endpoints if available
3. Manual cookie injection (planned for v0.5.0)

Full authentication support is planned for v1.0.0.

### Can DataScrapexter handle infinite scroll?

Infinite scroll requires JavaScript execution, not available in v0.1.0. Current alternatives:

1. Find the API endpoint that loads more data
2. Look for "View All" or pagination links
3. Use mobile site versions without infinite scroll

Browser automation for infinite scroll is planned for v0.5.0.

## Getting Help

### Where can I get help with DataScrapexter?

Support channels:

1. **GitHub Issues**: Bug reports and feature requests
2. **Discord Community**: Real-time help and discussions
3. **Stack Overflow**: Tag questions with `datascrapexter`
4. **Email Support**: support@datascrapexter.com (commercial)

### How do I report a bug?

Create a GitHub issue with:

1. DataScrapexter version (`datascrapexter version`)
2. Operating system and version
3. Configuration file (remove sensitive data)
4. Full error message and logs
5. Steps to reproduce the problem

### Can I request new features?

Yes! Feature requests are welcome:

1. Check existing issues first
2. Create a detailed feature request on GitHub
3. Explain the use case and benefits
4. Consider contributing the implementation

### How can I contribute to DataScrapexter?

Contributions are welcome:

1. **Code**: Fix bugs, add features, improve performance
2. **Documentation**: Improve guides, add examples
3. **Testing**: Report bugs, test new releases
4. **Community**: Help others, share configurations

See [CONTRIBUTING.md](../CONTRIBUTING.md) for guidelines.

### Is commercial support available?

Commercial support is planned for enterprise users, including:

- Priority bug fixes
- Custom feature development
- Training and consulting
- SLA guarantees

Contact support@datascrapexter.com for information.

---

This FAQ is regularly updated based on community questions. If your question isn't answered here, please check the documentation or ask in our community channels.
