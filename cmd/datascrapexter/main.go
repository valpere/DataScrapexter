// cmd/datascrapexter/main.go - Enhanced existing functions with error management
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/valpere/DataScrapexter/internal/errors"
	"github.com/valpere/DataScrapexter/internal/config"
	"github.com/valpere/DataScrapexter/internal/scraper"
	"github.com/valpere/DataScrapexter/internal/output"
	"gopkg.in/yaml.v3"
)

// Version information (set by build flags)
var (
	version   = "dev"
	buildTime = "unknown"
	gitCommit = "unknown"
)

// Global error service instance
var errorService = errors.NewService()

// Enhanced runScraper function (existing signature preserved)
func runScraper(configFile string) {
	// Check for verbose flag
	verbose := hasFlag("-v") || hasFlag("--verbose")
	errorService = errorService.WithVerbose(verbose)
	
	ctx := context.Background()
	
	// Execute with retry and error handling
	err := errorService.ExecuteWithRetry(ctx, func() error {
		return executeScrapingOperation(configFile, verbose)
	}, "scraping")
	
	if err != nil {
		fmt.Fprint(os.Stderr, errorService.FormatErrorForCLI(err))
		os.Exit(errorService.GetExitCode(err))
	}
}

// Enhanced validateConfig function (existing signature preserved)
func validateConfig(configFile string) {
	verbose := hasFlag("-v") || hasFlag("--verbose")
	errorService = errorService.WithVerbose(verbose)
	
	ctx := context.Background()
	
	err := errorService.ExecuteWithRetry(ctx, func() error {
		return executeValidation(configFile, verbose)
	}, "validation")
	
	if err != nil {
		fmt.Fprint(os.Stderr, errorService.FormatErrorForCLI(err))
		os.Exit(errorService.GetExitCode(err))
	}
	
	fmt.Printf("✓ Configuration file '%s' is valid\n", configFile)
}

// Enhanced generateTemplate function (existing signature preserved)
func generateTemplate(args []string) (string, error) {
	templateType := "basic"
	if len(args) > 0 && args[0] == "--type" && len(args) > 1 {
		templateType = args[1]
	}
	
	// Use existing template generation logic
	template := config.GenerateTemplate(templateType)
	
	// Convert to YAML string
	yamlData, err := yaml.Marshal(template)
	if err != nil {
		return "", fmt.Errorf("failed to marshal template to YAML: %w", err)
	}
	
	return string(yamlData), nil
}

// executeScrapingOperation performs the actual scraping with enhanced error handling
func executeScrapingOperation(configFile string, verbose bool) error {
	// Load configuration
	cfg, err := config.LoadFromFile(configFile)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}
	
	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}
	
	if verbose {
		fmt.Printf("Configuration loaded: %s\n", cfg.Name)
		fmt.Printf("Target URL: %s\n", cfg.BaseURL)
		fmt.Printf("Fields to extract: %d\n", len(cfg.Fields))
	}
	
	// Create engine with existing constructor
	engineConfig := convertToEngineConfig(cfg)
	engine, err := scraper.NewEngine(engineConfig)
	if err != nil {
		return fmt.Errorf("failed to create scraping engine: %w", err)
	}
	
	// Execute scraping
	if verbose {
		fmt.Printf("Starting scraping operation...\n")
	}
	
	// Convert config fields to FieldConfig for scraping
	fieldConfigs := make([]scraper.FieldConfig, len(cfg.Fields))
	for i, field := range cfg.Fields {
		fieldConfigs[i] = scraper.FieldConfig{
			Name:      field.Name,
			Selector:  field.Selector,
			Type:      field.Type,
			Required:  field.Required,
			Attribute: field.Attribute,
			Default:   field.Default,
		}
	}
	
	result, err := engine.Scrape(context.Background(), cfg.BaseURL, fieldConfigs)
	if err != nil {
		return fmt.Errorf("scraping failed: %w", err)
	}
	
	// Check for partial failures
	if !result.Success && result.Data != nil {
		fmt.Printf("⚠ Scraping completed with some errors, saving partial results\n")
	}
	
	// Save results using existing output manager
	outputManager, err := output.NewManager(&cfg.Output)
	if err != nil {
		return fmt.Errorf("failed to create output manager: %w", err)
	}
	
	outputData := []map[string]interface{}{result.Data}
	err = outputManager.WriteResults(outputData)
	if err != nil {
		return fmt.Errorf("failed to write results: %w", err)
	}
	
	if verbose {
		fmt.Printf("Results saved to: %s\n", cfg.Output.File)
		fmt.Printf("Fields extracted: %d\n", len(result.Data))
	} else {
		fmt.Printf("Scraping completed successfully. Results saved to %s\n", cfg.Output.File)
	}
	
	return nil
}

// executeValidation performs configuration validation
func executeValidation(configFile string, verbose bool) error {
	cfg, err := config.LoadFromFile(configFile)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}
	
	err = cfg.Validate()
	if err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}
	
	if verbose {
		fmt.Printf("Configuration details:\n")
		fmt.Printf("  Name: %s\n", cfg.Name)
		fmt.Printf("  Base URL: %s\n", cfg.BaseURL)
		fmt.Printf("  Fields: %d\n", len(cfg.Fields))
		fmt.Printf("  Output format: %s\n", cfg.Output.Format)
	}
	
	return nil
}

// convertToEngineConfig converts config to engine format (existing function enhanced)
func convertToEngineConfig(cfg *config.ScraperConfig) *scraper.Config {
	engineConfig := &scraper.Config{
		MaxRetries:      cfg.MaxRetries,
		Timeout:         30 * time.Second,
		FollowRedirects: true,
		MaxRedirects:    10,
		RateLimit:       1 * time.Second,
		BurstSize:       5,
		Headers:         cfg.Headers,
		UserAgents:      cfg.UserAgents,
	}

	// Convert browser configuration if present
	if cfg.Browser != nil {
		browserConfig := &scraper.BrowserConfig{
			Enabled:        cfg.Browser.Enabled,
			Headless:       cfg.Browser.Headless,
			UserDataDir:    cfg.Browser.UserDataDir,
			ViewportWidth:  cfg.Browser.ViewportWidth,
			ViewportHeight: cfg.Browser.ViewportHeight,
			WaitForElement: cfg.Browser.WaitForElement,
			UserAgent:      cfg.Browser.UserAgent,
			DisableImages:  cfg.Browser.DisableImages,
			DisableCSS:     cfg.Browser.DisableCSS,
			DisableJS:      cfg.Browser.DisableJS,
		}

		// Parse timeout strings
		if cfg.Browser.Timeout != "" {
			if duration, err := time.ParseDuration(cfg.Browser.Timeout); err == nil {
				browserConfig.Timeout = duration
			}
		}
		if cfg.Browser.WaitDelay != "" {
			if duration, err := time.ParseDuration(cfg.Browser.WaitDelay); err == nil {
				browserConfig.WaitDelay = duration
			}
		}

		engineConfig.Browser = browserConfig
	}

	// Convert proxy configuration if present
	if cfg.Proxy != nil {
		proxyConfig := &scraper.ProxyConfig{
			Enabled:          cfg.Proxy.Enabled,
			Rotation:         cfg.Proxy.Rotation,
			HealthCheck:      cfg.Proxy.HealthCheck,
			HealthCheckURL:   cfg.Proxy.HealthCheckURL,
			MaxRetries:       cfg.Proxy.MaxRetries,
			FailureThreshold: cfg.Proxy.FailureThreshold,
			Providers:        make([]scraper.ProxyProvider, len(cfg.Proxy.Providers)),
		}

		// Parse timeout strings
		if cfg.Proxy.Timeout != "" {
			if duration, err := time.ParseDuration(cfg.Proxy.Timeout); err == nil {
				proxyConfig.Timeout = duration
			}
		}
		if cfg.Proxy.RetryDelay != "" {
			if duration, err := time.ParseDuration(cfg.Proxy.RetryDelay); err == nil {
				proxyConfig.RetryDelay = duration
			}
		}
		if cfg.Proxy.HealthCheckRate != "" {
			if duration, err := time.ParseDuration(cfg.Proxy.HealthCheckRate); err == nil {
				proxyConfig.HealthCheckRate = duration
			}
		}
		if cfg.Proxy.RecoveryTime != "" {
			if duration, err := time.ParseDuration(cfg.Proxy.RecoveryTime); err == nil {
				proxyConfig.RecoveryTime = duration
			}
		}

		// Convert providers
		for i, provider := range cfg.Proxy.Providers {
			proxyConfig.Providers[i] = scraper.ProxyProvider{
				Name:     provider.Name,
				Type:     provider.Type,
				Host:     provider.Host,
				Port:     provider.Port,
				Username: provider.Username,
				Password: provider.Password,
				Weight:   provider.Weight,
				Enabled:  provider.Enabled,
			}
		}

		engineConfig.Proxy = proxyConfig
	}

	return engineConfig
}

// hasFlag checks if a flag is present in command line arguments
func hasFlag(flag string) bool {
	for _, arg := range os.Args {
		if arg == flag {
			return true
		}
	}
	return false
}

// main function handles CLI arguments and routes to appropriate functions
func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]
	
	switch command {
	case "run":
		if len(os.Args) < 3 {
			fmt.Fprintf(os.Stderr, "Error: config file required\n")
			fmt.Fprintf(os.Stderr, "Usage: datascrapexter run <config.yaml>\n")
			os.Exit(1)
		}
		runScraper(os.Args[2])
		
	case "validate":
		if len(os.Args) < 3 {
			fmt.Fprintf(os.Stderr, "Error: config file required\n")
			fmt.Fprintf(os.Stderr, "Usage: datascrapexter validate <config.yaml>\n")
			os.Exit(1)
		}
		validateConfig(os.Args[2])
		
	case "template":
		template, err := generateTemplate(os.Args[2:])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Print(template)
		
	case "version", "--version", "-v":
		printVersion()
		
	case "help", "--help", "-h":
		printUsage()
		
	default:
		fmt.Fprintf(os.Stderr, "Error: unknown command '%s'\n", command)
		printUsage()
		os.Exit(1)
	}
}

// printUsage displays help information
func printUsage() {
	fmt.Println("DataScrapexter - Professional Web Scraping Tool")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  datascrapexter run <config.yaml>        Run scraper with configuration file")
	fmt.Println("  datascrapexter validate <config.yaml>   Validate configuration file")
	fmt.Println("  datascrapexter template [--type <type>] Generate configuration template")
	fmt.Println("  datascrapexter version                  Show version information")
	fmt.Println("  datascrapexter help                     Show this help message")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -v, --verbose                           Enable verbose output")
	fmt.Println()
	fmt.Println("Template types:")
	fmt.Println("  basic       Basic scraping template (default)")
	fmt.Println("  ecommerce   E-commerce scraping template") 
	fmt.Println("  news        News article scraping template")
}

// printVersion displays version information
func printVersion() {
	fmt.Printf("DataScrapexter %s\n", version)
	fmt.Printf("Build time: %s\n", buildTime)
	fmt.Printf("Git commit: %s\n", gitCommit)
}
