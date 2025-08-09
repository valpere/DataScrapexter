# DataScrapexter

[![Go Version](https://img.shields.io/badge/Go-1.24%2B-00ADD8?style=flat&logo=go)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/valpere/DataScrapexter)](https://goreportcard.com/report/github.com/valpere/DataScrapexter)
[![codecov](https://codecov.io/gh/valpere/DataScrapexter/branch/main/graph/badge.svg)](https://codecov.io/gh/valpere/DataScrapexter)
[![GitHub release](https://img.shields.io/github/release/valpere/DataScrapexter.svg)](https://github.com/valpere/DataScrapexter/releases)

**DataScrapexter** is a professional web scraping platform built with Go 1.24+ that combines high performance, intelligent anti-detection mechanisms, and configuration-driven operation to enable seamless data extraction from any website structure.

![IN PROGRESS](./images/IN_PROGRESS.png)

## ✨ Key Features

### Core Capabilities

- **🎯 Universal Website Support**: Scrape any website type - e-commerce, news, directories, social media
- **⚙️ Configuration-Driven**: No-code setup through YAML configuration files
- **⚡ High Performance**: Go's concurrency model for processing 10,000+ pages per hour
- **🌐 JavaScript Support**: Headless browser automation for dynamic content
- **📊 Multiple Output Formats**: JSON, CSV, Excel, XML, YAML, databases, and cloud storage
- **📈 Real-time Monitoring**: Comprehensive metrics and health monitoring

### Anti-Detection

- **🛡️ Browser Fingerprinting Evasion**: Canvas, WebGL, audio spoofing
- **🔐 CAPTCHA Solving**: 2Captcha, Anti-Captcha, CapMonster integration
- **🔒 TLS Fingerprinting**: JA3/JA4 randomization
- **🌐 Proxy Management**: Residential and datacenter proxy rotation
- **🤖 Human-like Behavior**: Realistic timing and interaction patterns

### Enterprise Features

- **📊 Monitoring & Alerting**: Prometheus, Grafana, health checks
- **🏗️ High Availability**: Load balancing, failover, replication
- **☁️ Cloud Integration**: AWS, GCP, Azure support
- **🚢 Kubernetes Native**: Container orchestration ready
- **🔌 API Gateway Integration**: Enterprise architecture support
- **📋 Audit & Compliance**: GDPR compliance, audit logging

## 🚀 Quick Start

### 1. Install

```bash
# Download binary
curl -L https://github.com/valpere/DataScrapexter/releases/latest/download/datascrapexter-linux-amd64 -o datascrapexter
chmod +x datascrapexter

# Or install from source
go install github.com/valpere/DataScrapexter/cmd/datascrapexter@latest

# Or use Docker
docker pull ghcr.io/valpere/datascrapexter:latest
```

### 2. Create Configuration

```yaml
# config.yaml
name: "example_scraper"
base_url: "https://example.com"

# Data extraction
fields:
  - name: "title"
    selector: "h1"
    type: "text"
  - name: "price"
    selector: ".price"
    type: "text"

# Output
output:
  format: "json"
  file: "results.json"

# Optional: Rate limiting
rate_limit: "2s"
```

### 3. Run

```bash
# Validate configuration
datascrapexter validate config.yaml

# Run scraper
datascrapexter run config.yaml

# View results
cat results.json
```

## 📚 Complete Documentation

The documentation is organized in a logical reading order, from basic concepts to advanced enterprise features:

### 📚 [01. Getting Started](docs/01-getting-started.md)

*Start here for installation, first scraper, and basic concepts*

- Installation (binary, source, Docker)
- Quick start guide with first scraper
- Basic configuration structure
- Command line usage
- Next steps and learning path

### ⚙️ [02. Configuration Reference](docs/02-configuration-reference.md)

*Complete reference for all configuration options*

- Configuration file structure
- Request settings (rate limiting, timeouts, headers)
- Data extraction (fields, selectors, transformations)
- Pagination strategies
- Output formats and destinations
- Anti-detection settings
- Monitoring configuration
- Environment variables

### 🎯 [03. Examples and Tutorials](docs/03-examples-and-tutorials.md)

*Real-world examples and step-by-step tutorials*

- Basic examples (quotes, books)
- E-commerce scraping (price monitoring, product comparison)
- News and content collection
- Real estate listings
- Job board scraping
- Advanced scenarios (multi-site, SPA)
- Production configurations

### 🔧 [04. Advanced Features](docs/04-advanced-features.md)

*Sophisticated features for enterprise-grade scraping*

- Anti-detection technologies
- Monitoring and observability
- Output formats and destinations
- Browser automation
- Proxy management
- Performance optimization
- Enterprise features

### 🔌 [05. API Reference](docs/05-api-reference.md)

*Complete Go programming interface documentation*

- Core packages and interfaces
- Configuration types
- Scraping engine API
- Data processing and transformations
- Output management
- Monitoring integration
- Anti-detection features
- Error handling
- Complete code examples

### 🔍 [06. Troubleshooting](docs/06-troubleshooting.md)

*Solutions to common issues and debugging guide*

- Quick diagnostics
- Configuration issues
- Scraping failures
- Anti-detection problems
- Performance issues
- Output problems
- Monitoring and health issues
- Deployment issues
- Best practices for prevention

## 📖 Reading Paths for Different Users

Choose your learning path based on your role and needs:

- **🆕 New Users**: Start with [01. Getting Started](docs/01-getting-started.md) → [02. Configuration Reference](docs/02-configuration-reference.md) → [03. Examples and Tutorials](docs/03-examples-and-tutorials.md)
- **⚙️ Configuration Focus**: [02. Configuration Reference](docs/02-configuration-reference.md) → [03. Examples and Tutorials](docs/03-examples-and-tutorials.md) → [06. Troubleshooting](docs/06-troubleshooting.md)
- **👨‍💻 Developers**: [05. API Reference](docs/05-api-reference.md) → [04. Advanced Features](docs/04-advanced-features.md) → [06. Troubleshooting](docs/06-troubleshooting.md)
- **🚀 Production Users**: [04. Advanced Features](docs/04-advanced-features.md) → [05. API Reference](docs/05-api-reference.md) → [06. Troubleshooting](docs/06-troubleshooting.md)
- **🔧 Troubleshooting**: [06. Troubleshooting](docs/06-troubleshooting.md) → [02. Configuration Reference](docs/02-configuration-reference.md) → [05. API Reference](docs/05-api-reference.md)

## 💡 Usage Examples

<details>
<summary><strong>E-commerce Product Scraping</strong></summary>

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
    transform:
      - type: "regex"
        pattern: "\\$([0-9,]+\\.?[0-9]*)"
        replacement: "$1"
      - type: "parse_float"

pagination:
  type: "next_button"
  selector: ".pagination-next"
  max_pages: 50

output:
  format: "csv"
  file: "products.csv"
```

</details>

<details>
<summary><strong>News Article Extraction</strong></summary>

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

</details>

<details>
<summary><strong>JavaScript-Heavy Site (SPA)</strong></summary>

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

output:
  format: "json"
  file: "spa_data.json"
```

</details>

<details>
<summary><strong>Database Output (PostgreSQL)</strong></summary>

```yaml
name: "db_scraper"
base_url: "https://example.com"

fields:
  - name: "title"
    selector: "h1"
    type: "text"
  - name: "url"
    selector: "a"
    type: "attr"
    attribute: "href"

output:
  database:
    type: "postgresql"
    connection_string: "${DATABASE_URL}"
    table: "scraped_data"
    batch_size: 1000
    create_table: true
    on_conflict: "ignore"
```

</details>

## 🔧 Advanced Configuration

<details>
<summary><strong>Anti-Detection Setup</strong></summary>

```yaml
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

rate_limit:
  strategy: "adaptive"
  base_delay: "2s"
  max_delay: "30s"
```

</details>

<details>
<summary><strong>Docker Deployment</strong></summary>

```yaml
# docker-compose.yml
version: '3.8'
services:
  datascrapexter:
    image: ghcr.io/valpere/datascrapexter:latest
    volumes:
      - ./configs:/app/configs
      - ./output:/app/output
    environment:
      - DATABASE_URL=${DATABASE_URL}
      - CAPTCHA_API_KEY=${CAPTCHA_API_KEY}
    command: run /app/configs/production.yaml

  postgres:
    image: postgres:13
    environment:
      POSTGRES_DB: datascrapexter
      POSTGRES_USER: scraper
      POSTGRES_PASSWORD: password
```

</details>

<details>
<summary><strong>Programmatic Usage</strong></summary>

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

    // Create scraper
    s, err := scraper.New(cfg)
    if err != nil {
        log.Fatal(err)
    }

    // Run scraping
    results, err := s.Scrape(context.Background())
    if err != nil {
        log.Fatal(err)
    }

    // Process results
    for _, result := range results {
        log.Printf("Data: %+v", result.Data)
    }
}
```

</details>

## 🛠️ CLI Commands

```bash
# Configuration
datascrapexter validate <config.yaml>     # Validate configuration
datascrapexter template [--type <type>]   # Generate templates

# Execution
datascrapexter run <config.yaml>          # Run scraper
datascrapexter serve [--port 8080]        # Start API server

# Testing & Debugging
datascrapexter proxy-test --config <config.yaml>     # Test proxies
datascrapexter captcha-test --solver 2captcha        # Test CAPTCHA
datascrapexter health                                 # Health check
```

## 🏗️ Architecture

```plaintext
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   CLI/API       │    │  Scraping Engine │    │  Anti-Detection │
│                 │────│                  │────│                 │
│ • Configuration │    │ • HTTP Client    │    │ • Fingerprinting│
│ • Validation    │    │ • Browser Pool   │    │ • CAPTCHA Solver│
│ • Monitoring    │    │ • Rate Limiting  │    │ • Proxy Manager │
└─────────────────┘    └──────────────────┘    └─────────────────┘
         │                       │                       │
         v                       v                       v
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│  Data Pipeline  │    │   Output Engine  │    │   Monitoring    │
│                 │    │                  │    │                 │
│ • Extraction    │────│ • Format Writers │    │ • Metrics       │
│ • Validation    │    │ • Cloud Upload   │    │ • Health Checks │
│ • Transform     │    │ • Database Store │    │ • Dashboard     │
└─────────────────┘    └──────────────────┘    └─────────────────┘
```

## 📊 Performance & Monitoring

**Performance Metrics:**

- **Throughput**: 10,000+ pages/hour per instance
- **Concurrency**: 1,000+ concurrent goroutines
- **Memory**: <100MB for typical workloads
- **Latency**: <1200ns/op for HTTP operations

**Monitoring Endpoints:**

```bash
# Health checks
curl http://localhost:8080/health
curl http://localhost:8080/ready

# Prometheus metrics
curl http://localhost:9090/metrics
```

**Key Metrics:**

- `datascrapexter_requests_total` - Total requests
- `datascrapexter_requests_duration_seconds` - Request latency
- `datascrapexter_extraction_success_rate` - Success rate
- `datascrapexter_proxy_health` - Proxy status
- `datascrapexter_captcha_solve_rate` - CAPTCHA solving success rate

**Proxy Monitoring Configuration:**

DataScrapexter provides configurable retention policies for different deployment scenarios:

```yaml
proxy:
  monitoring:
    enabled: true
    history_retention: 24h      # Data retention period
    max_query_period: 168h      # Max query period (NEW: configurable)
    alerting_enabled: true
    budget_config:
      daily_budget: 100.0
```

**Historical Metrics API:**

```bash
# Query historical proxy metrics
curl "http://localhost:9090/metrics/historical/proxy_name?period=24h"
curl "http://localhost:9090/metrics/historical/proxy_name?period=72h"

# Deployment-specific configs:
# Development: max_query_period: 6h
# Production: max_query_period: 720h (1 month)
# High-volume: max_query_period: 168h (1 week)
```

## 🔌 Integrations

| Category | Supported Services |
|----------|-------------------|
| **Proxies** | Bright Data, Oxylabs, Smartproxy, ProxyMesh, Custom |
| **CAPTCHA** | 2Captcha, Anti-Captcha, CapMonster, DeathByCaptcha |
| **Databases** | PostgreSQL, MySQL, SQLite, MongoDB, Redis |
| **Storage** | AWS S3, Google Cloud, Azure Blob, MinIO |
| **Queues** | Kafka, RabbitMQ, Redis Streams, AWS SQS |

## ⚖️ Legal & Compliance

DataScrapexter includes built-in compliance features:

✅ **Robots.txt Respect** - Automatic parsing and compliance
✅ **Rate Limiting** - Configurable delays to prevent overload
✅ **Transparent Identification** - Proper User-Agent headers
✅ **Data Privacy** - GDPR-compliant handling
✅ **Terms Monitoring** - Compliance checking tools

**Best Practices:**

1. Always check robots.txt before scraping
2. Implement reasonable rate limiting (2+ seconds)
3. Respect website terms of service
4. Avoid scraping personal data without consent
5. Use scraped data responsibly and legally

## 🤝 Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md).

**Quick Development Setup:**

```bash
git clone https://github.com/valpere/DataScrapexter.git
cd DataScrapexter
go mod download
go test ./...
go build -o bin/datascrapexter cmd/datascrapexter/main.go
```

## 🚨 Troubleshooting

<details>
<summary><strong>Common Issues</strong></summary>

**High Error Rates:**

```bash
datascrapexter proxy-test --config config.yaml
datascrapexter validate config.yaml --strict
datascrapexter run config.yaml --log-level debug
```

**Memory Issues:**

```bash
datascrapexter run config.yaml --pprof-port 6060
go tool pprof http://localhost:6060/debug/pprof/heap
```

**CAPTCHA Problems:**

```bash
datascrapexter captcha-test --solver 2captcha --api-key YOUR_KEY
curl http://localhost:9090/metrics | grep captcha_solve_rate
```

</details>

**Need Help?** Check our [Troubleshooting Guide](docs/troubleshooting.md) or [FAQ](docs/faq.md).

## 🗺️ Roadmap

| Version | Status | Features |
|---------|--------|----------|
| **v0.1.0** | ✅ Current | HTTP scraping, CLI, anti-detection basics |
| **v0.5.0** | 🔄 In Progress | JavaScript support, advanced proxies, dashboard |
| **v1.0.0** | 📋 Planned | Distributed processing, enterprise features |
| **v1.5.0** | 💭 Future | AI-powered adaptation, ML content detection |

[Full Roadmap](docs/ARCHITECTURE.md#future-roadmap)

## 📄 License

MIT License - see [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

Built with: [Colly](https://go-colly.org/) • [Goquery](https://github.com/PuerkitoBio/goquery) • [chromedp](https://github.com/chromedp/chromedp) • [Cobra](https://github.com/spf13/cobra) • [Viper](https://github.com/spf13/viper)

## 📞 Support

- 📖 **Documentation**: [docs/](docs/)
- 🐛 **Issues**: [GitHub Issues](https://github.com/valpere/DataScrapexter/issues)
- 💬 **Discussions**: [GitHub Discussions](https://github.com/valpere/DataScrapexter/discussions)
- ✉️ **Email**: <support@datascrapexter.com>

---

⭐ **Star us on GitHub** if you find DataScrapexter useful!

*Made with ❤️ by the DataScrapexter team*
