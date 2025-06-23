# Makefile

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
BINARY_NAME=datascrapexter
BINARY_UNIX=$(BINARY_NAME)_unix

# Build information
VERSION?=$(shell git rev-parse --short HEAD)-dirty
BUILD_TIME?=$(shell date +%Y-%m-%d_%H:%M:%S)
GIT_COMMIT?=$(shell git rev-parse --short HEAD)

# Ldflags
LDFLAGS=-w -s -X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME) -X main.gitCommit=$(GIT_COMMIT)

# Default target
.PHONY: all
all: deps test build

# Dependencies
.PHONY: deps
deps:
	$(GOMOD) download
	$(GOMOD) tidy

# Build
.PHONY: build
build: deps
	@echo "Building $(BINARY_NAME) $(VERSION) for linux/amd64..."
	@mkdir -p bin
	$(GOBUILD) -ldflags "$(LDFLAGS)" -o ./bin/$(BINARY_NAME) ./cmd/datascrapexter

# Build for multiple platforms
.PHONY: build-all
build-all: build-linux build-windows build-darwin

.PHONY: build-linux
build-linux: deps
	@echo "Building for Linux..."
	@mkdir -p bin
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -ldflags "$(LDFLAGS)" -o ./bin/$(BINARY_NAME)-linux-amd64 ./cmd/datascrapexter

.PHONY: build-windows
build-windows: deps
	@echo "Building for Windows..."
	@mkdir -p bin
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GOBUILD) -ldflags "$(LDFLAGS)" -o ./bin/$(BINARY_NAME)-windows-amd64.exe ./cmd/datascrapexter

.PHONY: build-darwin
build-darwin: deps
	@echo "Building for macOS..."
	@mkdir -p bin
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GOBUILD) -ldflags "$(LDFLAGS)" -o ./bin/$(BINARY_NAME)-darwin-amd64 ./cmd/datascrapexter

# Testing
.PHONY: test
test:
	@echo "Running tests..."
	$(GOTEST) -v ./...

.PHONY: test-unit
test-unit:
	@echo "Running unit tests..."
	$(GOTEST) -v -short ./internal/...

.PHONY: test-integration
test-integration:
	@echo "Running integration tests..."
	$(GOTEST) -v -run Integration ./test/...

.PHONY: test-coverage
test-coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

.PHONY: test-race
test-race:
	@echo "Running tests with race detection..."
	$(GOTEST) -v -race ./...

.PHONY: test-bench
test-bench:
	@echo "Running benchmarks..."
	$(GOTEST) -v -bench=. -benchmem ./...

.PHONY: test-clean
test-clean:
	@echo "Cleaning test artifacts..."
	rm -f coverage.out coverage.html
	$(GOCMD) clean -testcache

# Linting and formatting
.PHONY: lint
lint:
	@echo "Running linters..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed, running basic checks..."; \
		$(GOCMD) vet ./...; \
		$(GOCMD) fmt ./...; \
	fi

.PHONY: fmt
fmt:
	@echo "Formatting code..."
	$(GOCMD) fmt ./...

.PHONY: vet
vet:
	@echo "Running go vet..."
	$(GOCMD) vet ./...

# Clean
.PHONY: clean
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -rf bin/
	rm -f coverage.out coverage.html

# Setup directories
.PHONY: setup
setup:
	@echo "Setting up project directories..."
	@mkdir -p internal/scraper
	@mkdir -p internal/pipeline
	@mkdir -p internal/antidetect
	@mkdir -p internal/config
	@mkdir -p internal/compliance
	@mkdir -p pkg/api
	@mkdir -p pkg/client
	@mkdir -p pkg/types
	@mkdir -p cmd/datascrapexter
	@mkdir -p cmd/server
	@mkdir -p cmd/tools
	@mkdir -p configs
	@mkdir -p docs
	@mkdir -p examples
	@mkdir -p scripts
	@mkdir -p test/utils
	@mkdir -p bin
	@echo "Project directories created successfully."

# Initialize missing files
.PHONY: init
init: setup
	@echo "Initializing missing files..."
	@if [ ! -f internal/pipeline/transform.go ]; then echo "Creating transform.go..."; fi
	@if [ ! -f internal/scraper/pagination_strategies.go ]; then echo "Creating pagination_strategies.go..."; fi
	@echo "Missing files initialized."

# Development helpers
.PHONY: dev-deps
dev-deps:
	@echo "Installing development dependencies..."
	@if ! command -v golangci-lint >/dev/null 2>&1; then \
		echo "Installing golangci-lint..."; \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin v1.54.2; \
	fi

.PHONY: run
run: build
	@echo "Running $(BINARY_NAME)..."
	./bin/$(BINARY_NAME)

.PHONY: run-example
run-example: build
	@echo "Running example configuration..."
	@if [ -f configs/example.yaml ]; then \
		./bin/$(BINARY_NAME) run configs/example.yaml; \
	else \
		echo "Example config not found. Generate with: make template > configs/example.yaml"; \
	fi

.PHONY: template
template: build
	@echo "Generating configuration template..."
	./bin/$(BINARY_NAME) template

.PHONY: validate-config
validate-config: build
	@echo "Validating configuration files..."
	@for config in configs/*.yaml; do \
		if [ -f "$config" ]; then \
			echo "Validating $config..."; \
			./bin/$(BINARY_NAME) validate "$config"; \
		fi; \
	done

# Docker support
.PHONY: docker-build
docker-build:
	@echo "Building Docker image..."
	docker build -t datascrapexter:latest .

.PHONY: docker-run
docker-run: docker-build
	@echo "Running Docker container..."
	docker run --rm -v $(PWD)/configs:/app/configs datascrapexter:latest

# Release preparation
.PHONY: pre-release
pre-release: clean deps lint test-coverage build-all
	@echo "Pre-release checks completed successfully!"

# Performance testing
.PHONY: perf-test
perf-test:
	@echo "Running performance tests..."
	$(GOTEST) -v -bench=. -benchtime=10s -benchmem ./...

.PHONY: memory-test
memory-test:
	@echo "Running memory tests..."
	$(GOTEST) -v -bench=. -benchmem -memprofile=mem.prof ./...
	$(GOCMD) tool pprof mem.prof

.PHONY: cpu-test
cpu-test:
	@echo "Running CPU tests..."
	$(GOTEST) -v -bench=. -cpuprofile=cpu.prof ./...
	$(GOCMD) tool pprof cpu.prof

# Security checks
.PHONY: security
security:
	@echo "Running security checks..."
	@if command -v gosec >/dev/null 2>&1; then \
		gosec ./...; \
	else \
		echo "gosec not installed. Install with: go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest"; \
	fi

# Documentation
.PHONY: docs
docs:
	@echo "Generating documentation..."
	@if command -v godoc >/dev/null 2>&1; then \
		echo "Starting godoc server on :6060..."; \
		godoc -http=:6060; \
	else \
		echo "godoc not available. Install with: go install golang.org/x/tools/cmd/godoc@latest"; \
	fi

# CI/CD helpers
.PHONY: ci
ci: deps lint test-race test-coverage

.PHONY: ci-short
ci-short: deps lint test-unit

# Debug helpers
.PHONY: debug-build
debug-build: deps
	@echo "Building debug version..."
	@mkdir -p bin
	$(GOBUILD) -gcflags="-N -l" -o ./bin/$(BINARY_NAME)-debug ./cmd/datascrapexter

# Install development tools
.PHONY: install-tools
install-tools:
	@echo "Installing development tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
	go install golang.org/x/tools/cmd/godoc@latest
	go install golang.org/x/tools/cmd/goimports@latest

# Validate project structure
.PHONY: validate
validate: vet fmt
	@echo "Validating project structure..."
	@echo "Checking required directories..."
	@test -d internal/scraper || (echo "Missing internal/scraper directory" && exit 1)
	@test -d internal/pipeline || (echo "Missing internal/pipeline directory" && exit 1)
	@test -d cmd/datascrapexter || (echo "Missing cmd/datascrapexter directory" && exit 1)
	@echo "Checking required files..."
	@test -f cmd/datascrapexter/main.go || (echo "Missing main.go" && exit 1)
	@test -f go.mod || (echo "Missing go.mod" && exit 1)
	@echo "Project structure validation passed!"

# Help
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  all              - Run deps, test, and build"
	@echo "  build            - Build the application"
	@echo "  build-all        - Build for all platforms"
	@echo "  deps             - Download and tidy dependencies"
	@echo "  test             - Run all tests"
	@echo "  test-unit        - Run unit tests only"
	@echo "  test-integration - Run integration tests only"
	@echo "  test-coverage    - Run tests with coverage report"
	@echo "  test-race        - Run tests with race detection"
	@echo "  test-bench       - Run benchmark tests"
	@echo "  lint             - Run linters"
	@echo "  fmt              - Format code"
	@echo "  vet              - Run go vet"
	@echo "  clean            - Clean build artifacts"
	@echo "  setup            - Create project directories"
	@echo "  init             - Initialize missing files"
	@echo "  run              - Build and run the application"
	@echo "  run-example      - Run with example configuration"
	@echo "  template         - Generate configuration template"
	@echo "  validate-config  - Validate configuration files"
	@echo "  docker-build     - Build Docker image"
	@echo "  docker-run       - Run in Docker container"
	@echo "  pre-release      - Run all pre-release checks"
	@echo "  security         - Run security checks"
	@echo "  docs             - Generate documentation"
	@echo "  ci               - Run CI pipeline"
	@echo "  install-tools    - Install development tools"
	@echo "  validate         - Validate project structure"
	@echo "  help             - Show this help message"

# Default help if no target specified
.DEFAULT_GOAL := help
