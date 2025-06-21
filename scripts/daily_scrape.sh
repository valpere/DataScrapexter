# Notification functions
send_notification() {
    local subject="$1"
    local message="$2"
    local status="${3:-info}"
    
    # Email notification
    if [ -n "$NOTIFY_EMAIL" ] && command -v mail &> /dev/null; then
        echo "$message" | mail -s "DataScrapexter: $subject" "$NOTIFY_EMAIL"
    fi
    
    # Webhook notification (Slack, Discord, etc.)
    if [ -n "$NOTIFY_WEBHOOK" ]; then
        local color="#808080"
        case "$status" in
            "success") color="#36a64f" ;;
            "error") color="#ff0000" ;;
            "warning") color="#ffaa00" ;;
        esac
        
        curl -X POST "$NOTIFY_WEBHOOK" \
            -H 'Content-Type: application/json' \
            -d "{\"text\":\"$subject\",\"attachments\":[{\"color\":\"$color\",\"text\":\"$message\"}]}" \
            2>/dev/null || true
    fi
}

# Function to run a single scraper
run_scraper() {
    local config_file="$1"
    local config_name=$(basename "$config_file" .yaml)
    local output_file="$OUTPUT_DIR/daily/$DATE/${config_name}_$TIMESTAMP.json"
    
    log "Starting scraper: $config_name"
    
    # Run the scraper with error handling
    if datascrapexter run "$config_file" -o "$output_file" >> "$LOG_FILE" 2>&1; then
        log_success "Completed: $config_name"
        
        # Validate output
        if [ -f "$output_file" ] && [ -s "$output_file" ]; then
            local record_count=$(grep -c '"url"' "$output_file" 2>/dev/null || echo "0")
            log "Extracted $record_count records from $config_name"
            return 0
        else
            log_error "Output file is empty for $config_name"
            return 1
        fi
    else
        log_error "Failed to run $config_name"
        return 1
    fi
}

# Function to validate configuration files
validate_configs() {
    local configs=("$@")
    local valid_configs=()
    
    log "Validating configuration files..."
    
    for config in "${configs[@]}"; do
        if [ -f "$config" ]; then
            if datascrapexter validate "$config" &>/dev/null; then
                valid_configs+=("$config")
                log "Validated: $(basename "$config")"
            else
                log_error "Invalid configuration: $(basename "$config")"
            fi
        else
            log_error "Configuration file not found: $config"
        fi
    done
    
    echo "${valid_configs[@]}"
}

# Function to compress old outputs
archive_old_outputs() {
    log "Archiving old outputs..."
    
    # Find outputs older than 7 days
    find "$OUTPUT_DIR/daily" -type f -name "*.json" -mtime +7 -print0 | while IFS= read -r -d '' file; do
        local dir_date=$(basename "$(dirname "$file")")
        local archive_dir="$OUTPUT_DIR/archive/$dir_date"
        
        mkdir -p "$archive_dir"
        
        # Compress and move
        if gzip -c "$file" > "$archive_dir/$(basename "$file").gz"; then
            rm "$file"
            log "Archived: $(basename "$file")"
        fi
    done
    
    # Remove empty directories
    find "$OUTPUT_DIR/daily" -type d -empty -delete 2>/dev/null || true
}

# Function to generate summary report
generate_summary() {
    local total_scrapers="$1"
    local successful="$2"
    local failed="$3"
    local duration="$4"
    
    local summary="Daily Scraping Summary - $DATE
=====================================
Total Scrapers: $total_scrapers
Successful: $successful
Failed: $failed
Duration: $duration

Detailed Results:
"
    
    # Add details from log
    summary+=$(grep -E "SUCCESS:|ERROR:" "$LOG_FILE" | tail -20)
    
    echo "$summary"
}

# Main execution
main() {
    log "======================================"
    log "Starting daily scraping job"
    log "======================================"
    
    local start_time=$(date +%s)
    
    # Find all configuration files
    local configs=()
    if [ -d "$CONFIG_DIR/production" ]; then
        configs=($(find "$CONFIG_DIR/production" -name "*.yaml" -type f))
    else
        configs=($(find "$CONFIG_DIR" -name "*.yaml" -type f))
    fi
    
    if [ ${#configs[@]} -eq 0 ]; then
        log_error "No configuration files found"
        exit 1
    fi
    
    log "Found ${#configs[@]} configuration files"
    
    # Validate configurations
    valid_configs=($(validate_configs "${configs[@]}"))
    
    if [ ${#valid_configs[@]} -eq 0 ]; then
        log_error "No valid configuration files"
        send_notification "Daily Scraping Failed" "No valid configuration files found" "error"
        exit 1
    fi
    
    # Run scrapers
    local successful=0
    local failed=0
    
    for config in "${valid_configs[@]}"; do
        if run_scraper "$config"; then
            ((successful++))
        else
            ((failed++))
        fi
        
        # Small delay between scrapers
        sleep 2
    done
    
    # Calculate duration
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    local duration_min=$((duration / 60))
    local duration_sec=$((duration % 60))
    local duration_str="${duration_min}m ${duration_sec}s"
    
    # Archive old outputs
    archive_old_outputs
    
    # Generate summary
    local summary=$(generate_summary "${#valid_configs[@]}" "$successful" "$failed" "$duration_str")
    
    # Log summary
    log "======================================"
    log "Daily scraping completed"
    log "Duration: $duration_str"
    log "Successful: $successful/${#valid_configs[@]}"
    log "======================================"
    
    # Send notifications
    if [ $failed -gt 0 ]; then
        log_error "Some scrapers failed"
        if [ "$NOTIFY_ON_FAILURE" = "true" ]; then
            send_notification "Daily Scraping Completed with Errors" "$summary" "error"
        fi
        exit 1
    else
        log_success "All scrapers completed successfully"
        if [ "$NOTIFY_ON_SUCCESS" = "true" ]; then
            send_notification "Daily Scraping Completed Successfully" "$summary" "success"
        fi
        exit 0
    fi
}

# Check if datascrapexter is available
if ! command -v datascrapexter &> /dev/null; then
    log_error "datascrapexter not found in PATH"
    exit 1
fi

# Run main function
main "$@"#!/bin/bash
# Daily scraping automation script for DataScrapexter
# This script handles scheduled scraping with logging, error handling, and notifications

set -e

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
CONFIG_DIR="$PROJECT_ROOT/configs"
OUTPUT_DIR="$PROJECT_ROOT/outputs"
LOG_DIR="$PROJECT_ROOT/logs"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
DATE=$(date +%Y-%m-%d)
LOG_FILE="$LOG_DIR/daily_scrape_$DATE.log"

# Create directories if they don't exist
mkdir -p "$OUTPUT_DIR/daily/$DATE"
mkdir -p "$LOG_DIR"
mkdir -p "$OUTPUT_DIR/archive"

# Notification settings (configure as needed)
NOTIFY_EMAIL="${SCRAPER_NOTIFY_EMAIL:-}"
NOTIFY_WEBHOOK="${SCRAPER_NOTIFY_WEBHOOK:-}"
NOTIFY_ON_SUCCESS="${SCRAPER_NOTIFY_SUCCESS:-false}"
NOTIFY_ON_FAILURE="${SCRAPER_NOTIFY_FAILURE:-true}"

# Colors for console output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Logging functions
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" | tee -a "$LOG_FILE"
}

log_error() {
    echo -e "${RED}[$(date '+%Y-%m-%d %H:%M:%S')] ERROR: $1${NC}" | tee -a "$LOG_FILE"
}

log_success() {
    echo -e "${GREEN}[$(date '+%Y-%m-%d %H:%M:%S')] SUCCESS: $1${NC}" | tee -a "$LOG_FILE"
}

log_warning() {
    echo -e "${YELLOW}[$(date '+%Y-%m-%d %H:%M:%S')] WARNING: $1${NC}" | tee -a "$LOG_FILE"
}

# Notification functions
send_notification() {
    local subject
