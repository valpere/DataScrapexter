#!/bin/bash
# Utility script to extract URLs from DataScrapexter output files
# Supports CSV and JSON formats

set -e

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Input parameters
INPUT_FILE="${1:-}"
OUTPUT_FILE="${2:-urls.txt}"
COLUMN="${3:-2}"  # Default to column 2 for CSV
FIELD="${4:-url}" # Default field name for JSON

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Functions
show_usage() {
    cat << EOF
Usage: $0 <input_file> [output_file] [column/field]

Extract URLs from DataScrapexter output files.

Arguments:
  input_file   - CSV or JSON file containing URLs
  output_file  - Output file for extracted URLs (default: urls.txt)
  column       - For CSV: column number (default: 2)
  field        - For JSON: field name (default: url)

Examples:
  # Extract from CSV (column 2)
  $0 products.csv product-urls.txt 2
  
  # Extract from JSON (field 'product_url')
  $0 products.json urls.txt product_url
  
  # Extract and filter
  $0 products.csv | grep "/product/" > product-urls.txt

EOF
}

detect_file_type() {
    local file="$1"
    local extension="${file##*.}"
    
    case "$extension" in
        csv|CSV)
            echo "csv"
            ;;
        json|JSON)
            echo "json"
            ;;
        *)
            # Try to detect by content
            if head -1 "$file" | grep -q "^[{[]"; then
                echo "json"
            else
                echo "csv"
            fi
            ;;
    esac
}

extract_from_csv() {
    local file="$1"
    local column="$2"
    local output="$3"
    
    echo -e "${GREEN}Extracting URLs from CSV column $column...${NC}"
    
    # Skip header and extract column
    if [ "$output" = "-" ]; then
        tail -n +2 "$file" | cut -d',' -f"$column" | sed 's/"//g' | grep -E "^https?://" || true
    else
        tail -n +2 "$file" | cut -d',' -f"$column" | sed 's/"//g' | grep -E "^https?://" > "$output" || true
    fi
    
    local count=$(grep -c "^https?://" "$output" 2>/dev/null || echo 0)
    echo -e "${GREEN}Extracted $count URLs${NC}"
}

extract_from_json() {
    local file="$1"
    local field="$2"
    local output="$3"
    
    echo -e "${GREEN}Extracting URLs from JSON field '$field'...${NC}"
    
    # Check if jq is available
    if ! command -v jq &> /dev/null; then
        echo -e "${YELLOW}Warning: jq not found. Falling back to grep method.${NC}"
        extract_from_json_grep "$file" "$field" "$output"
        return
    fi
    
    # Use jq to extract URLs
    local jq_filter=""
    
    # Detect JSON structure
    if jq -e 'if type == "array" then true else false end' "$file" &>/dev/null; then
        # Array of objects
        jq_filter=".[] | .${field} // .data.${field} // empty"
    else
        # Single object or nested structure
        jq_filter=".${field} // .data.${field} // .. | .${field}? // empty"
    fi
    
    if [ "$output" = "-" ]; then
        jq -r "$jq_filter" "$file" 2>/dev/null | grep -E "^https?://" || true
    else
        jq -r "$jq_filter" "$file" 2>/dev/null | grep -E "^https?://" > "$output" || true
    fi
    
    local count=$([ -f "$output" ] && grep -c "^https?://" "$output" || echo 0)
    echo -e "${GREEN}Extracted $count URLs${NC}"
}

extract_from_json_grep() {
    local file="$1"
    local field="$2"
    local output="$3"
    
    echo -e "${YELLOW}Using grep method for JSON extraction...${NC}"
    
    # Simple grep extraction
    local pattern="\"${field}\"\s*:\s*\"([^\"]+)\""
    
    if [ "$output" = "-" ]; then
        grep -oP "$pattern" "$file" | grep -oP 'https?://[^"]+' || true
    else
        grep -oP "$pattern" "$file" | grep -oP 'https?://[^"]+' > "$output" || true
    fi
    
    local count=$([ -f "$output" ] && grep -c "^https?://" "$output" || echo 0)
    echo -e "${GREEN}Extracted $count URLs${NC}"
}

validate_urls() {
    local file="$1"
    local invalid=0
    local total=0
    
    echo -e "\n${GREEN}Validating URLs...${NC}"
    
    while IFS= read -r url; do
        ((total++))
        if ! [[ "$url" =~ ^https?://[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)* ]]; then
            echo -e "${YELLOW}Invalid URL: $url${NC}"
            ((invalid++))
        fi
    done < "$file"
    
    echo "Total URLs: $total"
    echo "Valid URLs: $((total - invalid))"
    [ "$invalid" -gt 0 ] && echo -e "${YELLOW}Invalid URLs: $invalid${NC}"
}

deduplicate_urls() {
    local file="$1"
    local original_count=$(wc -l < "$file")
    local temp_file="${file}.tmp"
    
    echo -e "\n${GREEN}Removing duplicate URLs...${NC}"
    
    sort -u "$file" > "$temp_file"
    mv "$temp_file" "$file"
    
    local new_count=$(wc -l < "$file")
    local removed=$((original_count - new_count))
    
    echo "Original URLs: $original_count"
    echo "Unique URLs: $new_count"
    [ "$removed" -gt 0 ] && echo -e "${YELLOW}Duplicates removed: $removed${NC}"
}

filter_urls() {
    local file="$1"
    local pattern="${URL_FILTER:-}"
    
    if [ -n "$pattern" ]; then
        echo -e "\n${GREEN}Filtering URLs with pattern: $pattern${NC}"
        local temp_file="${file}.tmp"
        grep "$pattern" "$file" > "$temp_file" || true
        mv "$temp_file" "$file"
        
        local count=$(wc -l < "$file")
        echo "URLs after filtering: $count"
    fi
}

# Main execution
main() {
    # Check arguments
    if [ -z "$INPUT_FILE" ] || [ "$INPUT_FILE" = "--help" ] || [ "$INPUT_FILE" = "-h" ]; then
        show_usage
        exit 0
    fi
    
    # Validate input file
    if [ ! -f "$INPUT_FILE" ]; then
        echo -e "${RED}Error: Input file not found: $INPUT_FILE${NC}"
        exit 1
    fi
    
    # Detect file type
    FILE_TYPE=$(detect_file_type "$INPUT_FILE")
    echo -e "${GREEN}Detected file type: $FILE_TYPE${NC}"
    
    # Extract URLs based on file type
    case "$FILE_TYPE" in
        csv)
            extract_from_csv "$INPUT_FILE" "$COLUMN" "$OUTPUT_FILE"
            ;;
        json)
            extract_from_json "$INPUT_FILE" "$FIELD" "$OUTPUT_FILE"
            ;;
        *)
            echo -e "${RED}Error: Unsupported file type${NC}"
            exit 1
            ;;
    esac
    
    # Additional processing if output to file
    if [ "$OUTPUT_FILE" != "-" ] && [ -f "$OUTPUT_FILE" ]; then
        # Remove duplicates
        deduplicate_urls "$OUTPUT_FILE"
        
        # Apply filter if set
        filter_urls "$OUTPUT_FILE"
        
        # Validate URLs
        validate_urls "$OUTPUT_FILE"
        
        echo -e "\n${GREEN}URLs saved to: $OUTPUT_FILE${NC}"
    fi
}

# Run main function
main "$@"
