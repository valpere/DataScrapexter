#!/bin/bash
# Script to scrape individual products from a list of URLs
# Used for detailed product information extraction

set -e

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
CONFIG_FILE="${1:-$PROJECT_ROOT/configs/product-details.yaml}"
URL_FILE="${2:-$PROJECT_ROOT/outputs/product-urls.txt}"
OUTPUT_DIR="${3:-$PROJECT_ROOT/outputs/products}"
LOG_FILE="$PROJECT_ROOT/logs/product_scraping_$(date +%Y%m%d_%H%M%S).log"

# Create directories
mkdir -p "$OUTPUT_DIR"
mkdir -p "$(dirname "$LOG_FILE")"

# Rate limiting settings
DELAY_BETWEEN_PRODUCTS="${SCRAPE_DELAY:-3}"
MAX_CONCURRENT="${MAX_CONCURRENT:-1}"
BATCH_SIZE="${BATCH_SIZE:-10}"

# Progress tracking
TOTAL_URLS=0
PROCESSED=0
SUCCESSFUL=0
FAILED=0

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Functions
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" | tee -a "$LOG_FILE"
}

log_error() {
    echo -e "${RED}[$(date '+%Y-%m-%d %H:%M:%S')] ERROR: $1${NC}" | tee -a "$LOG_FILE"
}

log_success() {
    echo -e "${GREEN}[$(date '+%Y-%m-%d %H:%M:%S')] SUCCESS: $1${NC}" | tee -a "$LOG_FILE"
}

validate_prerequisites() {
    # Check if datascrapexter is available
    if ! command -v datascrapexter &> /dev/null; then
        log_error "datascrapexter not found in PATH"
        exit 1
    fi
    
    # Check if configuration file exists
    if [ ! -f "$CONFIG_FILE" ]; then
        log_error "Configuration file not found: $CONFIG_FILE"
        exit 1
    fi
    
    # Validate configuration
    if ! datascrapexter validate "$CONFIG_FILE" &>/dev/null; then
        log_error "Invalid configuration file: $CONFIG_FILE"
        exit 1
    fi
    
    # Check if URL file exists
    if [ ! -f "$URL_FILE" ]; then
        log_error "URL file not found: $URL_FILE"
        exit 1
    fi
    
    # Count total URLs
    TOTAL_URLS=$(grep -c "^http" "$URL_FILE" 2>/dev/null || echo 0)
    
    if [ "$TOTAL_URLS" -eq 0 ]; then
        log_error "No valid URLs found in $URL_FILE"
        exit 1
    fi
}

generate_safe_filename() {
    local url="$1"
    # Extract last part of URL and clean it
    local filename=$(basename "$url" | sed 's/[^a-zA-Z0-9-_]/_/g')
    
    # If filename is empty or too generic, use URL hash
    if [ -z "$filename" ] || [ "$filename" = "_" ]; then
        filename=$(echo -n "$url" | md5sum | cut -d' ' -f1)
    fi
    
    # Add timestamp to ensure uniqueness
    echo "${filename}_$(date +%s)"
}

scrape_product() {
    local url="$1"
    local output_file="$2"
    
    log "Scraping: $url"
    
    # Set environment variable for URL
    export PRODUCT_URL="$url"
    
    # Run scraper with timeout
    if timeout 120 datascrapexter run "$CONFIG_FILE" -o "$output_file" >> "$LOG_FILE" 2>&1; then
        # Validate output
        if [ -f "$output_file" ] && [ -s "$output_file" ]; then
            log_success "Scraped: $url"
            return 0
        else
            log_error "Output file is empty for: $url"
            rm -f "$output_file"
            return 1
        fi
    else
        log_error "Failed to scrape: $url"
        rm -f "$output_file"
        return 1
    fi
}

process_batch() {
    local -a batch=("$@")
    local batch_failed=0
    
    for url in "${batch[@]}"; do
        if [ -z "$url" ] || [[ ! "$url" =~ ^https?:// ]]; then
            log_error "Invalid URL: $url"
            ((FAILED++))
            continue
        fi
        
        # Generate output filename
        local filename=$(generate_safe_filename "$url")
        local output_file="$OUTPUT_DIR/${filename}.json"
        
        # Skip if already scraped
        if [ -f "$output_file" ]; then
            log "Already scraped: $url"
            ((SUCCESSFUL++))
            continue
        fi
        
        # Scrape the product
        if scrape_product "$url" "$output_file"; then
            ((SUCCESSFUL++))
        else
            ((FAILED++))
            ((batch_failed++))
        fi
        
        ((PROCESSED++))
        
        # Progress update
        local progress=$((PROCESSED * 100 / TOTAL_URLS))
        echo -ne "\rProgress: [$PROCESSED/$TOTAL_URLS] ${progress}% - Success: $SUCCESSFUL, Failed: $FAILED"
        
        # Rate limiting
        if [ "$PROCESSED" -lt "$TOTAL_URLS" ]; then
            sleep "$DELAY_BETWEEN_PRODUCTS"
        fi
    done
    
    return $batch_failed
}

show_summary() {
    echo -e "\n\n======================================"
    echo "Scraping Summary"
    echo "======================================"
    echo "Total URLs: $TOTAL_URLS"
    echo -e "Successful: ${GREEN}$SUCCESSFUL${NC}"
    echo -e "Failed: ${RED}$FAILED${NC}"
    echo "Success Rate: $((SUCCESSFUL * 100 / TOTAL_URLS))%"
    echo "Output Directory: $OUTPUT_DIR"
    echo "Log File: $LOG_FILE"
    echo "======================================"
}

main() {
    log "======================================"
    log "Starting Product Scraping"
    log "Configuration: $CONFIG_FILE"
    log "URL File: $URL_FILE"
    log "Output Directory: $OUTPUT_DIR"
    log "======================================"
    
    # Validate prerequisites
    validate_prerequisites
    
    log "Found $TOTAL_URLS URLs to process"
    
    # Read URLs into array
    mapfile -t urls < <(grep "^http" "$URL_FILE")
    
    # Process URLs in batches
    local batch_start=0
    
    while [ "$batch_start" -lt "$TOTAL_URLS" ]; do
        # Extract batch
        local batch_end=$((batch_start + BATCH_SIZE))
        if [ "$batch_end" -gt "$TOTAL_URLS" ]; then
            batch_end=$TOTAL_URLS
        fi
        
        local batch=("${urls[@]:$batch_start:$BATCH_SIZE}")
        
        # Process batch
        process_batch "${batch[@]}"
        
        # Move to next batch
        batch_start=$batch_end
        
        # Optional: pause between batches
        if [ "$batch_start" -lt "$TOTAL_URLS" ]; then
            log "Completed batch, pausing before next batch..."
            sleep 5
        fi
    done
    
    # Show summary
    show_summary
    
    # Exit with appropriate code
    if [ "$FAILED" -eq 0 ]; then
        exit 0
    elif [ "$SUCCESSFUL" -gt 0 ]; then
        exit 1  # Partial success
    else
        exit 2  # Complete failure
    fi
}

# Handle interrupts gracefully
trap 'echo -e "\n\nInterrupted! Showing summary..."; show_summary; exit 130' INT TERM

# Check arguments
if [ "$1" = "--help" ] || [ "$1" = "-h" ]; then
    echo "Usage: $0 [config_file] [url_file] [output_dir]"
    echo ""
    echo "Arguments:"
    echo "  config_file  - Product details scraper configuration (default: configs/product-details.yaml)"
    echo "  url_file     - File containing product URLs (default: outputs/product-urls.txt)"
    echo "  output_dir   - Directory for scraped products (default: outputs/products)"
    echo ""
    echo "Environment Variables:"
    echo "  SCRAPE_DELAY    - Delay between products in seconds (default: 3)"
    echo "  BATCH_SIZE      - Number of products per batch (default: 10)"
    echo "  MAX_CONCURRENT  - Maximum concurrent scrapers (default: 1)"
    exit 0
fi

# Run main function
main
