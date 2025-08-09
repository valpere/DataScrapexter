package utils

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// Benchmark tests for validation functions, specifically addressing the TODO
// for performance benchmarking of countCharsOptimized function.

// Test data for various scenarios
var (
	// Pure ASCII strings of different lengths
	shortASCII  = "Hello"
	mediumASCII = "This is a medium length ASCII string for testing performance characteristics"
	longASCII   = strings.Repeat("The quick brown fox jumps over the lazy dog. ", 100)
	
	// Unicode strings of different lengths
	shortUnicode  = "Hello 世界"
	mediumUnicode = "This is a médium length string with ünıcödé characters for testing performance"
	longUnicode   = strings.Repeat("Unicode test: 世界 🌍 测试 🧪 ", 100)
	
	// Mixed content
	mixedContent = "ASCII text mixed with Unicode: 世界 🌍 and more ASCII content"
	
	// Edge cases
	emptyString    = ""
	singleASCII    = "A"
	singleUnicode  = "世"
	onlyEmoji      = "🌍🧪🚀💻🎯"
	complexMixed   = strings.Repeat("Mix: ABC 世界 🌍 123 ", 50)
)

// BenchmarkCountCharsOptimized benchmarks the optimized character counting function
func BenchmarkCountCharsOptimized(b *testing.B) {
	testCases := []struct {
		name string
		str  string
	}{
		{"ShortASCII", shortASCII},
		{"MediumASCII", mediumASCII},
		{"LongASCII", longASCII},
		{"ShortUnicode", shortUnicode},
		{"MediumUnicode", mediumUnicode},
		{"LongUnicode", longUnicode},
		{"MixedContent", mixedContent},
		{"EmptyString", emptyString},
		{"SingleASCII", singleASCII},
		{"SingleUnicode", singleUnicode},
		{"OnlyEmoji", onlyEmoji},
		{"ComplexMixed", complexMixed},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				countCharsOptimized(tc.str)
			}
		})
	}
}

// BenchmarkCountCharsStandard benchmarks the standard library approach
func BenchmarkCountCharsStandard(b *testing.B) {
	testCases := []struct {
		name string
		str  string
	}{
		{"ShortASCII", shortASCII},
		{"MediumASCII", mediumASCII},
		{"LongASCII", longASCII},
		{"ShortUnicode", shortUnicode},
		{"MediumUnicode", mediumUnicode},
		{"LongUnicode", longUnicode},
		{"MixedContent", mixedContent},
		{"EmptyString", emptyString},
		{"SingleASCII", singleASCII},
		{"SingleUnicode", singleUnicode},
		{"OnlyEmoji", onlyEmoji},
		{"ComplexMixed", complexMixed},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				countCharsStandard(tc.str)
			}
		})
	}
}

// BenchmarkCountCharsComparison benchmarks both functions side by side
func BenchmarkCountCharsComparison(b *testing.B) {
	testCases := []struct {
		name string
		str  string
	}{
		{"ASCII_Short", shortASCII},
		{"ASCII_Long", longASCII},
		{"Unicode_Short", shortUnicode},
		{"Unicode_Long", longUnicode},
		{"Mixed_Complex", complexMixed},
	}

	for _, tc := range testCases {
		b.Run(tc.name+"_Optimized", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				countCharsOptimized(tc.str)
			}
		})
		
		b.Run(tc.name+"_Standard", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				countCharsStandard(tc.str)
			}
		})
	}
}

// BenchmarkASCIIDetection benchmarks different approaches for ASCII detection
func BenchmarkASCIIDetection(b *testing.B) {
	testStrings := []string{shortASCII, mediumASCII, longASCII, shortUnicode, longUnicode}
	
	b.Run("ByteByByte", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, s := range testStrings {
				isASCII := true
				for j := 0; j < len(s); j++ {
					if s[j] >= 128 {
						isASCII = false
						break
					}
				}
				_ = isASCII
			}
		}
	})
	
	b.Run("ContainsAny", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, s := range testStrings {
				// Check if string contains non-ASCII characters
				isASCII := !strings.ContainsFunc(s, func(r rune) bool {
					return r >= 128
				})
				_ = isASCII
			}
		}
	})
	
	b.Run("UTF8Valid", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, s := range testStrings {
				isValid := utf8.ValidString(s)
				_ = isValid
			}
		}
	})
}

// BenchmarkValidationFunctions benchmarks other key validation functions
func BenchmarkValidationFunctions(b *testing.B) {
	testEmails := []string{
		"test@example.com",
		"user.name+tag@domain.co.uk",
		"invalid-email",
		"",
	}
	
	testURLs := []string{
		"https://www.example.com/path?query=value",
		"http://localhost:8080",
		"ftp://files.example.com/file.txt",
		"javascript:alert('xss')",
		"invalid-url",
		"",
	}
	
	testSelectors := []string{
		"div.class-name",
		"#id-name",
		"div > p.content",
		"input[type='text']",
		":hover",
		"::before",
		"div, span, p",
		"@import url('malicious.css')",
		"",
	}
	
	b.Run("IsValidEmail", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, email := range testEmails {
				IsValidEmail(email)
			}
		}
	})
	
	b.Run("IsValidURL", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, url := range testURLs {
				IsValidURL(url)
			}
		}
	})
	
	b.Run("SelectorValidator", func(b *testing.B) {
		validator := &SelectorValidator{Required: false, Strict: true}
		for i := 0; i < b.N; i++ {
			for _, selector := range testSelectors {
				validator.Validate(selector)
			}
		}
	})
}

// BenchmarkRegexCompilation tests the impact of regex compilation
func BenchmarkRegexCompilation(b *testing.B) {
	testStrings := []string{"test123", "HELLO", "hello123world", "!@#$%", "αβγ"}
	
	b.Run("PreCompiledAlpha", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, s := range testStrings {
				IsAlpha(s)
			}
		}
	})
	
	b.Run("PreCompiledAlphaNumeric", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, s := range testStrings {
				IsAlphaNumeric(s)
			}
		}
	})
}

// BenchmarkStructValidation benchmarks struct validation performance
func BenchmarkStructValidation(b *testing.B) {
	type TestStruct struct {
		Name     string `validate:"required,min=3,max=50"`
		Email    string `validate:"required,email"`
		URL      string `validate:"url"`
		Age      int    `validate:"numeric,min=0,max=120"`
		Optional string `validate:"max=100"`
	}
	
	validStruct := TestStruct{
		Name:     "John Doe",
		Email:    "john@example.com",
		URL:      "https://example.com",
		Age:      30,
		Optional: "Some optional text",
	}
	
	invalidStruct := TestStruct{
		Name:     "Jo", // Too short
		Email:    "invalid-email",
		URL:      "not-a-url",
		Age:      -5, // Negative age
		Optional: strings.Repeat("x", 200), // Too long
	}
	
	b.Run("ValidStruct", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			ValidateStruct(validStruct, nil)
		}
	})
	
	b.Run("InvalidStruct", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			ValidateStruct(invalidStruct, nil)
		}
	})
}

// Helper function to measure memory allocations
func BenchmarkMemoryAllocation(b *testing.B) {
	testString := longASCII
	
	b.Run("CountCharsOptimized", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			countCharsOptimized(testString)
		}
	})
	
	b.Run("CountCharsStandard", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			countCharsStandard(testString)
		}
	})
}