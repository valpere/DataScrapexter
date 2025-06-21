# Docker Setup for DataScrapexter

This document provides comprehensive guidance for running DataScrapexter in Docker containers, including both production and development configurations.

## Prerequisites

Before proceeding with the Docker setup, ensure you have the following installed on your system:

- Docker Engine version 20.10 or higher
- Docker Compose version 2.0 or higher
- At least 4GB of available RAM (8GB recommended for running with Selenium Grid)
- 10GB of free disk space for images and data storage

## Quick Start

The simplest way to get DataScrapexter running with Docker is to use the provided docker-compose configuration:

```bash
# Clone the repository
git clone https://github.com/valpere/DataScrapexter.git
cd DataScrapexter

# Build and start the container
docker-compose up -d

# View logs
docker-compose logs -f datascrapexter

# Run a scraping job
docker-compose exec datascrapexter datascrapexter run configs/example.yaml
```

## Docker Image Details

The DataScrapexter Docker image is built using a multi-stage process to minimize the final image size while ensuring all necessary dependencies are included.

### Build Stage

The build stage uses the official Go Alpine image to compile the DataScrapexter binary. This stage includes all build dependencies such as Git, Make, GCC, and musl-dev. The application is compiled with optimizations for the target platform.

### Runtime Stage

The runtime stage is based on Alpine Linux and includes the following components:

- Chromium browser and ChromeDriver for web scraping capabilities
- Perl runtime with required modules for script execution
- Python 3 with PyYAML for configuration validation
- Essential utilities including bash, jq, and curl
- TLS certificates for secure HTTPS connections

The image is configured to run as a non-root user for enhanced security, with user ID 1000 and group ID 1000. All application directories are created with appropriate permissions.

## Configuration Options

### Environment Variables

The Docker container supports configuration through environment variables:

- `LOG_LEVEL`: Sets the logging verbosity (debug, info, warn, error)
- `SCRAPER_USER_AGENT`: Defines the User-Agent string for HTTP requests
- `SCRAPER_TIMEOUT`: Request timeout in seconds (default: 30)
- `SCRAPER_RETRY_ATTEMPTS`: Number of retry attempts for failed requests
- `SCRAPER_RETRY_DELAY`: Delay between retry attempts in seconds
- `TZ`: Timezone setting (default: UTC)

### Volume Mounts

The following directories should be mounted as volumes to persist data and configurations:

- `/app/configs`: Configuration files for scraping jobs
- `/app/outputs`: Scraped data output files
- `/app/logs`: Application and script logs
- `/app/data`: Persistent data storage
- `/app/scripts`: Custom scripts (optional, included in image)

## Using Docker Compose

The project includes several Docker Compose configurations for different use cases.

### Basic Setup

The default `docker-compose.yml` provides the core DataScrapexter service with essential volume mounts and network configuration. This setup is suitable for standalone operation where DataScrapexter handles all scraping tasks internally.

### Production Setup with Optional Services

The production configuration includes optional service profiles that can be enabled based on your requirements:

#### Selenium Grid Profile

Enable distributed scraping with Selenium Grid by activating the selenium profile:

```bash
docker-compose --profile selenium up -d
```

This starts a Selenium Hub with Chrome nodes for handling browser-based scraping at scale. The configuration includes three Chrome node replicas by default, which can be adjusted based on your workload.

#### Caching Profile

Enable Redis caching for improved performance:

```bash
docker-compose --profile cache up -d
```

Redis provides in-memory caching for frequently accessed data and can serve as a job queue for distributed scraping operations.

#### Database Profile

Enable PostgreSQL for structured data storage:

```bash
docker-compose --profile database up -d
```

The database is configured with a dedicated schema for DataScrapexter and includes health checks to ensure availability.

#### Monitoring Profile

Enable Prometheus and Grafana for metrics collection and visualization:

```bash
docker-compose --profile monitoring up -d
```

Access Grafana at http://localhost:3000 (default credentials: admin/admin) to view scraping metrics and system performance.

### Development Setup

For development work, use the development override configuration:

```bash
docker-compose -f docker-compose.yml -f docker-compose.dev.yml up
```

The development setup provides the following additional features:

- Source code hot reloading through volume mounts
- Debug logging enabled by default
- Mailhog for testing email notifications
- Adminer for database management
- Interactive terminal access for debugging

## Common Operations

### Running Scraping Jobs

Execute scraping jobs within the container:

```bash
# Single configuration
docker-compose exec datascrapexter datascrapexter run configs/products.yaml

# Multiple configurations
docker-compose exec datascrapexter bash -c "
  datascrapexter run configs/listing.yaml && \
  datascrapexter run configs/details.yaml
"
```

### Using Scripts

Run the included automation scripts:

```bash
# Daily scraping
docker-compose exec datascrapexter /app/scripts/daily_scrape.sh

# Price analysis
docker-compose exec datascrapexter /app/scripts/analyze_price_changes.pl \
  --data-dir /app/outputs --days 7

# Combine data
docker-compose exec datascrapexter /app/scripts/combine_data.pl \
  --sources "/app/outputs/*.json" -o /app/data/combined.json
```

### Managing Data

Access and manage scraped data:

```bash
# Copy data from container
docker cp datascrapexter:/app/outputs/products.json ./local-data/

# Archive old data
docker-compose exec datascrapexter bash -c "
  tar -czf /app/data/archive-$(date +%Y%m%d).tar.gz /app/outputs/
"

# Clean up old files
docker-compose exec datascrapexter find /app/outputs -mtime +30 -delete
```

### Monitoring and Debugging

View logs and debug issues:

```bash
# Application logs
docker-compose logs -f datascrapexter

# Script logs
docker-compose exec datascrapexter tail -f /app/logs/daily_scrape.log

# Interactive shell
docker-compose exec datascrapexter bash

# Check health status
docker-compose exec datascrapexter datascrapexter health
```

## Production Deployment

For production deployments, consider the following recommendations:

### Security Considerations

1. Change default passwords in environment variables and configuration files
2. Use Docker secrets for sensitive information instead of environment variables
3. Implement network policies to restrict container communication
4. Regularly update base images and dependencies
5. Enable SELinux or AppArmor profiles for additional container isolation

### Performance Optimization

1. Adjust container resource limits based on workload:
   ```yaml
   services:
     datascrapexter:
       deploy:
         resources:
           limits:
             cpus: '2.0'
             memory: 4G
           reservations:
             cpus: '1.0'
             memory: 2G
   ```

2. Configure appropriate shared memory size for Chrome:
   ```yaml
   services:
     datascrapexter:
       shm_size: 2gb
   ```

3. Use volume mounts with specific drivers for better I/O performance
4. Enable BuildKit for faster image builds:
   ```bash
   DOCKER_BUILDKIT=1 docker-compose build
   ```

### Scaling Considerations

For high-volume scraping operations:

1. Deploy multiple DataScrapexter instances behind a load balancer
2. Use external Redis cluster for distributed job management
3. Implement horizontal scaling with Kubernetes or Docker Swarm
4. Configure appropriate database connection pooling
5. Use object storage (S3, MinIO) for large output files

### Backup and Recovery

Implement regular backup procedures:

```bash
# Backup script example
#!/bin/bash
BACKUP_DIR="/backups/datascrapexter"
DATE=$(date +%Y%m%d_%H%M%S)

# Backup configurations
docker-compose exec datascrapexter tar -czf - /app/configs > \
  "${BACKUP_DIR}/configs_${DATE}.tar.gz"

# Backup data
docker-compose exec datascrapexter tar -czf - /app/data > \
  "${BACKUP_DIR}/data_${DATE}.tar.gz"

# Backup database if using PostgreSQL
docker-compose exec postgres pg_dump -U datascrapexter datascrapexter | \
  gzip > "${BACKUP_DIR}/database_${DATE}.sql.gz"
```

## Troubleshooting

### Container Fails to Start

If the container fails to start, check the following:

1. Verify Docker daemon is running: `docker info`
2. Check for port conflicts: `docker-compose ps`
3. Review container logs: `docker-compose logs datascrapexter`
4. Ensure sufficient disk space: `df -h`
5. Verify file permissions on mounted volumes

### Chrome/Chromium Issues

For browser-related problems:

1. Increase shared memory: `--shm-size=2g`
2. Add Chrome flags for stability:
   ```yaml
   environment:
     - CHROMIUM_FLAGS="--no-sandbox --disable-gpu --disable-dev-shm-usage"
   ```
3. Check Chrome process limits: `ulimit -n`

### Permission Errors

If encountering permission errors with mounted volumes:

```bash
# Fix ownership
sudo chown -R 1000:1000 ./configs ./outputs ./logs ./data

# Or run container as root (not recommended for production)
docker-compose exec -u root datascrapexter chown -R datascrapexter:datascrapexter /app
```

### Memory Issues

For out-of-memory errors:

1. Monitor memory usage: `docker stats`
2. Increase container memory limits
3. Reduce concurrent scraping jobs
4. Enable swap accounting on the host

## Building Custom Images

To build a custom DataScrapexter image with additional dependencies:

```dockerfile
# Custom Dockerfile
FROM datascrapexter:latest

# Switch to root for installation
USER root

# Add custom dependencies
RUN apk add --no-cache \
    postgresql-client \
    mongodb-tools \
    aws-cli

# Add custom scripts
COPY custom-scripts/ /app/custom-scripts/
RUN chmod +x /app/custom-scripts/*.sh

# Switch back to non-root user
USER datascrapexter
```

Build and use the custom image:

```bash
docker build -f Dockerfile.custom -t datascrapexter:custom .
docker-compose up -d
```

## Support and Resources

For additional help with Docker setup:

1. Review Docker logs for detailed error messages
2. Check the project's GitHub issues for similar problems
3. Consult the Docker documentation for platform-specific issues
4. Join the DataScrapexter community forum for support

Remember to keep your Docker installation and DataScrapexter image updated for the latest features and security patches.
