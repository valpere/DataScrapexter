// internal/output/excel.go
package output

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

// Logger interface for structured logging integration
// Applications can provide their own logger implementation
type Logger interface {
	Warnf(format string, args ...interface{})
	Infof(format string, args ...interface{})
}

// DefaultLogger implements Logger using standard log package
type DefaultLogger struct{}

func (l *DefaultLogger) Warnf(format string, args ...interface{}) {
	log.Printf("[WARN] "+format, args...)
}

func (l *DefaultLogger) Infof(format string, args ...interface{}) {
	log.Printf("[INFO] "+format, args...)
}

// Excel-specific default limits (can be overridden via ExcelConfig)
const (
	// DefaultExcelMaxCellLength is the default maximum characters in a single Excel cell
	DefaultExcelMaxCellLength = 32767
	// DefaultExcelMaxSheetRows is the default maximum rows per sheet in Excel
	DefaultExcelMaxSheetRows = 1048576
)

// ExcelWriter implements the Writer interface for Excel output
type ExcelWriter struct {
	file      *excelize.File
	config    ExcelConfig
	sheetName string
	headers   []string
	row       int
	records   []map[string]interface{}
}

// ExcelConfig configuration for Excel output
type ExcelConfig struct {
	FilePath         string         `json:"file"`
	SheetName        string         `json:"sheet_name"`
	IncludeHeaders   bool           `json:"include_headers"`
	AutoFilter       bool           `json:"auto_filter"`
	FreezePane       bool           `json:"freeze_pane"`
	BufferSize       int            `json:"buffer_size"`
	ColumnWidths     map[string]int `json:"column_widths"`
	HeaderStyle      ExcelCellStyle `json:"header_style"`
	DataStyle        ExcelCellStyle `json:"data_style"`
	DateFormat       string         `json:"date_format"`
	NumberFormat     string         `json:"number_format"`
	MaxSheetRows     int            `json:"max_sheet_rows"`
	MaxCellLength    int            `json:"max_cell_length"`
	MaxArrayElements int            `json:"max_array_elements"` // Maximum array elements to prevent memory issues
	CreateIndex      bool           `json:"create_index"`
	Compression      bool           `json:"compression"`
	Logger           Logger         `json:"-"` // Optional logger interface for structured logging
}

// ExcelCellStyle defines cell styling options
type ExcelCellStyle struct {
	Font      ExcelFont      `json:"font"`
	Fill      ExcelFill      `json:"fill"`
	Border    ExcelBorder    `json:"border"`
	Alignment ExcelAlignment `json:"alignment"`
	NumFmt    string         `json:"number_format"`
}

// ExcelFont defines font styling
type ExcelFont struct {
	Bold      bool   `json:"bold"`
	Italic    bool   `json:"italic"`
	Size      int    `json:"size"`
	Color     string `json:"color"`
	Family    string `json:"family"`
	Underline string `json:"underline"`
}

// ExcelFill defines cell fill/background
type ExcelFill struct {
	Type    string `json:"type"`
	Pattern int    `json:"pattern"`
	Color   string `json:"color"`
}

// ExcelBorder defines cell borders
type ExcelBorder struct {
	Type  string `json:"type"`
	Color string `json:"color"`
	Style int    `json:"style"`
}

// ExcelAlignment defines cell alignment
type ExcelAlignment struct {
	Horizontal string `json:"horizontal"`
	Vertical   string `json:"vertical"`
	WrapText   bool   `json:"wrap_text"`
}

// NewExcelWriter creates a new Excel writer
func NewExcelWriter(config ExcelConfig) (*ExcelWriter, error) {
	if config.FilePath == "" {
		return nil, fmt.Errorf("Excel file path is required")
	}

	// Set defaults
	if config.SheetName == "" {
		config.SheetName = "Sheet1"
	}
	if config.BufferSize == 0 {
		config.BufferSize = 1000
	}
	if config.DateFormat == "" {
		config.DateFormat = "yyyy-mm-dd hh:mm:ss"
	}
	if config.NumberFormat == "" {
		config.NumberFormat = "0.00"
	}
	if config.MaxSheetRows == 0 {
		config.MaxSheetRows = DefaultExcelMaxSheetRows
	}
	if config.MaxCellLength == 0 {
		config.MaxCellLength = DefaultExcelMaxCellLength
	}
	if config.Logger == nil {
		config.Logger = &DefaultLogger{}
	}

	file := excelize.NewFile()

	// Create or rename the default sheet
	defaultSheet := file.GetSheetName(0)
	if defaultSheet != config.SheetName {
		file.SetSheetName(defaultSheet, config.SheetName)
	}

	writer := &ExcelWriter{
		file:      file,
		config:    config,
		sheetName: config.SheetName,
		row:       1,
		records:   make([]map[string]interface{}, 0, config.BufferSize),
	}

	return writer, nil
}

// Write writes data to Excel file
func (w *ExcelWriter) Write(data []map[string]interface{}) error {
	for _, record := range data {
		if err := w.WriteRecord(record); err != nil {
			return err
		}
	}
	return nil
}

// WriteRecord writes a single record to Excel
func (w *ExcelWriter) WriteRecord(record map[string]interface{}) error {
	if len(w.records) >= w.config.BufferSize {
		if err := w.flush(); err != nil {
			return err
		}
	}

	w.records = append(w.records, record)
	return nil
}

// WriteContext writes data to Excel file with context
func (w *ExcelWriter) WriteContext(ctx context.Context, data interface{}) error {
	switch v := data.(type) {
	case []map[string]interface{}:
		return w.Write(v)
	case map[string]interface{}:
		return w.WriteRecord(v)
	case []interface{}:
		for _, item := range v {
			if record, ok := item.(map[string]interface{}); ok {
				if err := w.WriteRecord(record); err != nil {
					return err
				}
			} else {
				return fmt.Errorf("unsupported data type in slice: %T", item)
			}
		}
		return nil
	default:
		return fmt.Errorf("unsupported data type: %T", data)
	}
}

// Flush writes buffered records to file
func (w *ExcelWriter) Flush() error {
	return w.flush()
}

// Close closes the Excel writer and saves the file
func (w *ExcelWriter) Close() error {
	// Flush any remaining records
	if err := w.flush(); err != nil {
		return err
	}

	// Apply final formatting
	if err := w.applyFinalFormatting(); err != nil {
		return err
	}

	// Save the file
	return w.file.SaveAs(w.config.FilePath)
}

// GetType returns the output type
func (w *ExcelWriter) GetType() string {
	return "excel"
}

// flush writes buffered records to the worksheet
func (w *ExcelWriter) flush() error {
	if len(w.records) == 0 {
		return nil
	}

	// Extract headers if not already done
	if w.headers == nil {
		w.extractHeaders()
		if w.config.IncludeHeaders {
			if err := w.writeHeaders(); err != nil {
				return err
			}
		}
	}

	// Write records
	for _, record := range w.records {
		if err := w.writeRecord(record); err != nil {
			return err
		}
	}

	w.records = w.records[:0] // Clear the slice but keep capacity
	return nil
}

// extractHeaders extracts all unique column headers from records
func (w *ExcelWriter) extractHeaders() {
	headerSet := make(map[string]bool)

	for _, record := range w.records {
		for key := range record {
			headerSet[key] = true
		}
	}

	// Convert to sorted slice for consistent order
	w.headers = make([]string, 0, len(headerSet))
	for header := range headerSet {
		w.headers = append(w.headers, header)
	}
	sort.Strings(w.headers)

	// Add index column if requested
	if w.config.CreateIndex {
		w.headers = append([]string{"Index"}, w.headers...)
	}
}

// writeHeaders writes the header row
func (w *ExcelWriter) writeHeaders() error {
	for col, header := range w.headers {
		cell := columnName(col+1) + strconv.Itoa(w.row)
		if err := w.file.SetCellValue(w.sheetName, cell, header); err != nil {
			return err
		}

		// Apply header style
		if err := w.applyHeaderStyle(cell); err != nil {
			return err
		}
	}

	w.row++
	return nil
}

// writeRecord writes a single record to the worksheet
func (w *ExcelWriter) writeRecord(record map[string]interface{}) error {
	// Check if we need to create a new sheet (row limit reached)
	if w.row > w.config.MaxSheetRows {
		return w.createNewSheet()
	}

	for col, header := range w.headers {
		cell := columnName(col+1) + strconv.Itoa(w.row)

		var value interface{}
		if header == "Index" && w.config.CreateIndex {
			value = w.row - 1 // Subtract 1 for header row
			if !w.config.IncludeHeaders {
				value = w.row
			}
		} else {
			value = record[header]
		}

		// Process the value
		processedValue := w.processValue(value)

		if err := w.file.SetCellValue(w.sheetName, cell, processedValue); err != nil {
			return err
		}

		// Apply data style
		if err := w.applyDataStyle(cell, value); err != nil {
			return err
		}
	}

	w.row++
	return nil
}

// processValue processes a value for Excel output
func (w *ExcelWriter) processValue(value interface{}) interface{} {
	if value == nil {
		return ""
	}

	switch v := value.(type) {
	case time.Time:
		return v
	case []interface{}:
		// Convert arrays to comma-separated strings with configurable limits
		maxArrayElements := w.getMaxArrayElements()
		if len(v) > maxArrayElements {
			// Log truncation using configured logger - sanitize for security
			w.config.Logger.Warnf("Excel: Truncating array from %d to %d elements for memory efficiency",
				len(v), maxArrayElements)
			v = v[:maxArrayElements]
		}

		// Use efficient string builder for large arrays
		if len(v) > 100 {
			var builder strings.Builder
			builder.Grow(len(v) * 10) // Pre-allocate space estimation
			for i, item := range v {
				if i > 0 {
					builder.WriteString(", ")
				}
				builder.WriteString(fmt.Sprintf("%v", item))
			}
			return builder.String()
		} else {
			// Use slice approach for smaller arrays
			var parts []string
			for _, item := range v {
				parts = append(parts, fmt.Sprintf("%v", item))
			}
			return strings.Join(parts, ", ")
		}
	case map[string]interface{}:
		// Convert objects to JSON-like strings
		var parts []string
		for key, val := range v {
			parts = append(parts, fmt.Sprintf("%s: %v", key, val))
		}
		return "{" + strings.Join(parts, ", ") + "}"
	case string:
		// Handle very long strings using configurable limit
		maxLength := w.config.MaxCellLength
		if maxLength <= 0 {
			maxLength = DefaultExcelMaxCellLength
		}
		if len(v) > maxLength {
			// Log data truncation using configured logger - avoid exposing file paths for security
			w.config.Logger.Warnf("Excel: Truncating cell data from %d to %d characters (output_id: %p)",
				len(v), maxLength, w)
			return v[:maxLength]
		}
		return v
	default:
		return value
	}
}

// applyHeaderStyle applies styling to header cells
func (w *ExcelWriter) applyHeaderStyle(cell string) error {
	if w.config.HeaderStyle.Font.Size == 0 {
		// Set default header style
		style, err := w.file.NewStyle(&excelize.Style{
			Font: &excelize.Font{
				Bold: true,
				Size: 12,
			},
			Fill: excelize.Fill{
				Type:    "pattern",
				Color:   []string{"#E0E0E0"},
				Pattern: 1,
			},
			Border: []excelize.Border{
				{Type: "left", Color: "000000", Style: 1},
				{Type: "top", Color: "000000", Style: 1},
				{Type: "bottom", Color: "000000", Style: 1},
				{Type: "right", Color: "000000", Style: 1},
			},
		})
		if err != nil {
			return err
		}
		return w.file.SetCellStyle(w.sheetName, cell, cell, style)
	}

	// Apply custom header style
	return w.applyCustomStyle(cell, w.config.HeaderStyle)
}

// applyDataStyle applies styling to data cells
func (w *ExcelWriter) applyDataStyle(cell string, value interface{}) error {
	// Apply different styles based on data type
	switch value.(type) {
	case time.Time:
		return w.applyDateStyle(cell)
	case float64, float32, int, int64, int32:
		return w.applyNumberStyle(cell)
	default:
		if w.config.DataStyle.Font.Size > 0 {
			return w.applyCustomStyle(cell, w.config.DataStyle)
		}
	}
	return nil
}

// applyDateStyle applies date formatting
func (w *ExcelWriter) applyDateStyle(cell string) error {
	style, err := w.file.NewStyle(&excelize.Style{
		NumFmt: 22, // Date format
	})
	if err != nil {
		return err
	}
	return w.file.SetCellStyle(w.sheetName, cell, cell, style)
}

// applyNumberStyle applies number formatting
func (w *ExcelWriter) applyNumberStyle(cell string) error {
	style, err := w.file.NewStyle(&excelize.Style{
		NumFmt: 2, // Number format with 2 decimal places
	})
	if err != nil {
		return err
	}
	return w.file.SetCellStyle(w.sheetName, cell, cell, style)
}

// applyCustomStyle applies custom cell styling
func (w *ExcelWriter) applyCustomStyle(cell string, cellStyle ExcelCellStyle) error {
	style := &excelize.Style{}

	// Font
	if cellStyle.Font.Size > 0 || cellStyle.Font.Bold || cellStyle.Font.Color != "" {
		style.Font = &excelize.Font{
			Bold:   cellStyle.Font.Bold,
			Italic: cellStyle.Font.Italic,
			Size:   float64(cellStyle.Font.Size),
		}
		if cellStyle.Font.Color != "" {
			style.Font.Color = cellStyle.Font.Color
		}
		if cellStyle.Font.Family != "" {
			style.Font.Family = cellStyle.Font.Family
		}
	}

	// Fill
	if cellStyle.Fill.Color != "" {
		style.Fill = excelize.Fill{
			Type:    cellStyle.Fill.Type,
			Color:   []string{cellStyle.Fill.Color},
			Pattern: cellStyle.Fill.Pattern,
		}
		if style.Fill.Type == "" {
			style.Fill.Type = "pattern"
		}
		if style.Fill.Pattern == 0 {
			style.Fill.Pattern = 1
		}
	}

	// Alignment
	if cellStyle.Alignment.Horizontal != "" || cellStyle.Alignment.Vertical != "" {
		style.Alignment = &excelize.Alignment{
			Horizontal: cellStyle.Alignment.Horizontal,
			Vertical:   cellStyle.Alignment.Vertical,
			WrapText:   cellStyle.Alignment.WrapText,
		}
	}

	// Number format
	if cellStyle.NumFmt != "" {
		// This would need to be converted to Excel format ID
		style.NumFmt = 1 // Default to general format
	}

	styleID, err := w.file.NewStyle(style)
	if err != nil {
		return err
	}

	return w.file.SetCellStyle(w.sheetName, cell, cell, styleID)
}

// getMaxArrayElements returns the maximum number of array elements to process
// This prevents memory issues with very large arrays in Excel cells
func (w *ExcelWriter) getMaxArrayElements() int {
	if w.config.MaxArrayElements > 0 {
		return w.config.MaxArrayElements
	}
	// Default limit to prevent memory issues
	return 1000
}

// applyFinalFormatting applies final formatting to the worksheet
func (w *ExcelWriter) applyFinalFormatting() error {
	// Set column widths
	for col, header := range w.headers {
		colName := columnName(col + 1)
		width := 15.0 // Default width

		if w.config.ColumnWidths != nil {
			if customWidth, exists := w.config.ColumnWidths[header]; exists {
				width = float64(customWidth)
			}
		}

		if err := w.file.SetColWidth(w.sheetName, colName, colName, width); err != nil {
			return err
		}
	}

	// Apply auto filter
	if w.config.AutoFilter && len(w.headers) > 0 {
		lastCol := columnName(len(w.headers))
		lastRow := w.row - 1
		if w.config.IncludeHeaders {
			if err := w.file.AutoFilter(w.sheetName, "A1:"+lastCol+strconv.Itoa(lastRow), nil); err != nil {
				return err
			}
		}
	}

	// Freeze pane
	if w.config.FreezePane && w.config.IncludeHeaders {
		if err := w.file.SetPanes(w.sheetName, &excelize.Panes{
			Freeze: true,
			Split:  false,
			XSplit: 1,
			YSplit: 1,
		}); err != nil {
			return err
		}
	}

	return nil
}

// createNewSheet creates a new sheet when row limit is reached
func (w *ExcelWriter) createNewSheet() error {
	// Generate new sheet name
	newSheetName := fmt.Sprintf("%s_%d", w.config.SheetName, len(w.file.GetSheetList()))

	// Create new sheet
	index, err := w.file.NewSheet(newSheetName)
	if err != nil {
		return err
	}

	// Switch to new sheet
	w.file.SetActiveSheet(index)
	w.sheetName = newSheetName
	w.row = 1

	// Write headers if configured
	if w.config.IncludeHeaders {
		return w.writeHeaders()
	}

	return nil
}

// columnName converts a column number to Excel column name (A, B, C, ..., AA, AB, etc.)
func columnName(col int) string {
	name := ""
	for col > 0 {
		col--
		name = string(rune('A'+col%26)) + name
		col /= 26
	}
	return name
}

// ExcelWorkbook represents an Excel workbook with multiple sheets
type ExcelWorkbook struct {
	file    *excelize.File
	config  ExcelConfig
	writers map[string]*ExcelWriter
}

// NewExcelWorkbook creates a new Excel workbook
func NewExcelWorkbook(config ExcelConfig) (*ExcelWorkbook, error) {
	file := excelize.NewFile()

	return &ExcelWorkbook{
		file:    file,
		config:  config,
		writers: make(map[string]*ExcelWriter),
	}, nil
}

// GetOrCreateWriter gets or creates a writer for a specific sheet
func (wb *ExcelWorkbook) GetOrCreateWriter(sheetName string) (*ExcelWriter, error) {
	if writer, exists := wb.writers[sheetName]; exists {
		return writer, nil
	}

	// Create new sheet
	index, err := wb.file.NewSheet(sheetName)
	if err != nil {
		return nil, err
	}

	wb.file.SetActiveSheet(index)

	// Create writer for this sheet
	config := wb.config
	config.SheetName = sheetName

	writer := &ExcelWriter{
		file:      wb.file,
		config:    config,
		sheetName: sheetName,
		row:       1,
		records:   make([]map[string]interface{}, 0, config.BufferSize),
	}

	wb.writers[sheetName] = writer
	return writer, nil
}

// Save saves the workbook to file
func (wb *ExcelWorkbook) Save() error {
	// Flush all writers
	for _, writer := range wb.writers {
		if err := writer.Flush(); err != nil {
			return err
		}
		if err := writer.applyFinalFormatting(); err != nil {
			return err
		}
	}

	return wb.file.SaveAs(wb.config.FilePath)
}

// Close closes the workbook
func (wb *ExcelWorkbook) Close() error {
	return wb.Save()
}

// ValidateExcelConfig validates Excel configuration
func ValidateExcelConfig(config ExcelConfig) error {
	if config.FilePath == "" {
		return fmt.Errorf("file path is required")
	}

	if !strings.HasSuffix(strings.ToLower(config.FilePath), ".xlsx") {
		return fmt.Errorf("file path must end with .xlsx")
	}

	if config.BufferSize < 0 {
		return fmt.Errorf("buffer size must be non-negative")
	}

	if config.MaxSheetRows < 1 || config.MaxSheetRows > DefaultExcelMaxSheetRows {
		return fmt.Errorf("max sheet rows must be between 1 and 1048576")
	}

	return nil
}
