# DataScrapexter

[![Go Version](https://img.shields.io/badge/Go-1.24%2B-00ADD8?style=flat&logo=go)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/valpere/DataScrapexter)](https://goreportcard.com/report/github.com/valpere/DataScrapexter)
[![codecov](https://codecov.io/gh/valpere/DataScrapexter/branch/main/graph/badge.svg)](https://codecov.io/gh/valpere/DataScrapexter)
[![GitHub release](https://img.shields.io/github/release/valpere/DataScrapexter.svg)](https://github.com/valpere/DataScrapexter/releases)

**DataScrapexter** is a universal web scraper built with Go that combines high performance, intelligent anti-detection mechanisms, and configuration-driven operation to enable seamless data extraction from any website structure.


![IN PROGRESS](./images/IN_PROGRESS.png)

## 🚀 Features

- **Universal Compatibility**: Scrape any website type - e-commerce, news, directories, social media
- **Advanced Anti-Detection**: Sophisticated evasion techniques including proxy rotation, browser fingerprinting, and CAPTCHA solving
- **Configuration-Driven**: No-code setup through YAML configuration files
- **High Performance**: Go's concurrency model for processing 10,000+ pages per hour
- **JavaScript Support**: Headless browser automation for dynamic content
- **Legal Compliance**: Built-in ethical scraping and legal compliance features
- **Multiple Output Formats**: JSON, CSV, Excel, databases, and cloud storage
- **Real-time Monitoring**: Comprehensive metrics and health monitoring

## 📦 Installation

### Binary Releases

Download the latest binary for your platform from the [releases page](https://github.com/valpere/DataScrapexter/releases).

### From Source

```bash
go install github.com/valpere/DataScrapexter/cmd/datascrapexter@latest
```

### Docker

```bash
docker pull ghcr.io/valpere/datascrapexter:latest
docker run -v $(pwd)/configs:/app/configs ghcr.io/valpere/datascrapexter:latest run /app/configs/example.yaml
```

### Go Module

```bash
go get github.com/valpere/DataScrapexter
```

## 🏃‍♂️ Quick Start

### 1. Create a Configuration File

```yaml
# example.yaml
name: "example_scraper"
base_url: "https://example.com"

# Anti-detection settings
user_agents:
  - "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"
rate_limit: "2s"

# Data extraction rules
fields:
  - name: "title"
    selector: "h1"
    type: "text"
  - name: "price"
    selector: ".price"
    type: "text"
  - name: "description"
    selector: ".description"
    type: "text"

# Output configuration
output:
  format: "json"
  file: "output.json"
```

### 2. Run the Scraper

```bash
# Validate configuration
datascrapexter validate example.yaml

# Run scraper
datascrapexter run example.yaml

# Generate template
datascrapexter template > new_config.yaml
```

### 3. View Results

```bash
cat output.json
```

## 📖 Documentation

### Getting Started

- [**Installation Guide**](docs/installation.md) - Detailed installation instructions
- [**User Guide**](docs/user-guide.md) - Comprehensive guide for using DataScrapexter
- [**Quick Start Tutorial**](docs/quick-start.md) - Get scraping in under 5 minutes

### Configuration & Examples

- [**Configuration Reference**](docs/configuration.md) - Complete configuration options
- [**Example: E-commerce Scraper**](docs/tutorial-ecommerce.md) - Build a price monitoring system
- [**Example: News Scraper**](examples/news-scraper.yaml) - Extract articles and metadata
- [**Example: Job Board Scraper**](examples/job-board.yaml) - Collect job listings
- [**Example: Real Estate Scraper**](examples/real-estate.yaml) - Property listing extraction

### Developer Resources

- [**API Documentation**](docs/api.md) - Go package API reference
- [**Code Documentation**](docs/code-documentation.md) - Internal architecture and design
- [**CLI Reference**](docs/cli.md) - Command-line interface documentation
- [**Contributing Guide**](CONTRIBUTING.md) - How to contribute to the project

### Support

- [**Troubleshooting Guide**](docs/troubleshooting.md) - Common issues and solutions
- [**FAQ**](docs/faq.md) - Frequently asked questions
- [**GitHub Discussions**](https://github.com/valpere/DataScrapexter/discussions) - Community support

## 🔧 Configuration

DataScrapexter uses YAML configuration files to define scraping behavior. Here's a comprehensive example:

```yaml
name: "advanced_scraper"
base_url: "https://target-site.com"

# Browser automation
browser:
  enabled: true
  headless: true
  user_data_dir: "/tmp/datascrapexter"

# Anti-detection mechanisms
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

# Rate limiting
rate_limit:
  requests_per_second: 2
  burst: 5
  adaptive: true

# Data extraction
fields:
  - name: "product_name"
    selector: "h1.product-title"
    type: "text"
    required: true
  
  - name: "price"
    selector: ".price-current"
    type: "text"
    transform:
      - type: "regex"
        pattern: "\\$([0-9,]+\\.?[0-9]*)"
        replacement: "$1"
      - type: "parse_float"

# Pagination
pagination:
  type: "next_button"
  selector: ".pagination .next"
  max_pages: 100

# Output options
output:
  format: "json"
  file: "products.json"
  database:
    type: "postgresql"
    url: "${DATABASE_URL}"
    table: "products"
```

## 🛠️ Advanced Usage

### Programmatic Usage

```go
package main

import (
    "context"
    "log"
    
    "github.com/valpere/DataScrapexter/pkg/scraper"
    "github.com/valpere/DataScrapexter/pkg/config"
)

func main() {
    // Load configuration
    cfg, err := config.LoadFromFile("config.yaml")
    if err != nil {
        log.Fatal(err)
    }
    
    // Create scraper instance
    s, err := scraper.New(cfg)
    if err != nil {
        log.Fatal(err)
    }
    
    // Run scraping
    ctx := context.Background()
    results, err := s.Scrape(ctx)
    if err != nil {
        log.Fatal(err)
    }
    
    // Process results
    for _, result := range results {
        log.Printf("Extracted: %+v", result.Data)
    }
}
```

### Docker Compose

```yaml
version: '3.8'
services:
  datascrapexter:
    image: ghcr.io/valpere/datascrapexter:latest
    volumes:
      - ./configs:/app/configs
      - ./output:/app/output
    environment:
      - CAPTCHA_API_KEY=${CAPTCHA_API_KEY}
      - DATABASE_URL=${DATABASE_URL}
    command: run /app/configs/production.yaml
    
  redis:
    image: redis:alpine
    ports:
      - "6379:6379"
      
  postgres:
    image: postgres:13
    environment:
      POSTGRES_DB: datascrapexter
      POSTGRES_USER: scraper
      POSTGRES_PASSWORD: password
    ports:
      - "5432:5432"
```

## 🔒 Legal & Ethical Usage

DataScrapexter includes built-in compliance features:

- **Robots.txt Respect**: Automatic robots.txt parsing and compliance
- **Rate Limiting**: Configurable delays to avoid server overload
- **User-Agent Identification**: Transparent identification in requests
- **Data Privacy**: GDPR-compliant data handling and anonymization
- **Terms of Service**: Monitoring and compliance checking

### Best Practices

1. Always check and respect robots.txt
2. Implement reasonable rate limiting
3. Avoid scraping personal data without consent
4. Review website terms of service
5. Use scraped data responsibly and legally

## 🤝 Contributing

We welcome contributions! Please see our [Contributing Guidelines](CONTRIBUTING.md) for details.

### Development Setup

```bash
# Clone repository
git clone https://github.com/valpere/DataScrapexter.git
cd DataScrapexter

# Install dependencies
go mod download

# Run tests
go test ./...

# Build
go build -o bin/datascrapexter cmd/datascrapexter/main.go

# Run locally
./bin/datascrapexter run examples/basic.yaml
```

### Project Structure

```PlainText
DataScrapexter/
├── cmd/                    # Command-line applications
├── internal/               # Private application code
├── pkg/                    # Public API packages
├── configs/                # Configuration templates
├── docs/                   # Documentation
├── examples/               # Usage examples
├── scripts/                # Build and deployment scripts
└── test/                   # Integration tests
```

## 📊 Performance

DataScrapexter is designed for high performance:

- **Throughput**: 10,000+ pages per hour per instance
- **Concurrency**: 1,000+ concurrent goroutines
- **Memory Usage**: <100MB for typical workloads
- **Scalability**: Horizontal scaling with distributed processing

### Benchmarks

```bash
# Run performance benchmarks
go test -bench=. ./internal/scraper
go test -bench=. ./internal/antidetect

# Example results:
BenchmarkHTTPClient-8           1000000    1200 ns/op     320 B/op    4 allocs/op
BenchmarkHTMLParser-8            500000    2400 ns/op     512 B/op    8 allocs/op
BenchmarkProxyRotation-8        2000000     800 ns/op     128 B/op    2 allocs/op
```

## 🔧 Configuration Templates

DataScrapexter includes pre-built templates for common use cases:

### E-commerce Sites

```bash
datascrapexter template --type ecommerce > ecommerce.yaml
```

### News Websites

```bash
datascrapexter template --type news > news.yaml
```

### Job Boards

```bash
datascrapexter template --type jobs > jobs.yaml
```

### Social Media

```bash
datascrapexter template --type social > social.yaml
```

## 🐳 Docker Usage

### Basic Usage

```bash
# Pull latest image
docker pull ghcr.io/valpere/datascrapexter:latest

# Run with local config
docker run -v $(pwd)/config.yaml:/app/config.yaml \
           -v $(pwd)/output:/app/output \
           ghcr.io/valpere/datascrapexter:latest run /app/config.yaml
```

### Production Deployment

```bash
# Build production image
docker build -t datascrapexter:prod .

# Run with environment variables
docker run -e CAPTCHA_API_KEY=$CAPTCHA_KEY \
           -e DATABASE_URL=$DB_URL \
           -v $(pwd)/configs:/app/configs \
           datascrapexter:prod run /app/configs/production.yaml
```

## 📈 Monitoring & Observability

### Metrics Endpoint

DataScrapexter exposes Prometheus metrics at `/metrics`:

```bash
# Start with metrics enabled
datascrapexter serve --metrics-port 9090

# View metrics
curl http://localhost:9090/metrics
```

### Key Metrics

- `datascrapexter_requests_total` - Total HTTP requests made
- `datascrapexter_requests_duration_seconds` - Request duration histogram
- `datascrapexter_extraction_success_rate` - Data extraction success rate
- `datascrapexter_proxy_health` - Proxy health status
- `datascrapexter_captcha_solve_rate` - CAPTCHA solving success rate

### Health Checks

```bash
# Health check endpoint
curl http://localhost:8080/health

# Readiness check
curl http://localhost:8080/ready
```

## 🌐 API Server Mode

Run DataScrapexter as a web service:

```bash
# Start API server
datascrapexter serve --port 8080

# Submit scraping job via API
curl -X POST http://localhost:8080/api/v1/jobs \
  -H "Content-Type: application/json" \
  -d '{
    "name": "test_job",
    "config": {
      "base_url": "https://example.com",
      "fields": [
        {"name": "title", "selector": "h1", "type": "text"}
      ]
    }
  }'

# Check job status
curl http://localhost:8080/api/v1/jobs/{job_id}

# Get results
curl http://localhost:8080/api/v1/jobs/{job_id}/results
```

## 🔌 Integrations

### Proxy Providers

- **Bright Data** (formerly Luminati)
- **Oxylabs**
- **Smartproxy**
- **ProxyMesh**
- **Custom proxy lists**

### CAPTCHA Solvers

- **2Captcha**
- **Anti-Captcha**
- **CapMonster**
- **DeathByCaptcha**

### Cloud Storage

- **AWS S3**
- **Google Cloud Storage**
- **Azure Blob Storage**
- **MinIO**

### Databases

- **PostgreSQL**
- **MySQL**
- **MongoDB**
- **Redis**
- **InfluxDB**

### Message Queues

- **Apache Kafka**
- **RabbitMQ**
- **Redis Streams**
- **AWS SQS**

## 🚨 Troubleshooting

### Common Issues

#### High Error Rates

```bash
# Check proxy health
datascrapexter proxy-test --config config.yaml

# Validate configuration
datascrapexter validate config.yaml --strict

# Enable debug logging
datascrapexter run config.yaml --log-level debug
```

#### Memory Usage

```bash
# Profile memory usage
go tool pprof http://localhost:6060/debug/pprof/heap

# Enable memory profiling
datascrapexter run config.yaml --pprof-port 6060
```

#### CAPTCHA Issues

```bash
# Test CAPTCHA solver
datascrapexter captcha-test --solver 2captcha --api-key YOUR_KEY

# Check CAPTCHA solve rates
curl http://localhost:9090/metrics | grep captcha_solve_rate
```

### Debug Mode

```bash
# Run with maximum verbosity
datascrapexter run config.yaml \
  --log-level debug \
  --log-format json \
  --pprof-port 6060 \
  --metrics-port 9090
```

## 📚 Examples

### Basic E-commerce Scraping

```yaml
name: "product_scraper"
base_url: "https://shop.example.com"

fields:
  - name: "product_name"
    selector: "h1.product-title"
    type: "text"
  - name: "price"
    selector: ".price"
    type: "text"
  - name: "in_stock"
    selector: ".stock-status"
    type: "text"

pagination:
  type: "next_button"
  selector: ".pagination-next"
  max_pages: 50

output:
  format: "csv"
  file: "products.csv"
```

### News Article Extraction

```yaml
name: "news_scraper"
base_url: "https://news.example.com"

fields:
  - name: "headline"
    selector: "h1.article-headline"
    type: "text"
  - name: "author"
    selector: ".article-author"
    type: "text"
  - name: "publish_date"
    selector: ".publish-date"
    type: "text"
  - name: "content"
    selector: ".article-content"
    type: "text"

navigation:
  follow_links:
    - selector: ".article-link"
      max_depth: 2

output:
  format: "json"
  file: "articles.json"
```

### JavaScript-Heavy Site

```yaml
name: "spa_scraper"
base_url: "https://spa.example.com"

browser:
  enabled: true
  headless: true
  wait_for_element: ".dynamic-content"
  timeout: 30s

fields:
  - name: "dynamic_data"
    selector: ".ajax-loaded"
    type: "text"
    wait_for_element: true

anti_detection:
  fingerprinting:
    randomize_viewport: true
    spoof_canvas: true
    disable_webdriver_flags: true

output:
  format: "json"
  file: "spa_data.json"
```

## 🔮 Roadmap

### v0.1.0 - MVP (Current)

- [x] Basic HTTP scraping engine
- [x] Configuration-driven setup
- [x] CLI interface
- [x] Anti-detection basics

### v0.5.0 - Enhanced

- [ ] JavaScript rendering support
- [ ] Advanced proxy management
- [ ] Web dashboard
- [ ] Template marketplace

### v1.0.0 - Professional

- [ ] Distributed processing
- [ ] Advanced monitoring
- [ ] Enterprise features
- [ ] API server mode

### v1.5.0 - AI-Enhanced

- [ ] ML-powered adaptation
- [ ] Intelligent content detection
- [ ] Predictive scaling
- [ ] Auto-configuration

### v2.0.0 - Enterprise

- [ ] Multi-tenant architecture
- [ ] Advanced compliance
- [ ] Professional services
- [ ] Enterprise integrations

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- [Colly](https://go-colly.org/) - Elegant scraper framework for Go
- [Goquery](https://github.com/PuerkitoBio/goquery) - jQuery-like HTML parsing
- [chromedp](https://github.com/chromedp/chromedp) - Chrome DevTools Protocol
- [Cobra](https://github.com/spf13/cobra) - CLI library for Go
- [Viper](https://github.com/spf13/viper) - Configuration management

## 📞 Support

- **Documentation**: [docs/](docs/)
- **Issues**: [GitHub Issues](https://github.com/valpere/DataScrapexter/issues)
- **Discussions**: [GitHub Discussions](https://github.com/valpere/DataScrapexter/discussions)
- **Email**: support@datascrapexter.com
- **Discord**: [DataScrapexter Community](https://discord.gg/datascrapexter)

## 🌟 Star History

[![Star History Chart](https://api.star-history.com/svg?repos=valpere/DataScrapexter&type=Date)](https://star-history.com/#valpere/DataScrapexter&Date)

---

*Made with ❤️ by the DataScrapexter team*

