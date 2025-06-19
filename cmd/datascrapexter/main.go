package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"

	"github.com/valpere/DataScrapexter/internal/output"
	"github.com/valpere/DataScrapexter/internal/scraper"
	"github.com/valpere/DataScrapexter/pkg/api"
)

var (
	version   = "0.1.0"
	buildTime = "unknown"
	gitCommit = "unknown"
)

var (
	cfgFile     string
	logLevel    string
	logFormat   string
	concurrency int
	outputFile  string
)

// rootCmd represents the base command
var rootCmd = &cobra.Command{
	Use:   "datascrapexter",
	Short: "Universal web scraper with advanced anti-detection capabilities",
	Long: `DataScrapexter is a high-performance, configuration-driven web scraper
built with Go that can extract data from any website while intelligently
avoiding detection mechanisms.`,
}

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run [config-file]",
	Short: "Run a scraping job using the specified configuration file",
	Args:  cobra.ExactArgs(1),
	RunE:  runScraper,
}

// validateCmd represents the validate command
var validateCmd = &cobra.Command{
	Use:   "validate [config-file]",
	Short: "Validate a scraper configuration file",
	Args:  cobra.ExactArgs(1),
	RunE:  validateConfig,
}

// templateCmd represents the template command
var templateCmd = &cobra.Command{
	Use:   "template",
	Short: "Generate a template configuration file",
	RunE:  generateTemplate,
}

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("DataScrapexter %s\n", version)
		fmt.Printf("Build Time: %s\n", buildTime)
		fmt.Printf("Git Commit: %s\n", gitCommit)
	},
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.datascrapexter.yaml)")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "info", "log level (debug, info, warn, error)")
	rootCmd.PersistentFlags().StringVar(&logFormat, "log-format", "text", "log format (text, json)")

	// Run command flags
	runCmd.Flags().IntVar(&concurrency, "concurrency", 1, "number of concurrent scrapers")
	runCmd.Flags().StringVarP(&outputFile, "output", "o", "", "override output file path")

	// Add commands
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(templateCmd)
	rootCmd.AddCommand(versionCmd)
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".datascrapexter")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runScraper(cmd *cobra.Command, args []string) error {
	configFile := args[0]

	// Load configuration
	config, err := loadScraperConfig(configFile)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Validate configuration
	if err := validateScraperConfig(config); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Override outputFile file if specified
	if outputFile != "" {
		config.Output.File = outputFile
	}

	// Create scraping engine
	engineConfig := createEngineConfig(config)
	engine, err := scraper.NewEngine(engineConfig)
	if err != nil {
		return fmt.Errorf("failed to create scraping engine: %w", err)
	}

	// Convert API fields to scraper field extractors
	extractors := make([]scraper.FieldExtractor, len(config.Fields))
	for i, field := range config.Fields {
		// Convert TransformRule from API to scraper format
		transformRules := make([]scraper.TransformRule, len(field.Transform))
		for j, rule := range field.Transform {
			transformRules[j] = scraper.TransformRule{
				Type:        rule.Type,
				Pattern:     rule.Pattern,
				Replacement: rule.Replacement,
			}
		}

		extractors[i] = scraper.FieldExtractor{
			Name:      field.Name,
			Selector:  field.Selector,
			Type:      field.Type,
			Attribute: field.Attribute,
			Required:  field.Required,
			Transform: transformRules,
		}
	}

	// Create context
	ctx := context.Background()

	// Start scraping
	log.Printf("Starting scraper '%s' for URL: %s", config.Name, config.BaseURL)

	startTime := time.Now()
	result, err := engine.Scrape(ctx, config.BaseURL, extractors)
	if err != nil {
		return fmt.Errorf("scraping failed: %w", err)
	}

	duration := time.Since(startTime)
	log.Printf("Scraping completed in %v", duration)

	// Handle pagination if configured
	results := []*scraper.Result{result}
	if config.Pagination != nil {
		log.Printf("Pagination enabled: type=%s, max_pages=%d", config.Pagination.Type, config.Pagination.MaxPages)

		paginatedScraper, err := scraper.NewPaginatedScraper(engine, config, extractors)
		if err != nil {
			log.Printf("Warning: Failed to create paginated scraper: %v", err)
			log.Println("Continuing with single page result")
		} else {
			// Scrape all pages
			allResults, err := paginatedScraper.ScrapeAll(ctx)
			if err != nil {
				log.Printf("Warning: Pagination error: %v", err)
			} else {
				results = allResults
			}
		}
	}

	// Output results
	if err := outputResults(results, config.Output); err != nil {
		return fmt.Errorf("failed to output results: %w", err)
	}

	log.Printf("Successfully scraped %d pages", len(results))
	return nil
}

func validateConfig(cmd *cobra.Command, args []string) error {
	configFile := args[0]

	config, err := loadScraperConfig(configFile)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	if err := validateScraperConfig(config); err != nil {
		fmt.Printf("❌ Configuration is invalid:\n%v\n", err)
		return err
	}

	fmt.Println("✅ Configuration is valid!")
	return nil
}

func generateTemplate(cmd *cobra.Command, args []string) error {
	template := api.ScraperConfig{
		Name:       "example_scraper",
		BaseURL:    "https://example.com",
		RateLimit:  "2s",
		Timeout:    "30s",
		MaxRetries: 3,
		Headers: map[string]string{
			"Accept-Language": "en-US,en;q=0.9",
		},
		Fields: []api.Field{
			{
				Name:     "title",
				Selector: "h1",
				Type:     "text",
				Required: true,
			},
			{
				Name:     "price",
				Selector: ".price",
				Type:     "text",
				Transform: []api.TransformRule{
					{
						Type:        "regex",
						Pattern:     `\$([0-9,]+\.?[0-9]*)`,
						Replacement: "$1",
					},
					{
						Type: "parse_float",
					},
				},
			},
			{
				Name:     "description",
				Selector: ".description",
				Type:     "text",
			},
			{
				Name:      "image",
				Selector:  "img.product-image",
				Type:      "attr",
				Attribute: "src",
			},
		},
		Pagination: &api.PaginationConfig{
			Type:     "next_button",
			Selector: ".pagination .next",
			MaxPages: 10,
		},
		Output: api.OutputConfig{
			Format: "json",
			File:   "output.json",
		},
	}

	data, err := yaml.Marshal(template)
	if err != nil {
		return fmt.Errorf("failed to marshal template: %w", err)
	}

	fmt.Println(string(data))
	return nil
}

func loadScraperConfig(filename string) (*api.ScraperConfig, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config api.ScraperConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Expand environment variables
	config.BaseURL = os.ExpandEnv(config.BaseURL)
	if config.Proxy != nil {
		config.Proxy.URL = os.ExpandEnv(config.Proxy.URL)
	}
	if config.Output.Database != nil {
		config.Output.Database.URL = os.ExpandEnv(config.Output.Database.URL)
	}

	return &config, nil
}

func validateScraperConfig(config *api.ScraperConfig) error {
	// Basic validation
	if config.Name == "" {
		return fmt.Errorf("scraper name is required")
	}
	if config.BaseURL == "" {
		return fmt.Errorf("base URL is required")
	}
	if len(config.Fields) == 0 {
		return fmt.Errorf("at least one field must be defined")
	}

	// Validate fields
	for i, field := range config.Fields {
		if field.Name == "" {
			return fmt.Errorf("field[%d]: name is required", i)
		}
		if field.Selector == "" {
			return fmt.Errorf("field[%d]: selector is required", i)
		}
		if field.Type == "" {
			return fmt.Errorf("field[%d]: type is required", i)
		}
		// Validate field type
		switch field.Type {
		case "text", "html", "attr", "list":
			// Valid types
		default:
			return fmt.Errorf("field[%d]: invalid type '%s'", i, field.Type)
		}
		// Validate attr type has attribute
		if field.Type == "attr" && field.Attribute == "" {
			return fmt.Errorf("field[%d]: attribute is required for type 'attr'", i)
		}
	}

	// Validate output
	if config.Output.Format == "" {
		config.Output.Format = "json" // Default
	}
	switch config.Output.Format {
	case "json", "csv", "excel":
		// Valid formats
	default:
		return fmt.Errorf("invalid output format '%s'", config.Output.Format)
	}

	// Validate rate limit
	if config.RateLimit != "" {
		if _, err := time.ParseDuration(config.RateLimit); err != nil {
			return fmt.Errorf("invalid rate limit duration: %w", err)
		}
	}

	// Validate timeout
	if config.Timeout != "" {
		if _, err := time.ParseDuration(config.Timeout); err != nil {
			return fmt.Errorf("invalid timeout duration: %w", err)
		}
	}

	return nil
}

func createEngineConfig(config *api.ScraperConfig) *scraper.Config {
	engineConfig := scraper.DefaultConfig()

	// Parse durations
	if config.RateLimit != "" {
		if d, err := time.ParseDuration(config.RateLimit); err == nil {
			engineConfig.RateLimit = d
		}
	}
	if config.Timeout != "" {
		if d, err := time.ParseDuration(config.Timeout); err == nil {
			engineConfig.Timeout = d
		}
	}

	// Set other configurations
	if config.MaxRetries > 0 {
		engineConfig.MaxRetries = config.MaxRetries
	}
	engineConfig.Headers = config.Headers

	// Set proxy if configured
	if config.Proxy != nil && config.Proxy.Enabled && config.Proxy.URL != "" {
		engineConfig.ProxyURL = config.Proxy.URL
	}

	return engineConfig
}

func outputResults(results []*scraper.Result, outputConfig api.OutputConfig) error {
	switch outputConfig.Format {
	case "json":
		return outputJSON(results, outputConfig.File)
	case "csv":
		return outputCSV(results, outputConfig.File)
	case "excel":
		return fmt.Errorf("Excel output not yet implemented")
	default:
		return fmt.Errorf("unsupported output format: %s", outputConfig.Format)
	}
}

func outputJSON(results []*scraper.Result, filename string) error {
	// Convert scraper results to API results
	apiResults := make([]api.ScrapeResult, len(results))
	for i, result := range results {
		apiResults[i] = api.ScrapeResult{
			URL:        result.URL,
			StatusCode: result.StatusCode,
			Data:       result.Data,
			Timestamp:  result.Timestamp,
		}
		if result.Error != nil {
			apiResults[i].Error = result.Error.Error()
		}
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(apiResults, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal results: %w", err)
	}

	// Write to file or stdout
	if filename == "" || filename == "-" {
		fmt.Println(string(data))
		return nil
	}

	// Create directory if needed
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Write to file
	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	log.Printf("Results written to %s", filename)
	return nil
}

func outputCSV(results []*scraper.Result, filename string) error {
	return output.WriteResultsToCSV(results, filename)
}
