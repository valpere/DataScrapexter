// internal/output/xml.go
package output

import (
	"context"
	"encoding/xml"
	"fmt"
	"os"
	"reflect"
	"strings"
	"time"
)

// XMLWriter implements the Writer interface for XML output
type XMLWriter struct {
	file       *os.File
	encoder    *xml.Encoder
	config     XMLConfig
	records    []map[string]interface{}
	rootName   string
	recordName string
}

// XMLConfig configuration for XML output
type XMLConfig struct {
	FilePath     string `json:"file"`
	RootElement  string `json:"root_element"`
	RecordElement string `json:"record_element"`
	Indent       bool   `json:"indent"`
	IndentString string `json:"indent_string"`
	PrettyPrint  bool   `json:"pretty_print"`
	Encoding     string `json:"encoding"`
	Standalone   bool   `json:"standalone"`
	Version      string `json:"version"`
	BufferSize   int    `json:"buffer_size"`
	FlushInterval time.Duration `json:"flush_interval"`
}

// NewXMLWriter creates a new XML writer
func NewXMLWriter(config XMLConfig) (*XMLWriter, error) {
	if config.FilePath == "" {
		return nil, fmt.Errorf("XML file path is required")
	}
	
	// Set defaults
	if config.RootElement == "" {
		config.RootElement = "data"
	}
	if config.RecordElement == "" {
		config.RecordElement = "record"
	}
	if config.IndentString == "" {
		config.IndentString = "  "
	}
	if config.Encoding == "" {
		config.Encoding = "UTF-8"
	}
	if config.Version == "" {
		config.Version = "1.0"
	}
	if config.BufferSize == 0 {
		config.BufferSize = 1000
	}
	
	file, err := os.Create(config.FilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create XML file: %w", err)
	}
	
	encoder := xml.NewEncoder(file)
	if config.Indent || config.PrettyPrint {
		encoder.Indent("", config.IndentString)
	}
	
	writer := &XMLWriter{
		file:       file,
		encoder:    encoder,
		config:     config,
		records:    make([]map[string]interface{}, 0, config.BufferSize),
		rootName:   config.RootElement,
		recordName: config.RecordElement,
	}
	
	// Write XML declaration
	if err := writer.writeXMLDeclaration(); err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to write XML declaration: %w", err)
	}
	
	// Write root element start tag
	if err := writer.writeRootStart(); err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to write root element: %w", err)
	}
	
	return writer, nil
}

// Write writes data to XML file
func (w *XMLWriter) Write(data []map[string]interface{}) error {
	for _, record := range data {
		if err := w.WriteRecord(record); err != nil {
			return err
		}
	}
	return nil
}

// WriteRecord writes a single record to XML
func (w *XMLWriter) WriteRecord(record map[string]interface{}) error {
	if len(w.records) >= w.config.BufferSize {
		if err := w.flush(); err != nil {
			return err
		}
	}
	
	w.records = append(w.records, record)
	return nil
}

// Write with context writes data to XML file
func (w *XMLWriter) WriteContext(ctx context.Context, data interface{}) error {
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
func (w *XMLWriter) Flush() error {
	return w.flush()
}

// Close closes the XML writer and finalizes the file
func (w *XMLWriter) Close() error {
	// Flush any remaining records
	if err := w.flush(); err != nil {
		return err
	}
	
	// Write root element end tag
	if err := w.writeRootEnd(); err != nil {
		return err
	}
	
	// Close encoder and file
	if err := w.encoder.Flush(); err != nil {
		return err
	}
	
	return w.file.Close()
}

// GetType returns the output type
func (w *XMLWriter) GetType() string {
	return "xml"
}

// flush writes buffered records to the file
func (w *XMLWriter) flush() error {
	for _, record := range w.records {
		if err := w.writeRecord(record); err != nil {
			return err
		}
	}
	
	w.records = w.records[:0] // Clear the slice but keep capacity
	return w.encoder.Flush()
}

// writeXMLDeclaration writes the XML declaration
func (w *XMLWriter) writeXMLDeclaration() error {
	declaration := fmt.Sprintf(`<?xml version="%s" encoding="%s"`, w.config.Version, w.config.Encoding)
	if w.config.Standalone {
		declaration += ` standalone="yes"`
	}
	declaration += "?>\n"
	
	_, err := w.file.WriteString(declaration)
	return err
}

// writeRootStart writes the root element start tag
func (w *XMLWriter) writeRootStart() error {
	startElement := xml.StartElement{
		Name: xml.Name{Local: w.rootName},
		Attr: []xml.Attr{
			{Name: xml.Name{Local: "generated"}, Value: time.Now().Format(time.RFC3339)},
			{Name: xml.Name{Local: "generator"}, Value: "DataScrapexter"},
		},
	}
	return w.encoder.EncodeToken(startElement)
}

// writeRootEnd writes the root element end tag
func (w *XMLWriter) writeRootEnd() error {
	endElement := xml.EndElement{Name: xml.Name{Local: w.rootName}}
	return w.encoder.EncodeToken(endElement)
}

// writeRecord writes a single record as XML
func (w *XMLWriter) writeRecord(record map[string]interface{}) error {
	startElement := xml.StartElement{Name: xml.Name{Local: w.recordName}}
	if err := w.encoder.EncodeToken(startElement); err != nil {
		return err
	}
	
	// Sort keys for consistent output
	keys := make([]string, 0, len(record))
	for key := range record {
		keys = append(keys, key)
	}
	
	for _, key := range keys {
		value := record[key]
		if err := w.writeElement(key, value); err != nil {
			return err
		}
	}
	
	endElement := xml.EndElement{Name: xml.Name{Local: w.recordName}}
	return w.encoder.EncodeToken(endElement)
}

// writeElement writes a single XML element
func (w *XMLWriter) writeElement(name string, value interface{}) error {
	// Sanitize element name
	elementName := sanitizeXMLName(name)
	
	if value == nil {
		// Empty element for nil values
		element := xml.StartElement{
			Name: xml.Name{Local: elementName},
			Attr: []xml.Attr{{Name: xml.Name{Local: "nil"}, Value: "true"}},
		}
		if err := w.encoder.EncodeToken(element); err != nil {
			return err
		}
		return w.encoder.EncodeToken(xml.EndElement{Name: xml.Name{Local: elementName}})
	}
	
	switch v := value.(type) {
	case map[string]interface{}:
		return w.writeComplexElement(elementName, v)
	case []interface{}:
		return w.writeArrayElement(elementName, v)
	case []map[string]interface{}:
		return w.writeArrayOfMapsElement(elementName, v)
	default:
		return w.writeSimpleElement(elementName, value)
	}
}

// writeSimpleElement writes a simple text element
func (w *XMLWriter) writeSimpleElement(name string, value interface{}) error {
	startElement := xml.StartElement{
		Name: xml.Name{Local: name},
		Attr: []xml.Attr{{Name: xml.Name{Local: "type"}, Value: getXMLType(value)}},
	}
	
	if err := w.encoder.EncodeToken(startElement); err != nil {
		return err
	}
	
	text := fmt.Sprintf("%v", value)
	if err := w.encoder.EncodeToken(xml.CharData(text)); err != nil {
		return err
	}
	
	endElement := xml.EndElement{Name: xml.Name{Local: name}}
	return w.encoder.EncodeToken(endElement)
}

// writeComplexElement writes a complex object element
func (w *XMLWriter) writeComplexElement(name string, obj map[string]interface{}) error {
	startElement := xml.StartElement{
		Name: xml.Name{Local: name},
		Attr: []xml.Attr{{Name: xml.Name{Local: "type"}, Value: "object"}},
	}
	
	if err := w.encoder.EncodeToken(startElement); err != nil {
		return err
	}
	
	for key, value := range obj {
		if err := w.writeElement(key, value); err != nil {
			return err
		}
	}
	
	endElement := xml.EndElement{Name: xml.Name{Local: name}}
	return w.encoder.EncodeToken(endElement)
}

// writeArrayElement writes an array element
func (w *XMLWriter) writeArrayElement(name string, arr []interface{}) error {
	startElement := xml.StartElement{
		Name: xml.Name{Local: name},
		Attr: []xml.Attr{
			{Name: xml.Name{Local: "type"}, Value: "array"},
			{Name: xml.Name{Local: "length"}, Value: fmt.Sprintf("%d", len(arr))},
		},
	}
	
	if err := w.encoder.EncodeToken(startElement); err != nil {
		return err
	}
	
	for i, item := range arr {
		itemName := fmt.Sprintf("item_%d", i)
		if err := w.writeElement(itemName, item); err != nil {
			return err
		}
	}
	
	endElement := xml.EndElement{Name: xml.Name{Local: name}}
	return w.encoder.EncodeToken(endElement)
}

// writeArrayOfMapsElement writes an array of maps element
func (w *XMLWriter) writeArrayOfMapsElement(name string, arr []map[string]interface{}) error {
	startElement := xml.StartElement{
		Name: xml.Name{Local: name},
		Attr: []xml.Attr{
			{Name: xml.Name{Local: "type"}, Value: "array"},
			{Name: xml.Name{Local: "length"}, Value: fmt.Sprintf("%d", len(arr))},
		},
	}
	
	if err := w.encoder.EncodeToken(startElement); err != nil {
		return err
	}
	
	for i, item := range arr {
		itemName := fmt.Sprintf("item_%d", i)
		if err := w.writeComplexElement(itemName, item); err != nil {
			return err
		}
	}
	
	endElement := xml.EndElement{Name: xml.Name{Local: name}}
	return w.encoder.EncodeToken(endElement)
}

// Helper functions

// sanitizeXMLName ensures the name is valid for XML
func sanitizeXMLName(name string) string {
	// Replace invalid characters with underscores
	sanitized := strings.Map(func(r rune) rune {
		if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || 
		   (r >= '0' && r <= '9') || r == '_' || r == '-' || r == '.' {
			return r
		}
		return '_'
	}, name)
	
	// Ensure it doesn't start with a number
	if len(sanitized) > 0 && sanitized[0] >= '0' && sanitized[0] <= '9' {
		sanitized = "_" + sanitized
	}
	
	// Ensure it's not empty
	if sanitized == "" {
		sanitized = "element"
	}
	
	return sanitized
}

// getXMLType returns the XML type for a value
func getXMLType(value interface{}) string {
	if value == nil {
		return "nil"
	}
	
	switch value.(type) {
	case bool:
		return "boolean"
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return "integer"
	case float32, float64:
		return "float"
	case string:
		return "string"
	case time.Time:
		return "datetime"
	case []interface{}:
		return "array"
	case map[string]interface{}:
		return "object"
	default:
		return "string"
	}
}

// XMLElement represents a generic XML element for marshaling
type XMLElement struct {
	XMLName xml.Name
	Attrs   []xml.Attr `xml:",attr"`
	Content interface{}
}

// XMLRecord represents a record for XML output
type XMLRecord struct {
	XMLName xml.Name               `xml:"record"`
	Data    map[string]interface{} `xml:",omitempty"`
}

// MarshalXML implements custom XML marshaling
func (r XMLRecord) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	if err := e.EncodeToken(start); err != nil {
		return err
	}
	
	for key, value := range r.Data {
		element := XMLElement{
			XMLName: xml.Name{Local: sanitizeXMLName(key)},
			Content: value,
		}
		if err := e.Encode(element); err != nil {
			return err
		}
	}
	
	return e.EncodeToken(xml.EndElement{Name: start.Name})
}

// ValidateXMLConfig validates XML configuration
func ValidateXMLConfig(config XMLConfig) error {
	if config.FilePath == "" {
		return fmt.Errorf("file path is required")
	}
	
	if config.RootElement != "" && !isValidXMLName(config.RootElement) {
		return fmt.Errorf("invalid root element name: %s", config.RootElement)
	}
	
	if config.RecordElement != "" && !isValidXMLName(config.RecordElement) {
		return fmt.Errorf("invalid record element name: %s", config.RecordElement)
	}
	
	if config.BufferSize < 0 {
		return fmt.Errorf("buffer size must be non-negative")
	}
	
	return nil
}

// isValidXMLName checks if a string is a valid XML name
func isValidXMLName(name string) bool {
	if len(name) == 0 {
		return false
	}
	
	// Check first character
	first := rune(name[0])
	if !((first >= 'A' && first <= 'Z') || (first >= 'a' && first <= 'z') || first == '_') {
		return false
	}
	
	// Check remaining characters
	for _, r := range name[1:] {
		if !((r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || 
		     (r >= '0' && r <= '9') || r == '_' || r == '-' || r == '.') {
			return false
		}
	}
	
	return true
}