# DataScrapexter Makefile
# High-performance universal web scraper built with Go

# Variables
BINARY_NAME = datascrapexter
GO_CMD = go
GO_BUILD = $(GO_CMD) build
GO_TEST = $(GO_CMD) test
GO_CLEAN = $(GO_CMD) clean
GO_GET = $(GO_CMD) get
GO_MOD = $(GO_CMD) mod
GO_FMT = $(GO_CMD) fmt
GO_VET = $(GO_CMD) vet
GO_LINT = golangci-lint

# Build variables
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "0.1.0-dev")
BUILD_TIME = $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT = $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS = -ldflags "-w -s -X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME) -X main.gitCommit=$(GIT_COMMIT)"

# Directories
CMD_DIR = ./cmd/datascrapexter
BUILD_DIR = ./bin
DIST_DIR = ./dist
COVERAGE_DIR = ./coverage

# Platform-specific variables
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)

# Docker variables
DOCKER_IMAGE = datascrapexter
DOCKER_TAG ?= latest
DOCKER_REGISTRY ?= ghcr.io/valpere

# Default target
.DEFAULT_GOAL := help

# Help target
.PHONY: help
help: ## Display this help message
	@echo "DataScrapexter - Universal Web Scraper"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@awk 'BEGIN {FS = ":.*##"; printf "\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  %-20s %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

# Development targets
.PHONY: all
all: clean fmt vet lint test build ## Run all build steps

.PHONY: build
build: ## Build the binary for current platform
	@echo "Building $(BINARY_NAME) $(VERSION) for $(GOOS)/$(GOARCH)..."
	@mkdir -p $(BUILD_DIR)
	$(GO_BUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

.PHONY: build-all
build-all: ## Build binaries for all supported platforms
	@echo "Building binaries for all platforms..."
	@mkdir -p $(BUILD_DIR)
	@$(MAKE) build-linux-amd64
	@$(MAKE) build-linux-arm64
	@$(MAKE) build-darwin-amd64
	@$(MAKE) build-darwin-arm64
	@$(MAKE) build-windows-amd64
	@echo "All builds complete"

.PHONY: build-linux-amd64
build-linux-amd64: ## Build for Linux AMD64
	@echo "Building for linux/amd64..."
	@mkdir -p $(BUILD_DIR)/linux-amd64
	GOOS=linux GOARCH=amd64 $(GO_BUILD) $(LDFLAGS) -o $(BUILD_DIR)/linux-amd64/$(BINARY_NAME) $(CMD_DIR)

.PHONY: build-linux-arm64
build-linux-arm64: ## Build for Linux ARM64
	@echo "Building for linux/arm64..."
	@mkdir -p $(BUILD_DIR)/linux-arm64
	GOOS=linux GOARCH=arm64 $(GO_BUILD) $(LDFLAGS) -o $(BUILD_DIR)/linux-arm64/$(BINARY_NAME) $(CMD_DIR)

.PHONY: build-darwin-amd64
build-darwin-amd64: ## Build for macOS AMD64
	@echo "Building for darwin/amd64..."
	@mkdir -p $(BUILD_DIR)/darwin-amd64
	GOOS=darwin GOARCH=amd64 $(GO_BUILD) $(LDFLAGS) -o $(BUILD_DIR)/darwin-amd64/$(BINARY_NAME) $(CMD_DIR)

.PHONY: build-darwin-arm64
build-darwin-arm64: ## Build for macOS ARM64 (M1/M2)
	@echo "Building for darwin/arm64..."
	@mkdir -p $(BUILD_DIR)/darwin-arm64
	GOOS=darwin GOARCH=arm64 $(GO_BUILD) $(LDFLAGS) -o $(BUILD_DIR)/darwin-arm64/$(BINARY_NAME) $(CMD_DIR)

.PHONY: build-windows-amd64
build-windows-amd64: ## Build for Windows AMD64
	@echo "Building for windows/amd64..."
	@mkdir -p $(BUILD_DIR)/windows-amd64
	GOOS=windows GOARCH=amd64 $(GO_BUILD) $(LDFLAGS) -o $(BUILD_DIR)/windows-amd64/$(BINARY_NAME).exe $(CMD_DIR)

.PHONY: install
install: build ## Install the binary to $GOPATH/bin
	@echo "Installing $(BINARY_NAME)..."
	@cp $(BUILD_DIR)/$(BINARY_NAME) $(GOPATH)/bin/
	@echo "Installation complete"

.PHONY: run
run: build ## Build and run the application
	@echo "Running $(BINARY_NAME)..."
	@$(BUILD_DIR)/$(BINARY_NAME)

.PHONY: run-example
run-example: build ## Run with example configuration
	@echo "Running example scraper..."
	@$(BUILD_DIR)/$(BINARY_NAME) run examples/basic.yaml

# Testing targets
.PHONY: test
test: ## Run unit tests
	@echo "Running tests..."
	$(GO_TEST) -v -race -timeout 30s ./...

.PHONY: test-coverage
test-coverage: ## Run tests with coverage report
	@echo "Running tests with coverage..."
	@mkdir -p $(COVERAGE_DIR)
	$(GO_TEST) -v -race -coverprofile=$(COVERAGE_DIR)/coverage.out -covermode=atomic ./...
	$(GO_CMD) tool cover -html=$(COVERAGE_DIR)/coverage.out -o $(COVERAGE_DIR)/coverage.html
	@echo "Coverage report generated: $(COVERAGE_DIR)/coverage.html"

.PHONY: test-integration
test-integration: ## Run integration tests
	@echo "Running integration tests..."
	$(GO_TEST) -v -tags=integration -timeout 5m ./test/...

.PHONY: benchmark
benchmark: ## Run benchmarks
	@echo "Running benchmarks..."
	$(GO_TEST) -bench=. -benchmem -run=^$ ./...

# Code quality targets
.PHONY: fmt
fmt: ## Format code using gofmt
	@echo "Formatting code..."
	$(GO_FMT) ./...

.PHONY: vet
vet: ## Run go vet
	@echo "Running go vet..."
	$(GO_VET) ./...

.PHONY: lint
lint: ## Run golangci-lint
	@echo "Running linter..."
	@if command -v golangci-lint > /dev/null; then \
		$(GO_LINT) run ./...; \
	else \
		echo "golangci-lint not installed. Install with: make install-tools"; \
	fi

.PHONY: security
security: ## Run security checks with gosec
	@echo "Running security scan..."
	@if command -v gosec > /dev/null; then \
		gosec -quiet ./...; \
	else \
		echo "gosec not installed. Install with: make install-tools"; \
	fi

# Dependency management
.PHONY: deps
deps: ## Download dependencies
	@echo "Downloading dependencies..."
	$(GO_MOD) download

.PHONY: deps-update
deps-update: ## Update dependencies
	@echo "Updating dependencies..."
	$(GO_MOD) tidy
	$(GO_GET) -u ./...
	$(GO_MOD) tidy

.PHONY: deps-verify
deps-verify: ## Verify dependencies
	@echo "Verifying dependencies..."
	$(GO_MOD) verify

# Docker targets
.PHONY: docker-build
docker-build: ## Build Docker image
	@echo "Building Docker image..."
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) \
		--build-arg VERSION=$(VERSION) \
		--build-arg BUILD_TIME=$(BUILD_TIME) \
		--build-arg GIT_COMMIT=$(GIT_COMMIT) \
		.

.PHONY: docker-push
docker-push: ## Push Docker image to registry
	@echo "Pushing Docker image..."
	docker tag $(DOCKER_IMAGE):$(DOCKER_TAG) $(DOCKER_REGISTRY)/$(DOCKER_IMAGE):$(DOCKER_TAG)
	docker push $(DOCKER_REGISTRY)/$(DOCKER_IMAGE):$(DOCKER_TAG)

.PHONY: docker-run
docker-run: ## Run Docker container
	@echo "Running Docker container..."
	docker run --rm -it \
		-v $(PWD)/configs:/app/configs \
		-v $(PWD)/output:/app/output \
		$(DOCKER_IMAGE):$(DOCKER_TAG)

# Release targets
.PHONY: release
release: clean test build-all ## Create release artifacts
	@echo "Creating release $(VERSION)..."
	@mkdir -p $(DIST_DIR)
	@for platform in linux-amd64 linux-arm64 darwin-amd64 darwin-arm64 windows-amd64; do \
		echo "Packaging $$platform..."; \
		if [ "$$platform" = "windows-amd64" ]; then \
			cd $(BUILD_DIR)/$$platform && zip -q ../../$(DIST_DIR)/$(BINARY_NAME)-$(VERSION)-$$platform.zip $(BINARY_NAME).exe && cd ../..; \
		else \
			cd $(BUILD_DIR)/$$platform && tar -czf ../../$(DIST_DIR)/$(BINARY_NAME)-$(VERSION)-$$platform.tar.gz $(BINARY_NAME) && cd ../..; \
		fi; \
	done
	@echo "Release artifacts created in $(DIST_DIR)"

.PHONY: release-notes
release-notes: ## Generate release notes
	@echo "Generating release notes for $(VERSION)..."
	@echo "# Release Notes - $(VERSION)" > RELEASE_NOTES.md
	@echo "" >> RELEASE_NOTES.md
	@echo "## Changes" >> RELEASE_NOTES.md
	@git log --pretty=format:"- %s" $(shell git describe --tags --abbrev=0 2>/dev/null || echo "")..HEAD >> RELEASE_NOTES.md
	@echo "" >> RELEASE_NOTES.md
	@echo "Release notes generated: RELEASE_NOTES.md"

# Development tools
.PHONY: install-tools
install-tools: ## Install development tools
	@echo "Installing development tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/securego/gosec/v2/cmd/gosec@latest
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/google/go-licenses@latest
	@echo "Tools installation complete"

.PHONY: generate
generate: ## Run go generate
	@echo "Running code generation..."
	$(GO_CMD) generate ./...

.PHONY: proto
proto: ## Compile protocol buffers (if any)
	@echo "Compiling protocol buffers..."
	@if [ -d "proto" ]; then \
		protoc --go_out=. --go-grpc_out=. proto/*.proto; \
	else \
		echo "No proto files found"; \
	fi

# Documentation
.PHONY: docs
docs: ## Generate documentation
	@echo "Generating documentation..."
	@if command -v godoc > /dev/null; then \
		echo "Starting godoc server on http://localhost:6060"; \
		godoc -http=:6060; \
	else \
		echo "godoc not installed. Install with: go install golang.org/x/tools/cmd/godoc@latest"; \
	fi

.PHONY: docs-check
docs-check: ## Check documentation coverage
	@echo "Checking documentation coverage..."
	@go doc -all ./... | grep -E "^(func|type|const|var)" | wc -l

# Utility targets
.PHONY: clean
clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	$(GO_CLEAN)
	@rm -rf $(BUILD_DIR) $(DIST_DIR) $(COVERAGE_DIR)
	@echo "Clean complete"

.PHONY: info
info: ## Display build information
	@echo "DataScrapexter Build Information"
	@echo "================================"
	@echo "Version:     $(VERSION)"
	@echo "Build Time:  $(BUILD_TIME)"
	@echo "Git Commit:  $(GIT_COMMIT)"
	@echo "Go Version:  $(shell go version)"
	@echo "Platform:    $(GOOS)/$(GOARCH)"

.PHONY: check-mod
check-mod: ## Check for outdated modules
	@echo "Checking for outdated modules..."
	@go list -u -m all

.PHONY: license-check
license-check: ## Check licenses of dependencies
	@echo "Checking licenses..."
	@if command -v go-licenses > /dev/null; then \
		go-licenses check ./cmd/datascrapexter; \
	else \
		echo "go-licenses not installed. Install with: make install-tools"; \
	fi

# Git hooks
.PHONY: install-hooks
install-hooks: ## Install git hooks
	@echo "Installing git hooks..."
	@cp scripts/pre-commit .git/hooks/pre-commit
	@chmod +x .git/hooks/pre-commit
	@echo "Git hooks installed"

# Development shortcuts
.PHONY: dev
dev: fmt vet lint test ## Run all development checks

.PHONY: quick
quick: fmt build ## Quick build without tests

.PHONY: watch
watch: ## Watch for changes and rebuild
	@echo "Watching for changes..."
	@if command -v air > /dev/null; then \
		air; \
	else \
		echo "air not installed. Install with: go install github.com/cosmtrek/air@latest"; \
	fi
