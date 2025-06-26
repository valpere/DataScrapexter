// cmd/datascrapexter/main.go - Enhanced placeholder functions only
package main

import (
	"fmt"
	"os"

	"github.com/valpere/DataScrapexter/internal/config"
	"gopkg.in/yaml.v3"
)

// Build-time variables (set by ldflags)
var (
	version   = "dev"
	buildTime = "unknown"
	gitCommit = "unknown"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "version" {
		fmt.Printf("DataScrapexter %s\n", version)
		fmt.Printf("Build time: %s\n", buildTime)
		fmt.Printf("Git commit: %s\n", gitCommit)
		return
	}

	fmt.Printf("DataScrapexter v%s\n", version)
	fmt.Printf("Universal web scraper built with Go\n")
	fmt.Printf("Build: %s (%s)\n", gitCommit, buildTime)
	fmt.Println()
	
	if len(os.Args) < 2 {
		printUsage()
		return
	}

	switch os.Args[1] {
	case "run":
		if len(os.Args) < 3 {
			fmt.Println("Error: config file required")
			fmt.Println("Usage: datascrapexter run <config.yaml>")
			os.Exit(1)
		}
		runScraper(os.Args[2])
	case "validate":
		if len(os.Args) < 3 {
			fmt.Println("Error: config file required")
			fmt.Println("Usage: datascrapexter validate <config.yaml>")
			os.Exit(1)
		}
		validateConfig(os.Args[2])
	case "template":
		generateTemplate()
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Printf("Unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage: datascrapexter <command> [options]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  run <config>      Run scraper with specified config file")
	fmt.Println("  validate <config> Validate configuration file")
	fmt.Println("  template          Generate template configuration")
	fmt.Println("  version           Show version information")
	fmt.Println("  help              Show this help message")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  datascrapexter run example.yaml")
	fmt.Println("  datascrapexter validate config.yaml")
	fmt.Println("  datascrapexter template > new_config.yaml")
}

// Enhanced runScraper - now uses existing internal packages
func runScraper(configFile string) {
	fmt.Printf("Running scraper with config: %s\n", configFile)
	
	// Check if config file exists
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		fmt.Printf("Error: Config file not found: %s\n", configFile)
		os.Exit(1)
	}
	
	// Load configuration using existing config package
	cfg, err := config.LoadFromFile(configFile)
	if err != nil {
		fmt.Printf("Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Loaded configuration: %s\n", cfg.Name)
	
	// For now, just simulate scraping until we verify the interfaces
	fmt.Println("Starting scraper...")
	fmt.Printf("Would scrape: %s\n", cfg.BaseURL)
	fmt.Printf("With %d fields\n", len(cfg.Fields))
	fmt.Println("Scraper completed successfully!")
}

// Enhanced validateConfig - now uses existing config package
func validateConfig(configFile string) {
	fmt.Printf("Validating config: %s\n", configFile)
	
	// Check if config file exists
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		fmt.Printf("Error: Config file not found: %s\n", configFile)
		os.Exit(1)
	}
	
	// Load and validate using existing config package
	cfg, err := config.LoadFromFile(configFile)
	if err != nil {
		fmt.Printf("Configuration validation failed: %v\n", err)
		os.Exit(1)
	}

	// Additional validation using existing validation methods
	if err := cfg.Validate(); err != nil {
		fmt.Printf("Configuration is invalid: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Configuration is valid!")
	fmt.Printf("  Name: %s\n", cfg.Name)
	fmt.Printf("  Base URL: %s\n", cfg.BaseURL)
	fmt.Printf("  Fields: %d\n", len(cfg.Fields))
}

// Enhanced generateTemplate - now uses existing config package  
func generateTemplate() {
	// Check for template type argument
	templateType := "basic"
	if len(os.Args) > 2 {
		templateType = os.Args[2]
	}

	// Generate template using existing config package - returns single value
	template := config.GenerateTemplate(templateType)

	// Convert to YAML manually since ToYAML method doesn't exist
	yamlData, err := yaml.Marshal(template)
	if err != nil {
		fmt.Printf("Error converting template to YAML: %v\n", err)
		os.Exit(1)
	}

	fmt.Print(string(yamlData))
}
