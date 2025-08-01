# DataScrapexter Scripts Documentation

## Overview

The DataScrapexter project includes a comprehensive collection of scripts for automation, monitoring, and data analysis. These scripts work together to create a robust web scraping ecosystem that handles everything from daily operations to health monitoring and reporting.

## Script Files Reference

### 1. analyze_price_changes.pl

**Purpose**: Analyzes price changes across multiple scraping sessions to identify trends and significant price movements.

**Usage**:
```bash
./scripts/analyze_price_changes.pl --data-dir outputs --days 30 --csv --json
```

**Key Features**:
- Historical price trend analysis
- Configurable time range analysis (default: 30 days)
- Multiple export formats (text report, CSV, JSON alerts)
- Statistical analysis including average, median, and percentage changes
- Alert generation for significant price changes (configurable thresholds)

**Required Arguments**:
- `--data-dir DIR`: Directory containing scraped data files

**Optional Arguments**:
- `--output-dir DIR`: Output directory for reports (default: price_analysis)
- `--days N`: Number of days to analyze (default: 30)
- `--csv`: Export detailed CSV report
- `--json`: Generate JSON alerts file
- `--help`: Show help message

### 2. combine_data.pl

**Purpose**: Merges data from multiple scraping sessions into a unified dataset, handling duplicates and enriching data with calculated fields.

**Usage**:
```bash
./scripts/combine_data.pl --sources outputs/daily --output combined_data.json --format json
```

**Key Features**:
- Supports multiple input formats (CSV, JSON)
- Intelligent duplicate detection using configurable hash fields
- Data enrichment with timestamps and calculated fields
- Support for glob patterns in source specification
- Detailed merge statistics

**Required Arguments**:
- `--sources PATTERN`: Source files or directories (supports wildcards)

**Optional Arguments**:
- `--output FILE`: Output file name (default: combined_data.json)
- `--format FORMAT`: Output format - json or csv (default: json)
- `--merge-field FIELD`: Field to use for matching records (default: url)
- `--hash-fields FIELDS`: Comma-separated fields for duplicate detection
- `--exclude PATTERN`: Pattern to exclude files
- `--dry-run`: Preview without creating output

### 3. daily_scrape.sh

**Purpose**: Automated daily scraping orchestration with comprehensive logging, error handling, and notification capabilities.

**Usage**:
```bash
./scripts/daily_scrape.sh [profile1 profile2 ...]
```

**Key Features**:
- Multiple configuration profile support
- Comprehensive error handling with retry logic
- Email and webhook notifications
- Automatic output archiving
- Integration with health monitoring
- Detailed performance metrics

**Environment Variables**:
- `SCRAPER_NOTIFY_EMAIL`: Email for notifications
- `SCRAPER_NOTIFY_WEBHOOK`: Webhook URL for alerts
- `SCRAPER_NOTIFY_SUCCESS`: Notify on success (default: false)
- `SCRAPER_NOTIFY_FAILURE`: Notify on failure (default: true)
- `RETRY_ATTEMPTS`: Number of retry attempts (default: 3)
- `RETRY_DELAY`: Delay between retries in seconds (default: 60)

**Cron Example**:
```bash
0 6 * * * cd /path/to/datascrapexter && ./scripts/daily_scrape.sh
```

### 4. extract_urls.sh

**Purpose**: Extracts URLs from DataScrapexter output files for further processing or analysis.

**Usage**:
```bash
./scripts/extract_urls.sh products.csv product-urls.txt 2
./scripts/extract_urls.sh products.json urls.txt product_url
```

**Key Features**:
- Automatic file type detection (CSV/JSON)
- Configurable column/field extraction
- URL validation and deduplication
- Filtering capabilities
- Batch processing support
- Progress indicators

**Arguments**:
1. `input_file`: CSV or JSON file containing URLs
2. `output_file`: Output file for extracted URLs (default: urls.txt)
3. `column/field`: For CSV - column number; For JSON - field name

### 5. generate_weekly_report.sh

**Purpose**: Generates comprehensive weekly reports analyzing scraping performance, data quality, and trends.

**Usage**:
```bash
./scripts/generate_weekly_report.sh [--email recipient@example.com]
```

**Key Features**:
- Weekly data aggregation and analysis
- Performance metrics calculation
- Price trend visualization
- Data quality assessment
- Multi-format report generation (text, JSON, HTML)
- Email distribution capability

**Options**:
- `--email ADDRESS`: Send report to email address
- `--format FORMAT`: Report format (text/json/html)
- `--days N`: Number of days to include (default: 7)
- `--output DIR`: Output directory

### 6. monitor_health.sh

**Purpose**: Monitors the health of the scraping system, checking various metrics and sending alerts when issues are detected.

**Usage**:
```bash
./scripts/monitor_health.sh
```

**Key Features**:
- Recent output file monitoring
- Error rate tracking
- System resource monitoring (CPU, memory, disk)
- Data quality validation
- Response time analysis
- Alert integration (email, webhooks)
- External health check endpoint support

**Environment Variables**:
- `MONITOR_ALERT_EMAIL`: Email for alerts
- `MONITOR_ALERT_WEBHOOK`: Webhook URL for alerts
- `MONITOR_HEALTH_URL`: External health check endpoint

**Monitoring Thresholds** (configurable):
- Error threshold: 10 errors
- Warning threshold: 5 errors
- Minimum output files: 1 per hour
- Maximum response time: 30 seconds
- Disk usage warning: 80%
- Memory warning: 80%
- CPU warning: 80%

### 7. pre-commit

**Purpose**: Git pre-commit hook ensuring code quality and consistency before commits.

**Installation**:
```bash
cp scripts/pre-commit .git/hooks/pre-commit
chmod +x .git/hooks/pre-commit
```

**Checks Performed**:
1. Go code formatting (gofmt)
2. Static analysis (go vet)
3. TODO/FIXME comment detection
4. License header verification
5. Module dependency cleanliness
6. Build verification
7. Unit test execution
8. Large file detection
9. YAML configuration validation
10. Commit message format suggestions

### 8. scrape-products.sh

**Purpose**: Scrapes individual product details from a list of URLs with rate limiting and progress tracking.

**Usage**:
```bash
./scripts/scrape-products.sh [config_file] [url_file] [output_dir]
```

**Key Features**:
- Batch URL processing
- Configurable rate limiting
- Progress tracking with statistics
- Error handling and logging
- Resume capability
- Summary report generation

**Environment Variables**:
- `SCRAPE_DELAY`: Delay between products (default: 3 seconds)
- `BATCH_SIZE`: Products per batch (default: 10)
- `MAX_CONCURRENT`: Maximum concurrent scrapers (default: 1)

### 9. scrape_new_products.pl

**Purpose**: Discovers and scrapes products that are new since the last scraping session.

**Usage**:
```bash
./scripts/scrape_new_products.pl \
    --listings outputs/products-listing.csv \
    --history outputs/products \
    --config configs/product-details.yaml \
    --max 20
```

**Key Features**:
- Automatic new product discovery
- Historical data comparison
- Configurable scraping limits
- Metadata enrichment
- Detailed reporting
- Integration with main scraper

**Required Arguments**:
- `--listings FILE`: Current product listings file
- `--history DIR`: Directory with historical product data
- `--config FILE`: DataScrapexter configuration file

**Optional Arguments**:
- `--output DIR`: Output directory (default: outputs/new_products)
- `--max N`: Maximum new products to scrape (default: 50)
- `--delay N`: Delay between products in seconds (default: 3)
- `--verbose`: Enable verbose output

## Script Integration Workflow

### Daily Operation Flow

1. **Morning Scrape** (6:00 AM)
   - `daily_scrape.sh` runs main scraping tasks
   - Collects product listings and updates

2. **New Product Discovery** (Every 4 hours)
   - `scrape_new_products.pl` identifies new items
   - `scrape-products.sh` fetches detailed information

3. **Data Processing** (After each scrape)
   - `combine_data.pl` merges new data with existing
   - `extract_urls.sh` prepares URLs for next cycle

4. **Analysis** (Daily)
   - `analyze_price_changes.pl` tracks price movements
   - Generates alerts for significant changes

5. **Reporting** (Weekly)
   - `generate_weekly_report.sh` creates comprehensive reports
   - Distributes to stakeholders

6. **Monitoring** (Hourly)
   - `monitor_health.sh` checks system health
   - Sends alerts if issues detected

### Setting Up Automated Workflow

1. **Install Pre-commit Hook**:
   ```bash
   make install-hooks
   ```

2. **Configure Cron Jobs**:
   ```bash
   # Edit crontab
   crontab -e

   # Add scheduled tasks
   0 6 * * * cd /path/to/datascrapexter && ./scripts/daily_scrape.sh
   0 */4 * * * cd /path/to/datascrapexter && ./scripts/scrape_new_products.pl --listings outputs/latest.csv --history outputs/products --config configs/product.yaml
   0 9 * * 1 cd /path/to/datascrapexter && ./scripts/generate_weekly_report.sh --email admin@example.com
   0 * * * * cd /path/to/datascrapexter && ./scripts/monitor_health.sh
   ```

3. **Set Environment Variables**:
   ```bash
   export SCRAPER_NOTIFY_EMAIL="admin@example.com"
   export SCRAPER_NOTIFY_WEBHOOK="https://hooks.slack.com/services/YOUR/WEBHOOK/URL"
   export MONITOR_ALERT_EMAIL="alerts@example.com"
   ```

## Best Practices

### Error Handling
- All scripts include comprehensive error handling
- Failed operations are logged with detailed error messages
- Retry logic is implemented where appropriate
- Non-zero exit codes indicate failures

### Logging
- Structured logging with timestamps
- Multiple log levels (INFO, WARNING, ERROR)
- Log rotation handled by daily_scrape.sh
- Centralized log directory at `logs/`

### Performance
- Rate limiting prevents server overload
- Batch processing for efficiency
- Progress indicators for long-running operations
- Resource monitoring prevents system exhaustion

### Data Integrity
- Duplicate detection in combine_data.pl
- Data validation in all processing scripts
- Backup creation before modifications
- Atomic operations where possible

## Troubleshooting

### Common Issues

1. **Script Permission Errors**
   ```bash
   chmod +x scripts/*.sh scripts/*.pl
   ```

2. **Missing Dependencies**
   - Bash scripts: Require bash 4.0+, coreutils
   - Perl scripts: Require JSON, File::Find, Getopt::Long modules
   ```bash
   cpan install JSON Getopt::Long
   ```

3. **Configuration Not Found**
   - Ensure paths in scripts match your directory structure
   - Use absolute paths in cron jobs
   - Check CONFIG_DIR and OUTPUT_DIR variables

4. **Memory Issues with Large Datasets**
   - Adjust batch sizes in scraping scripts
   - Use streaming processing in combine_data.pl
   - Monitor system resources during operation

### Debug Mode

Most scripts support verbose output:
```bash
# Bash scripts
DEBUG=1 ./scripts/daily_scrape.sh

# Perl scripts
./scripts/analyze_price_changes.pl --verbose
```

## Contributing

When adding new scripts:

1. Follow existing naming conventions
2. Include comprehensive help text
3. Implement proper error handling
4. Add logging capabilities
5. Document in this file
6. Create example usage in configs/

## License

All scripts are part of the DataScrapexter project and are licensed under the MIT License. See the LICENSE file in the project root for details.
