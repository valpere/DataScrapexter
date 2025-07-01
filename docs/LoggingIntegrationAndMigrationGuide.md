# Logging Integration and Migration Guide

## Current Logging Issues Identified

The DataScrapexter codebase currently uses multiple logging approaches:

1. **`fmt.Printf/fmt.Println`** - Direct console output in main.go
2. **`log.Printf`** - Standard Go logging in config/watcher.go
3. **`printf`** - Shell scripts logging
4. **Unused logger** - Sophisticated logger in internal/utils/logger.go

## Unified Logging Solution

The enhanced `internal/utils/logger.go` provides:

- **Centralized logging** with consistent format
- **Component-based logging** for different modules
- **Level-based filtering** (DEBUG, INFO, WARN, ERROR)
- **Structured logging** with fields
- **Environment configuration** support
- **Standard log intercept** to catch existing log.Printf calls

## Migration Steps

### 1. Replace fmt.Printf with structured logging

**Before:**
```go
fmt.Printf("Running scraper with config: %s\n", configFile)
fmt.Printf("Error: Config file not found: %s\n", configFile)
```

**After:**
```go
logger.WithField("config", configFile).Info("Starting scraper")
logger.WithField("file", configFile).Error("Config file not found")
```

### 2. Replace log.Printf with utils logging

**Before (in internal/config/watcher.go):**
```go
log.Printf("Warning: failed to watch config directory: %v", err)
log.Printf("Config watcher error: %v", err)
```

**After:**
```go
var logger = utils.NewComponentLogger("config-watcher")

logger.WithField("error", err).Warn("Failed to watch config directory")
logger.WithField("error", err).Error("Config watcher error")
```

### 3. Component-specific loggers

Create dedicated loggers for each component:

```go
// In scraper package
var scraperLogger = utils.NewComponentLogger("scraper")

// In pipeline package  
var pipelineLogger = utils.NewComponentLogger("pipeline")

// In output package
var outputLogger = utils.NewComponentLogger("output")
```

### 4. Environment configuration

Set logging behavior via environment variables:

```bash
# Set log level
export LOG_LEVEL=debug

# Log to file
export LOG_FILE=/var/log/datascrapexter.log

# Or configure in code
utils.ConfigureFromEnv()
```

## Integration Example

### Updated main.go

The main.go has been updated to demonstrate proper logging integration:

- **Component logger**: `utils.NewComponentLogger("cli")`
- **Structured fields**: `.WithField("config", configFile)`
- **Level-based output**: Debug logs only in verbose mode
- **Error handling**: Consistent error logging with context

### Key Benefits

1. **Unified format**: All logs follow consistent timestamp and level format
2. **Searchable logs**: Structured fields enable easy searching and filtering
3. **Component tracing**: Easy to identify which component generated each log
4. **Environment control**: Log level and output configurable without code changes
5. **Standard log compatibility**: Existing log.Printf calls automatically work

## Usage Patterns

### Basic logging
```go
utils.Info("Application started")
utils.Errorf("Failed to connect: %v", err)
```

### Structured logging
```go
logger.WithFields(map[string]interface{}{
    "url":        "https://example.com",
    "status":     200,
    "duration":   "1.2s",
}).Info("Request completed")
```

### Component logging
```go
scraperLogger := utils.NewComponentLogger("scraper")
scraperLogger.WithField("url", url).Debug("Processing URL")
```

### Environment configuration
```go
// Configure once at startup
utils.ConfigureFromEnv()

// Optionally override in code
if verbose {
    utils.SetGlobalLogLevel(utils.DebugLevel)
}
```

## Migration Checklist

- [ ] Replace fmt.Printf with utils.Info/Error in main.go ✅
- [ ] Replace log.Printf with component loggers in internal/config/watcher.go
- [ ] Add component loggers to internal/scraper/*.go
- [ ] Add component loggers to internal/pipeline/*.go  
- [ ] Add component loggers to internal/output/*.go
- [ ] Update shell scripts to log to files for processing
- [ ] Add LOG_LEVEL=debug to development environment
- [ ] Add LOG_FILE configuration for production

## Testing Integration

### Development
```bash
# Run with debug logging
LOG_LEVEL=debug go run cmd/datascrapexter/main.go -v run config.yaml

# Log to file
LOG_FILE=debug.log go run cmd/datascrapexter/main.go run config.yaml
```

### Production
```bash
# Standard info logging
datascrapexter run config.yaml

# Error-only logging
LOG_LEVEL=error datascrapexter run config.yaml

# Log to system log
LOG_FILE=/var/log/datascrapexter.log datascrapexter run config.yaml
```

## Next Steps

1. **Apply to remaining components**: Update internal packages to use component loggers
2. **Add request tracing**: Include request IDs for distributed logging
3. **Metrics integration**: Connect logging with metrics collection
4. **Log rotation**: Configure log file rotation for production use
5. **Structured output**: Add JSON format option for log aggregation systems

This unified logging approach provides a solid foundation for debugging, monitoring, and operational visibility across the entire DataScrapexter application.
