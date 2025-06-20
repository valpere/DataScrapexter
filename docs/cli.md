# DataScrapexter CLI Reference

## Overview

DataScrapexter provides a comprehensive command-line interface designed for both interactive use and automation. The CLI follows conventional Unix patterns while offering user-friendly features for newcomers to command-line tools. This reference documents all available commands, their options, and usage patterns to help you effectively utilize DataScrapexter's capabilities.

The CLI design emphasizes clarity and consistency across commands. Each command serves a specific purpose in the web scraping workflow, from initial configuration validation through execution and result analysis. Global options apply across all commands, while command-specific options provide fine-grained control over individual operations.

## Command Structure

DataScrapexter commands follow a hierarchical structure with the main executable followed by a command and its associated options and arguments. The general syntax pattern is:

```bash
datascrapexter [global-options] <command> [command-options] [arguments]
```

Global options modify the behavior of DataScrapexter across all commands, such as controlling log levels or specifying configuration file locations. Commands represent specific actions like running a scraper or validating configuration. Command options provide parameters specific to individual commands, while arguments supply required information such as file paths or URLs.

## Global Options

Global options can be specified before any command and affect the overall operation of DataScrapexter. These options provide control over logging, configuration paths, and output formatting that apply regardless of the specific command being executed.

### --config

The config option specifies an alternative location for DataScrapexter's global configuration file. By default, the application looks for .datascrapexter.yaml in your home directory. This option is particularly useful when managing multiple DataScrapexter installations or when operating in containerized environments.

```bash
datascrapexter --config /etc/datascrapexter/config.yaml run scraper.yaml
```

### --log-level

Controls the verbosity of log output, accepting values of debug, info, warn, and error. The default level is info, providing a balance between useful operational information and console clarity. Debug level reveals detailed operation traces useful for troubleshooting, while error level shows only critical issues.

```bash
datascrapexter --log-level debug run config.yaml
```

### --log-format

Determines the structure of log output, supporting text for human-readable logs and json for structured logging suitable for log aggregation systems. JSON format is particularly valuable in production environments where logs are processed by centralized logging infrastructure.

```bash
datascrapexter --log-format json run config.yaml
```

### --quiet

Suppresses all non-error output, useful for automated scripts where only critical failures should produce console output. This option overrides log-level settings, showing only error messages and the final exit status.

```bash
datascrapexter --quiet run config.yaml
```

### --verbose

Increases output verbosity beyond the standard info level, equivalent to --log-level debug but more intuitive for users familiar with common CLI conventions. Multiple -v flags can be used for increasing verbosity levels.

```bash
datascrapexter -v run config.yaml        # Verbose output
datascrapexter -vv run config.yaml       # Very verbose output
```

### --no-color

Disables colored output in terminals, useful when piping output to files or when using terminals that don't support ANSI color codes. This ensures clean output in automated environments and log files.

```bash
datascrapexter --no-color run config.yaml > output.log
```

## Commands

### run

The run command executes a web scraping operation based on a configuration file. This is the primary command for DataScrapexter, orchestrating the entire scraping process from initialization through data extraction and output generation.

```bash
datascrapexter run [options] <config-file>
```

#### Options

**--output, -o**: Overrides the output file specified in the configuration. This is particularly useful when running the same configuration multiple times with different output destinations or when testing configurations.

```bash
datascrapexter run -o results.json config.yaml
datascrapexter run --output /data/scrapes/$(date +%Y%m%d).csv config.yaml
```

**--concurrency**: Sets the number of concurrent scraping workers. The default value of 1 ensures sequential processing, while higher values enable parallel scraping for improved performance. Consider target website capacity and rate limits when adjusting this value.

```bash
datascrapexter run --concurrency 5 config.yaml
```

**--dry-run**: Validates the configuration and simulates execution without making actual HTTP requests. This option helps verify configuration correctness and understand what operations would be performed.

```bash
datascrapexter run --dry-run config.yaml
```

**--continue-on-error**: Allows scraping to continue even when individual page extractions fail. By default, DataScrapexter stops on critical errors. This option is valuable for large-scale scraping where partial results are acceptable.

```bash
datascrapexter run --continue-on-error config.yaml
```

**--start-url**: Overrides the base_url specified in the configuration file. This enables using the same configuration for different starting points without modification.

```bash
datascrapexter run --start-url "https://example.com/page/2" config.yaml
```

#### Examples

Basic execution with standard configuration:

```bash
datascrapexter run product-scraper.yaml
```

Override output location with timestamp:

```bash
datascrapexter run -o "outputs/products-$(date +%Y%m%d-%H%M%S).json" config.yaml
```

Run with increased concurrency and error tolerance:

```bash
datascrapexter run --concurrency 10 --continue-on-error large-catalog.yaml
```

Execute with environment variable substitution:

```bash
SITE_URL="https://example.com" datascrapexter run template-config.yaml
```

### validate

The validate command checks configuration files for syntax errors and logical inconsistencies without executing any scraping operations. This pre-flight check helps catch configuration problems early, saving time and preventing failed scraping attempts.

```bash
datascrapexter validate [options] <config-file>
```

#### Options

**--strict**: Enables additional validation checks beyond basic syntax verification. Strict mode validates selector syntax, transformation rule parameters, and logical consistency between configuration sections.

```bash
datascrapexter validate --strict config.yaml
```

**--json**: Outputs validation results in JSON format for programmatic processing. This facilitates integration with CI/CD pipelines and automated configuration testing.

```bash
datascrapexter validate --json config.yaml | jq '.valid'
```

#### Examples

Basic validation:

```bash
datascrapexter validate scraper-config.yaml
```

Validate multiple configurations:

```bash
for config in configs/*.yaml; do
    echo "Validating $config"
    datascrapexter validate "$config"
done
```

Strict validation with JSON output for automation:

```bash
if datascrapexter validate --strict --json config.yaml | jq -e '.valid'; then
    echo "Configuration valid, proceeding with scraping"
    datascrapexter run config.yaml
else
    echo "Configuration invalid, aborting"
    exit 1
fi
```

### template

The template command generates example configuration files for common scraping scenarios. These templates serve as starting points for creating custom configurations, demonstrating best practices and available features.

```bash
datascrapexter template [options]
```

#### Options

**--type**: Specifies the template type to generate. Available types include basic, ecommerce, news, jobs, and social. Each template is tailored to common patterns found in its respective website category.

```bash
datascrapexter template --type ecommerce > shop-scraper.yaml
```

**--list**: Lists all available template types with brief descriptions. This helps users discover appropriate starting points for their scraping needs.

```bash
datascrapexter template --list
```

**--output, -o**: Writes the template to a specified file instead of stdout. This convenience option eliminates the need for shell redirection.

```bash
datascrapexter template --type news -o news-config.yaml
```

#### Examples

Generate a basic template:

```bash
datascrapexter template > my-scraper.yaml
```

Create an e-commerce scraper configuration:

```bash
datascrapexter template --type ecommerce -o product-monitor.yaml
```

View available templates:

```bash
datascrapexter template --list
```

### version

The version command displays detailed information about the DataScrapexter installation, including version number, build time, and git commit hash. This information is essential for bug reports and ensuring compatibility.

```bash
datascrapexter version [options]
```

#### Options

**--json**: Outputs version information in JSON format for automated processing and integration with monitoring systems.

```bash
datascrapexter version --json
```

**--short**: Displays only the version number without additional build information, useful for simple version checks in scripts.

```bash
datascrapexter version --short
```

#### Examples

Display full version information:

```bash
datascrapexter version
# Output:
# DataScrapexter v0.1.0
# Build Time: 2024-06-20_10:30:45
# Git Commit: a4b5c6d
```

Get version for automated checks:

```bash
VERSION=$(datascrapexter version --short)
echo "Running DataScrapexter $VERSION"
```

### serve (Future)

The serve command will launch DataScrapexter in server mode, providing REST API access to scraping functionality. This enables integration with web applications and distributed systems.

```bash
datascrapexter serve [options]
```

Planned options include --port for API server port, --metrics-port for Prometheus metrics endpoint, and --auth for authentication configuration.

### list (Future)

The list command will display information about scraping jobs, configurations, and results. This provides visibility into DataScrapexter's operation history and current status.

```bash
datascrapexter list [options] <resource>
```

Planned resources include jobs for active and completed scraping jobs, configs for available configurations, and results for stored scraping outputs.

## Exit Codes

DataScrapexter uses standard exit codes to indicate operation results, facilitating integration with shell scripts and automation systems. Understanding these codes helps in building robust automation around DataScrapexter.

- **0**: Success - Command completed successfully
- **1**: General error - Unspecified error occurred
- **2**: Configuration error - Invalid or missing configuration
- **3**: Network error - Failed to connect to target website
- **4**: Extraction error - Failed to extract required data
- **5**: Output error - Failed to write results
- **64**: Usage error - Invalid command line arguments
- **130**: Interrupted - Operation cancelled by user (Ctrl+C)

## Environment Variables

DataScrapexter recognizes several environment variables that modify its behavior without requiring command-line options. These variables are particularly useful in containerized deployments and CI/CD pipelines.

### DATASCRAPEXTER_HOME

Sets the base directory for DataScrapexter operations, including default locations for configurations, outputs, and logs. If not set, defaults to $HOME/datascrapexter.

```bash
export DATASCRAPEXTER_HOME=/opt/datascrapexter
datascrapexter run config.yaml
```

### DATASCRAPEXTER_LOG_LEVEL

Provides a default log level that can be overridden by the --log-level command-line option. Useful for setting consistent logging across multiple invocations.

```bash
export DATASCRAPEXTER_LOG_LEVEL=debug
datascrapexter run config.yaml  # Will use debug logging
```

### DATASCRAPEXTER_CONFIG_PATH

Specifies additional directories to search for configuration files. Multiple directories can be specified using the system path separator (: on Unix, ; on Windows).

```bash
export DATASCRAPEXTER_CONFIG_PATH=/etc/datascrapexter:/opt/configs
datascrapexter run myconfig.yaml  # Will search in specified directories
```

### HTTP_PROXY / HTTPS_PROXY

Standard proxy environment variables respected by DataScrapexter's HTTP client. These apply globally unless overridden in configuration files.

```bash
export HTTP_PROXY=http://proxy.company.com:8080
export HTTPS_PROXY=http://proxy.company.com:8080
datascrapexter run config.yaml
```

### NO_PROXY

Specifies hosts that should bypass proxy settings, useful for internal websites or local development.

```bash
export NO_PROXY=localhost,127.0.0.1,internal.company.com
datascrapexter run internal-site.yaml
```

## Shell Completion

DataScrapexter supports shell completion for bash, zsh, and fish shells, enhancing command-line productivity by providing automatic suggestions for commands and options.

### Installing Completions

For bash:

```bash
datascrapexter completion bash > /etc/bash_completion.d/datascrapexter
# Or for user-specific installation:
datascrapexter completion bash >> ~/.bashrc
```

For zsh:

```bash
datascrapexter completion zsh > "${fpath[1]}/_datascrapexter"
```

For fish:

```bash
datascrapexter completion fish > ~/.config/fish/completions/datascrapexter.fish
```

### Using Completions

Once installed, shell completions provide intelligent suggestions:

```bash
datascrapexter <TAB>           # Shows available commands
datascrapexter run --<TAB>     # Shows options for run command
datascrapexter run ~/configs/<TAB>  # Shows yaml files in directory
```

## Advanced Usage Patterns

### Chaining Commands

Combine DataScrapexter commands with shell utilities for complex workflows:

```bash
# Validate all configurations before running
find configs -name "*.yaml" -exec datascrapexter validate {} \; && \
datascrapexter run main-config.yaml
```

### Parallel Execution

Run multiple scrapers concurrently using shell job control:

```bash
# Run multiple scrapers in parallel
datascrapexter run config1.yaml &
datascrapexter run config2.yaml &
datascrapexter run config3.yaml &
wait  # Wait for all background jobs to complete
```

### Conditional Execution

Use exit codes for conditional logic:

```bash
if datascrapexter run --dry-run config.yaml; then
    echo "Dry run successful, executing real scraping"
    datascrapexter run config.yaml
else
    echo "Configuration has issues, aborting"
    exit 1
fi
```

### Output Processing

Pipe DataScrapexter output through other tools:

```bash
# Extract specific fields from JSON output
datascrapexter run -o - config.yaml | jq '.[] | {title, price}'

# Convert JSON to CSV
datascrapexter run -o - config.yaml | jq -r '.[] | [.title, .price] | @csv'

# Real-time monitoring
datascrapexter run config.yaml 2>&1 | tee scraping.log | grep ERROR
```

### Scheduled Execution

Integrate with cron for automated scraping:

```bash
# Add to crontab
0 6 * * * cd /opt/scrapers && datascrapexter run daily-products.yaml >> logs/daily.log 2>&1

# With error notification
0 6 * * * datascrapexter run config.yaml || echo "Scraping failed" | mail -s "Scraper Error" admin@example.com
```

## Debugging and Troubleshooting

### Verbose Output

Use increasing verbosity levels to diagnose issues:

```bash
datascrapexter -v run config.yaml     # Basic verbose output
datascrapexter -vv run config.yaml    # Detailed debugging
datascrapexter --log-level debug run config.yaml  # Maximum detail
```

### Debug Output Analysis

Filter debug output for specific components:

```bash
# Focus on HTTP requests
datascrapexter -vv run config.yaml 2>&1 | grep HTTP

# Watch selector matching
datascrapexter -vv run config.yaml 2>&1 | grep -i selector

# Monitor rate limiting
datascrapexter -vv run config.yaml 2>&1 | grep -i "rate limit"
```

### Dry Run Testing

Test configurations without making requests:

```bash
# Verify what would be scraped
datascrapexter run --dry-run config.yaml

# Check pagination logic
datascrapexter run --dry-run --log-level debug config.yaml | grep -i pagination
```

## Best Practices

### Configuration Management

Organize configurations systematically:

```bash
configs/
├── production/
│   ├── daily-news.yaml
│   └── product-monitor.yaml
├── development/
│   └── test-scraper.yaml
└── templates/
    └── base-config.yaml
```

### Logging Strategy

Implement comprehensive logging for production use:

```bash
# Create dated log files
LOG_DIR="/var/log/datascrapexter"
LOG_FILE="$LOG_DIR/scraper-$(date +%Y%m%d).log"
datascrapexter run config.yaml >> "$LOG_FILE" 2>&1
```

### Error Handling

Build robust error handling into automation:

```bash
#!/bin/bash
set -e  # Exit on error

# Function to handle errors
handle_error() {
    echo "Error occurred in scraping process" >&2
    # Add notification logic here
    exit 1
}

# Set error trap
trap handle_error ERR

# Run scraper with error checking
datascrapexter run config.yaml || handle_error
```

### Performance Monitoring

Track scraping performance over time:

```bash
# Time execution
time datascrapexter run config.yaml

# Monitor resource usage
/usr/bin/time -v datascrapexter run config.yaml

# Profile performance
datascrapexter run --pprof-port 6060 config.yaml &
go tool pprof http://localhost:6060/debug/pprof/profile
```

This CLI reference provides comprehensive coverage of DataScrapexter's command-line interface. Regular consultation ensures effective use of all available features and options. As DataScrapexter evolves, new commands and options will be added to enhance functionality while maintaining backward compatibility.
