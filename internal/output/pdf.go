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

	"github.com/jung-kurt/gofpdf/v2"
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
	// Determine page size and orientation from options
	pageSize := "A4"
	orientation := "P" // Portrait
	
	if options != nil {
		if options.PageSize != "" {
			pageSize = options.PageSize
		}
		if options.Orientation != "" {
			if strings.ToLower(options.Orientation) == "landscape" {
				orientation = "L"
			} else {
				orientation = "P"
			}
		}
	}
	
	// Create new PDF document using configuration from options
	pdf := gofpdf.New(orientation, "mm", pageSize, "")
	
	// Set document properties
	rt.setDocumentProperties(pdf, options)
	
	// Add first page
	pdf.AddPage()
	
	// Write document header
	rt.writeDocumentHeader(pdf, options)
	
	// Write executive summary
	rt.writeExecutiveSummaryPDF(pdf, data, options)
	
	// Write data sections
	rt.writeDataSectionsPDF(pdf, data, options)
	
	// Write appendices
	rt.writeAppendicesPDF(pdf, data, options)
	
	// Write footer on all pages
	rt.writeFooterPDF(pdf, options)
	
	// Generate PDF bytes
	var buf bytes.Buffer
	err := pdf.Output(&buf)
	if err != nil {
		return nil, fmt.Errorf("failed to generate PDF: %w", err)
	}
	
	return buf.Bytes(), nil
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
	// Determine page size and orientation from options
	pageSize := "A4"
	orientation := "P" // Portrait
	
	if options != nil {
		if options.PageSize != "" {
			pageSize = options.PageSize
		}
		if options.Orientation != "" {
			if strings.ToLower(options.Orientation) == "landscape" {
				orientation = "L"
			} else {
				orientation = "P"
			}
		}
	}
	
	// Create new PDF document using configuration from options
	pdf := gofpdf.New(orientation, "mm", pageSize, "")
	pdf.AddPage()
	
	// Set document properties
	if options != nil {
		if options.Title != "" {
			pdf.SetTitle(options.Title, false)
		}
		if options.Author != "" {
			pdf.SetAuthor(options.Author, false)
		}
		if options.Subject != "" {
			pdf.SetSubject(options.Subject, false)
		}
		if len(options.Keywords) > 0 {
			pdf.SetKeywords(strings.Join(options.Keywords, ", "), false)
		}
	}
	
	// Generate table content using gofpdf
	tt.generatePDFTable(pdf, data, options)
	
	// Generate PDF bytes
	var buf bytes.Buffer
	err := pdf.Output(&buf)
	if err != nil {
		return nil, fmt.Errorf("failed to generate PDF: %w", err)
	}
	
	return buf.Bytes(), nil
}

// generatePDFTable creates a proper PDF table using gofpdf
func (tt *TableTemplate) generatePDFTable(pdf *gofpdf.Fpdf, data []map[string]interface{}, options *PDFOptions) {
	if len(data) == 0 {
		pdf.SetFont("Arial", "", 12)
		pdf.Cell(40, 10, "No data available")
		return
	}
	
	// Extract column names from first record
	var columns []string
	for key := range data[0] {
		columns = append(columns, key)
	}
	
	// Sort columns for consistent output
	sort.Strings(columns)
	
	// Set up fonts and colors
	headerFont := "Arial"
	headerSize := 10.0
	cellFont := "Arial"
	cellSize := 9.0
	
	if options != nil && options.TableOptions != nil {
		if options.TableOptions.HeaderStyle != nil && options.TableOptions.HeaderStyle.Font != nil {
			if options.TableOptions.HeaderStyle.Font.Family != "" {
				headerFont = options.TableOptions.HeaderStyle.Font.Family
			}
			if options.TableOptions.HeaderStyle.Font.Size > 0 {
				headerSize = options.TableOptions.HeaderStyle.Font.Size
			}
		}
		if options.TableOptions.RowStyle != nil && options.TableOptions.RowStyle.Font != nil {
			if options.TableOptions.RowStyle.Font.Family != "" {
				cellFont = options.TableOptions.RowStyle.Font.Family
			}
			if options.TableOptions.RowStyle.Font.Size > 0 {
				cellSize = options.TableOptions.RowStyle.Font.Size
			}
		}
	}
	
	// Calculate column widths
	pageWidth, _ := pdf.GetPageSize()
	margins := 20.0 // Total left + right margins
	availableWidth := pageWidth - margins
	columnWidth := availableWidth / float64(len(columns))
	
	// Add title if specified
	if options != nil && options.Title != "" {
		pdf.SetFont(headerFont, "B", headerSize+2)
		pdf.Cell(0, 10, options.Title)
		pdf.Ln(15)
	}
	
	// Draw table header
	pdf.SetFont(headerFont, "B", headerSize)
	pdf.SetFillColor(200, 200, 200) // Light gray background
	
	for _, col := range columns {
		pdf.CellFormat(columnWidth, 8, col, "1", 0, "C", true, 0, "")
	}
	pdf.Ln(-1)
	
	// Draw table rows
	pdf.SetFont(cellFont, "", cellSize)
	pdf.SetFillColor(240, 240, 240) // Very light gray for alternating rows
	
	for i, record := range data {
		// Alternate row colors
		fill := i%2 == 1
		
		for _, col := range columns {
			value := ""
			if v, exists := record[col]; exists {
				value = fmt.Sprintf("%v", v)
			}
			
			// Truncate long values to fit in cell
			if len(value) > 30 {
				value = value[:27] + "..."
			}
			
			pdf.CellFormat(columnWidth, 6, value, "1", 0, "L", fill, 0, "")
		}
		pdf.Ln(-1)
		
		// Add page break if needed
		if pdf.GetY() > 250 { // Near bottom of page
			pdf.AddPage()
			
			// Re-draw header on new page
			pdf.SetFont(headerFont, "B", headerSize)
			pdf.SetFillColor(200, 200, 200)
			
			for _, col := range columns {
				pdf.CellFormat(columnWidth, 8, col, "1", 0, "C", true, 0, "")
			}
			pdf.Ln(-1)
			pdf.SetFont(cellFont, "", cellSize)
		}
	}
	
	// Add summary information
	pdf.Ln(10)
	pdf.SetFont("Arial", "I", 8)
	pdf.Cell(0, 5, fmt.Sprintf("Generated on: %s | Total records: %d", 
		time.Now().Format("2006-01-02 15:04:05"), len(data)))
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
	// Determine page size and orientation from options
	pageSize := "A4"
	orientation := "P" // Portrait
	
	if options != nil {
		if options.PageSize != "" {
			pageSize = options.PageSize
		}
		if options.Orientation != "" {
			if strings.ToLower(options.Orientation) == "landscape" {
				orientation = "L"
			} else {
				orientation = "P"
			}
		}
	}
	
	// Create new PDF document using configuration from options
	pdf := gofpdf.New(orientation, "mm", pageSize, "")
	pdf.AddPage()
	
	// Set document properties
	if options != nil {
		if options.Title != "" {
			pdf.SetTitle(options.Title, false)
		}
		if options.Author != "" {
			pdf.SetAuthor(options.Author, false)
		}
		if options.Subject != "" {
			pdf.SetSubject(options.Subject, false)
		}
		if len(options.Keywords) > 0 {
			pdf.SetKeywords(strings.Join(options.Keywords, ", "), false)
		}
	}
	
	// Generate compact layout using smaller fonts and multiple columns
	ct.generateCompactLayout(pdf, data, options)
	
	// Generate PDF bytes
	var buf bytes.Buffer
	err := pdf.Output(&buf)
	if err != nil {
		return nil, fmt.Errorf("failed to generate compact PDF: %w", err)
	}
	
	return buf.Bytes(), nil
}

// generateCompactLayout creates space-efficient PDF layout with smaller fonts and multiple columns
func (ct *CompactTemplate) generateCompactLayout(pdf *gofpdf.Fpdf, data []map[string]interface{}, options *PDFOptions) {
	if len(data) == 0 {
		pdf.SetFont("Arial", "", ct.fontSize)
		pdf.Cell(40, 6, "No data available")
		return
	}
	
	// Extract all unique keys from all records for complete field coverage
	fieldSet := make(map[string]bool)
	for _, record := range data {
		for key := range record {
			fieldSet[key] = true
		}
	}
	
	var fields []string
	for field := range fieldSet {
		fields = append(fields, field)
	}
	sort.Strings(fields)
	
	// Set up compact fonts (smaller than table template)
	headerFont := "Arial"
	headerSize := ct.fontSize + 1
	cellFont := "Arial"
	cellSize := ct.fontSize
	
	// Get page dimensions
	pageWidth, _ := pdf.GetPageSize()
	margins := 10.0 // Reduced margins for more space
	availableWidth := pageWidth - margins
	
	// Add compact title
	if options != nil && options.Title != "" {
		pdf.SetFont(headerFont, "B", headerSize+1)
		pdf.Cell(0, 8, options.Title)
		pdf.Ln(12)
	}
	
	// Calculate how many columns per row based on data density
	recordsPerRow := ct.columnsPerRow
	if len(data) > 50 {
		recordsPerRow = 4 // More columns for larger datasets
	}
	
	columnWidth := availableWidth / float64(recordsPerRow)
	
	// Generate compact multi-column layout
	pdf.SetFont(cellFont, "", cellSize)
	
	for i := 0; i < len(data); i += recordsPerRow {
		// Start new row
		startY := pdf.GetY()
		
		// Process records in this row
		for col := 0; col < recordsPerRow && i+col < len(data); col++ {
			record := data[i+col]
			xPos := 10 + float64(col)*columnWidth
			
			// Position for this column
			pdf.SetXY(xPos, startY)
			
			// Draw compact record box
			ct.drawCompactRecord(pdf, record, fields, columnWidth-2, options)
		}
		
		// Move to next row
		pdf.SetY(startY + ct.calculateRecordHeight(fields, columnWidth))
		
		// Check for page break
		if pdf.GetY() > 260 {
			pdf.AddPage()
		}
	}
	
	// Add compact summary
	pdf.Ln(5)
	pdf.SetFont("Arial", "I", ct.fontSize-1)
	pdf.Cell(0, 4, fmt.Sprintf("Generated: %s | Records: %d | Layout: %d columns", 
		time.Now().Format("2006-01-02 15:04"), len(data), recordsPerRow))
}

// drawCompactRecord draws a single record in compact format
func (ct *CompactTemplate) drawCompactRecord(pdf *gofpdf.Fpdf, record map[string]interface{}, fields []string, width float64, options *PDFOptions) {
	startX := pdf.GetX()
	startY := pdf.GetY()
	
	// Draw border
	pdf.SetDrawColor(200, 200, 200)
	pdf.SetLineWidth(0.2)
	
	rowHeight := 3.0
	
	// Draw each field in the record
	for i, field := range fields {
		value := ""
		if v, exists := record[field]; exists {
			value = fmt.Sprintf("%v", v)
		}
		
		// Truncate long values for compact display
		maxLen := int(width / 2) // Approximate character limit based on width
		if len(value) > maxLen {
			value = value[:maxLen-3] + "..."
		}
		
		// Alternate background for readability
		if i%2 == 0 {
			pdf.SetFillColor(248, 248, 248)
		} else {
			pdf.SetFillColor(255, 255, 255)
		}
		
		// Field label (bold, smaller)
		pdf.SetXY(startX, startY+float64(i)*rowHeight)
		pdf.SetFont("Arial", "B", ct.fontSize-1)
		pdf.CellFormat(width*0.4, rowHeight, field+":", "LTB", 0, "L", true, 0, "")
		
		// Field value
		pdf.SetFont("Arial", "", ct.fontSize-1)
		pdf.CellFormat(width*0.6, rowHeight, value, "RTB", 0, "L", true, 0, "")
	}
}

// calculateRecordHeight calculates the height needed for a record
func (ct *CompactTemplate) calculateRecordHeight(fields []string, width float64) float64 {
	rowHeight := 3.0
	return float64(len(fields))*rowHeight + 2.0 // +2 for padding
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

// PDF Helper Methods for gofpdf implementation

// setDocumentProperties sets PDF document metadata and properties
func (rt *ReportTemplate) setDocumentProperties(pdf *gofpdf.Fpdf, options *PDFOptions) {
	pdf.SetCreator("DataScrapexter", true)
	pdf.SetCreationDate(time.Now())
	
	if options != nil {
		if options.Author != "" {
			pdf.SetAuthor(options.Author, true)
		}
		if options.Title != "" {
			pdf.SetTitle(options.Title, true)
			rt.title = options.Title
		}
		if options.Subject != "" {
			pdf.SetSubject(options.Subject, true)
			rt.subtitle = options.Subject
		}
		if len(options.Keywords) > 0 {
			pdf.SetKeywords(strings.Join(options.Keywords, ", "), true)
		}
	}
	
	// Set default title if not provided
	if rt.title == "" {
		rt.title = "DataScrapexter Report"
		pdf.SetTitle(rt.title, true)
	}
	
	// Set default fonts
	pdf.SetFont("Arial", "", 12)
}

// writeDocumentHeader writes the PDF document header
func (rt *ReportTemplate) writeDocumentHeader(pdf *gofpdf.Fpdf, options *PDFOptions) {
	// Add header with title
	pdf.SetFont("Arial", "B", 16)
	pdf.CellFormat(0, 10, rt.title, "0", 1, "C", false, 0, "")
	
	if rt.subtitle != "" {
		pdf.SetFont("Arial", "", 12)
		pdf.CellFormat(0, 8, rt.subtitle, "0", 1, "C", false, 0, "")
	}
	
	// Add generation date
	pdf.SetFont("Arial", "", 10)
	pdf.CellFormat(0, 6, "Generated: "+time.Now().Format("January 2, 2006 at 3:04 PM"), "0", 1, "C", false, 0, "")
	
	// Add spacing
	pdf.Ln(5)
}

// writeExecutiveSummaryPDF writes executive summary section to PDF
func (rt *ReportTemplate) writeExecutiveSummaryPDF(pdf *gofpdf.Fpdf, data []map[string]interface{}, options *PDFOptions) {
	pdf.SetFont("Arial", "B", 14)
	pdf.CellFormat(0, 8, "Executive Summary", "0", 1, "L", false, 0, "")
	pdf.Ln(2)
	
	pdf.SetFont("Arial", "", 10)
	pdf.CellFormat(0, 6, fmt.Sprintf("Total Records: %d", len(data)), "0", 1, "L", false, 0, "")
	
	// Analyze data structure
	if len(data) > 0 {
		fields := make(map[string]int)
		for _, record := range data {
			for field := range record {
				fields[field]++
			}
		}
		
		pdf.CellFormat(0, 6, fmt.Sprintf("Data Fields: %d", len(fields)), "0", 1, "L", false, 0, "")
		pdf.Ln(3)
	}
}

// writeDataSectionsPDF writes data sections to PDF
func (rt *ReportTemplate) writeDataSectionsPDF(pdf *gofpdf.Fpdf, data []map[string]interface{}, options *PDFOptions) {
	pdf.SetFont("Arial", "B", 14)
	pdf.CellFormat(0, 8, "Data Analysis", "0", 1, "L", false, 0, "")
	pdf.Ln(2)
	
	if len(data) == 0 {
		pdf.SetFont("Arial", "", 10)
		pdf.CellFormat(0, 6, "No data available", "0", 1, "L", false, 0, "")
		return
	}
	
	// Display first few records as examples
	pdf.SetFont("Arial", "B", 10)
	pdf.CellFormat(0, 6, "Sample Records:", "0", 1, "L", false, 0, "")
	pdf.Ln(1)
	
	maxRecords := 5
	if len(data) < maxRecords {
		maxRecords = len(data)
	}
	
	pdf.SetFont("Arial", "", 9)
	for i := 0; i < maxRecords; i++ {
		record := data[i]
		pdf.CellFormat(0, 5, fmt.Sprintf("Record %d:", i+1), "0", 1, "L", false, 0, "")
		
		for key, value := range record {
			valueStr := rt.formatValueForPDF(value)
			if len(valueStr) > 80 {
				valueStr = valueStr[:77] + "..."
			}
			pdf.CellFormat(0, 4, fmt.Sprintf("  %s: %s", key, valueStr), "0", 1, "L", false, 0, "")
		}
		pdf.Ln(1)
	}
}

// writeAppendicesPDF writes appendices to PDF
func (rt *ReportTemplate) writeAppendicesPDF(pdf *gofpdf.Fpdf, data []map[string]interface{}, options *PDFOptions) {
	pdf.AddPage()
	
	pdf.SetFont("Arial", "B", 14)
	pdf.CellFormat(0, 8, "Appendices", "0", 1, "L", false, 0, "")
	pdf.Ln(2)
	
	// Appendix A: Technical Details
	pdf.SetFont("Arial", "B", 12)
	pdf.CellFormat(0, 6, "Appendix A: Technical Details", "0", 1, "L", false, 0, "")
	pdf.SetFont("Arial", "", 10)
	pdf.CellFormat(0, 5, fmt.Sprintf("Processing Engine: %s", "DataScrapexter v1.0.0"), "0", 1, "L", false, 0, "")
	pdf.CellFormat(0, 5, fmt.Sprintf("Output Format: %s", "PDF"), "0", 1, "L", false, 0, "")
	pdf.CellFormat(0, 5, fmt.Sprintf("Character Encoding: %s", "UTF-8"), "0", 1, "L", false, 0, "")
	pdf.Ln(3)
	
	// Appendix B: Field Analysis
	if len(data) > 0 {
		pdf.SetFont("Arial", "B", 12)
		pdf.CellFormat(0, 6, "Appendix B: Field Analysis", "0", 1, "L", false, 0, "")
		
		fields := make(map[string]int)
		for _, record := range data {
			for field := range record {
				fields[field]++
			}
		}
		
		pdf.SetFont("Arial", "", 10)
		for field, count := range fields {
			coverage := float64(count) / float64(len(data)) * 100
			pdf.CellFormat(0, 4, fmt.Sprintf("  %s: %d records (%.1f%%)", field, count, coverage), "0", 1, "L", false, 0, "")
		}
	}
}

// writeFooterPDF writes footer to PDF
func (rt *ReportTemplate) writeFooterPDF(pdf *gofpdf.Fpdf, options *PDFOptions) {
	pdf.SetFooterFunc(func() {
		pdf.SetY(-15)
		pdf.SetFont("Arial", "I", 8)
		pdf.CellFormat(0, 10, fmt.Sprintf("Page %d", pdf.PageNo()), "", 0, "C", false, 0, "")
		pdf.SetX(10)
		pdf.CellFormat(0, 10, "Generated by DataScrapexter", "", 0, "L", false, 0, "")
		
		if options != nil && options.HeaderFooter != nil && options.HeaderFooter.Footer != nil {
			footer := options.HeaderFooter.Footer
			if footer.Text != "" {
				pdf.SetX(180)
				pdf.CellFormat(0, 10, footer.Text, "", 0, "R", false, 0, "")
			}
		}
	})
}

// formatValueForPDF formats a value for display in PDF
func (rt *ReportTemplate) formatValueForPDF(value interface{}) string {
	if value == nil {
		return "<nil>"
	}
	
	switch v := value.(type) {
	case string:
		return v
	case int, int64, int32:
		return fmt.Sprintf("%d", v)
	case float32, float64:
		return fmt.Sprintf("%.2f", v)
	case bool:
		return fmt.Sprintf("%t", v)
	default:
		return fmt.Sprintf("%v", v)
	}
}