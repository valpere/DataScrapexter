// internal/pipeline/pipeline.go
package pipeline

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// DataPipeline orchestrates the entire data processing pipeline
type DataPipeline struct {
	Extractor     *DataExtractor
	Transformer   *DataTransformer
	Validator     *DataValidator
	Deduplicator  *RecordDeduplicator
	Enricher      *DataEnricher
	OutputManager *OutputManager
	
	// Configuration
	Config *PipelineConfig
	
	// State management
	mu       sync.RWMutex
	metrics  *PipelineMetrics
	logger   Logger
}

// PipelineConfig holds pipeline configuration
type PipelineConfig struct {
	BufferSize     int           `yaml:"buffer_size" json:"buffer_size"`
	WorkerCount    int           `yaml:"worker_count" json:"worker_count"`
	Timeout        time.Duration `yaml:"timeout" json:"timeout"`
	EnableMetrics  bool          `yaml:"enable_metrics" json:"enable_metrics"`
	RetryAttempts  int           `yaml:"retry_attempts" json:"retry_attempts"`
	RetryDelay     time.Duration `yaml:"retry_delay" json:"retry_delay"`
}

// PipelineMetrics tracks pipeline performance
type PipelineMetrics struct {
	ProcessedCount   int64         `json:"processed_count"`
	SuccessCount     int64         `json:"success_count"`
	ErrorCount       int64         `json:"error_count"`
	AverageTime      time.Duration `json:"average_time"`
	TotalTime        time.Duration `json:"total_time"`
	LastProcessedAt  time.Time     `json:"last_processed_at"`
}

// ProcessedData represents data that has been processed through the pipeline
type ProcessedData struct {
	Raw         map[string]interface{} `json:"raw"`
	Extracted   map[string]interface{} `json:"extracted"`
	Transformed map[string]interface{} `json:"transformed"`
	Validated   map[string]interface{} `json:"validated"`
	Enriched    map[string]interface{} `json:"enriched"`
	
	Metadata    ProcessingMetadata     `json:"metadata"`
	Errors      []ProcessingError      `json:"errors,omitempty"`
}

// ProcessingMetadata contains metadata about the processing
type ProcessingMetadata struct {
	ProcessedAt  time.Time     `json:"processed_at"`
	ProcessingID string        `json:"processing_id"`
	SourceURL    string        `json:"source_url"`
	PipelineID   string        `json:"pipeline_id"`
	Duration     time.Duration `json:"duration"`
	Stage        string        `json:"stage"`
}

// ProcessingError represents an error that occurred during processing
type ProcessingError struct {
	Stage   string    `json:"stage"`
	Message string    `json:"message"`
	Time    time.Time `json:"time"`
	Fatal   bool      `json:"fatal"`
}

// Logger interface for pipeline logging
type Logger interface {
	Debug(msg string, fields ...interface{})
	Info(msg string, fields ...interface{})
	Warn(msg string, fields ...interface{})
	Error(msg string, fields ...interface{})
}

// NewDataPipeline creates a new data processing pipeline
func NewDataPipeline(config *PipelineConfig) *DataPipeline {
	if config == nil {
		config = &PipelineConfig{
			BufferSize:    1000,
			WorkerCount:   10,
			Timeout:       30 * time.Second,
			EnableMetrics: true,
			RetryAttempts: 3,
			RetryDelay:    1 * time.Second,
		}
	}

	return &DataPipeline{
		Config:  config,
		metrics: &PipelineMetrics{},
	}
}

// Process processes data through the entire pipeline
func (dp *DataPipeline) Process(ctx context.Context, rawData map[string]interface{}) (*ProcessedData, error) {
	startTime := time.Now()
	
	result := &ProcessedData{
		Raw: rawData,
		Metadata: ProcessingMetadata{
			ProcessedAt:  startTime,
			ProcessingID: generateProcessingID(),
			Stage:        "started",
		},
	}

	// Create a timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, dp.Config.Timeout)
	defer cancel()

	// Stage 1: Extraction
	if dp.Extractor != nil {
		result.Metadata.Stage = "extraction"
		extracted, err := dp.Extractor.Extract(timeoutCtx, rawData)
		if err != nil {
			result.Errors = append(result.Errors, ProcessingError{
				Stage:   "extraction",
				Message: err.Error(),
				Time:    time.Now(),
				Fatal:   true,
			})
			return result, fmt.Errorf("extraction failed: %w", err)
		}
		result.Extracted = extracted
	} else {
		result.Extracted = rawData
	}

	// Stage 2: Transformation
	if dp.Transformer != nil {
		result.Metadata.Stage = "transformation"
		transformed, err := dp.Transformer.TransformData(timeoutCtx, result.Extracted)
		if err != nil {
			result.Errors = append(result.Errors, ProcessingError{
				Stage:   "transformation",
				Message: err.Error(),
				Time:    time.Now(),
				Fatal:   true,
			})
			return result, fmt.Errorf("transformation failed: %w", err)
		}
		result.Transformed = transformed
	} else {
		result.Transformed = result.Extracted
	}

	// Stage 3: Validation
	if dp.Validator != nil {
		result.Metadata.Stage = "validation"
		validated, err := dp.Validator.Validate(timeoutCtx, result.Transformed)
		if err != nil {
			result.Errors = append(result.Errors, ProcessingError{
				Stage:   "validation",
				Message: err.Error(),
				Time:    time.Now(),
				Fatal:   true,
			})
			return result, fmt.Errorf("validation failed: %w", err)
		}
		result.Validated = validated
	} else {
		result.Validated = result.Transformed
	}

	// Stage 4: Deduplication
	if dp.Deduplicator != nil {
		result.Metadata.Stage = "deduplication"
		deduplicated, err := dp.Deduplicator.Deduplicate(timeoutCtx, result.Validated)
		if err != nil {
			result.Errors = append(result.Errors, ProcessingError{
				Stage:   "deduplication",
				Message: err.Error(),
				Time:    time.Now(),
				Fatal:   false, // Non-fatal error
			})
			// Continue with original data if deduplication fails
		} else {
			result.Validated = deduplicated
		}
	}

	// Stage 5: Enrichment
	if dp.Enricher != nil {
		result.Metadata.Stage = "enrichment"
		enriched, err := dp.Enricher.Enrich(timeoutCtx, result.Validated)
		if err != nil {
			result.Errors = append(result.Errors, ProcessingError{
				Stage:   "enrichment",
				Message: err.Error(),
				Time:    time.Now(),
				Fatal:   false, // Non-fatal error
			})
			// Continue with validated data if enrichment fails
			result.Enriched = result.Validated
		} else {
			result.Enriched = enriched
		}
	} else {
		result.Enriched = result.Validated
	}

	// Update metadata
	result.Metadata.Stage = "completed"
	result.Metadata.Duration = time.Since(startTime)

	// Update metrics
	dp.updateMetrics(result)

	return result, nil
}

// ProcessBatch processes multiple data items through the pipeline
func (dp *DataPipeline) ProcessBatch(ctx context.Context, batchData []map[string]interface{}) ([]*ProcessedData, error) {
	results := make([]*ProcessedData, 0, len(batchData))
	
	// Create worker pool for concurrent processing
	workerCount := dp.Config.WorkerCount
	if workerCount <= 0 {
		workerCount = 10
	}

	jobs := make(chan map[string]interface{}, len(batchData))
	resultChan := make(chan *ProcessedData, len(batchData))
	errorChan := make(chan error, len(batchData))

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for data := range jobs {
				result, err := dp.Process(ctx, data)
				if err != nil {
					errorChan <- err
				} else {
					resultChan <- result
				}
			}
		}()
	}

	// Send jobs
	go func() {
		defer close(jobs)
		for _, data := range batchData {
			select {
			case jobs <- data:
			case <-ctx.Done():
				return
			}
		}
	}()

	// Wait for completion
	go func() {
		wg.Wait()
		close(resultChan)
		close(errorChan)
	}()

	// Collect results
	var errors []error
	for {
		select {
		case result, ok := <-resultChan:
			if !ok {
				resultChan = nil
			} else {
				results = append(results, result)
			}
		case err, ok := <-errorChan:
			if !ok {
				errorChan = nil
			} else {
				errors = append(errors, err)
			}
		case <-ctx.Done():
			return results, ctx.Err()
		}

		if resultChan == nil && errorChan == nil {
			break
		}
	}

	if len(errors) > 0 {
		return results, fmt.Errorf("batch processing errors: %v", errors)
	}

	return results, nil
}

// GetMetrics returns current pipeline metrics
func (dp *DataPipeline) GetMetrics() *PipelineMetrics {
	dp.mu.RLock()
	defer dp.mu.RUnlock()
	
	metricsCopy := *dp.metrics
	return &metricsCopy
}

// updateMetrics updates pipeline metrics
func (dp *DataPipeline) updateMetrics(result *ProcessedData) {
	if !dp.Config.EnableMetrics {
		return
	}

	dp.mu.Lock()
	defer dp.mu.Unlock()

	dp.metrics.ProcessedCount++
	dp.metrics.LastProcessedAt = time.Now()
	dp.metrics.TotalTime += result.Metadata.Duration

	if len(result.Errors) == 0 || !hassFatalError(result.Errors) {
		dp.metrics.SuccessCount++
	} else {
		dp.metrics.ErrorCount++
	}

	// Calculate average time
	if dp.metrics.ProcessedCount > 0 {
		dp.metrics.AverageTime = dp.metrics.TotalTime / time.Duration(dp.metrics.ProcessedCount)
	}
}

// hassFatalError checks if there are any fatal errors
func hassFatalError(errors []ProcessingError) bool {
	for _, err := range errors {
		if err.Fatal {
			return true
		}
	}
	return false
}

// generateProcessingID generates a unique processing ID
func generateProcessingID() string {
	return fmt.Sprintf("proc_%d", time.Now().UnixNano())
}

// SetLogger sets the logger for the pipeline
func (dp *DataPipeline) SetLogger(logger Logger) {
	dp.logger = logger
}

// SetExtractor sets the data extractor
func (dp *DataPipeline) SetExtractor(extractor *DataExtractor) {
	dp.Extractor = extractor
}

// SetTransformer sets the data transformer
func (dp *DataPipeline) SetTransformer(transformer *DataTransformer) {
	dp.Transformer = transformer
}

// SetValidator sets the data validator
func (dp *DataPipeline) SetValidator(validator *DataValidator) {
	dp.Validator = validator
}

// SetDeduplicator sets the record deduplicator
func (dp *DataPipeline) SetDeduplicator(deduplicator *RecordDeduplicator) {
	dp.Deduplicator = deduplicator
}

// SetEnricher sets the data enricher
func (dp *DataPipeline) SetEnricher(enricher *DataEnricher) {
	dp.Enricher = enricher
}

// SetOutputManager sets the output manager
func (dp *DataPipeline) SetOutputManager(outputManager *OutputManager) {
	dp.OutputManager = outputManager
}
