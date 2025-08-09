# CLAUDE.md - DataScrapexter Project Documentation

## Project Overview

DataScrapexter is a professional web scraping platform built with Go 1.24+ that combines high performance, intelligent anti-detection mechanisms, and configuration-driven operation to enable seamless data extraction from any website structure.

### Core Capabilities
- **Universal Compatibility**: Scrape any website type - e-commerce, news, directories, social media
- **Advanced Anti-Detection**: Sophisticated evasion techniques including proxy rotation, browser fingerprinting, and CAPTCHA solving
- **Configuration-Driven**: No-code setup through YAML configuration files
- **High Performance**: Go's concurrency model for processing 10,000+ pages per hour
- **JavaScript Support**: Headless browser automation for dynamic content
- **Legal Compliance**: Built-in ethical scraping and legal compliance features
- **Multiple Output Formats**: JSON, CSV, Excel, databases, and cloud storage
- **Real-time Monitoring**: Comprehensive metrics and health monitoring

## Technical Stack

### Primary Technologies
- **Language**: Go 1.24+
- **Auxiliary Languages**: Bash, Perl (for scripts and utilities)
- **IDE**: VSCode

### Core Dependencies
- **Web Scraping Framework**: Colly
- **HTML Parsing**: Goquery (jQuery-like HTML parsing)
- **Browser Automation**: chromedp (Chrome DevTools Protocol)
- **CLI Framework**: Cobra
- **Configuration Management**: Viper
- **ORM**: GORM
- **Logging**: logrus
- **Metrics**: Prometheus

### Anti-Detection Technologies
- chromedp for browser automation
- Rod (alternative browser automation)
- 2Captcha, Anti-Captcha, CapMonster, DeathByCaptcha integration
- TLS fingerprinting (JA3/JA4 randomization)
- Canvas/WebGL spoofing

## Project Structure

```
DataScrapexter/
├── cmd/                      # Command-line applications
│   └── datascrapexter/      # Main CLI application
│       └── main.go          # Entry point
├── internal/                 # Private application code
│   ├── types.go             # Common types for internal package
│   ├── scraper/             # Core scraping logic
│   │   ├── types.go         # Scraper-specific types
│   │   ├── engine.go        # Main scraping engine
│   │   ├── config.go        # Configuration handling
│   │   └── extractor.go     # Data extraction logic
│   ├── output/              # Output handling
│   │   ├── types.go         # Output-specific types
│   │   ├── json.go          # JSON output formatter
│   │   ├── csv.go           # CSV output formatter
│   │   └── database.go      # Database output handler
│   ├── antidetect/          # Anti-detection mechanisms
│   │   ├── types.go         # Anti-detection types
│   │   ├── proxy.go         # Proxy management
│   │   ├── fingerprint.go   # Browser fingerprinting
│   │   └── captcha.go       # CAPTCHA solving
│   ├── browser/             # Browser automation
│   │   ├── types.go         # Browser-specific types
│   │   ├── pool.go          # Browser pool management
│   │   └── driver.go        # Browser driver implementation
│   └── monitor/             # Monitoring and metrics
│       ├── types.go         # Monitoring types
│       ├── metrics.go       # Prometheus metrics
│       └── health.go        # Health check endpoints
├── pkg/                     # Public API packages
│   ├── scraper/             # Public scraping API
│   ├── config/              # Configuration API
│   └── client/              # Client library
├── configs/                 # Configuration templates
│   ├── examples/            # Example configurations
│   └── templates/           # Reusable templates
├── docs/                    # Documentation
│   ├── api.md              # API documentation
│   ├── configuration.md    # Configuration reference
│   └── architecture.md     # Architecture documentation
├── examples/                # Usage examples
│   ├── basic.yaml          # Basic scraping example
│   ├── ecommerce.yaml      # E-commerce scraping
│   └── news.yaml           # News scraping
├── scripts/                 # Build and deployment scripts
│   ├── build.sh            # Build script
│   └── deploy.sh           # Deployment script
├── test/                    # Integration tests
│   ├── integration/        # Integration test suites
│   └── fixtures/           # Test fixtures
├── .gitignore
├── .golangci.yml           # Go linting configuration
├── Dockerfile              # Container image definition
├── docker-compose.yml      # Local development setup
├── go.mod                  # Go module definition
├── go.sum                  # Go dependency checksums
├── LICENSE                 # MIT License
├── Makefile               # Build automation
└── README.md              # Project documentation
```

## Design Principles

The project strictly adheres to the following design principles:

### Core Principles
1. **DRY (Don't Repeat Yourself)**: Avoid code duplication by creating reusable components and functions
2. **YAGNI (You Aren't Gonna Need It)**: Only implement features when they're actually needed
3. **KISS (Keep It Simple, Stupid)**: Prefer simple, straightforward solutions over complex ones
4. **Encapsulation**: Hide implementation details and expose clean interfaces
5. **PoLA (Principle of Least Astonishment)**: Design APIs and behaviors to be intuitive and predictable

### SOLID Principles
- **Single Responsibility**: Each type/function should have one clear purpose
- **Open/Closed**: Open for extension, closed for modification
- **Liskov Substitution**: Derived types must be substitutable for base types
- **Interface Segregation**: Many specific interfaces are better than one general interface
- **Dependency Inversion**: Depend on abstractions, not concretions

### GRASP (General Responsibility Assignment Software Patterns)
- Proper responsibility assignment based on information expert, creator, and controller patterns

## Type Organization Rules

### Type Hierarchy
Types are organized in a hierarchical structure to avoid conflicts and maintain clarity:

1. **Common Types** (`internal/types.go`):
   - Types used across multiple internal packages
   - Base interfaces and structs
   - Common error types

2. **Package-Specific Types** (`internal/*/types.go`):
   - Types specific to a package domain
   - Located at package root (e.g., `internal/scraper/types.go`)
   - Take precedence over lower-level types

3. **Conflict Resolution**:
   - When type conflicts occur, keep types in the highest-level `types.go` file
   - Remove duplicates from lower-level files
   - Types in parent directories override types in subdirectories

### Example Type Organization
```go
// internal/types.go
package internal

type Config struct {
    Name    string
    BaseURL string
}

type Result struct {
    Data  map[string]interface{}
    Error error
}

// internal/scraper/types.go
package scraper

type Engine struct {
    config *internal.Config
    // scraper-specific fields
}

type ExtractorFunc func(doc *goquery.Document) (*internal.Result, error)

// internal/output/types.go
package output

type Formatter interface {
    Format(result *internal.Result) ([]byte, error)
}

type OutputConfig struct {
    Format string
    Path   string
}
```

## Configuration System

DataScrapexter uses YAML-based configuration files managed by Viper. Configuration files define:

### Basic Configuration Structure
```yaml
name: "scraper_name"
base_url: "https://target-site.com"

# Rate limiting
rate_limit: "2s"  # or complex configuration

# User agents
user_agents:
  - "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"
  - "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36"

# Data extraction
fields:
  - name: "title"
    selector: "h1"
    type: "text"
    required: true
  - name: "price"
    selector: ".price"
    type: "text"
    transform:
      - type: "regex"
        pattern: "\\$([0-9,]+\\.?[0-9]*)"
        replacement: "$1"
      - type: "parse_float"

# Output configuration
output:
  format: "json"
  file: "output.json"
```

### Advanced Configuration Features
```yaml
# Browser automation
browser:
  enabled: true
  headless: true
  user_data_dir: "/tmp/datascrapexter"
  wait_for_element: ".dynamic-content"
  timeout: 30s

# Anti-detection
anti_detection:
  proxy:
    enabled: true
    rotation: "random"
    providers:
      - "http://proxy1:8080"
      - "http://proxy2:8080"
  captcha:
    solver: "2captcha"
    api_key: "${CAPTCHA_API_KEY}"
  fingerprinting:
    randomize_viewport: true
    spoof_canvas: true
    rotate_user_agents: true

# Pagination
pagination:
  type: "next_button"
  selector: ".pagination .next"
  max_pages: 100

# Database output
output:
  database:
    type: "postgresql"
    url: "${DATABASE_URL}"
    table: "scraped_data"
```

## CLI Commands

The CLI is built with Cobra and provides the following commands:

### Core Commands
```bash
# Run scraper
datascrapexter run <config.yaml>

# Validate configuration
datascrapexter validate <config.yaml>

# Generate template
datascrapexter template [--type <type>] > new_config.yaml

# Server mode
datascrapexter serve [--port 8080] [--metrics-port 9090]
```

### Utility Commands
```bash
# Test proxy configuration
datascrapexter proxy-test --config <config.yaml>

# Test CAPTCHA solver
datascrapexter captcha-test --solver 2captcha --api-key <key>

# Check health
datascrapexter health

# View metrics
datascrapexter metrics --job-id <id>
```

## API Design

### Public Package API (`pkg/scraper`)
```go
package scraper

import (
    "context"
    "github.com/valpere/DataScrapexter/pkg/config"
)

// Scraper interface defines the public API
type Scraper interface {
    Scrape(ctx context.Context) ([]*Result, error)
    Stop() error
    Status() *Status
}

// New creates a new scraper instance
func New(cfg *config.Config) (Scraper, error)

// Result represents scraped data
type Result struct {
    URL       string
    Data      map[string]interface{}
    Timestamp time.Time
}

// Status represents scraper status
type Status struct {
    Running      bool
    PagesScraped int
    Errors       int
    StartTime    time.Time
}
```

### REST API (Server Mode)
```
POST   /api/v1/jobs              - Submit scraping job
GET    /api/v1/jobs/{id}         - Get job status
GET    /api/v1/jobs/{id}/results - Get job results
DELETE /api/v1/jobs/{id}         - Cancel job
GET    /health                   - Health check
GET    /ready                    - Readiness check
GET    /metrics                  - Prometheus metrics
```

## Package Evolution

### v0.1.0 - Basic Package (MVP)
- Core HTTP scraping with Goquery
- YAML configuration support
- JSON/CSV output
- Basic rate limiting and User-Agent rotation
- CLI interface

### v0.5.0 - Standard Package
- JavaScript rendering (chromedp)
- Proxy rotation
- Database support (SQLite, PostgreSQL)
- Configuration hot-reloading
- Browser pool management

### v1.0.0 - Premium Package
- Advanced anti-detection (CAPTCHA solving, fingerprinting)
- Distributed processing with Redis
- Web dashboard
- RESTful API
- Prometheus monitoring
- Enterprise proxy support

### v1.5.0 - Advanced Package
- AI-powered adaptation
- Machine learning for extraction rules
- Natural language processing
- Kubernetes-based auto-scaling
- Advanced analytics

### v2.0.0 - Custom Features
- Bespoke anti-detection strategies
- Custom integrations
- Private deployments
- Compliance automation
- Professional services

## Development Guidelines

### Code Style
- Use Go standard formatting (`gofmt`)
- Follow effective Go patterns
- Write clear, self-documenting code
- Add comments for complex logic
- Use meaningful variable names (short names for loops: `i`, `j`, `e` for events)

### Testing
```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run benchmarks
go test -bench=. ./internal/scraper

# Run integration tests
go test ./test/integration/...
```

### Building
```bash
# Build for current platform
go build -o bin/datascrapexter cmd/datascrapexter/main.go

# Build with version info
go build -ldflags "-X main.version=1.0.0" -o bin/datascrapexter cmd/datascrapexter/main.go

# Build Docker image
docker build -t datascrapexter:latest .

# Cross-platform builds
GOOS=linux GOARCH=amd64 go build -o bin/datascrapexter-linux-amd64
GOOS=darwin GOARCH=amd64 go build -o bin/datascrapexter-darwin-amd64
GOOS=windows GOARCH=amd64 go build -o bin/datascrapexter-windows-amd64.exe
```

## Performance Characteristics

### Benchmarks
- HTTP Client: ~1200 ns/op, 320 B/op, 4 allocs/op
- HTML Parser: ~2400 ns/op, 512 B/op, 8 allocs/op
- Proxy Rotation: ~800 ns/op, 128 B/op, 2 allocs/op

### Resource Usage
- Memory: <100MB for typical workloads
- CPU: Scales with goroutine count
- Throughput: 10,000+ pages/hour per instance
- Concurrency: 1,000+ concurrent goroutines

## Monitoring and Observability

### Prometheus Metrics
- `datascrapexter_requests_total` - Total HTTP requests
- `datascrapexter_requests_duration_seconds` - Request duration histogram
- `datascrapexter_extraction_success_rate` - Data extraction success rate
- `datascrapexter_proxy_health` - Proxy health status
- `datascrapexter_captcha_solve_rate` - CAPTCHA solving success rate

### Health Endpoints
- `/health` - Basic health check
- `/ready` - Readiness check
- `/metrics` - Prometheus metrics endpoint

### Proxy Monitoring Configuration

DataScrapexter provides comprehensive proxy monitoring with configurable retention policies:

```yaml
# Proxy monitoring configuration
proxy:
  monitoring:
    enabled: true
    metrics_port: 9090
    detailed_metrics: true
    
    # Data retention configuration
    history_retention: 24h      # How long to keep historical data
    max_query_period: 168h      # Maximum allowed query period (7 days default)
    
    # Alerting configuration
    alerting_enabled: true
    alert_thresholds:
      failure_rate: 0.05        # 5% failure rate threshold
      latency_p95: 5000         # 5 second P95 latency threshold
      budget_threshold: 0.80    # 80% of budget threshold
    
    # Budget monitoring
    budget_config:
      daily_budget: 100.0       # $100/day
      hourly_budget: 5.0        # $5/hour
    
    realtime_updates: true
    export_prometheus: true
    export_interval: 1m
```

#### Configuration Options

- **`history_retention`**: How long to keep historical data in memory/storage
- **`max_query_period`**: Maximum allowed period for historical queries (prevents excessive memory usage)
  - Default: 168h (7 days) if not specified
  - Configurable for different deployment scenarios
  - Queries exceeding this limit are automatically clamped with warnings

#### Deployment-Specific Configurations

**Development Environment:**
```yaml
history_retention: 2h
max_query_period: 6h          # Short retention for testing
```

**Production Environment:**
```yaml
history_retention: 168h       # 1 week
max_query_period: 720h        # 1 month for comprehensive analysis
```

**High-Volume Production:**
```yaml
history_retention: 72h        # 3 days
max_query_period: 168h        # 1 week (balanced performance)
```

#### Historical Metrics API

Access historical proxy metrics with configurable query periods:

```bash
# Valid queries (within max_query_period)
GET /metrics/historical/proxy_name?period=24h
GET /metrics/historical/proxy_name?period=72h

# Queries exceeding limit are clamped with warnings
GET /metrics/historical/proxy_name?period=200h  # Clamped to max_query_period
```

## Integration Points

### Supported Proxies
- Bright Data (Luminati)
- Oxylabs
- Smartproxy
- ProxyMesh
- Custom proxy lists

### CAPTCHA Solvers
- 2Captcha
- Anti-Captcha
- CapMonster
- DeathByCaptcha

### Storage Backends
- Local filesystem
- AWS S3
- Google Cloud Storage
- Azure Blob Storage
- MinIO

### Databases
- PostgreSQL
- MySQL
- MongoDB
- Redis
- InfluxDB

### Message Queues
- Apache Kafka
- RabbitMQ
- Redis Streams
- AWS SQS

## Error Handling

### Error Types
```go
// internal/types.go
type ScraperError struct {
    Type    ErrorType
    Message string
    Cause   error
}

type ErrorType int

const (
    ErrorTypeNetwork ErrorType = iota
    ErrorTypeParsing
    ErrorTypeValidation
    ErrorTypeRateLimit
    ErrorTypeCaptcha
    ErrorTypeProxy
)
```

### Retry Logic
- Exponential backoff for network errors
- Proxy rotation on connection failures
- CAPTCHA retry with different solvers
- Configurable max retries per error type

## Security Considerations

### Credential Management
- Environment variables for sensitive data
- Integration with HashiCorp Vault
- Encrypted configuration files
- No hardcoded credentials

### Network Security
- TLS certificate validation
- Proxy authentication support
- Rate limiting to prevent abuse
- User-Agent rotation

## Legal and Compliance

### Built-in Features
- Robots.txt parsing and compliance
- Rate limiting and politeness delays
- User-Agent identification
- GDPR/CCPA compliance tools
- Audit logging

### Best Practices
- Always check robots.txt
- Implement reasonable delays
- Respect website terms of service
- Handle personal data responsibly
- Document data usage

## Deployment Options

### Docker
```dockerfile
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o datascrapexter cmd/datascrapexter/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY --from=builder /app/datascrapexter /usr/local/bin/
ENTRYPOINT ["datascrapexter"]
```

### Kubernetes
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: datascrapexter
spec:
  replicas: 3
  template:
    spec:
      containers:
      - name: scraper
        image: datascrapexter:latest
        resources:
          limits:
            memory: "512Mi"
            cpu: "1000m"
```

### Docker Compose
```yaml
version: '3.8'
services:
  datascrapexter:
    image: datascrapexter:latest
    volumes:
      - ./configs:/app/configs
      - ./output:/app/output
    environment:
      - CAPTCHA_API_KEY=${CAPTCHA_API_KEY}
      - DATABASE_URL=${DATABASE_URL}
    command: run /app/configs/production.yaml
```

## Common Issues and Solutions

### Rate Limiting
- Implement exponential backoff
- Use proxy rotation
- Respect server-indicated delays
- Configure reasonable concurrency limits

### Dynamic Content
- Enable browser mode for JavaScript
- Wait for specific elements
- Use appropriate timeouts
- Handle AJAX requests

### Anti-Bot Protection
- Rotate User-Agents
- Use residential proxies
- Implement human-like delays
- Solve CAPTCHAs automatically

## Future Roadmap

### Short Term (3-6 months)
- Enhanced browser fingerprinting evasion
- More CAPTCHA solver integrations
- Improved configuration validation
- Better error recovery mechanisms

### Medium Term (6-12 months)
- Web-based configuration builder
- Real-time configuration updates
- Advanced monitoring dashboard
- Template marketplace

### Long Term (12+ months)
- AI-powered site adaptation
- Automated compliance reporting
- Multi-tenant SaaS platform
- Enterprise API gateway

## Contributing Guidelines

### Code Contributions
1. Fork the repository
2. Create a feature branch
3. Write tests for new functionality
4. Ensure all tests pass
5. Submit a pull request

### Documentation
- Update relevant documentation
- Add examples for new features
- Keep README.md current
- Document breaking changes

### Testing Requirements
- Unit tests for all new functions
- Integration tests for new features
- Benchmark tests for performance-critical code
- Example configurations for new functionality

## Support and Resources

### Documentation
- Main site: https://valpere.github.io/datascrapexter/
- API docs: https://valpere.github.io/datascrapexter/docs/
- GitHub: https://github.com/valpere/DataScrapexter

### Community
- GitHub Issues: Bug reports and feature requests
- GitHub Discussions: Community support
- Discord: Real-time chat (planned)
- Email: support@datascrapexter.com (planned)

### Commercial Support
- Premium packages available
- Custom feature development
- Professional services
- Training and consulting

## License

DataScrapexter is licensed under the MIT License. See LICENSE file for details.
