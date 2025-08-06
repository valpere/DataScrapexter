// internal/output/mongodb.go - MongoDB database connector with advanced features
package output

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/valpere/DataScrapexter/internal/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
)

var mongoLogger = utils.NewComponentLogger("mongodb-output")

// MongoDBWriter implements Writer interface for MongoDB output
type MongoDBWriter struct {
	client     *mongo.Client
	collection *mongo.Collection
	config     MongoDBOptions
	buffer     []interface{}
	metadata   *MongoDBMetadata
	
	// Connection state
	connected   bool
	ctx         context.Context
	cancelFunc  context.CancelFunc
	
	// Statistics
	totalWrites   int64
	totalErrors   int64
	totalBytes    int64
	lastWriteTime time.Time
	startTime     time.Time
}

// MongoDBOptions defines MongoDB-specific configuration options
type MongoDBOptions struct {
	ConnectionString    string            `yaml:"connection_string" json:"connection_string"`
	Database            string            `yaml:"database" json:"database"`
	Collection          string            `yaml:"collection" json:"collection"`
	BatchSize           int               `yaml:"batch_size,omitempty" json:"batch_size,omitempty"`
	WriteConcern        *WriteConcernOptions `yaml:"write_concern,omitempty" json:"write_concern,omitempty"`
	ReadPreference      string            `yaml:"read_preference,omitempty" json:"read_preference,omitempty"`
	MaxPoolSize         int               `yaml:"max_pool_size,omitempty" json:"max_pool_size,omitempty"`
	MinPoolSize         int               `yaml:"min_pool_size,omitempty" json:"min_pool_size,omitempty"`
	MaxConnIdleTime     time.Duration     `yaml:"max_conn_idle_time,omitempty" json:"max_conn_idle_time,omitempty"`
	Timeout             time.Duration     `yaml:"timeout,omitempty" json:"timeout,omitempty"`
	RetryWrites         bool              `yaml:"retry_writes,omitempty" json:"retry_writes,omitempty"`
	RetryReads          bool              `yaml:"retry_reads,omitempty" json:"retry_reads,omitempty"`
	CreateCollection    bool              `yaml:"create_collection,omitempty" json:"create_collection,omitempty"`
	OnConflict          ConflictStrategy  `yaml:"on_conflict,omitempty" json:"on_conflict,omitempty"`
	Indexes             []MongoIndexSpec  `yaml:"indexes,omitempty" json:"indexes,omitempty"`
	ValidationRules     *ValidationRules  `yaml:"validation_rules,omitempty" json:"validation_rules,omitempty"`
	Compression         string            `yaml:"compression,omitempty" json:"compression,omitempty"` // snappy, zlib, zstd
	TLS                 *MongoTLSOptions  `yaml:"tls,omitempty" json:"tls,omitempty"`
	Authentication      *MongoAuthOptions `yaml:"authentication,omitempty" json:"authentication,omitempty"`
	TransformFields     map[string]string `yaml:"transform_fields,omitempty" json:"transform_fields,omitempty"`
	IncludeMetadata     bool              `yaml:"include_metadata,omitempty" json:"include_metadata,omitempty"`
	TimeStampField      string            `yaml:"timestamp_field,omitempty" json:"timestamp_field,omitempty"`
}

// WriteConcernOptions defines MongoDB write concern settings
type WriteConcernOptions struct {
	W         interface{} `yaml:"w,omitempty" json:"w,omitempty"`           // int or string ("majority")
	Journal   *bool       `yaml:"journal,omitempty" json:"journal,omitempty"`
	WTimeout  time.Duration `yaml:"wtimeout,omitempty" json:"wtimeout,omitempty"`
}

// MongoIndexSpec defines MongoDB index specification
type MongoIndexSpec struct {
	Keys        bson.D                 `yaml:"keys" json:"keys"`
	Options     *options.IndexOptions  `yaml:"options,omitempty" json:"options,omitempty"`
	Name        string                 `yaml:"name,omitempty" json:"name,omitempty"`
	Unique      bool                   `yaml:"unique,omitempty" json:"unique,omitempty"`
	Background  bool                   `yaml:"background,omitempty" json:"background,omitempty"`
	Sparse      bool                   `yaml:"sparse,omitempty" json:"sparse,omitempty"`
	TTL         time.Duration          `yaml:"ttl,omitempty" json:"ttl,omitempty"`
	PartialFilter bson.D               `yaml:"partial_filter,omitempty" json:"partial_filter,omitempty"`
}

// ValidationRules defines MongoDB document validation rules
type ValidationRules struct {
	Validator    bson.M `yaml:"validator,omitempty" json:"validator,omitempty"`
	Level        string `yaml:"level,omitempty" json:"level,omitempty"`         // strict, moderate
	Action       string `yaml:"action,omitempty" json:"action,omitempty"`       // error, warn
}

// MongoTLSOptions defines MongoDB TLS/SSL configuration
type MongoTLSOptions struct {
	Enabled               bool   `yaml:"enabled" json:"enabled"`
	CertificateFile       string `yaml:"certificate_file,omitempty" json:"certificate_file,omitempty"`
	PrivateKeyFile        string `yaml:"private_key_file,omitempty" json:"private_key_omitempty"`
	CAFile                string `yaml:"ca_file,omitempty" json:"ca_file,omitempty"`
	InsecureSkipVerify    bool   `yaml:"insecure_skip_verify,omitempty" json:"insecure_skip_verify,omitempty"`
	AllowInvalidHostnames bool   `yaml:"allow_invalid_hostnames,omitempty" json:"allow_invalid_hostnames,omitempty"`
}

// MongoAuthOptions defines MongoDB authentication configuration
type MongoAuthOptions struct {
	Mechanism           string `yaml:"mechanism,omitempty" json:"mechanism,omitempty"`           // SCRAM-SHA-1, SCRAM-SHA-256, MONGODB-CR, etc.
	Source              string `yaml:"source,omitempty" json:"source,omitempty"`                 // Auth database
	Username            string `yaml:"username,omitempty" json:"username,omitempty"`
	Password            string `yaml:"password,omitempty" json:"password,omitempty"`
	MechanismProperties map[string]string `yaml:"mechanism_properties,omitempty" json:"mechanism_properties,omitempty"`
}

// MongoDBMetadata contains metadata about MongoDB operations
type MongoDBMetadata struct {
	Database          string              `json:"database"`
	Collection        string              `json:"collection"`
	ServerInfo        *MongoServerInfo    `json:"server_info,omitempty"`
	CollectionStats   *MongoCollectionStats `json:"collection_stats,omitempty"`
	IndexInfo         []MongoIndexInfo    `json:"index_info,omitempty"`
	WriteConcern      *WriteConcernOptions `json:"write_concern,omitempty"`
	LastOpTime        primitive.Timestamp `json:"last_op_time,omitempty"`
	ConnectionState   string              `json:"connection_state"`
}

// MongoServerInfo contains MongoDB server information
type MongoServerInfo struct {
	Version        string            `json:"version"`
	GitVersion     string            `json:"git_version,omitempty"`
	Modules        []string          `json:"modules,omitempty"`
	StorageEngine  map[string]interface{} `json:"storage_engine,omitempty"`
	ReplicaSet     string            `json:"replica_set,omitempty"`
	MaxBSONSize    int32             `json:"max_bson_size"`
	MaxMessageSize int32             `json:"max_message_size"`
}

// MongoCollectionStats contains collection statistics
type MongoCollectionStats struct {
	DocumentCount int64   `json:"document_count"`
	AvgObjSize    float64 `json:"avg_obj_size"`
	DataSize      int64   `json:"data_size"`
	StorageSize   int64   `json:"storage_size"`
	IndexCount    int     `json:"index_count"`
	IndexSize     int64   `json:"index_size"`
	Sharded       bool    `json:"sharded"`
}

// MongoIndexInfo contains index information
type MongoIndexInfo struct {
	Name     string `json:"name"`
	Keys     bson.D `json:"keys"`
	Unique   bool   `json:"unique"`
	Sparse   bool   `json:"sparse"`
	TTL      int32  `json:"ttl,omitempty"`
	Size     int64  `json:"size"`
}

// NewMongoDBWriter creates a new MongoDB writer
func NewMongoDBWriter(options MongoDBOptions) (*MongoDBWriter, error) {
	if options.ConnectionString == "" {
		return nil, fmt.Errorf("MongoDB connection string is required")
	}
	if options.Database == "" {
		return nil, fmt.Errorf("MongoDB database name is required")
	}
	if options.Collection == "" {
		return nil, fmt.Errorf("MongoDB collection name is required")
	}

	// Set defaults
	if options.BatchSize == 0 {
		options.BatchSize = 1000
	}
	if options.Timeout == 0 {
		options.Timeout = 30 * time.Second
	}
	if options.MaxPoolSize == 0 {
		options.MaxPoolSize = 100
	}
	if options.MinPoolSize == 0 {
		options.MinPoolSize = 5
	}
	if options.MaxConnIdleTime == 0 {
		options.MaxConnIdleTime = 10 * time.Minute
	}
	if options.OnConflict == "" {
		options.OnConflict = ConflictIgnore
	}
	if options.TimeStampField == "" {
		options.TimeStampField = "created_at"
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), options.Timeout)

	writer := &MongoDBWriter{
		config:     options,
		buffer:     make([]interface{}, 0, options.BatchSize),
		ctx:        ctx,
		cancelFunc: cancel,
		startTime:  time.Now(),
		metadata: &MongoDBMetadata{
			Database:        options.Database,
			Collection:      options.Collection,
			ConnectionState: "disconnected",
		},
	}

	// Connect to MongoDB
	if err := writer.connect(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	mongoLogger.Info(fmt.Sprintf("Connected to MongoDB database %s, collection %s", 
		options.Database, options.Collection))

	return writer, nil
}

// connect establishes connection to MongoDB
func (mw *MongoDBWriter) connect() error {
	// Build client options
	clientOptions := options.Client()
	clientOptions.ApplyURI(mw.config.ConnectionString)

	// Connection pool settings
	clientOptions.SetMaxPoolSize(uint64(mw.config.MaxPoolSize))
	clientOptions.SetMinPoolSize(uint64(mw.config.MinPoolSize))
	clientOptions.SetMaxConnIdleTime(mw.config.MaxConnIdleTime)

	// Retry settings
	clientOptions.SetRetryWrites(mw.config.RetryWrites)
	clientOptions.SetRetryReads(mw.config.RetryReads)

	// Compression
	if mw.config.Compression != "" {
		clientOptions.SetCompressors([]string{mw.config.Compression})
	}

	// Write concern
	if mw.config.WriteConcern != nil {
		wc := buildWriteConcern(mw.config.WriteConcern)
		clientOptions.SetWriteConcern(wc)
	}

	// Read preference
	if mw.config.ReadPreference != "" {
		rp, err := buildReadPreference(mw.config.ReadPreference)
		if err != nil {
			return fmt.Errorf("invalid read preference: %w", err)
		}
		clientOptions.SetReadPreference(rp)
	}

	// TLS configuration
	if mw.config.TLS != nil && mw.config.TLS.Enabled {
		tlsConfig, err := buildTLSConfig(mw.config.TLS)
		if err != nil {
			return fmt.Errorf("failed to build TLS config: %w", err)
		}
		clientOptions.SetTLSConfig(tlsConfig)
	}

	// Authentication
	if mw.config.Authentication != nil {
		credential := buildCredential(mw.config.Authentication)
		clientOptions.SetAuth(credential)
	}

	// Connect to MongoDB
	client, err := mongo.Connect(mw.ctx, clientOptions)
	if err != nil {
		return fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Ping to verify connection
	if err := client.Ping(mw.ctx, nil); err != nil {
		client.Disconnect(mw.ctx)
		return fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	mw.client = client
	mw.collection = client.Database(mw.config.Database).Collection(mw.config.Collection)
	mw.connected = true
	mw.metadata.ConnectionState = "connected"

	// Initialize collection and indexes
	if err := mw.initializeCollection(); err != nil {
		return fmt.Errorf("failed to initialize collection: %w", err)
	}

	// Gather server and collection metadata
	if err := mw.gatherMetadata(); err != nil {
		mongoLogger.Warn(fmt.Sprintf("Failed to gather metadata: %v", err))
	}

	return nil
}

// initializeCollection creates collection and indexes if needed
func (mw *MongoDBWriter) initializeCollection() error {
	if !mw.config.CreateCollection {
		return nil
	}

	// Check if collection exists
	collections, err := mw.client.Database(mw.config.Database).ListCollectionNames(mw.ctx, bson.D{})
	if err != nil {
		return fmt.Errorf("failed to list collections: %w", err)
	}

	collectionExists := false
	for _, name := range collections {
		if name == mw.config.Collection {
			collectionExists = true
			break
		}
	}

	// Create collection if it doesn't exist
	if !collectionExists {
		createOptions := options.CreateCollection()
		
		// Set validation rules if provided
		if mw.config.ValidationRules != nil {
			if mw.config.ValidationRules.Validator != nil {
				createOptions.SetValidator(mw.config.ValidationRules.Validator)
			}
			if mw.config.ValidationRules.Level != "" {
				createOptions.SetValidationLevel(mw.config.ValidationRules.Level)
			}
			if mw.config.ValidationRules.Action != "" {
				createOptions.SetValidationAction(mw.config.ValidationRules.Action)
			}
		}

		if err := mw.client.Database(mw.config.Database).CreateCollection(mw.ctx, mw.config.Collection, createOptions); err != nil {
			return fmt.Errorf("failed to create collection: %w", err)
		}

		mongoLogger.Info(fmt.Sprintf("Created collection %s.%s", mw.config.Database, mw.config.Collection))
	}

	// Create indexes
	if len(mw.config.Indexes) > 0 {
		if err := mw.createIndexes(); err != nil {
			return fmt.Errorf("failed to create indexes: %w", err)
		}
	}

	return nil
}

// createIndexes creates the specified indexes
func (mw *MongoDBWriter) createIndexes() error {
	indexModels := make([]mongo.IndexModel, 0, len(mw.config.Indexes))

	for _, indexSpec := range mw.config.Indexes {
		indexOptions := options.Index()

		if indexSpec.Name != "" {
			indexOptions.SetName(indexSpec.Name)
		}
		if indexSpec.Unique {
			indexOptions.SetUnique(true)
		}
		if indexSpec.Background {
			indexOptions.SetBackground(true)
		}
		if indexSpec.Sparse {
			indexOptions.SetSparse(true)
		}
		if indexSpec.TTL > 0 {
			indexOptions.SetExpireAfterSeconds(int32(indexSpec.TTL.Seconds()))
		}
		if len(indexSpec.PartialFilter) > 0 {
			indexOptions.SetPartialFilterExpression(indexSpec.PartialFilter)
		}

		// Merge with user-provided options
		if indexSpec.Options != nil {
			// This would require more complex merging logic in a full implementation
			// For now, we use the basic options set above
		}

		indexModel := mongo.IndexModel{
			Keys:    indexSpec.Keys,
			Options: indexOptions,
		}
		indexModels = append(indexModels, indexModel)
	}

	// Create indexes
	createdIndexes, err := mw.collection.Indexes().CreateMany(mw.ctx, indexModels)
	if err != nil {
		return fmt.Errorf("failed to create indexes: %w", err)
	}

	mongoLogger.Info(fmt.Sprintf("Created %d indexes: %v", len(createdIndexes), createdIndexes))
	return nil
}

// gatherMetadata collects server and collection metadata
func (mw *MongoDBWriter) gatherMetadata() error {
	// Get server information
	var serverStatus bson.M
	if err := mw.client.Database("admin").RunCommand(mw.ctx, bson.D{{"serverStatus", 1}}).Decode(&serverStatus); err == nil {
		if version, ok := serverStatus["version"].(string); ok {
			mw.metadata.ServerInfo = &MongoServerInfo{
				Version: version,
			}
		}
	}

	// Get collection stats
	var collStats bson.M
	if err := mw.client.Database(mw.config.Database).RunCommand(
		mw.ctx, 
		bson.D{{"collStats", mw.config.Collection}},
	).Decode(&collStats); err == nil {
		if count, ok := collStats["count"].(int64); ok {
			if avgObjSize, ok := collStats["avgObjSize"].(float64); ok {
				mw.metadata.CollectionStats = &MongoCollectionStats{
					DocumentCount: count,
					AvgObjSize:   avgObjSize,
				}
			}
		}
	}

	return nil
}

// Write writes data to MongoDB collection
func (mw *MongoDBWriter) Write(data []map[string]interface{}) error {
	if !mw.connected {
		return fmt.Errorf("not connected to MongoDB")
	}

	// Transform and buffer data
	for _, record := range data {
		transformedRecord := mw.transformRecord(record)
		mw.buffer = append(mw.buffer, transformedRecord)

		// Flush if buffer is full
		if len(mw.buffer) >= mw.config.BatchSize {
			if err := mw.flush(); err != nil {
				return err
			}
		}
	}

	mongoLogger.Debug(fmt.Sprintf("Buffered %d records (buffer size: %d)", len(data), len(mw.buffer)))
	return nil
}

// transformRecord applies field transformations and adds metadata
func (mw *MongoDBWriter) transformRecord(record map[string]interface{}) map[string]interface{} {
	transformed := make(map[string]interface{})

	// Copy and transform fields
	for key, value := range record {
		// Apply field name transformation
		if newKey, exists := mw.config.TransformFields[key]; exists {
			transformed[newKey] = mw.transformValue(value)
		} else {
			transformed[key] = mw.transformValue(value)
		}
	}

	// Add timestamp if configured
	if mw.config.TimeStampField != "" {
		transformed[mw.config.TimeStampField] = time.Now()
	}

	// Add metadata if requested
	if mw.config.IncludeMetadata {
		metadata := map[string]interface{}{
			"_datascrapexter_ingested_at": time.Now(),
			"_datascrapexter_source":      "DataScrapexter",
			"_datascrapexter_version":     "1.0.0",
		}
		transformed["_metadata"] = metadata
	}

	return transformed
}

// transformValue converts values to MongoDB-compatible types
func (mw *MongoDBWriter) transformValue(value interface{}) interface{} {
	if value == nil {
		return nil
	}

	switch v := value.(type) {
	case string:
		// Try to parse as time if it looks like a timestamp
		if HasTimeFormatPattern(v) {
			if parsed, err := time.Parse(time.RFC3339, v); err == nil {
				return primitive.NewDateTimeFromTime(parsed)
			}
			if parsed, err := time.Parse("2006-01-02 15:04:05", v); err == nil {
				return primitive.NewDateTimeFromTime(parsed)
			}
			if parsed, err := time.Parse("2006-01-02", v); err == nil {
				return primitive.NewDateTimeFromTime(parsed)
			}
		}
		return v
	case int, int32, int64:
		return v
	case float32, float64:
		return v
	case bool:
		return v
	case time.Time:
		return primitive.NewDateTimeFromTime(v)
	case []interface{}:
		// Transform array elements
		transformed := make([]interface{}, len(v))
		for i, item := range v {
			transformed[i] = mw.transformValue(item)
		}
		return transformed
	case map[string]interface{}:
		// Transform nested object
		transformed := make(map[string]interface{})
		for key, val := range v {
			transformed[key] = mw.transformValue(val)
		}
		return transformed
	default:
		// Convert to string as fallback
		return fmt.Sprintf("%v", v)
	}
}

// flush writes buffered data to MongoDB
func (mw *MongoDBWriter) flush() error {
	if len(mw.buffer) == 0 {
		return nil
	}

	startTime := time.Now()

	// Handle conflicts based on strategy
	switch mw.config.OnConflict {
	case ConflictIgnore:
		if err := mw.insertIgnore(); err != nil {
			return err
		}
	case ConflictReplace:
		if err := mw.upsertRecords(); err != nil {
			return err
		}
	default: // ConflictError
		if err := mw.insertRecords(); err != nil {
			return err
		}
	}

	duration := time.Since(startTime)
	mw.totalWrites += int64(len(mw.buffer))
	mw.lastWriteTime = time.Now()

	mongoLogger.Debug(fmt.Sprintf("Flushed %d records in %v", len(mw.buffer), duration))

	// Clear buffer
	mw.buffer = mw.buffer[:0]
	return nil
}

// insertRecords performs regular insert (fails on duplicate keys)
func (mw *MongoDBWriter) insertRecords() error {
	result, err := mw.collection.InsertMany(mw.ctx, mw.buffer)
	if err != nil {
		mw.totalErrors++
		return fmt.Errorf("failed to insert records: %w", err)
	}

	mongoLogger.Debug(fmt.Sprintf("Inserted %d records", len(result.InsertedIDs)))
	return nil
}

// insertIgnore performs insert with ignore duplicates
func (mw *MongoDBWriter) insertIgnore() error {
	// MongoDB doesn't have a direct "INSERT IGNORE" equivalent
	// We need to insert one by one and ignore duplicate key errors
	insertedCount := 0
	ignoredCount := 0

	for _, doc := range mw.buffer {
		_, err := mw.collection.InsertOne(mw.ctx, doc)
		if err != nil {
			// Check if it's a duplicate key error
			if mongo.IsDuplicateKeyError(err) {
				ignoredCount++
				continue
			}
			mw.totalErrors++
			return fmt.Errorf("failed to insert record: %w", err)
		}
		insertedCount++
	}

	mongoLogger.Debug(fmt.Sprintf("Inserted %d records, ignored %d duplicates", insertedCount, ignoredCount))
	return nil
}

// upsertRecords performs upsert (insert or replace)
func (mw *MongoDBWriter) upsertRecords() error {
	operations := make([]mongo.WriteModel, len(mw.buffer))

	for i, doc := range mw.buffer {
		// Use the entire document as filter for replacement
		// In a real implementation, you'd want to use specific fields as unique identifiers
		filter := bson.M{}
		if docMap, ok := doc.(map[string]interface{}); ok {
			// Try to find a unique identifier
			if id, exists := docMap["id"]; exists {
				filter["id"] = id
			} else if id, exists := docMap["_id"]; exists {
				filter["_id"] = id
			} else {
				// Use the first field as identifier (not ideal, but works for demo)
				for key, value := range docMap {
					filter[key] = value
					break
				}
			}
		}

		replaceModel := mongo.NewReplaceOneModel().SetFilter(filter).SetReplacement(doc).SetUpsert(true)
		operations[i] = replaceModel
	}

	result, err := mw.collection.BulkWrite(mw.ctx, operations)
	if err != nil {
		mw.totalErrors++
		return fmt.Errorf("failed to upsert records: %w", err)
	}

	mongoLogger.Debug(fmt.Sprintf("Upserted %d records (inserted: %d, modified: %d)", 
		len(mw.buffer), result.InsertedCount, result.ModifiedCount))
	return nil
}

// Close closes the MongoDB connection and flushes remaining data
func (mw *MongoDBWriter) Close() error {
	if !mw.connected {
		return nil
	}

	// Flush remaining data
	if len(mw.buffer) > 0 {
		if err := mw.flush(); err != nil {
			mongoLogger.Error(fmt.Sprintf("Failed to flush remaining data: %v", err))
		}
	}

	// Disconnect from MongoDB
	if mw.client != nil {
		if err := mw.client.Disconnect(mw.ctx); err != nil {
			mongoLogger.Error(fmt.Sprintf("Failed to disconnect from MongoDB: %v", err))
		}
	}

	// Cancel context
	if mw.cancelFunc != nil {
		mw.cancelFunc()
	}

	mw.connected = false
	mw.metadata.ConnectionState = "disconnected"

	duration := time.Since(mw.startTime)
	mongoLogger.Info(fmt.Sprintf("Closed MongoDB connection. Total writes: %d, errors: %d, duration: %v", 
		mw.totalWrites, mw.totalErrors, duration))

	return nil
}

// GetMetadata returns MongoDB operation metadata
func (mw *MongoDBWriter) GetMetadata() *MongoDBMetadata {
	// Update current stats
	mw.metadata.WriteConcern = mw.config.WriteConcern
	return mw.metadata
}

// GetStatistics returns detailed statistics about MongoDB operations
func (mw *MongoDBWriter) GetStatistics() map[string]interface{} {
	duration := time.Since(mw.startTime)
	
	stats := map[string]interface{}{
		"total_writes":     mw.totalWrites,
		"total_errors":     mw.totalErrors,
		"total_bytes":      mw.totalBytes,
		"duration":         duration,
		"writes_per_second": float64(mw.totalWrites) / duration.Seconds(),
		"last_write_time":  mw.lastWriteTime,
		"buffer_size":      len(mw.buffer),
		"connected":        mw.connected,
		"database":         mw.config.Database,
		"collection":       mw.config.Collection,
		"batch_size":       mw.config.BatchSize,
	}

	if mw.metadata.CollectionStats != nil {
		stats["collection_stats"] = mw.metadata.CollectionStats
	}

	return stats
}

// Helper functions

func buildWriteConcern(wc *WriteConcernOptions) *writeconcern.WriteConcern {
	var w writeconcern.WMode
	
	switch v := wc.W.(type) {
	case int:
		w = writeconcern.W(v)
	case string:
		if v == "majority" {
			w = writeconcern.WMajority()
		} else {
			w = writeconcern.WTagSet(v)
		}
	default:
		w = writeconcern.W(1) // Default
	}
	
	options := []writeconcern.Option{w}
	
	if wc.Journal != nil {
		if *wc.Journal {
			options = append(options, writeconcern.J(true))
		} else {
			options = append(options, writeconcern.J(false))
		}
	}
	
	if wc.WTimeout > 0 {
		options = append(options, writeconcern.WTimeout(wc.WTimeout))
	}
	
	return writeconcern.New(options...)
}

func buildReadPreference(rp string) (*options.ReadPreference, error) {
	switch strings.ToLower(rp) {
	case "primary":
		return options.ReadPreference().SetMode("primary"), nil
	case "primarypreferred":
		return options.ReadPreference().SetMode("primaryPreferred"), nil
	case "secondary":
		return options.ReadPreference().SetMode("secondary"), nil
	case "secondarypreferred":
		return options.ReadPreference().SetMode("secondaryPreferred"), nil
	case "nearest":
		return options.ReadPreference().SetMode("nearest"), nil
	default:
		return nil, fmt.Errorf("invalid read preference: %s", rp)
	}
}

func buildTLSConfig(tls *MongoTLSOptions) (*options.ClientEncryption, error) {
	// This is a simplified implementation
	// In a real implementation, you would build proper TLS configuration
	// MongoDB driver has specific methods for TLS configuration
	return nil, fmt.Errorf("TLS configuration not fully implemented")
}

func buildCredential(auth *MongoAuthOptions) *options.Credential {
	credential := &options.Credential{
		Username: auth.Username,
		Password: auth.Password,
	}
	
	if auth.Source != "" {
		credential.AuthSource = auth.Source
	}
	
	if auth.Mechanism != "" {
		credential.AuthMechanism = auth.Mechanism
	}
	
	if len(auth.MechanismProperties) > 0 {
		credential.AuthMechanismProperties = auth.MechanismProperties
	}
	
	return credential
}

// Utility functions for MongoDB operations

// EnsureIndex creates an index if it doesn't exist
func (mw *MongoDBWriter) EnsureIndex(keys bson.D, options *options.IndexOptions) error {
	indexModel := mongo.IndexModel{
		Keys:    keys,
		Options: options,
	}
	
	_, err := mw.collection.Indexes().CreateOne(mw.ctx, indexModel)
	return err
}

// DropIndex drops an index by name
func (mw *MongoDBWriter) DropIndex(indexName string) error {
	_, err := mw.collection.Indexes().DropOne(mw.ctx, indexName)
	return err
}

// Count returns the number of documents in the collection
func (mw *MongoDBWriter) Count(filter bson.M) (int64, error) {
	if filter == nil {
		filter = bson.M{}
	}
	return mw.collection.CountDocuments(mw.ctx, filter)
}

// FindOne finds a single document
func (mw *MongoDBWriter) FindOne(filter bson.M) (*mongo.SingleResult, error) {
	return mw.collection.FindOne(mw.ctx, filter), nil
}

// Find finds multiple documents
func (mw *MongoDBWriter) Find(filter bson.M, opts ...*options.FindOptions) (*mongo.Cursor, error) {
	return mw.collection.Find(mw.ctx, filter, opts...)
}

// Aggregate performs aggregation pipeline
func (mw *MongoDBWriter) Aggregate(pipeline bson.A, opts ...*options.AggregateOptions) (*mongo.Cursor, error) {
	return mw.collection.Aggregate(mw.ctx, pipeline, opts...)
}

// CreateTextIndex creates a text search index on specified fields
func (mw *MongoDBWriter) CreateTextIndex(fields ...string) error {
	keys := bson.D{}
	for _, field := range fields {
		keys = append(keys, bson.E{Key: field, Value: "text"})
	}
	
	indexOptions := options.Index().SetName("text_search_index")
	return mw.EnsureIndex(keys, indexOptions)
}

// Default MongoDB configuration
func GetDefaultMongoDBOptions() MongoDBOptions {
	return MongoDBOptions{
		BatchSize:       1000,
		RetryWrites:     true,
		RetryReads:      true,
		MaxPoolSize:     100,
		MinPoolSize:     5,
		MaxConnIdleTime: 10 * time.Minute,
		Timeout:         30 * time.Second,
		OnConflict:      ConflictIgnore,
		TimeStampField:  "created_at",
		IncludeMetadata: false,
		WriteConcern: &WriteConcernOptions{
			W:        "majority",
			Journal:  boolPtr(true),
			WTimeout: 5 * time.Second,
		},
		ReadPreference: "primary",
	}
}

// Helper function to create bool pointer
func boolPtr(b bool) *bool {
	return &b
}