// cmd/datascrapexter/main.go
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/valpere/DataScrapexter/internal/config"
	"github.com/valpere/DataScrapexter/internal/output"
	"github.com/valpere/DataScrapexter/internal/pipeline"
	"github.com/valpere/DataScrapexter/internal/scraper"
)

// Build-time variables (set by ldflags)
var (
	version   = "dev"
	buildTime = "unknown"
	gitCommit = "unknown"
)

// Global flags
var (
	verbose   bool
	outputFile string
	dryRun    bool
)

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		printUsage()
		return
	}

	// Parse global flags
	args = parseGlobalFlags(args)

	command := args[0]
	commandArgs := args[1:]

	switch command {
	case "run":
		if len(commandArgs) < 1 {
			fmt.Println("Error: configuration file required")
			fmt.Println("Usage: datascrapexter run <config.yaml>")
			os.Exit(1)
		}
		runScraper(commandArgs[0])
	case "validate":
		if len(commandArgs) < 1 {
			fmt.Println("Error: configuration file required")
			fmt.Println("Usage: datascrapexter validate <config.yaml>")
			os.Exit(1)
		}
		validateConfig(commandArgs[0])
	case "template":
		generateTemplate()
	case "version":
		printVersion()
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Printf("Error: unknown command '%s'\n", command)
		printUsage()
		os.Exit(1)
	}
}

func parseGlobalFlags(args []string) []string {
	var remaining []string

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-v", "--verbose":
			verbose = true
		case "-o", "--output":
			if i+1 < len(args) {
				outputFile = args[i+1]
				i++ // Skip next argument
			}
		case "--dry-run":
			dryRun = true
		default:
			remaining = append(remaining, args[i])
		}
	}

	return remaining
}

func printUsage() {
	fmt.Printf("DataScrapexter %s - Universal web scraper\n\n", version)
	fmt.Println("Usage:")
	fmt.Println("  datascrapexter [global-options] <command> [arguments]")
	fmt.Println()
	fmt.Println("Global Options:")
	fmt.Println("  -v, --verbose     Enable verbose logging")
	fmt.Println("  -o, --output FILE Override output file")
	fmt.Println("  --dry-run         Validate configuration without scraping")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  run <config.yaml>      Run scraper with configuration")
	fmt.Println("  validate <config.yaml> Validate configuration file")
	fmt.Println("  template               Generate configuration template")
	fmt.Println("  version                Show version information")
	fmt.Println("  help                   Show this help message")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  datascrapexter run examples/basic.yaml")
	fmt.Println("  datascrapexter -v run config.yaml")
	fmt.Println("  datascrapexter --dry-run run config.yaml")
	fmt.Println("  datascrapexter template > myconfig.yaml")
}

func printVersion() {
	fmt.Printf("DataScrapexter %s\n", version)
	fmt.Printf("Build time: %s\n", buildTime)
	fmt.Printf("Git commit: %s\n", gitCommit)
}

func runScraper(configFile string) {
	if verbose {
		fmt.Printf("Starting scraper with config: %s\n", configFile)
	}

	// Load configuration
	cfg, err := config.LoadFromFile(configFile)
	if err != nil {
		fmt.Printf("Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	if verbose {
		fmt.Printf("Configuration loaded: %s\n", cfg.Name)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		fmt.Printf("Configuration validation failed: %v\n", err)
		os.Exit(1)
	}

	// Override output file if specified
	if outputFile != "" {
		cfg.Output.File = outputFile
	}

	// Dry run mode
	if dryRun {
		fmt.Println("Configuration is valid")
		return
	}

	// Convert config to engine config
	engineConfig := convertToEngineConfig(cfg)

	// Create scraping engine
	engine, err := scraper.NewScrapingEngine(engineConfig)
	if err != nil {
		fmt.Printf("Error creating scraping engine: %v\n", err)
		os.Exit(1)
	}
	defer engine.Close()

	// Create output manager
	outputManager, err := output.NewManager(&cfg.Output)
	if err != nil {
		fmt.Printf("Error creating output manager: %v\n", err)
		os.Exit(1)
	}

	// Perform scraping
	ctx := context.Background()
	
	// Determine target URL
	targetURL := cfg.BaseURL
	if len(cfg.URLs) > 0 {
		targetURL = cfg.URLs[0] // Use first URL for now
	}
	
	if verbose {
		fmt.Printf("Scraping URL: %s\n", targetURL)
	}
	
	result, err := engine.Scrape(ctx, targetURL)
	if err != nil {
		fmt.Printf("Scraping failed: %v\n", err)
		os.Exit(1)
	}

	// Check for scraping errors
	if !result.Success {
		fmt.Printf("Scraping completed with errors: %v\n", result.Errors)
	}

	// Convert single result to slice for output
	outputData := []map[string]interface{}{result.Data}
	
	// Write results
	err = outputManager.WriteResults(outputData)
	if err != nil {
		fmt.Printf("Error writing results: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Scraping completed successfully. Results written to %s\n", cfg.Output.File)
}

func validateConfig(configFile string) {
	if verbose {
		fmt.Printf("Validating configuration: %s\n", configFile)
	}

	cfg, err := config.LoadFromFile(configFile)
	if err != nil {
		fmt.Printf("Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	if err := cfg.Validate(); err != nil {
		fmt.Printf("Configuration validation failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Configuration is valid")
}

func generateTemplate() {
	template := `# DataScrapexter Configuration Template
name: "example_scraper"
base_url: "https://example.com"

# Optional: Multiple URLs to scrape
# urls:
#   - "https://example.com/page1"
#   - "https://example.com/page2"

# Request configuration
user_agents:
  - "DataScrapexter/1.0"
rate_limit: "2s"
timeout: "30s"
max_retries: 3

# Optional headers and cookies
# headers:
#   Accept: "text/html,application/xhtml+xml"
# cookies:
#   session: "abc123"

# Fields to extract
fields:
  - name: "title"
    selector: "h1"
    type: "text"
    required: true
    
  - name: "description"
    selector: ".description"
    type: "text"
    required: false
    transform:
      - type: "trim"
      - type: "normalize_spaces"
      
  - name: "price"
    selector: ".price"
    type: "text"
    required: false
    transform:
      - type: "regex"
        pattern: "\\$([0-9,]+\\.?[0-9]*)"
        replacement: "$1"
      - type: "parse_float"

  - name: "image_url"
    selector: "img.main-image"
    type: "attr"
    attribute: "src"
    required: false

  - name: "tags"
    selector: ".tag"
    type: "list"
    required: false

# Optional pagination
# pagination:
#   type: "next_button"
#   selector: ".next-page"
#   max_pages: 10

# Output configuration
output:
  format: "json"
  file: "results.json"
`

	fmt.Print(template)
}

// convertToEngineConfig converts config.ScraperConfig to scraper.EngineConfig
func convertToEngineConfig(cfg *config.ScraperConfig) *scraper.EngineConfig {
	// Convert fields
	var fields []scraper.FieldConfig
	for _, field := range cfg.Fields {
		fieldConfig := scraper.FieldConfig{
			Name:      field.Name,
			Selector:  field.Selector,
			Type:      field.Type,
			Required:  field.Required,
			Attribute: field.Attribute,
			Default:   field.Default,
		}
		
		// Convert transform rules
		for _, transform := range field.Transform {
			fieldConfig.Transform = append(fieldConfig.Transform, convertTransformRule(transform))
		}
		
		fields = append(fields, fieldConfig)
	}

	// Parse timeout
	timeout := 30 * time.Second
	if cfg.Timeout != "" {
		if parsed, err := time.ParseDuration(cfg.Timeout); err == nil {
			timeout = parsed
		}
	}

	// Parse rate limit
	rateLimit := time.Duration(0)
	if cfg.RateLimit != "" {
		if parsed, err := time.ParseDuration(cfg.RateLimit); err == nil {
			rateLimit = parsed
		}
	}

	return &scraper.EngineConfig{
		Fields:         fields,
		UserAgents:     cfg.UserAgents,
		RequestTimeout: timeout,
		RetryAttempts:  cfg.MaxRetries,
		RateLimit:      rateLimit,
		ExtractionConfig: scraper.ExtractionConfig{
			StrictMode:      false,
			ContinueOnError: true,
		},
	}
}

// convertTransformRule converts config.TransformRule to pipeline.TransformRule
func convertTransformRule(rule config.TransformRule) pipeline.TransformRule {
	return pipeline.TransformRule{
		Type:        rule.Type,
		Pattern:     rule.Pattern,
		Replacement: rule.Replacement,
	}
}
