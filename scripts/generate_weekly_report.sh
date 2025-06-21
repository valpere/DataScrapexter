#!/bin/bash
# Weekly report generation script for DataScrapexter
# Generates comprehensive reports on scraping activity and data insights

set -e

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
OUTPUT_DIR="$PROJECT_ROOT/outputs"
LOG_DIR="$PROJECT_ROOT/logs"
REPORTS_DIR="$PROJECT_ROOT/reports"
METRICS_DIR="$PROJECT_ROOT/metrics"

# Time periods
WEEK_START=$(date -d "last monday" +%Y-%m-%d)
WEEK_END=$(date +%Y-%m-%d)
REPORT_DATE=$(date +%Y%m%d)
REPORT_TIMESTAMP=$(date +%Y%m%d_%H%M%S)

# Create directories
mkdir -p "$REPORTS_DIR/weekly"
mkdir -p "$REPORTS_DIR/data"

# Report files
REPORT_FILE="$REPORTS_DIR/weekly/report_${REPORT_TIMESTAMP}.md"
SUMMARY_JSON="$REPORTS_DIR/data/summary_${REPORT_TIMESTAMP}.json"
METRICS_CSV="$REPORTS_DIR/data/metrics_${REPORT_TIMESTAMP}.csv"

# Email settings
REPORT_EMAIL="${REPORT_EMAIL:-}"
REPORT_SUBJECT="DataScrapexter Weekly Report - $WEEK_START to $WEEK_END"

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Functions
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1"
}

calculate_stats() {
    local values=("$@")
    local count=${#values[@]}
    
    if [ "$count" -eq 0 ]; then
        echo "0,0,0,0"
        return
    fi
    
    # Calculate sum and average
    local sum=0
    for val in "${values[@]}"; do
        sum=$((sum + val))
    done
    local avg=$((sum / count))
    
    # Find min and max
    local min=${values[0]}
    local max=${values[0]}
    for val in "${values[@]}"; do
        [ "$val" -lt "$min" ] && min=$val
        [ "$val" -gt "$max" ] && max=$val
    done
    
    echo "$sum,$avg,$min,$max"
}

analyze_scraping_activity() {
    log "Analyzing scraping activity..."
    
    local total_runs=0
    local successful_runs=0
    local failed_runs=0
    local total_items=0
    
    # Count scraping runs from logs
    if [ -d "$LOG_DIR" ]; then
        for log_file in $(find "$LOG_DIR" -name "*.log" -mtime -7 -type f); do
            if grep -q "Starting scraper" "$log_file" 2>/dev/null; then
                ((total_runs++))
                
                if grep -q "SUCCESS: Completed" "$log_file" 2>/dev/null; then
                    ((successful_runs++))
                else
                    ((failed_runs++))
                fi
            fi
        done
    fi
    
    # Count output files
    local output_files=$(find "$OUTPUT_DIR" -name "*.json" -o -name "*.csv" -mtime -7 2>/dev/null | wc -l)
    
    # Sample files to count items
    for file in $(find "$OUTPUT_DIR" -name "*.json" -mtime -7 2>/dev/null | head -20); do
        if [ -f "$file" ]; then
            local items=$(grep -c '"url"' "$file" 2>/dev/null || echo 0)
            total_items=$((total_items + items))
        fi
    done
    
    cat >> "$REPORT_FILE" << EOF
## Scraping Activity

- **Total Scraping Runs**: $total_runs
- **Successful Runs**: $successful_runs ($(( total_runs > 0 ? successful_runs * 100 / total_runs : 0 ))%)
- **Failed Runs**: $failed_runs
- **Output Files Generated**: $output_files
- **Total Items Scraped**: ~$total_items

EOF
}

analyze_performance_metrics() {
    log "Analyzing performance metrics..."
    
    local response_times=()
    local error_counts=()
    local daily_volumes=()
    
    # Collect performance data from logs
    for day in $(seq 0 6); do
        local date=$(date -d "$WEEK_START +$day days" +%Y%m%d)
        local day_errors=0
        local day_volume=0
        
        # Count errors for the day
        if [ -d "$LOG_DIR" ]; then
            day_errors=$(grep -c "ERROR" "$LOG_DIR"/*"$date"*.log 2>/dev/null || echo 0)
        fi
        error_counts+=("$day_errors")
        
        # Count output volume for the day
        day_volume=$(find "$OUTPUT_DIR" -name "*$date*.json" -o -name "*$date*.csv" 2>/dev/null | wc -l)
        daily_volumes+=("$day_volume")
    done
    
    # Calculate statistics
    IFS=',' read -r error_sum error_avg error_min error_max <<< "$(calculate_stats "${error_counts[@]}")"
    IFS=',' read -r volume_sum volume_avg volume_min volume_max <<< "$(calculate_stats "${daily_volumes[@]}")"
    
    cat >> "$REPORT_FILE" << EOF
## Performance Metrics

### Error Analysis
- **Total Errors**: $error_sum
- **Average Daily Errors**: $error_avg
- **Peak Error Day**: $error_max errors
- **Best Day**: $error_min errors

### Volume Analysis
- **Total Files**: $volume_sum
- **Average Daily Output**: $volume_avg files
- **Peak Output Day**: $volume_max files
- **Lowest Output Day**: $volume_min files

EOF
}

analyze_data_quality() {
    log "Analyzing data quality..."
    
    local empty_files=0
    local incomplete_records=0
    local total_size=0
    
    # Check for data quality issues
    empty_files=$(find "$OUTPUT_DIR" -name "*.json" -o -name "*.csv" -size 0 -mtime -7 2>/dev/null | wc -l)
    
    # Calculate total data size
    if [ -d "$OUTPUT_DIR" ]; then
        total_size=$(find "$OUTPUT_DIR" -mtime -7 -type f -exec du -c {} + 2>/dev/null | tail -1 | cut -f1)
        total_size_human=$(du -h -c $(find "$OUTPUT_DIR" -mtime -7 -type f 2>/dev/null) 2>/dev/null | tail -1 | cut -f1)
    fi
    
    cat >> "$REPORT_FILE" << EOF
## Data Quality

- **Empty Files Detected**: $empty_files
- **Total Data Volume**: ${total_size_human:-0}
- **Average File Size**: $(( output_files > 0 ? total_size / output_files : 0 )) KB

EOF
}

analyze_system_health() {
    log "Analyzing system health..."
    
    local disk_usage="N/A"
    local log_size="N/A"
    local oldest_output="N/A"
    
    # Check disk usage
    if [ -d "$OUTPUT_DIR" ]; then
        disk_usage=$(df -h "$OUTPUT_DIR" | tail -1 | awk '{print $5}')
    fi
    
    # Check log directory size
    if [ -d "$LOG_DIR" ]; then
        log_size=$(du -sh "$LOG_DIR" 2>/dev/null | cut -f1)
    fi
    
    # Find oldest output file
    oldest_output=$(find "$OUTPUT_DIR" -type f -printf '%T+ %p\n' 2>/dev/null | sort | head -1 | cut -d' ' -f1 || echo "N/A")
    
    cat >> "$REPORT_FILE" << EOF
## System Health

- **Disk Usage**: $disk_usage
- **Log Directory Size**: $log_size
- **Oldest Output File**: $oldest_output
- **Active Scrapers**: $(pgrep -f "datascrapexter run" | wc -l)

EOF
}

generate_recommendations() {
    log "Generating recommendations..."
    
    cat >> "$REPORT_FILE" << EOF
## Recommendations

Based on this week's analysis:

EOF
    
    # Check for high error rates
    if [ "$error_sum" -gt 50 ]; then
        cat >> "$REPORT_FILE" << EOF
1. **High Error Rate Detected**: Consider reviewing error logs and adjusting scraper configurations.
EOF
    fi
    
    # Check for low success rate
    local success_rate=$(( total_runs > 0 ? successful_runs * 100 / total_runs : 100 ))
    if [ "$success_rate" -lt 80 ]; then
        cat >> "$REPORT_FILE" << EOF
2. **Low Success Rate**: Only $success_rate% of runs succeeded. Review failed configurations.
EOF
    fi
    
    # Check for disk usage
    if [[ "$disk_usage" =~ ^([0-9]+)% ]]; then
        if [ "${BASH_REMATCH[1]}" -gt 80 ]; then
            cat >> "$REPORT_FILE" << EOF
3. **High Disk Usage**: Currently at $disk_usage. Consider archiving old data.
EOF
        fi
    fi
    
    # Check for empty files
    if [ "$empty_files" -gt 5 ]; then
        cat >> "$REPORT_FILE" << EOF
4. **Empty Files Found**: $empty_files empty files detected. Check selectors and site structure.
EOF
    fi
    
    echo "" >> "$REPORT_FILE"
}

generate_summary_json() {
    log "Generating JSON summary..."
    
    cat > "$SUMMARY_JSON" << EOF
{
  "report_date": "$(date -Iseconds)",
  "period": {
    "start": "$WEEK_START",
    "end": "$WEEK_END"
  },
  "scraping": {
    "total_runs": $total_runs,
    "successful_runs": $successful_runs,
    "failed_runs": $failed_runs,
    "success_rate": $(( total_runs > 0 ? successful_runs * 100 / total_runs : 0 ))
  },
  "output": {
    "files_generated": $output_files,
    "total_items": $total_items,
    "empty_files": $empty_files,
    "total_size_kb": $total_size
  },
  "errors": {
    "total": $error_sum,
    "daily_average": $error_avg,
    "peak_day": $error_max
  },
  "system": {
    "disk_usage": "$disk_usage",
    "active_scrapers": $(pgrep -f "datascrapexter run" | wc -l)
  }
}
EOF
}

send_report() {
    if [ -n "$REPORT_EMAIL" ] && command -v mail &> /dev/null; then
        log "Sending report to $REPORT_EMAIL..."
        
        # Convert markdown to plain text for email
        if command -v pandoc &> /dev/null; then
            pandoc -f markdown -t plain "$REPORT_FILE" | mail -s "$REPORT_SUBJECT" "$REPORT_EMAIL"
        else
            cat "$REPORT_FILE" | mail -s "$REPORT_SUBJECT" "$REPORT_EMAIL"
        fi
    fi
}

# Main execution
main() {
    log "======================================"
    log "Generating Weekly Report"
    log "Period: $WEEK_START to $WEEK_END"
    log "======================================"
    
    # Initialize report
    cat > "$REPORT_FILE" << EOF
# DataScrapexter Weekly Report

**Report Period**: $WEEK_START to $WEEK_END  
**Generated**: $(date '+%Y-%m-%d %H:%M:%S')

---

EOF
    
    # Run analyses
    analyze_scraping_activity
    analyze_performance_metrics
    analyze_data_quality
    analyze_system_health
    generate_recommendations
    
    # Add footer
    cat >> "$REPORT_FILE" << EOF
---

*This report was automatically generated by DataScrapexter monitoring system.*
EOF
    
    # Generate additional outputs
    generate_summary_json
    
    # Display summary
    echo -e "\n${GREEN}Report generated successfully!${NC}"
    echo "Report file: $REPORT_FILE"
    echo "Summary JSON: $SUMMARY_JSON"
    
    # Send report if configured
    send_report
    
    log "Weekly report generation completed"
}

# Run main function
main "$@"
