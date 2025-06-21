#!/bin/bash
# Health monitoring script for DataScrapexter
# Monitors scraper health, performance, and sends alerts

set -e

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
OUTPUT_DIR="$PROJECT_ROOT/outputs"
LOG_DIR="$PROJECT_ROOT/logs"
METRICS_DIR="$PROJECT_ROOT/metrics"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
DATE=$(date +%Y-%m-%d)

# Create directories
mkdir -p "$METRICS_DIR"

# Monitoring thresholds
ERROR_THRESHOLD=10
WARNING_THRESHOLD=5
MIN_OUTPUT_FILES=1
MAX_RESPONSE_TIME=30
DISK_USAGE_WARNING=80
DISK_USAGE_CRITICAL=90
MEMORY_WARNING=80
CPU_WARNING=80

# Alert settings
ALERT_EMAIL="${MONITOR_ALERT_EMAIL:-}"
ALERT_WEBHOOK="${MONITOR_ALERT_WEBHOOK:-}"
HEALTH_CHECK_URL="${MONITOR_HEALTH_URL:-}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Health status
HEALTH_STATUS="healthy"
HEALTH_ISSUES=()

# Functions
log_metric() {
    local metric_name="$1"
    local metric_value="$2"
    local metric_file="$METRICS_DIR/${metric_name}_$(date +%Y%m%d).txt"
    echo "$(date +%s) $metric_value" >> "$metric_file"
}

check_recent_outputs() {
    echo "Checking recent outputs..."
    
    # Count files created in last hour
    local recent_files=$(find "$OUTPUT_DIR" -name "*.json" -o -name "*.csv" -mmin -60 2>/dev/null | wc -l)
    log_metric "recent_output_files" "$recent_files"
    
    if [ "$recent_files" -lt "$MIN_OUTPUT_FILES" ]; then
        HEALTH_STATUS="degraded"
        HEALTH_ISSUES+=("No recent output files (last hour)")
        echo -e "${YELLOW}Warning: Only $recent_files output files in last hour${NC}"
    else
        echo -e "${GREEN}✓ Found $recent_files recent output files${NC}"
    fi
    
    # Check for empty files
    local empty_files=$(find "$OUTPUT_DIR" -name "*.json" -o -name "*.csv" -size 0 -mtime -1 2>/dev/null | wc -l)
    if [ "$empty_files" -gt 0 ]; then
        HEALTH_STATUS="degraded"
        HEALTH_ISSUES+=("Found $empty_files empty output files")
        echo -e "${YELLOW}Warning: Found $empty_files empty output files${NC}"
    fi
}

check_error_rates() {
    echo "Checking error rates..."
    
    if [ ! -d "$LOG_DIR" ]; then
        echo -e "${YELLOW}Warning: Log directory not found${NC}"
        return
    fi
    
    # Count errors in recent logs
    local error_count=0
    local warning_count=0
    
    for log_file in $(find "$LOG_DIR" -name "*.log" -mtime -1 2>/dev/null); do
        if [ -f "$log_file" ]; then
            local file_errors=$(grep -c "ERROR" "$log_file" 2>/dev/null || echo 0)
            local file_warnings=$(grep -c "WARNING" "$log_file" 2>/dev/null || echo 0)
            error_count=$((error_count + file_errors))
            warning_count=$((warning_count + file_warnings))
        fi
    done
    
    log_metric "error_count" "$error_count"
    log_metric "warning_count" "$warning_count"
    
    if [ "$error_count" -gt "$ERROR_THRESHOLD" ]; then
        HEALTH_STATUS="unhealthy"
        HEALTH_ISSUES+=("High error count: $error_count errors in last 24h")
        echo -e "${RED}Critical: $error_count errors found${NC}"
    elif [ "$warning_count" -gt "$WARNING_THRESHOLD" ]; then
        HEALTH_STATUS="degraded"
        HEALTH_ISSUES+=("Elevated warnings: $warning_count warnings in last 24h")
        echo -e "${YELLOW}Warning: $warning_count warnings found${NC}"
    else
        echo -e "${GREEN}✓ Error rates normal (E:$error_count W:$warning_count)${NC}"
    fi
}

check_disk_usage() {
    echo "Checking disk usage..."
    
    # Check output directory usage
    if [ -d "$OUTPUT_DIR" ]; then
        local disk_usage=$(df -h "$OUTPUT_DIR" | tail -1 | awk '{print $5}' | sed 's/%//')
        log_metric "disk_usage_percent" "$disk_usage"
        
        if [ "$disk_usage" -gt "$DISK_USAGE_CRITICAL" ]; then
            HEALTH_STATUS="unhealthy"
            HEALTH_ISSUES+=("Critical disk usage: ${disk_usage}%")
            echo -e "${RED}Critical: Disk usage at ${disk_usage}%${NC}"
        elif [ "$disk_usage" -gt "$DISK_USAGE_WARNING" ]; then
            HEALTH_STATUS="degraded"
            HEALTH_ISSUES+=("High disk usage: ${disk_usage}%")
            echo -e "${YELLOW}Warning: Disk usage at ${disk_usage}%${NC}"
        else
            echo -e "${GREEN}✓ Disk usage normal: ${disk_usage}%${NC}"
        fi
        
        # Check total output size
        local output_size=$(du -sh "$OUTPUT_DIR" 2>/dev/null | cut -f1)
        echo "  Output directory size: $output_size"
    fi
}

check_process_health() {
    echo "Checking DataScrapexter processes..."
    
    # Check if any scrapers are running
    local running_scrapers=$(pgrep -f "datascrapexter run" | wc -l)
    log_metric "running_scrapers" "$running_scrapers"
    
    if [ "$running_scrapers" -gt 0 ]; then
        echo -e "${GREEN}✓ $running_scrapers scraper(s) currently running${NC}"
        
        # Check resource usage
        for pid in $(pgrep -f "datascrapexter run"); do
            if [ -n "$pid" ]; then
                # Get CPU and memory usage
                local stats=$(ps -p "$pid" -o pid,pcpu,pmem,etime,comm --no-headers 2>/dev/null || true)
                if [ -n "$stats" ]; then
                    local cpu=$(echo "$stats" | awk '{print $2}')
                    local mem=$(echo "$stats" | awk '{print $3}')
                    local elapsed=$(echo "$stats" | awk '{print $4}')
                    
                    echo "  PID $pid: CPU ${cpu}%, MEM ${mem}%, Runtime $elapsed"
                    
                    # Check for high resource usage
                    if (( $(echo "$cpu > $CPU_WARNING" | bc -l) )); then
                        HEALTH_STATUS="degraded"
                        HEALTH_ISSUES+=("High CPU usage: ${cpu}% for PID $pid")
                    fi
                    
                    if (( $(echo "$mem > $MEMORY_WARNING" | bc -l) )); then
                        HEALTH_STATUS="degraded"
                        HEALTH_ISSUES+=("High memory usage: ${mem}% for PID $pid")
                    fi
                fi
            fi
        done
    else
        echo "  No scrapers currently running"
    fi
}

check_response_times() {
    echo "Checking scraper performance..."
    
    # Analyze recent log files for response times
    local total_time=0
    local request_count=0
    
    for log_file in $(find "$LOG_DIR" -name "*scrape*.log" -mtime -1 2>/dev/null | head -5); do
        if [ -f "$log_file" ]; then
            # Extract response times (adjust regex based on log format)
            while read -r response_time; do
                if [ -n "$response_time" ]; then
                    total_time=$(echo "$total_time + $response_time" | bc)
                    request_count=$((request_count + 1))
                fi
            done < <(grep -oP "response_time=\K[\d.]+" "$log_file" 2>/dev/null || true)
        fi
    done
    
    if [ "$request_count" -gt 0 ]; then
        local avg_response_time=$(echo "scale=2; $total_time / $request_count" | bc)
        log_metric "avg_response_time" "$avg_response_time"
        
        if (( $(echo "$avg_response_time > $MAX_RESPONSE_TIME" | bc -l) )); then
            HEALTH_STATUS="degraded"
            HEALTH_ISSUES+=("Slow response times: ${avg_response_time}s average")
            echo -e "${YELLOW}Warning: Average response time ${avg_response_time}s${NC}"
        else
            echo -e "${GREEN}✓ Average response time: ${avg_response_time}s${NC}"
        fi
    fi
}

check_data_quality() {
    echo "Checking data quality..."
    
    # Sample recent output files
    local sample_files=$(find "$OUTPUT_DIR" -name "*.json" -mtime -1 2>/dev/null | head -5)
    local total_records=0
    local files_checked=0
    
    for file in $sample_files; do
        if [ -f "$file" ] && [ -s "$file" ]; then
            # Count records (adjust based on JSON structure)
            local records=$(grep -c '"url"' "$file" 2>/dev/null || echo 0)
            total_records=$((total_records + records))
            files_checked=$((files_checked + 1))
            
            # Check for common data issues
            if [ "$records" -eq 0 ]; then
                HEALTH_STATUS="degraded"
                HEALTH_ISSUES+=("Empty data file: $(basename "$file")")
            fi
        fi
    done
    
    if [ "$files_checked" -gt 0 ]; then
        local avg_records=$((total_records / files_checked))
        log_metric "avg_records_per_file" "$avg_records"
        echo "  Average records per file: $avg_records"
    fi
}

generate_health_report() {
    local report_file="$METRICS_DIR/health_report_$TIMESTAMP.json"
    
    # Create JSON health report
    cat > "$report_file" << EOF
{
  "timestamp": "$(date -Iseconds)",
  "status": "$HEALTH_STATUS",
  "issues": [$(printf '"%s",' "${HEALTH_ISSUES[@]}" | sed 's/,$//')],
  "metrics": {
    "recent_outputs": $(find "$OUTPUT_DIR" -mmin -60 2>/dev/null | wc -l),
    "error_count": $(grep -c "ERROR" "$LOG_DIR"/*.log 2>/dev/null || echo 0),
    "disk_usage": "$(df -h "$OUTPUT_DIR" | tail -1 | awk '{print $5}')",
    "running_scrapers": $(pgrep -f "datascrapexter run" | wc -l)
  }
}
EOF
    
    echo -e "\nHealth report saved to: $report_file"
}

send_alerts() {
    if [ "$HEALTH_STATUS" = "healthy" ]; then
        return
    fi
    
    local alert_message="DataScrapexter Health Alert - Status: $HEALTH_STATUS\n\nIssues:\n"
    for issue in "${HEALTH_ISSUES[@]}"; do
        alert_message+="- $issue\n"
    done
    
    # Send email alert
    if [ -n "$ALERT_EMAIL" ] && command -v mail &> /dev/null; then
        echo -e "$alert_message" | mail -s "DataScrapexter Health Alert: $HEALTH_STATUS" "$ALERT_EMAIL"
    fi
    
    # Send webhook alert
    if [ -n "$ALERT_WEBHOOK" ]; then
        local color="#ffaa00"
        [ "$HEALTH_STATUS" = "unhealthy" ] && color="#ff0000"
        
        curl -X POST "$ALERT_WEBHOOK" \
            -H 'Content-Type: application/json' \
            -d "{
                \"text\":\"DataScrapexter Health Alert\",
                \"attachments\":[{
                    \"color\":\"$color\",
                    \"title\":\"Status: $HEALTH_STATUS\",
                    \"text\":\"$alert_message\"
                }]
            }" 2>/dev/null || true
    fi
}

# Main monitoring flow
main() {
    echo "======================================"
    echo "DataScrapexter Health Monitor"
    echo "Time: $(date)"
    echo "======================================"
    echo ""
    
    # Run health checks
    check_recent_outputs
    check_error_rates
    check_disk_usage
    check_process_health
    check_response_times
    check_data_quality
    
    # Generate report
    echo ""
    echo "======================================"
    echo "Overall Health Status: $HEALTH_STATUS"
    
    if [ ${#HEALTH_ISSUES[@]} -gt 0 ]; then
        echo "Issues Found:"
        for issue in "${HEALTH_ISSUES[@]}"; do
            echo "  - $issue"
        done
    fi
    echo "======================================"
    
    generate_health_report
    
    # Send alerts if needed
    if [ "$HEALTH_STATUS" != "healthy" ]; then
        send_alerts
    fi
    
    # Update external health check endpoint if configured
    if [ -n "$HEALTH_CHECK_URL" ]; then
        curl -X POST "$HEALTH_CHECK_URL" \
            -H 'Content-Type: application/json' \
            -d "{\"status\":\"$HEALTH_STATUS\",\"timestamp\":\"$(date -Iseconds)\"}" \
            2>/dev/null || true
    fi
    
    # Exit with appropriate code
    case "$HEALTH_STATUS" in
        "healthy") exit 0 ;;
        "degraded") exit 1 ;;
        "unhealthy") exit 2 ;;
    esac
}

# Run main function
main "$@"
