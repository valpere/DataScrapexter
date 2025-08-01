// internal/output/csv_output.go
package output

import (
	"encoding/csv"
	"fmt"
	"os"

	"github.com/valpere/DataScrapexter/internal/utils"
)

var logger = utils.NewComponentLogger("csv-output")

// WriteCSVToFile writes data as CSV to specified file
func WriteCSVToFile(filename string, data []map[string]interface{}) error {
	if len(data) == 0 {
		logger.Warn("No data to write to CSV")
		return nil
	}

	file, err := os.Create(filename)
	if err != nil {
		logger.WithField("error", err).Error("Failed to create CSV file")
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Extract headers from first record
	var headers []string
	for key := range data[0] {
		headers = append(headers, key)
	}

	// Write headers
	if err := writer.Write(headers); err != nil {
		logger.WithField("error", err).Error("Failed to write CSV headers")
		return fmt.Errorf("failed to write headers: %w", err)
	}

	// Write data rows
	for i, record := range data {
		var row []string
		for _, header := range headers {
			value := record[header]
			row = append(row, fmt.Sprintf("%v", value))
		}

		if err := writer.Write(row); err != nil {
			logger.WithFields(map[string]interface{}{
				"row_index": i,
				"error":     err,
			}).Error("Failed to write CSV row")
			return fmt.Errorf("failed to write row %d: %w", i, err)
		}
	}

	logger.WithFields(map[string]interface{}{
		"filename":    filename,
		"record_count": len(data),
		"column_count": len(headers),
	}).Info("CSV file written successfully")

	return nil
}
