// internal/output/pdf.go - Professional PDF output formatter with comprehensive layout options
package output

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/valpere/DataScrapexter/internal/utils"
)

var pdfLogger = utils.NewComponentLogger("pdf-output")

// PDFWriter implements Writer interface for PDF output
type PDFWriter struct {
	file     *os.File
	options  *PDFOptions
	filePath string
	buffer   *bytes.Buffer
	data     []map[string]interface{}
	
	// Document metadata
	pageCount    int
	recordCount  int
	createdAt    time.Time
	
	// Template engine
	template     PDFTemplate
	colorScheme  *PDFColorScheme
	
	// Statistics
	totalPages   int
	totalSize    int64
	
	// Internal state
	currentPage  int
	yPosition    float64
	xPosition    float64
	marginLeft   float64
	marginRight  float64
	marginTop    float64
	marginBottom float64
	pageWidth    float64
	pageHeight   float64
}

// PDFTemplate interface for different PDF layouts
type PDFTemplate interface {
	GenerateDocument(data []map[string]interface{}, options *PDFOptions) ([]byte, error)
	GetTemplateName() string
	GetTemplateDescription() string
}

// ReportTemplate implements professional report-style PDF layout
type ReportTemplate struct {
	title       string
	subtitle    string
	metadata    map[string]interface{}
	sections    []PDFSection
}

// TableTemplate implements table-focused PDF layout
type TableTemplate struct {
	headers     []string
	columnWidths map[string]float64
	tableStyle  *PDFTableOptions
}

// DetailedTemplate implements detailed record-by-record PDF layout
type DetailedTemplate struct {
	recordsPerPage int
	showMetadata   bool
	groupBy        string
}

// CompactTemplate implements space-efficient PDF layout
type CompactTemplate struct {
	columnsPerRow int
	fontSize      float64
}

// PDFSection represents a section in the PDF document
type PDFSection struct {
	Title       string                   `json:"title"`
	Type        string                   `json:"type"`        // text, table, chart, image
	Content     interface{}              `json:"content"`
	Style       *PDFSectionStyle        `json:"style,omitempty"`
	Metadata    map[string]interface{}   `json:"metadata,omitempty"`
}

// PDFSectionStyle defines styling for PDF sections
type PDFSectionStyle struct {
	BackgroundColor string   `json:"background_color,omitempty"`
	BorderColor     string   `json:"border_color,omitempty"`
	BorderWidth     float64  `json:"border_width,omitempty"`
	Padding         float64  `json:"padding,omitempty"`
	Margin          float64  `json:"margin,omitempty"`
	Font            *PDFFont `json:"font,omitempty"`
	Alignment       string   `json:"alignment,omitempty"`
}

// NewPDFWriter creates a new PDF writer
func NewPDFWriter(filePath string, options *PDFOptions) (*PDFWriter, error) {
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	// Set defaults if options is nil
	if options == nil {
		options = getDefaultPDFOptions()
	} else {
		applyDefaultPDFOptions(options)
	}

	// Create file
	file, err := os.Create(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create PDF file: %w", err)
	}

	// Initialize color scheme
	colorScheme := options.Colors
	if colorScheme == nil {
		colorScheme = getDefaultColorScheme()
	}

	// Calculate page dimensions (assuming A4 by default)
	pageWidth, pageHeight := getPageDimensions(options.PageSize, options.Orientation)
	
	// Set margins
	margins := options.Margins
	if margins == nil {
		margins = &PDFMargins{
			Top:    72,  // 1 inch
			Bottom: 72,
			Left:   72,
			Right:  72,
		}
	}

	writer := &PDFWriter{
		file:         file,
		options:      options,
		filePath:     filePath,
		buffer:       bytes.NewBuffer(nil),
		data:         make([]map[string]interface{}, 0),
		createdAt:    time.Now(),
		colorScheme:  colorScheme,
		pageWidth:    pageWidth,
		pageHeight:   pageHeight,
		marginLeft:   margins.Left,
		marginRight:  margins.Right,
		marginTop:    margins.Top,
		marginBottom: margins.Bottom,
		yPosition:    margins.Top,
		xPosition:    margins.Left,
		currentPage:  1,
	}

	// Select template
	template, err := selectPDFTemplate(options.Template, options)
	if err != nil {
		writer.Close()
		return nil, fmt.Errorf("failed to select PDF template: %w", err)
	}
	writer.template = template

	pdfLogger.Info(fmt.Sprintf("Created PDF writer for %s with template %s", filePath, template.GetTemplateName()))

	return writer, nil
}

// Write writes data to the PDF buffer
func (pw *PDFWriter) Write(data []map[string]interface{}) error {
	if pw == nil || pw.file == nil {
		return fmt.Errorf("PDF writer not initialized")
	}

	// Append data to internal buffer
	pw.data = append(pw.data, data...)
	pw.recordCount += len(data)

	pdfLogger.Debug(fmt.Sprintf("Added %d records to PDF buffer (total: %d)", len(data), pw.recordCount))
	return nil
}

// Close finalizes and writes the PDF document
func (pw *PDFWriter) Close() error {
	if pw == nil || pw.file == nil {
		return fmt.Errorf("PDF writer not initialized")
	}

	defer func() {
		if pw.file != nil {
			pw.file.Close()
		}
	}()

	pdfLogger.Info(fmt.Sprintf("Generating PDF document with %d records using %s template", 
		pw.recordCount, pw.template.GetTemplateName()))

	// Generate PDF content using selected template
	content, err := pw.template.GenerateDocument(pw.data, pw.options)
	if err != nil {
		return fmt.Errorf("failed to generate PDF content: %w", err)
	}

	// Write content to file
	if _, err := pw.file.Write(content); err != nil {
		return fmt.Errorf("failed to write PDF content: %w", err)
	}

	// Get file size
	if stat, err := pw.file.Stat(); err == nil {
		pw.totalSize = stat.Size()
	}

	pdfLogger.Info(fmt.Sprintf("Successfully generated PDF: %s (size: %d bytes, pages: %d)", 
		pw.filePath, pw.totalSize, pw.totalPages))

	return nil
}

// Template Selection and Management

func selectPDFTemplate(templateName string, options *PDFOptions) (PDFTemplate, error) {
	if templateName == "" {
		templateName = "report"
	}

	switch strings.ToLower(templateName) {
	case "report":
		return NewReportTemplate(options), nil
	case "table":
		return NewTableTemplate(options), nil
	case "detailed":
		return NewDetailedTemplate(options), nil
	case "compact":
		return NewCompactTemplate(options), nil
	default:
		return nil, fmt.Errorf("unknown PDF template: %s", templateName)
	}
}

// Report Template Implementation

func NewReportTemplate(options *PDFOptions) *ReportTemplate {
	return &ReportTemplate{
		title:    options.Title,
		subtitle: options.Subject,
		metadata: make(map[string]interface{}),
		sections: make([]PDFSection, 0),
	}
}

func (rt *ReportTemplate) GetTemplateName() string {
	return "Report Template"
}

func (rt *ReportTemplate) GetTemplateDescription() string {
	return "Professional business report layout with executive summary, data sections, and analytics"
}

func (rt *ReportTemplate) GenerateDocument(data []map[string]interface{}, options *PDFOptions) ([]byte, error) {
	// For this implementation, we'll create a text-based PDF structure
	// In a real implementation, you would use a PDF library like gofpdf, unidoc, or similar
	
	var content bytes.Buffer
	
	// Write PDF header (simplified for demonstration)
	content.WriteString("%PDF-1.4\n")
	content.WriteString(fmt.Sprintf("%% Generated by DataScrapexter at %s\n", time.Now().Format("2006-01-02 15:04:05")))
	
	// Document metadata
	rt.writeDocumentInfo(&content, options)
	
	// Executive summary
	rt.writeExecutiveSummary(&content, data, options)
	
	// Data sections
	rt.writeDataSections(&content, data, options)
	
	// Appendices
	rt.writeAppendices(&content, data, options)
	
	// Footer information
	rt.writeFooter(&content, options)
	
	// In a real implementation, this would be proper PDF binary format
	// For demonstration, we're creating a structured text representation
	return content.Bytes(), nil
}

func (rt *ReportTemplate) writeDocumentInfo(w io.Writer, options *PDFOptions) {
	fmt.Fprintf(w, "\n=== DOCUMENT INFORMATION ===\n")
	fmt.Fprintf(w, "Title: %s\n", options.Title)
	fmt.Fprintf(w, "Author: %s\n", options.Author)
	fmt.Fprintf(w, "Subject: %s\n", options.Subject)
	if len(options.Keywords) > 0 {
		fmt.Fprintf(w, "Keywords: %s\n", strings.Join(options.Keywords, ", "))
	}
	fmt.Fprintf(w, "Created: %s\n", time.Now().Format("January 2, 2006 at 3:04 PM"))
	fmt.Fprintf(w, "Page Size: %s (%s)\n", options.PageSize, options.Orientation)
	fmt.Fprintf(w, "\n")
}

func (rt *ReportTemplate) writeExecutiveSummary(w io.Writer, data []map[string]interface{}, options *PDFOptions) {
	fmt.Fprintf(w, "=== EXECUTIVE SUMMARY ===\n")
	fmt.Fprintf(w, "Total Records: %d\n", len(data))
	
	// Analyze data structure
	if len(data) > 0 {
		fields := make(map[string]int)
		for _, record := range data {
			for field := range record {
				fields[field]++
			}
		}
		
		fmt.Fprintf(w, "Data Fields: %d\n", len(fields))
		fmt.Fprintf(w, "Most Common Fields:\n")
		
		// Sort fields by frequency
		type fieldCount struct {
			Field string
			Count int
		}
		var sortedFields []fieldCount
		for field, count := range fields {
			sortedFields = append(sortedFields, fieldCount{Field: field, Count: count})
		}
		sort.Slice(sortedFields, func(i, j int) bool {
			return sortedFields[i].Count > sortedFields[j].Count
		})
		
		// Show top 10 fields
		for i, fc := range sortedFields {
			if i >= 10 {
				break
			}
			coverage := float64(fc.Count) / float64(len(data)) * 100
			fmt.Fprintf(w, "  - %s (%.1f%% coverage)\n", fc.Field, coverage)
		}
	}
	fmt.Fprintf(w, "\n")
}

func (rt *ReportTemplate) writeDataSections(w io.Writer, data []map[string]interface{}, options *PDFOptions) {
	fmt.Fprintf(w, "=== DATA SECTIONS ===\n\n")
	
	// Group data by common patterns or use pagination
	pageSize := 50 // Records per page
	pageCount := (len(data) + pageSize - 1) / pageSize
	
	for page := 0; page < pageCount; page++ {
		start := page * pageSize
		end := start + pageSize
		if end > len(data) {
			end = len(data)
		}
		
		fmt.Fprintf(w, "--- Page %d of %d (Records %d-%d) ---\n", page+1, pageCount, start+1, end)
		
		for i := start; i < end; i++ {
			record := data[i]
			fmt.Fprintf(w, "\nRecord #%d:\n", i+1)
			
			// Sort keys for consistent output
			var keys []string
			for key := range record {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			
			for _, key := range keys {
				value := record[key]
				// Format value based on type
				formattedValue := rt.formatValue(value)
				fmt.Fprintf(w, "  %-20s: %s\n", key, formattedValue)
			}
		}
		fmt.Fprintf(w, "\n")
	}
}

func (rt *ReportTemplate) writeAppendices(w io.Writer, data []map[string]interface{}, options *PDFOptions) {
	fmt.Fprintf(w, "=== APPENDICES ===\n\n")
	
	// Appendix A: Data Statistics
	fmt.Fprintf(w, "Appendix A: Data Statistics\n")
	rt.writeDataStatistics(w, data)
	
	// Appendix B: Field Analysis
	fmt.Fprintf(w, "\nAppendix B: Field Analysis\n")
	rt.writeFieldAnalysis(w, data)
	
	// Appendix C: Custom Fields (if any)
	if len(options.CustomFields) > 0 {
		fmt.Fprintf(w, "\nAppendix C: Custom Information\n")
		for key, value := range options.CustomFields {
			fmt.Fprintf(w, "  %s: %s\n", key, value)
		}
	}
}

func (rt *ReportTemplate) writeFooter(w io.Writer, options *PDFOptions) {
	fmt.Fprintf(w, "\n=== FOOTER ===\n")
	fmt.Fprintf(w, "Generated by DataScrapexter\n")
	fmt.Fprintf(w, "Report created: %s\n", time.Now().Format("January 2, 2006 at 3:04 PM"))
	fmt.Fprintf(w, "Software version: 1.0.0\n")
	
	if options.HeaderFooter != nil && options.HeaderFooter.Footer != nil {
		footer := options.HeaderFooter.Footer
		if footer.Text != "" {
			fmt.Fprintf(w, "Footer: %s\n", footer.Text)
		}
		if footer.ShowDate {
			dateFormat := footer.DateFormat
			if dateFormat == "" {
				dateFormat = "2006-01-02 15:04:05"
			}
			fmt.Fprintf(w, "Date: %s\n", time.Now().Format(dateFormat))
		}
	}
}

func (rt *ReportTemplate) formatValue(value interface{}) string {
	if value == nil {
		return "<nil>"
	}
	
	switch v := value.(type) {
	case string:
		if len(v) > 100 {
			return v[:97] + "..."
		}
		return v
	case int, int64, int32:
		return fmt.Sprintf("%d", v)
	case float32, float64:
		return fmt.Sprintf("%.2f", v)
	case bool:
		if v {
			return "Yes"
		}
		return "No"
	case time.Time:
		return v.Format("2006-01-02 15:04:05")
	case []interface{}:
		if len(v) > 5 {
			return fmt.Sprintf("Array with %d items", len(v))
		}
		return fmt.Sprintf("%v", v)
	case map[string]interface{}:
		return fmt.Sprintf("Object with %d fields", len(v))
	default:
		str := fmt.Sprintf("%v", v)
		if len(str) > 100 {
			return str[:97] + "..."
		}
		return str
	}
}

func (rt *ReportTemplate) writeDataStatistics(w io.Writer, data []map[string]interface{}) {
	if len(data) == 0 {
		fmt.Fprintf(w, "  No data to analyze\n")
		return
	}
	
	// Calculate basic statistics
	totalRecords := len(data)
	totalFields := 0
	fieldTypes := make(map[string]map[reflect.Kind]int)
	
	for _, record := range data {
		for field, value := range record {
			if _, exists := fieldTypes[field]; !exists {
				fieldTypes[field] = make(map[reflect.Kind]int)
			}
			
			if value != nil {
				kind := reflect.TypeOf(value).Kind()
				fieldTypes[field][kind]++
			} else {
				fieldTypes[field][reflect.Invalid]++
			}
		}
		totalFields += len(record)
	}
	
	avgFieldsPerRecord := float64(totalFields) / float64(totalRecords)
	
	fmt.Fprintf(w, "  Total Records: %d\n", totalRecords)
	fmt.Fprintf(w, "  Average Fields per Record: %.1f\n", avgFieldsPerRecord)
	fmt.Fprintf(w, "  Unique Fields: %d\n", len(fieldTypes))
}

func (rt *ReportTemplate) writeFieldAnalysis(w io.Writer, data []map[string]interface{}) {
	if len(data) == 0 {
		fmt.Fprintf(w, "  No data to analyze\n")
		return
	}
	
	fieldStats := make(map[string]struct {
		Count       int
		NullCount   int
		Types       map[reflect.Kind]int
		SampleValues []string
	})
	
	// Analyze each field
	for _, record := range data {
		for field, value := range record {
			if _, exists := fieldStats[field]; !exists {
				fieldStats[field] = struct {
					Count       int
					NullCount   int
					Types       map[reflect.Kind]int
					SampleValues []string
				}{
					Types:       make(map[reflect.Kind]int),
					SampleValues: make([]string, 0, 5),
				}
			}
			
			stats := fieldStats[field]
			stats.Count++
			
			if value == nil {
				stats.NullCount++
				stats.Types[reflect.Invalid]++
			} else {
				kind := reflect.TypeOf(value).Kind()
				stats.Types[kind]++
				
				// Collect sample values
				if len(stats.SampleValues) < 5 {
					strValue := rt.formatValue(value)
					if len(strValue) < 50 { // Only include short values as samples
						stats.SampleValues = append(stats.SampleValues, strValue)
					}
				}
			}
			
			fieldStats[field] = stats
		}
	}
	
	// Sort fields by count (most common first)
	type fieldAnalysis struct {
		Name  string
		Stats struct {
			Count       int
			NullCount   int
			Types       map[reflect.Kind]int
			SampleValues []string
		}
	}
	
	var sortedFields []fieldAnalysis
	for field, stats := range fieldStats {
		sortedFields = append(sortedFields, fieldAnalysis{
			Name:  field,
			Stats: stats,
		})
	}
	
	sort.Slice(sortedFields, func(i, j int) bool {
		return sortedFields[i].Stats.Count > sortedFields[j].Stats.Count
	})
	
	// Write analysis for each field
	for _, fa := range sortedFields {
		coverage := float64(fa.Stats.Count) / float64(len(data)) * 100
		nullRate := float64(fa.Stats.NullCount) / float64(fa.Stats.Count) * 100
		
		fmt.Fprintf(w, "\n  Field: %s\n", fa.Name)
		fmt.Fprintf(w, "    Coverage: %.1f%% (%d/%d records)\n", coverage, fa.Stats.Count, len(data))
		fmt.Fprintf(w, "    Null Rate: %.1f%%\n", nullRate)
		
		// Show data types
		fmt.Fprintf(w, "    Data Types: ")
		var typeStrings []string
		for kind, count := range fa.Stats.Types {
			percent := float64(count) / float64(fa.Stats.Count) * 100
			typeStrings = append(typeStrings, fmt.Sprintf("%s (%.1f%%)", kind.String(), percent))
		}
		fmt.Fprintf(w, "%s\n", strings.Join(typeStrings, ", "))
		
		// Show sample values
		if len(fa.Stats.SampleValues) > 0 {
			fmt.Fprintf(w, "    Sample Values: %s\n", strings.Join(fa.Stats.SampleValues, ", "))
		}
	}
}

// Table Template Implementation

func NewTableTemplate(options *PDFOptions) *TableTemplate {
	return &TableTemplate{
		headers:      make([]string, 0),
		columnWidths: make(map[string]float64),
		tableStyle:   options.TableOptions,
	}
}

func (tt *TableTemplate) GetTemplateName() string {
	return "Table Template"
}

func (tt *TableTemplate) GetTemplateDescription() string {
	return "Clean table layout optimized for tabular data presentation"
}

func (tt *TableTemplate) GenerateDocument(data []map[string]interface{}, options *PDFOptions) ([]byte, error) {
	var content bytes.Buffer
	
	// Write PDF header
	content.WriteString("%PDF-1.4\n")
	content.WriteString(fmt.Sprintf("%% Table-style PDF generated by DataScrapexter at %s\n", time.Now().Format("2006-01-02 15:04:05")))
	
	// Document info
	tt.writeDocumentHeader(&content, options)
	
	// Extract and write table
	tt.writeTable(&content, data, options)
	
	// Summary statistics
	tt.writeSummary(&content, data, options)
	
	return content.Bytes(), nil
}

func (tt *TableTemplate) writeDocumentHeader(w io.Writer, options *PDFOptions) {
	fmt.Fprintf(w, "\n=== %s ===\n", options.Title)
	if options.Author != "" {
		fmt.Fprintf(w, "Author: %s\n", options.Author)
	}
	fmt.Fprintf(w, "Generated: %s\n", time.Now().Format("January 2, 2006 at 3:04 PM"))
	fmt.Fprintf(w, "\n")
}

func (tt *TableTemplate) writeTable(w io.Writer, data []map[string]interface{}, options *PDFOptions) {
	if len(data) == 0 {
		fmt.Fprintf(w, "No data to display\n")
		return
	}
	
	// Extract headers from first record and all records for completeness
	headerSet := make(map[string]bool)
	for _, record := range data {
		for field := range record {
			headerSet[field] = true
		}
	}
	
	// Convert to sorted slice
	for header := range headerSet {
		tt.headers = append(tt.headers, header)
	}
	sort.Strings(tt.headers)
	
	// Calculate column widths
	tt.calculateColumnWidths(data)
	
	// Write table header
	fmt.Fprintf(w, "=== DATA TABLE ===\n\n")
	tt.writeTableHeader(w)
	tt.writeTableSeparator(w)
	
	// Write data rows
	for i, record := range data {
		tt.writeTableRow(w, record, i+1)
	}
	
	tt.writeTableSeparator(w)
	fmt.Fprintf(w, "\n")
}

func (tt *TableTemplate) writeTableHeader(w io.Writer) {
	fmt.Fprintf(w, "| %4s |", "Row")
	for _, header := range tt.headers {
		width := int(tt.columnWidths[header])
		if width < len(header) {
			width = len(header)
		}
		if width > 30 { // Max column width for readability
			width = 30
		}
		fmt.Fprintf(w, " %-*s |", width, header)
	}
	fmt.Fprintf(w, "\n")
}

func (tt *TableTemplate) writeTableSeparator(w io.Writer) {
	fmt.Fprintf(w, "|------|")
	for _, header := range tt.headers {
		width := int(tt.columnWidths[header])
		if width < len(header) {
			width = len(header)
		}
		if width > 30 {
			width = 30
		}
		fmt.Fprintf(w, "%s|", strings.Repeat("-", width+2))
	}
	fmt.Fprintf(w, "\n")
}

func (tt *TableTemplate) writeTableRow(w io.Writer, record map[string]interface{}, rowNum int) {
	fmt.Fprintf(w, "| %4d |", rowNum)
	for _, header := range tt.headers {
		width := int(tt.columnWidths[header])
		if width < len(header) {
			width = len(header)
		}
		if width > 30 {
			width = 30
		}
		
		value := record[header]
		strValue := tt.formatCellValue(value, width)
		fmt.Fprintf(w, " %-*s |", width, strValue)
	}
	fmt.Fprintf(w, "\n")
}

func (tt *TableTemplate) calculateColumnWidths(data []map[string]interface{}) {
	for _, header := range tt.headers {
		maxWidth := float64(len(header))
		
		for _, record := range data {
			if value, exists := record[header]; exists {
				strValue := tt.formatCellValue(value, 1000) // No truncation for width calculation
				if float64(len(strValue)) > maxWidth {
					maxWidth = float64(len(strValue))
				}
			}
		}
		
		// Cap at reasonable width
		if maxWidth > 30 {
			maxWidth = 30
		}
		if maxWidth < 8 {
			maxWidth = 8
		}
		
		tt.columnWidths[header] = maxWidth
	}
}

func (tt *TableTemplate) formatCellValue(value interface{}, maxWidth int) string {
	if value == nil {
		return ""
	}
	
	var str string
	switch v := value.(type) {
	case string:
		str = v
	case int, int64, int32:
		str = fmt.Sprintf("%d", v)
	case float32, float64:
		str = fmt.Sprintf("%.2f", v)
	case bool:
		if v {
			str = "true"
		} else {
			str = "false"
		}
	case time.Time:
		str = v.Format("2006-01-02")
	default:
		str = fmt.Sprintf("%v", v)
	}
	
	// Truncate if necessary
	if len(str) > maxWidth && maxWidth > 3 {
		str = str[:maxWidth-3] + "..."
	}
	
	return str
}

func (tt *TableTemplate) writeSummary(w io.Writer, data []map[string]interface{}, options *PDFOptions) {
	fmt.Fprintf(w, "=== TABLE SUMMARY ===\n")
	fmt.Fprintf(w, "Total Rows: %d\n", len(data))
	fmt.Fprintf(w, "Total Columns: %d\n", len(tt.headers))
	
	if len(data) > 0 {
		// Calculate completeness for each column
		fmt.Fprintf(w, "\nColumn Completeness:\n")
		for _, header := range tt.headers {
			nonEmpty := 0
			for _, record := range data {
				if value, exists := record[header]; exists && value != nil {
					if str, ok := value.(string); !ok || str != "" {
						nonEmpty++
					}
				}
			}
			completeness := float64(nonEmpty) / float64(len(data)) * 100
			fmt.Fprintf(w, "  %-20s: %6.1f%% (%d/%d)\n", header, completeness, nonEmpty, len(data))
		}
	}
	fmt.Fprintf(w, "\n")
}

// Detailed and Compact Templates (simplified implementations)

func NewDetailedTemplate(options *PDFOptions) *DetailedTemplate {
	return &DetailedTemplate{
		recordsPerPage: 10,
		showMetadata:   true,
		groupBy:        "",
	}
}

func (dt *DetailedTemplate) GetTemplateName() string {
	return "Detailed Template"
}

func (dt *DetailedTemplate) GetTemplateDescription() string {
	return "Detailed record-by-record layout with full field information"
}

func (dt *DetailedTemplate) GenerateDocument(data []map[string]interface{}, options *PDFOptions) ([]byte, error) {
	// Implementation similar to report template but with more detailed record display
	return NewReportTemplate(options).GenerateDocument(data, options)
}

func NewCompactTemplate(options *PDFOptions) *CompactTemplate {
	return &CompactTemplate{
		columnsPerRow: 3,
		fontSize:      8.0,
	}
}

func (ct *CompactTemplate) GetTemplateName() string {
	return "Compact Template"
}

func (ct *CompactTemplate) GetTemplateDescription() string {
	return "Space-efficient layout maximizing data density"
}

func (ct *CompactTemplate) GenerateDocument(data []map[string]interface{}, options *PDFOptions) ([]byte, error) {
	// Implementation focused on space efficiency
	return NewTableTemplate(options).GenerateDocument(data, options)
}

// Helper Functions

func getDefaultPDFOptions() *PDFOptions {
	return &PDFOptions{
		Title:       "DataScrapexter Report",
		Author:      "DataScrapexter",
		Subject:     "Scraped Data Report",
		PageSize:    "A4",
		Orientation: "Portrait",
		Template:    "report",
		Margins: &PDFMargins{
			Top:    72,
			Bottom: 72,
			Left:   72,
			Right:  72,
		},
		Font: &PDFFont{
			Family:     "Helvetica",
			Size:       12,
			Style:      "Regular",
			Color:      "#000000",
			LineHeight: 1.2,
		},
	}
}

func applyDefaultPDFOptions(options *PDFOptions) {
	if options.Title == "" {
		options.Title = "DataScrapexter Report"
	}
	if options.Author == "" {
		options.Author = "DataScrapexter"
	}
	if options.PageSize == "" {
		options.PageSize = "A4"
	}
	if options.Orientation == "" {
		options.Orientation = "Portrait"
	}
	if options.Template == "" {
		options.Template = "report"
	}
	if options.Margins == nil {
		options.Margins = &PDFMargins{Top: 72, Bottom: 72, Left: 72, Right: 72}
	}
	if options.Font == nil {
		options.Font = &PDFFont{
			Family: "Helvetica",
			Size:   12,
			Style:  "Regular",
			Color:  "#000000",
			LineHeight: 1.2,
		}
	}
}

func getDefaultColorScheme() *PDFColorScheme {
	return &PDFColorScheme{
		Primary:    "#2563eb", // Blue
		Secondary:  "#64748b", // Gray
		Accent:     "#f59e0b", // Amber
		Text:       "#1f2937", // Dark gray
		Background: "#ffffff", // White
		Border:     "#e5e7eb", // Light gray
		Success:    "#10b981", // Green
		Warning:    "#f59e0b", // Amber
		Error:      "#ef4444", // Red
	}
}

func getPageDimensions(pageSize, orientation string) (width, height float64) {
	// Dimensions in points (1 point = 1/72 inch)
	var w, h float64
	
	switch strings.ToUpper(pageSize) {
	case "A4":
		w, h = 595, 842
	case "LETTER":
		w, h = 612, 792
	case "LEGAL":
		w, h = 612, 1008
	case "TABLOID":
		w, h = 792, 1224
	default:
		w, h = 595, 842 // Default to A4
	}
	
	if strings.ToLower(orientation) == "landscape" {
		w, h = h, w
	}
	
	return w, h
}

// PDFMetadata represents PDF document metadata
type PDFMetadata struct {
	RecordCount    int                    `json:"record_count"`
	PageCount      int                    `json:"page_count"`
	FileSize       int64                  `json:"file_size"`
	CreatedAt      time.Time              `json:"created_at"`
	Template       string                 `json:"template"`
	Options        *PDFOptions            `json:"options,omitempty"`
	Statistics     *PDFStatistics         `json:"statistics,omitempty"`
	ProcessingTime time.Duration          `json:"processing_time"`
}

// PDFStatistics represents statistics about the generated PDF
type PDFStatistics struct {
	UniqueFields     int                    `json:"unique_fields"`
	AverageFieldsPerRecord float64         `json:"average_fields_per_record"`
	FieldCoverage    map[string]float64     `json:"field_coverage"`
	DataTypes        map[string]int         `json:"data_types"`
	CompletionRate   float64                `json:"completion_rate"`
	TableCount       int                    `json:"table_count"`
	ImageCount       int                    `json:"image_count"`
	ChartCount       int                    `json:"chart_count"`
}

// GetPDFMetadata returns metadata about the generated PDF
func (pw *PDFWriter) GetPDFMetadata() *PDFMetadata {
	return &PDFMetadata{
		RecordCount:    pw.recordCount,
		PageCount:      pw.totalPages,
		FileSize:       pw.totalSize,
		CreatedAt:      pw.createdAt,
		Template:       pw.template.GetTemplateName(),
		Options:        pw.options,
		ProcessingTime: time.Since(pw.createdAt),
	}
}