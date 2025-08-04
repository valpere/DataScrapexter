// Package utils provides enhanced type safety utilities
// for the DataScrapexter platform.
package utils

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Platform-specific integer limits
const (
	maxInt = int(^uint(0) >> 1)
	minInt = -maxInt - 1
)

// TypeConverter provides safe type conversion utilities
type TypeConverter struct {
	strict bool // If true, conversions that lose precision will fail
}

// NewTypeConverter creates a new type converter
func NewTypeConverter(strict bool) *TypeConverter {
	return &TypeConverter{strict: strict}
}

// ConversionResult represents the result of a type conversion
type ConversionResult struct {
	Value   interface{} `json:"value"`
	Success bool        `json:"success"`
	Error   error       `json:"error,omitempty"`
	Warning string      `json:"warning,omitempty"`
}

// ToString safely converts a value to string
func (tc *TypeConverter) ToString(value interface{}) ConversionResult {
	if value == nil {
		return ConversionResult{Value: "", Success: true}
	}

	switch v := value.(type) {
	case string:
		return ConversionResult{Value: v, Success: true}
	case int, int8, int16, int32, int64:
		return ConversionResult{Value: fmt.Sprintf("%d", v), Success: true}
	case uint, uint8, uint16, uint32, uint64:
		return ConversionResult{Value: fmt.Sprintf("%d", v), Success: true}
	case float32, float64:
		return ConversionResult{Value: fmt.Sprintf("%g", v), Success: true}
	case bool:
		return ConversionResult{Value: fmt.Sprintf("%t", v), Success: true}
	case time.Time:
		return ConversionResult{Value: v.Format(time.RFC3339), Success: true}
	case fmt.Stringer:
		return ConversionResult{Value: v.String(), Success: true}
	default:
		// Try JSON marshaling as last resort
		if data, err := json.Marshal(v); err == nil {
			return ConversionResult{
				Value:   string(data),
				Success: true,
				Warning: "Converted complex type using JSON marshaling",
			}
		}
		return ConversionResult{
			Value:   fmt.Sprintf("%v", v),
			Success: false,
			Error:   fmt.Errorf("unable to safely convert %T to string", v),
		}
	}
}

// ToInt safely converts a value to int
func (tc *TypeConverter) ToInt(value interface{}) ConversionResult {
	if value == nil {
		return ConversionResult{Value: 0, Success: true}
	}

	switch v := value.(type) {
	case int:
		return ConversionResult{Value: v, Success: true}
	case int8:
		return ConversionResult{Value: int(v), Success: true}
	case int16:
		return ConversionResult{Value: int(v), Success: true}
	case int32:
		return ConversionResult{Value: int(v), Success: true}
	case int64:
		if tc.strict && (v > int64(maxInt) || v < int64(minInt)) {
			return ConversionResult{
				Value:   0,
				Success: false,
				Error:   fmt.Errorf("int64 value %d would overflow int", v),
			}
		}
		return ConversionResult{Value: int(v), Success: true}
	case uint, uint8, uint16, uint32:
		return ConversionResult{Value: int(reflect.ValueOf(v).Uint()), Success: true}
	case uint64:
		if tc.strict && v > uint64(int(^uint(0)>>1)) {
			return ConversionResult{
				Value:   0,
				Success: false,
				Error:   fmt.Errorf("uint64 value %d would overflow int", v),
			}
		}
		return ConversionResult{Value: int(v), Success: true}
	case float32:
		if tc.strict && (v != float32(int(v))) {
			return ConversionResult{
				Value:   0,
				Success: false,
				Error:   fmt.Errorf("float32 value %f would lose precision when converted to int", v),
			}
		}
		return ConversionResult{Value: int(v), Success: true}
	case float64:
		if tc.strict && (v != float64(int(v))) {
			return ConversionResult{
				Value:   0,
				Success: false,
				Error:   fmt.Errorf("float64 value %f would lose precision when converted to int", v),
			}
		}
		return ConversionResult{Value: int(v), Success: true}
	case string:
		if val, err := strconv.Atoi(v); err == nil {
			return ConversionResult{Value: val, Success: true}
		} else {
			return ConversionResult{
				Value:   0,
				Success: false,
				Error:   fmt.Errorf("cannot convert string %q to int: %w", v, err),
			}
		}
	case bool:
		if v {
			return ConversionResult{Value: 1, Success: true}
		}
		return ConversionResult{Value: 0, Success: true}
	default:
		return ConversionResult{
			Value:   0,
			Success: false,
			Error:   fmt.Errorf("cannot convert %T to int", v),
		}
	}
}

// ToFloat64 safely converts a value to float64
func (tc *TypeConverter) ToFloat64(value interface{}) ConversionResult {
	if value == nil {
		return ConversionResult{Value: 0.0, Success: true}
	}

	switch v := value.(type) {
	case float64:
		return ConversionResult{Value: v, Success: true}
	case float32:
		return ConversionResult{Value: float64(v), Success: true}
	case int, int8, int16, int32, int64:
		return ConversionResult{Value: float64(reflect.ValueOf(v).Int()), Success: true}
	case uint, uint8, uint16, uint32, uint64:
		return ConversionResult{Value: float64(reflect.ValueOf(v).Uint()), Success: true}
	case string:
		if val, err := strconv.ParseFloat(v, 64); err == nil {
			return ConversionResult{Value: val, Success: true}
		} else {
			return ConversionResult{
				Value:   0.0,
				Success: false,
				Error:   fmt.Errorf("cannot convert string %q to float64: %w", v, err),
			}
		}
	case bool:
		if v {
			return ConversionResult{Value: 1.0, Success: true}
		}
		return ConversionResult{Value: 0.0, Success: true}
	default:
		return ConversionResult{
			Value:   0.0,
			Success: false,
			Error:   fmt.Errorf("cannot convert %T to float64", v),
		}
	}
}

// ToBool safely converts a value to bool
func (tc *TypeConverter) ToBool(value interface{}) ConversionResult {
	if value == nil {
		return ConversionResult{Value: false, Success: true}
	}

	switch v := value.(type) {
	case bool:
		return ConversionResult{Value: v, Success: true}
	case int, int8, int16, int32, int64:
		return ConversionResult{Value: reflect.ValueOf(v).Int() != 0, Success: true}
	case uint, uint8, uint16, uint32, uint64:
		return ConversionResult{Value: reflect.ValueOf(v).Uint() != 0, Success: true}
	case float32, float64:
		return ConversionResult{Value: reflect.ValueOf(v).Float() != 0, Success: true}
	case string:
		switch strings.ToLower(strings.TrimSpace(v)) {
		case "true", "yes", "1", "on", "enabled":
			return ConversionResult{Value: true, Success: true}
		case "false", "no", "0", "off", "disabled", "":
			return ConversionResult{Value: false, Success: true}
		default:
			return ConversionResult{
				Value:   false,
				Success: false,
				Error:   fmt.Errorf("cannot convert string %q to bool", v),
			}
		}
	default:
		return ConversionResult{
			Value:   false,
			Success: false,
			Error:   fmt.Errorf("cannot convert %T to bool", v),
		}
	}
}

// ToTime safely converts a value to time.Time
func (tc *TypeConverter) ToTime(value interface{}) ConversionResult {
	if value == nil {
		return ConversionResult{Value: time.Time{}, Success: true}
	}

	switch v := value.(type) {
	case time.Time:
		return ConversionResult{Value: v, Success: true}
	case string:
		// Try common time formats
		formats := []string{
			time.RFC3339,
			time.RFC3339Nano,
			"2006-01-02 15:04:05",
			"2006-01-02",
			"15:04:05",
			"2006/01/02",
			"01/02/2006",
			"Jan 2, 2006",
			"January 2, 2006",
		}
		
		for _, format := range formats {
			if t, err := time.Parse(format, v); err == nil {
				return ConversionResult{Value: t, Success: true}
			}
		}
		
		return ConversionResult{
			Value:   time.Time{},
			Success: false,
			Error:   fmt.Errorf("cannot parse time string %q", v),
		}
	case int64:
		// Assume Unix timestamp
		return ConversionResult{Value: time.Unix(v, 0), Success: true}
	default:
		return ConversionResult{
			Value:   time.Time{},
			Success: false,
			Error:   fmt.Errorf("cannot convert %T to time.Time", v),
		}
	}
}

// TypeGuard provides runtime type checking utilities
type TypeGuard struct{}

// NewTypeGuard creates a new type guard
func NewTypeGuard() *TypeGuard {
	return &TypeGuard{}
}

// IsString checks if a value is a string
func (tg *TypeGuard) IsString(value interface{}) bool {
	_, ok := value.(string)
	return ok
}

// IsNumeric checks if a value is any numeric type
func (tg *TypeGuard) IsNumeric(value interface{}) bool {
	switch value.(type) {
	case int, int8, int16, int32, int64:
		return true
	case uint, uint8, uint16, uint32, uint64:
		return true
	case float32, float64:
		return true
	default:
		return false
	}
}

// IsInteger checks if a value is an integer type
func (tg *TypeGuard) IsInteger(value interface{}) bool {
	switch value.(type) {
	case int, int8, int16, int32, int64:
		return true
	case uint, uint8, uint16, uint32, uint64:
		return true
	default:
		return false
	}
}

// IsFloat checks if a value is a floating-point type
func (tg *TypeGuard) IsFloat(value interface{}) bool {
	switch value.(type) {
	case float32, float64:
		return true
	default:
		return false
	}
}

// IsBool checks if a value is a boolean
func (tg *TypeGuard) IsBool(value interface{}) bool {
	_, ok := value.(bool)
	return ok
}

// IsNil checks if a value is nil
func (tg *TypeGuard) IsNil(value interface{}) bool {
	return value == nil
}

// IsEmpty checks if a value is considered "empty"
func (tg *TypeGuard) IsEmpty(value interface{}) bool {
	if value == nil {
		return true
	}
	
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.String:
		return v.String() == ""
	case reflect.Slice, reflect.Array, reflect.Map, reflect.Chan:
		return v.Len() == 0
	case reflect.Ptr, reflect.Interface:
		return v.IsNil()
	default:
		return false
	}
}

// GetTypeName returns a human-readable type name
func (tg *TypeGuard) GetTypeName(value interface{}) string {
	if value == nil {
		return "nil"
	}
	
	t := reflect.TypeOf(value)
	if t.Kind() == reflect.Ptr {
		return "*" + t.Elem().Name()
	}
	return t.Name()
}

// Optional represents an optional value that may or may not be present
type Optional[T any] struct {
	value   T
	present bool
}

// Some creates an Optional with a value
func Some[T any](value T) Optional[T] {
	return Optional[T]{value: value, present: true}
}

// None creates an empty Optional
func None[T any]() Optional[T] {
	return Optional[T]{present: false}
}

// IsPresent returns true if the Optional contains a value
func (o Optional[T]) IsPresent() bool {
	return o.present
}

// IsEmpty returns true if the Optional is empty
func (o Optional[T]) IsEmpty() bool {
	return !o.present
}

// Get returns the value if present, otherwise the zero value
func (o Optional[T]) Get() T {
	return o.value
}

// GetOrElse returns the value if present, otherwise the provided default
func (o Optional[T]) GetOrElse(defaultValue T) T {
	if o.present {
		return o.value
	}
	return defaultValue
}

// IfPresent executes a function if the Optional contains a value
func (o Optional[T]) IfPresent(fn func(T)) {
	if o.present {
		fn(o.value)
	}
}

// Map transforms the Optional value if present
func Map[T, U any](o Optional[T], fn func(T) U) Optional[U] {
	if o.present {
		return Some(fn(o.value))
	}
	return None[U]()
}

// Filter returns the Optional if the predicate is satisfied, otherwise None
func (o Optional[T]) Filter(predicate func(T) bool) Optional[T] {
	if o.present && predicate(o.value) {
		return o
	}
	return None[T]()
}

// Result represents a value that can be either successful or an error
type Result[T any] struct {
	value T
	err   error
}

// Ok creates a successful Result
func Ok[T any](value T) Result[T] {
	return Result[T]{value: value}
}

// Err creates an error Result
func Err[T any](err error) Result[T] {
	return Result[T]{err: err}
}

// IsOk returns true if the Result is successful
func (r Result[T]) IsOk() bool {
	return r.err == nil
}

// IsErr returns true if the Result contains an error
func (r Result[T]) IsErr() bool {
	return r.err != nil
}

// Unwrap returns the value or panics if there's an error
func (r Result[T]) Unwrap() T {
	if r.err != nil {
		panic(r.err)
	}
	return r.value
}

// UnwrapOr returns the value if successful, otherwise the provided default
func (r Result[T]) UnwrapOr(defaultValue T) T {
	if r.err != nil {
		return defaultValue
	}
	return r.value
}

// Error returns the error if present
func (r Result[T]) Error() error {
	return r.err
}

// MapResult transforms a successful Result value
func MapResult[T, U any](r Result[T], fn func(T) U) Result[U] {
	if r.err != nil {
		return Err[U](r.err)
	}
	return Ok(fn(r.value))
}

// MapError transforms a Result error
func (r Result[T]) MapError(fn func(error) error) Result[T] {
	if r.err != nil {
		return Err[T](fn(r.err))
	}
	return r
}

// AndThen chains Results (flatMap)
func AndThen[T, U any](r Result[T], fn func(T) Result[U]) Result[U] {
	if r.err != nil {
		return Err[U](r.err)
	}
	return fn(r.value)
}

// Recover allows error recovery with a fallback function
func (r Result[T]) Recover(fn func(error) T) T {
	if r.err != nil {
		return fn(r.err)
	}
	return r.value
}

// TypeSafeMap provides type-safe map operations
type TypeSafeMap[K comparable, V any] struct {
	data map[K]V
	mutex sync.RWMutex
}

// NewTypeSafeMap creates a new type-safe map
func NewTypeSafeMap[K comparable, V any]() *TypeSafeMap[K, V] {
	return &TypeSafeMap[K, V]{
		data: make(map[K]V),
	}
}

// Set sets a key-value pair
func (tsm *TypeSafeMap[K, V]) Set(key K, value V) {
	tsm.mutex.Lock()
	defer tsm.mutex.Unlock()
	tsm.data[key] = value
}

// Get gets a value by key
func (tsm *TypeSafeMap[K, V]) Get(key K) Optional[V] {
	tsm.mutex.RLock()
	defer tsm.mutex.RUnlock()
	
	if value, exists := tsm.data[key]; exists {
		return Some(value)
	}
	return None[V]()
}

// Delete removes a key-value pair
func (tsm *TypeSafeMap[K, V]) Delete(key K) bool {
	tsm.mutex.Lock()
	defer tsm.mutex.Unlock()
	
	if _, exists := tsm.data[key]; exists {
		delete(tsm.data, key)
		return true
	}
	return false
}

// Contains checks if a key exists
func (tsm *TypeSafeMap[K, V]) Contains(key K) bool {
	tsm.mutex.RLock()
	defer tsm.mutex.RUnlock()
	
	_, exists := tsm.data[key]
	return exists
}

// Size returns the number of items in the map
func (tsm *TypeSafeMap[K, V]) Size() int {
	tsm.mutex.RLock()
	defer tsm.mutex.RUnlock()
	return len(tsm.data)
}

// Keys returns all keys
func (tsm *TypeSafeMap[K, V]) Keys() []K {
	tsm.mutex.RLock()
	defer tsm.mutex.RUnlock()
	
	keys := make([]K, 0, len(tsm.data))
	for k := range tsm.data {
		keys = append(keys, k)
	}
	return keys
}

// Values returns all values
func (tsm *TypeSafeMap[K, V]) Values() []V {
	tsm.mutex.RLock()
	defer tsm.mutex.RUnlock()
	
	values := make([]V, 0, len(tsm.data))
	for _, v := range tsm.data {
		values = append(values, v)
	}
	return values
}

// Clear removes all items
func (tsm *TypeSafeMap[K, V]) Clear() {
	tsm.mutex.Lock()
	defer tsm.mutex.Unlock()
	tsm.data = make(map[K]V)
}

// ForEach iterates over all key-value pairs
func (tsm *TypeSafeMap[K, V]) ForEach(fn func(K, V)) {
	tsm.mutex.RLock()
	defer tsm.mutex.RUnlock()
	
	for k, v := range tsm.data {
		fn(k, v)
	}
}
