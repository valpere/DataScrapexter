#### Documentation

1. **Code Comments**:
   - Document why, not what
   - Keep comments up-to-date
   - Use examples in doc comments

2. **README Files**:
   - Keep main README concise
   - Use separate docs for details
   - Include examples

#### Performance

1. **Profiling**:
   ```bash
   # CPU profiling
   go test -cpuprofile=cpu.prof -bench=.
   go tool pprof cpu.prof
   
   # Memory profiling
   go test -memprofile=mem.prof -bench=.
   go tool pprof mem.prof
   ```

2. **Benchmarks**:
   - Write benchmarks for critical paths
   - Track performance over time
   - Use `benchstat` for comparisons

### Security Considerations

#### Static Analysis

1. **Security Scanning**:
   ```bash
   # Run gosec
   make security
   
   # Check for secrets
   detect-secrets scan
   ```

2. **Dependency Scanning**:
   ```bash
   # Check for vulnerabilities
   go list -m all | nancy sleuth
   
   # Update dependencies
   go get -u ./...
   go mod tidy
   ```

#### Best Practices

1. **Never commit**:
   - API keys or tokens
   - Passwords or credentials
   - Private keys or certificates
   - Internal URLs or IPs

2. **Use environment variables**:
   - For configuration
   - For secrets
   - Document required vars

### Contributing Guidelines

#### Pull Request Process

1. **Before Creating PR**:
   - Run all tests
   - Update documentation
   - Add tests for new features
   - Ensure CI passes

2. **PR Description**:
   - Describe changes clearly
   - Link related issues
   - Include test results
   - Add screenshots if UI changes

3. **Code Review**:
   - Address all comments
   - Keep discussions professional
   - Update based on feedback

#### Commit Messages

Follow conventional commits:
```
feat(scraper): add retry logic for failed requests
fix(config): correct YAML parsing for nested fields
docs(readme): update installation instructions
test(engine): add benchmarks for concurrent scraping
refactor(output): simplify CSV generation logic
```

### Resources

#### Official Documentation

- [Go Documentation](https://golang.org/doc/)
- [EditorConfig](https://editorconfig.org/)
- [GolangCI-Lint](https://golangci-lint.run/)
- [Pre-commit](https://pre-commit.com/)

#### VSCode Extensions

- [Go Extension Guide](https://code.visualstudio.com/docs/languages/go)
- [Debugging Go](https://github.com/golang/vscode-go/wiki/debugging)
- [Go Tools](https://github.com/golang/vscode-go/wiki/tools)

#### Style Guides

- [Effective Go](https://golang.org/doc/effective_go)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md)

## Summary

These development tools and configurations ensure:

1. **Consistent code style** across all contributors
2. **Automated quality checks** before commits
3. **Efficient debugging** and testing workflows
4. **Security best practices** enforcement
5. **Smooth collaboration** through standardized tooling

By following these guidelines and using the provided configurations, developers can focus on writing quality code while the tooling handles formatting, linting, and other routine tasks automatically.

### Advanced Configuration

#### Custom Linting Rules

Create project-specific linting rules:

```yaml
# .golangci-custom.yml
linters-settings:
  custom:
    datascrapexter-rules:
      - pattern: 'panic\('
        message: "Use errors instead of panic"
      - pattern: 'fmt\.Print'
        message: "Use structured logging"
```

#### Performance Profiling

1. **Enable profiling in code**:
   ```go
   import _ "net/http/pprof"
   go func() {
       log.Println(http.ListenAndServe("localhost:6060", nil))
   }()
   ```

2. **Analyze performance**:
   ```bash
   # CPU profile
   go tool pprof http://localhost:6060/debug/pprof/profile
   
   # Memory profile
   go tool pprof http://localhost:6060/debug/pprof/heap
   
   # Goroutine profile
   go tool pprof http://localhost:6060/debug/pprof/goroutine
   ```

#### Custom Git Hooks

Beyond pre-commit, add custom hooks:

```bash
# .git/hooks/commit-msg
#!/bin/bash
# Verify commit message format

commit_regex='^(feat|fix|docs|style|refactor|test|chore)(\(.+\))?: .{1,50}'
if ! grep -qE "$commit_regex" "$1"; then
    echo "Invalid commit message format!"
    echo "Format: type(scope): description"
    exit 1
fi
```

### Integration with CI/CD

#### GitHub Actions Integration

```yaml
# .github/workflows/ci.yml
name: CI
on: [push, pull_request]
jobs:
  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      - name: golangci-lint
        uses: golangci/golangci-lint-action@v3
        with:
          args: --config=.golangci.yml
```

#### Docker Development

```bash
# Use development container
docker-compose -f docker-compose.yml -f docker-compose.dev.yml up

# Run tests in container
docker-compose exec datascrapexter go test ./...

# Debug in container
docker-compose exec datascrapexter dlv debug ./cmd/datascrapexter
```

### Productivity Tips

#### VSCode Snippets

Create custom snippets for common patterns:

```json
// .vscode/datascrapexter.code-snippets
{
  "Error Handling": {
    "prefix": "dserr",
    "body": [
      "if err != nil {",
      "\treturn fmt.Errorf(\"${1:context}: %w\", err)",
      "}"
    ]
  },
  "Logger": {
    "prefix": "dslog",
    "body": [
      "log.WithFields(log.Fields{",
      "\t\"${1:key}\": ${2:value},",
      "}).${3|Info,Debug,Error,Warn|}(\"${4:message}\")"
    ]
  }
}
```

#### Keyboard Shortcuts

Customize shortcuts for common tasks:

```json
// keybindings.json
[
  {
    "key": "ctrl+shift+t",
    "command": "workbench.action.tasks.runTask",
    "args": "Run Tests"
  },
  {
    "key": "ctrl+shift+l",
    "command": "workbench.action.tasks.runTask",
    "args": "Run Linter"
  }
]
```

### Monitoring Code Quality

#### Metrics Dashboard

Track project health:

```bash
# Generate metrics
go test -coverprofile=coverage.out ./...
golangci-lint run --out-format json > lint-report.json
gocyclo -avg . > complexity.txt

# View in dashboard
go tool cover -html=coverage.out -o coverage.html
```

#### Code Review Checklist

- [ ] Tests added/updated for changes
- [ ] Documentation updated
- [ ] No linting errors
- [ ] Coverage maintained/improved
- [ ] Performance impact considered
- [ ] Security implications reviewed
- [ ] Breaking changes documented

### Team Collaboration

#### Shared Configuration

1. **Sync settings**:
   ```bash
   # Export settings
   cp .vscode/settings.json .vscode/settings.shared.json
   
   # Team members import
   cp .vscode/settings.shared.json .vscode/settings.json
   ```

2. **Document conventions**:
   - Create `DEVELOPMENT.md` for team practices
   - Maintain `ARCHITECTURE.md` for design decisions
   - Use `CHANGELOG.md` for version history

#### Code Documentation

1. **Package documentation**:
   ```go
   // Package scraper provides web scraping functionality with
   // anti-detection measures and configurable extraction rules.
   //
   // Basic usage:
   //
   //     engine := scraper.NewEngine(config)
   //     results, err := engine.Scrape(ctx, url)
   package scraper
   ```

2. **Function documentation**:
   ```go
   // Scrape extracts data from the given URL according to the
   // configured extraction rules. It returns structured data
   // or an error if the operation fails.
   //
   // The context can be used to cancel long-running operations:
   //
   //     ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
   //     defer cancel()
   //     results, err := engine.Scrape(ctx, url)
   func (e *Engine) Scrape(ctx context.Context, url string) (*Results, error) {
   ```

### Maintenance Tasks

#### Regular Updates

```bash
# Update Go modules
go get -u ./...
go mod tidy

# Update tools
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Update pre-commit hooks
pre-commit autoupdate
```

#### Clean Up

```bash
# Remove unused dependencies
go mod tidy

# Clean build artifacts
make clean

# Remove old branches
git branch --merged | grep -v "\*\|main\|develop" | xargs -n 1 git branch -d
```

### Conclusion

This comprehensive development environment configuration provides:

1. **Consistency**: Uniform code style and quality across the team
2. **Automation**: Reduced manual work through tools and scripts
3. **Quality**: Early detection of issues and bugs
4. **Productivity**: Optimized workflows and shortcuts
5. **Collaboration**: Shared standards and practices

The configuration is designed to scale with the project and team, supporting both individual productivity and team collaboration. Regular maintenance and updates ensure the development environment remains current with best practices and tool improvements.

For questions or improvements to these configurations, please open an issue or submit a pull request to the DataScrapexter repository.#### Documentation

1. **Code Comments**:
   - Document why, not what
   - Keep comments up-to-date
   - Use examples in doc comments

2. **README Files**:
   - Keep main README concise
   - Use separate docs for details
   - Include examples#### Using VSCode Debugger

1. **Set Breakpoints**: Click in the gutter next to line numbers
2. **Start Debugging**: Press F5 or use Run menu
3. **Debug Console**: View output and execute commands
4. **Variables**: Inspect values in the sidebar

#### Debug Configurations

- **CLI Debug**: Debug DataScrapexter commands
- **Server Debug**: Debug API server
- **Test Debug**: Debug specific tests
- **Remote Debug**: Attach to running process

### Testing

#### Running Tests

```bash
# All tests
make test

# With coverage
make test-coverage

# Integration tests
make test-integration

# Specific package
go test -v ./internal/scraper

# Specific test
go test -v -run TestEngineScrape ./internal/scraper
```

#### Writing Tests

1. **Test Files**: Name with `_test.go` suffix
2. **Test Functions**: Start with `Test`
3. **Table-Driven Tests**: Use for multiple cases
4. **Mocks**: Use interfaces for dependencies

### Continuous Integration

#### Pre-commit Checks

All commits trigger:
- Code formatting
- Linting
- Basic tests
- Security scans

#### Pre-push Checks

Before pushing:
- Full test suite
- Coverage requirements (>70%)
- Integration tests

### Troubleshooting

#### Common Issues

1. **Linter Errors**:
   ```bash
   # Run specific linter
   golangci-lint run --no-config --disable-all --enable=errcheck
   
   # Fix automatically
   golangci-lint run --fix
   ```

2. **Import Issues**:
   ```bash
   # Update imports
   goimports -w -local github.com/valpere/DataScrapexter .
   
   # Tidy modules
   go mod tidy
   ```

3. **Pre-commit Failures**:
   ```bash
   # Skip hooks temporarily
   git commit --no-verify
   
   # Run specific hook
   pre-commit run check-yaml --all-files
   ```

#### VSCode Issues

1. **Go Tools Missing**:
   - Open Command Palette: `Ctrl+Shift+P`
   - Run: "Go: Install/Update Tools"

2. **Debugging Not Working**:
   - Ensure `dlv` is installed
   - Check launch.json configuration
   - Verify binary paths

### Best Practices

#### Code Organization

1. **Package Structure**:
   - Keep packages focused and cohesive
   - Use internal/ for private packages
   - Separate concerns clearly

2. **Dependencies**:
   - Minimize external dependencies
   - Use go mod for version management
   - Document why each dependency is needed

3. **Testing**:
   - Write tests alongside code
   - Aim for >80% coverage
   - Include integration tests

#### Documentation

1. **Code Comments**:
   - Document why, not what
   - Keep comments up-to-date
   - Use examples inPre-commit hooks will automatically run these checks.

#### VSCode Shortcuts

- **Build**: `Ctrl+Shift+B` (Cmd+Shift+B on Mac)
- **Run Tests**: `Ctrl+Shift+T`
- **Debug**: `F5`
- **Format Document**: `Shift+Alt+F` (Shift+Option+F on Mac)

### Code Style Guidelines

#### Go Code

1. **Formatting**:
   - Use `gofmt` and `goimports`
   - Group imports: stdlib, external, internal
   - Maximum line length: 120 characters

2. **Naming**:
   - Exported names start with capital letter
   - Use camelCase for variables and functions
   - Acronyms should be all caps (URL, API, ID)

3. **Comments**:
   - Start with the name being declared
   - Use complete sentences
   - Document all exported types and functions

4. **Error Handling**:
   ```go
   if err != nil {
       return fmt.Errorf("failed to process: %w", err)
   }
   ```

#### YAML Configuration

1. **Indentation**: 2 spaces
2. **Quote Style**: Double quotes for strings with special characters
3. **Comments**: Use `#` with a space after
4. **Lists**: Use `-` with proper indentation

#### Shell Scripts

1. **Shebang**: `#!/bin/bash` or `#!/usr/bin/env bash`
2. **Set Options**: `set -euo pipefail`
3. **Variables**: Use `${VAR}` syntax
4. **Functions**: Descriptive names with comments

### Debugging

#### Using VSCode Debugger

1. **Set Breakpoints**: Click in the gutter next to line numbers
2. **Start Debugging**: Press F5 or use Run menu
3. **Debug Console**: View output an### 6. `.pre-commit-config.yaml`

Pre-commit hooks for automated quality checks:

- **File checks**: Large files, merge conflicts, private keys
- **Go checks**: Formatting, imports, linting, tests
- **Security**: Secret detection, GitLeaks
- **Documentation**: Markdown linting
- **Scripts**: Shell and Perl syntax checking
- **Custom checks**: Config validation, test coverage

### 7. Additional Configuration Files

#### `.dockerignore`
Optimize Docker builds by excluding unnecessary files:
- Development dependencies
- Test files and fixtures
- Documentation
- Git history
- Local environment files

#### `go.mod` and `go.sum`
Go module configuration:
- Module path: `github.com/valpere/DataScrapexter`
- Go version requirement: 1.21
- Direct and indirect dependencies
- Cryptographic checksums for reproducible builds

#### `.env.example`
Template for environment variables:
```bash
# Application settings
LOG_LEVEL=info
DEBUG=false

# Scraper configuration
USER_AGENT="DataScrapexter/1.0"
TIMEOUT=30
RETRY_ATTEMPTS=3

# Database (optional)
DATABASE_URL=postgresql://user:pass@localhost/datascrapexter

# API keys (optional)
API_KEY=your-api-key-here
```

### 8. Makefile Configuration

The Makefile provides standardized commands for development tasks:

#### Build Targets:
- `make build`: Build for current platform
- `make build-all`: Build for all supported platforms
- `make install`: Install binary to $GOPATH/bin
- `make run`: Build and run the application
- `make run-example`: Run with example configuration

#### Testing Targets:
- `make test`: Run unit tests
- `make test-coverage`: Generate coverage report
- `make test-integration`: Run integration tests
- `make benchmark`: Run performance benchmarks

#### Quality Targets:
- `make fmt`: Format code with gofmt
- `make vet`: Run go vet static analysis
- `make lint`: Run golangci-lint
- `make security`: Run gosec security scan

#### Development Targets:
- `make deps`: Download dependencies
- `make deps-update`: Update all dependencies
- `make generate`: Run code generation
- `make clean`: Remove build artifacts

#### Docker Targets:
- `make docker-build`: Build Docker image
- `make docker-run`: Run in container
- `make docker-push`: Push to registry

#### Release Targets:
- `make release`: Create release artifacts
- `make release-notes`: Generate changelog

### 9. Environment Configuration

#### Development Environment Setup:

**Required Tools**:
- Go 1.21+
- Git
- Make
- Docker (optional)
- golangci-lint
- pre-commit

**Optional Tools**:
- air (hot reloading)
- dlv (debugger)
- godoc (documentation)
- benchstat (benchmark comparison)

**Environment Variables**:
```bash
# Development
export GO111MODULE=on
export GOPROXY=https://proxy.golang.org,direct
export GOSUMDB=sum.golang.org
export CGO_ENABLED=0

# DataScrapexter specific
export DATASCRAPEXTER_HOME=$HOME/datascrapexter
export DATASCRAPEXTER_CONFIG_PATH=$DATASCRAPEXTER_HOME/configs
export DATASCRAPEXTER_OUTPUT_PATH=$DATASCRAPEXTER_HOME/outputs
export DATASCRAPEXTER_LOG_LEVEL=debug
```

### 10. CI/CD Integration

#### GitHub Actions Workflow:

```yaml
name: CI
on:
  push:
    branches: [main, develop]
  pull_request:
    branches: [main]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      - run: make deps
      - run: make lint
      - run: make test-coverage
      - uses: codecov/codecov-action@v3

  build:
    runs-on: ubuntu-latest
    needs: test
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
      - run: make build-all
      - uses: actions/upload-artifact@v3
        with:
          name: binaries
          path: bin/
```

#### Pre-commit Integration:

```bash
# Install pre-commit
pip install pre-commit

# Install hooks
pre-commit install
pre-commit install --hook-type commit-msg

# Run manually
pre-commit run --all-files

# Update hooks
pre-commit autoupdate
```

## Setup Instructions

### Initial Setup

1. **Install Development Tools**:
   ```bash
   # Install Go tools
   make install-tools
   
   # Install pre-commit
   pip install pre-commit
   pre-commit install
   
   # Install golangci-lint
   curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin
   ```

2. **Configure Git**:
   ```bash
   # Set up Git attributes
   git config core.attributesfile .gitattributes
   
   # Enable Git hooks
   cp scripts/pre-commit .git/hooks/pre-commit
   chmod +x .git/hooks/pre-commit
   ```

3. **VSCode Setup**:
   - Open the project in VSCode
   - Install recommended extensions when prompted
   - Reload window to activate settings

### Daily Workflow

#### Before Committing

1. **Format Code**:
   ```bash
   make fmt
   ```

2. **Run Linters**:
   ```bash
   make lint
   ```

3. **Run Tests**:
   ```bash
   make test
   ```

4. **Check Coverage**:
   ```bash
   make test-coverage
   ```

Pre-commit hooks will automatically run these checks.

#### VSCode Shortcuts

- **Build**: `Ctrl+Shift+B`### 6. `.pre-commit-config.yaml`

Pre-commit hooks for automated quality checks:

- **File checks**: Large files, merge conflicts, private keys
- **Go checks**: Formatting, imports, linting, tests
- **Security**: Secret detection, GitLeaks
- **Documentation**: Markdown linting
- **Scripts**: Shell and Perl syntax checking
- **Custom checks**: Config validation, test#### Enabled Linters:
- Standard: `errcheck`, `gosimple`, `govet`, `ineffassign`, `staticcheck`
- Security: `gosec` with comprehensive security checks
- Style: `gofmt`, `goimports`, `gofumpt`
- Complexity: `gocyclo`, `gocognit`, `funlen`
- Best practices: `gocritic`, `prealloc`, `exhaustive`

### 4. VSCode Configuration

#### `.vscode/settings.json`
Workspace-specific settings for optimal Go development:

- **Go tools**: Configured to use golangci-lint and goimports
- **Format on save**: Enabled with automatic import organization
- **Test coverage**: Visual indicators in the editor gutter
- **File associations**: Proper syntax highlighting for all file types

#### `.vscode/launch.json`
Debug configurations for various scenarios:

- Debug CLI commands
- Debug server mode
- Debug unit and integration tests
- Debug shell and Perl scripts
- Remote debugging support

#### `.vscode/tasks.json`
Automated tasks for common operations:

- Build tasks (single platform or all)
- Test execution with coverage
- Linting and formatting
- Docker operations
- Server startup

#### `.vscode/extensions.json`
Recommended extensions for the project:

- **Go development**: Official Go extension, test explorer
- **Code quality**: EditorConfig, linters, spell checker
- **Version control**: GitLens, Git Graph
- **Containers**: Docker, Remote Containers
- **Productivity**: Better Comments, Todo Tree

### 5. `.gitattributes`

Git attributes for proper file handling:

- **Line endings**: Enforce LF for all text files
- **Binary files**: Properly marked to prevent corruption
- **Diff settings**: Language-specific diff drivers
- **Export ignore**: Exclude development files from archives
- **Linguist overrides**: Accurate language statistics

### 6. `.pre-commit-config.yaml`

Pre-commit hooks for automated quality# Development Tools Configuration Guide

## Overview

This guide explains the development tools configuration for the DataScrapexter project. These configurations ensure code quality, consistency, and efficient development workflows across all contributors.

## Configuration Files

### 1. `.gitignore`

The `.gitignore` file prevents unnecessary files from being tracked by Git:

- **Build artifacts**: `bin/`, `dist/`, `build/`
- **IDE files**: `.idea/`, `.vscode/*` (except specific configs)
- **OS files**: `.DS_Store`, `Thumbs.db`
- **DataScrapexter outputs**: `outputs/`, `logs/`, `cache/`
- **Sensitive files**: `.env`, `*.pem`, `credentials/`
- **Test data**: `test-data/`, `*.test.yaml`

Key patterns:
```gitignore
# Keep example configs while ignoring other JSON/YAML
*.json
*.yaml
!configs/*.yaml
!examples/*.yaml
```

### 2. `.editorconfig`

EditorConfig ensures consistent coding styles across different editors:

- **Go files**: Use tabs with size 4
- **YAML files**: Use 2 spaces
- **Scripts**: Use 4 spaces
- **Line endings**: LF for all files
- **Final newline**: Required for all files

### 3. `.golangci.yml`

Comprehensive linter configuration for Go code quality:

#### Key Settings:
- **Timeout**: 5 minutes for analysis
- **Go version**: 1.21
- **Line length**: Maximum 120 characters
- **Cyclomatic complexity**: Maximum 15
- **Function length**: Maximum 100 lines

#### Enabled Linters:
- Standard: `errcheck`, `gosimple`, `govet`, `ineffassign`, `staticcheck`
- Security: `gosec` with comprehensive security checks
- Style: `gofmt`, `goimports`, `gofumpt`
- Complexity: `gocyclo`, `gocognit`,
