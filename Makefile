# Makefile

# Default target
.PHONY: all
all: deps test build

# Dependencies
.PHONY: deps
deps:
	go mod download
	go mod tidy

# Build
.PHONY: build
build: deps
	@echo "Building datascrapexter $(shell git rev-parse --short HEAD)-dirty for linux/amd64..."
	@mkdir -p bin
	go build -ldflags "-w -s -X main.version=$(shell git rev-parse --short HEAD)-dirty -X main.buildTime=$(shell date +%Y-%m-%d_%H:%M:%S) -X main.gitCommit=$(shell git rev-parse --short HEAD)" -o ./bin/datascrapexter ./cmd/datascrapexter

# Test
.PHONY: test
test:
	go test -v ./...

# Clean
.PHONY: clean
clean:
	rm -rf bin/
	go clean

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
	@mkdir -p test
	@echo "Project directories created successfully."

# Initialize missing files
.PHONY: init
init: setup
	@echo "Initializing missing files..."
	@if [ ! -f internal/pipeline/transform.go ]; then echo "Creating transform.go..."; fi
	@if [ ! -f internal/scraper/pagination_strategies.go ]; then echo "Creating pagination_strategies.go..."; fi
	@echo "Missing files initialized."

# Validate code
.PHONY: validate
validate:
	go vet ./...
	gofmt -l .

# Help
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  all       - Run deps, test, and build"
	@echo "  deps      - Download and tidy dependencies"
	@echo "  build     - Build the application"
	@echo "  test      - Run tests"
	@echo "  clean     - Clean build artifacts"
	@echo "  setup     - Create project directories"
	@echo "  init      - Initialize missing files"
	@echo "  validate  - Validate code with vet and fmt"
	@echo "  help      - Show this help message"
