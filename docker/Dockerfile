# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make gcc musl-dev

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN make build

# Runtime stage
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    chromium \
    chromium-chromedriver \
    bash \
    perl \
    perl-json \
    perl-text-csv \
    perl-digest-md5 \
    perl-getopt-long \
    python3 \
    py3-pip \
    jq \
    curl \
    tzdata

# Install Python dependencies for YAML validation
RUN pip3 install --no-cache-dir pyyaml

# Create non-root user
RUN addgroup -g 1000 -S datascrapexter && \
    adduser -u 1000 -S datascrapexter -G datascrapexter

# Create necessary directories
RUN mkdir -p /app/configs /app/outputs /app/logs /app/scripts /app/data && \
    chown -R datascrapexter:datascrapexter /app

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/bin/datascrapexter /usr/local/bin/

# Copy scripts
COPY --chown=datascrapexter:datascrapexter scripts/ /app/scripts/

# Make scripts executable
RUN chmod +x /app/scripts/*.sh /app/scripts/*.pl

# Copy example configurations
COPY --chown=datascrapexter:datascrapexter configs/ /app/configs/
COPY --chown=datascrapexter:datascrapexter examples/ /app/examples/

# Set environment variables
ENV CHROME_BIN=/usr/bin/chromium-browser \
    CHROME_PATH=/usr/lib/chromium/ \
    CHROMIUM_FLAGS="--disable-software-rasterizer --disable-dev-shm-usage" \
    PATH="/app/scripts:${PATH}"

# Switch to non-root user
USER datascrapexter

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD datascrapexter version || exit 1

# Default command
CMD ["datascrapexter", "--help"]
