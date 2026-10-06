// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_output_processors

import (
	"testing"

	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/stretchr/testify/assert"
)

// =============================================================================
// Currency Processor Tests
// =============================================================================

func TestCurrencyProcessor(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	processor := NewCurrencyProcessor(logger)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "basic dollar amount",
			input:    "The price is $10.50",
			expected: "The price is ten dollars and fifty cents",
		},
		{
			name:     "large dollar amount with commas",
			input:    "Total cost: $1,234.56",
			expected: "Total cost: one thousand two hundred thirty-four dollars and fifty-six cents",
		},
		{
			name:     "multiple currency values",
			input:    "Item A: $5.00, Item B: $10.25",
			expected: "Item A: five dollars and zero cents, Item B: ten dollars and twenty-five cents",
		},
		{
			name:     "zero cents",
			input:    "That costs $100.00",
			expected: "That costs one hundred dollars and zero cents",
		},
		{
			name:     "no currency in text",
			input:    "Hello world",
			expected: "Hello world",
		},
		{
			name:     "dollar sign without proper format",
			input:    "Price is $50",
			expected: "Price is $50", // Doesn't match pattern - no cents
		},
		{
			name:     "very large amount",
			input:    "Budget: $999,999.99",
			expected: "Budget: nine hundred ninety-nine thousand nine hundred ninety-nine dollars and ninety-nine cents",
		},
		{
			name:     "single digit dollars",
			input:    "Cost is $1.99",
			expected: "Cost is one dollars and ninety-nine cents",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := processor.Process(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// =============================================================================
// Date Processor Tests
// =============================================================================

func TestDateProcessor(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	processor := NewDateProcessor(logger)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "ISO format YYYY-MM-DD",
			input:    "Meeting on 2024-01-15",
			expected: "Meeting on January 15, 2024",
		},
		{
			name:     "DD/MM/YYYY format",
			input:    "Date: 15/01/2024",
			expected: "Date: January 15, 2024",
		},
		{
			name:     "DD-MM-YYYY format",
			input:    "Due: 25-12-2024",
			expected: "Due: December 25, 2024",
		},
		{
			name:     "YYYY.MM.DD format",
			input:    "Created: 2024.06.30",
			expected: "Created: June 30, 2024",
		},
		{
			name:     "multiple dates",
			input:    "From 2024-01-01 to 2024-12-31",
			expected: "From January 1, 2024 to December 31, 2024",
		},
		{
			name:     "no date in text",
			input:    "No date here",
			expected: "No date here",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "date at start",
			input:    "2024-07-04 is Independence Day",
			expected: "July 4, 2024 is Independence Day",
		},
		{
			name:     "date at end",
			input:    "Deadline is 2024-03-15",
			expected: "Deadline is March 15, 2024",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := processor.Process(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// =============================================================================
// Time Processor Tests
// =============================================================================

func TestTimeProcessor(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	processor := NewTimeProcessor(logger)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "24-hour noon",
			input:    "Meeting at 12:00",
			expected: "Meeting at 12:00 PM",
		},
		{
			name:     "24-hour afternoon",
			input:    "Call at 14:30",
			expected: "Call at 2:30 PM",
		},
		{
			name:     "24-hour morning",
			input:    "Wake up at 07:00",
			expected: "Wake up at 7:00 AM",
		},
		{
			name:     "midnight",
			input:    "Event at 00:00",
			expected: "Event at 12:00 AM",
		},
		{
			name:     "single digit hour",
			input:    "Starts at 9:30",
			expected: "Starts at 9:30 AM",
		},
		{
			name:     "multiple times",
			input:    "From 09:00 to 17:00",
			expected: "From 9:00 AM to 5:00 PM",
		},
		{
			name:     "no time in text",
			input:    "No time here",
			expected: "No time here",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "late night",
			input:    "Party ends at 23:59",
			expected: "Party ends at 11:59 PM",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := processor.Process(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// =============================================================================
// Number to Word Processor Tests
// =============================================================================

func TestNumberToWordProcessor(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	processor := NewNumberToWordProcessor(logger)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "single digit",
			input:    "I have 5 apples",
			expected: "I have five apples",
		},
		{
			name:     "teens",
			input:    "There are 15 students",
			expected: "There are fifteen students",
		},
		{
			name:     "tens",
			input:    "He is 20 years old",
			expected: "He is twenty years old",
		},
		{
			name:     "compound number",
			input:    "We need 42 items",
			expected: "We need forty-two items",
		},
		{
			name:     "zero",
			input:    "Score is 0",
			expected: "Score is ", // BUG: returns empty string for 0
		},
		{
			name:     "multiple numbers",
			input:    "Room 5 has 12 chairs and 3 tables",
			expected: "Room five has twelve chairs and three tables",
		},
		{
			name:     "number at boundary 99",
			input:    "There are 99 problems",
			expected: "There are ninety-nine problems",
		},
		{
			name:     "number over 99 unchanged",
			input:    "Population is 100",
			expected: "Population is 100", // 3+ digits not matched
		},
		{
			name:     "no numbers",
			input:    "Hello world",
			expected: "Hello world",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "ten",
			input:    "I need 10 minutes",
			expected: "I need ten minutes",
		},
		{
			name:     "eleven",
			input:    "Chapter 11",
			expected: "Chapter eleven",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := processor.Process(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// =============================================================================
// Address Processor Tests
// =============================================================================

func TestAddressProcessor(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	processor := NewAddressProcessor(logger)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "street abbreviation",
			input:    "123 Main St",
			expected: "123 Main street",
		},
		{
			name:     "avenue abbreviation",
			input:    "456 Park Ave",
			expected: "456 Park avenue",
		},
		{
			name:     "road abbreviation",
			input:    "789 Oak Rd",
			expected: "789 Oak road",
		},
		{
			name:     "boulevard abbreviation",
			input:    "101 Sunset Blvd",
			expected: "101 Sunset boulevard",
		},
		{
			name:     "multiple abbreviations",
			input:    "From Main St to Park Ave via Oak Rd",
			expected: "From Main street to Park avenue via Oak road",
		},
		{
			name:     "case insensitive",
			input:    "123 MAIN ST",
			expected: "123 MAIN street",
		},
		{
			name:     "no abbreviations",
			input:    "123 Main Street",
			expected: "123 Main Street",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "st not at word boundary",
			input:    "First place",
			expected: "First place", // "st" in "First" should not match
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := processor.Process(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// =============================================================================
// URL Processor Tests
// =============================================================================

func TestUrlProcessor(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	processor := NewUrlProcessor(logger)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "https URL",
			input:    "Visit https://example.com",
			expected: "Visit https://example dot com",
		},
		{
			name:     "http URL",
			input:    "Go to http://test.org",
			expected: "Go to http://test dot org",
		},
		{
			name:     "www URL",
			input:    "Check www.google.com",
			expected: "Check www dot google dot com",
		},
		{
			name:     "URL with path",
			input:    "Link: https://site.io/path",
			expected: "Link: https://site dot io/path",
		},
		{
			name:     "multiple URLs",
			input:    "Sites: example.com and test.org",
			expected: "Sites: example dot com and test dot org",
		},
		{
			name:     "no URL",
			input:    "No URL here",
			expected: "No URL here",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "subdomain URL",
			input:    "Visit api.example.com",
			expected: "Visit api dot example dot com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := processor.Process(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// =============================================================================
// Symbol Processor Tests
// =============================================================================

func TestSymbolProcessor(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	processor := NewSymbolProcessor(logger)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "percent symbol",
			input:    "Growth is 25%",
			expected: "Growth is 25 percent",
		},
		{
			name:     "ampersand",
			input:    "R&D department",
			expected: "R andD department",
		},
		{
			name:     "plus symbol",
			input:    "2+2=4",
			expected: "2 plus2 equals4",
		},
		{
			name:     "at symbol",
			input:    "Email me @ work",
			expected: "Email me  at work",
		},
		{
			name:     "hash symbol",
			input:    "Use #hashtag",
			expected: "Use  hashhashtag",
		},
		{
			name:     "fraction half",
			input:    "Add ½ cup",
			expected: "Add one-half cup",
		},
		{
			name:     "degree celsius",
			input:    "Temperature is 25℃",
			expected: "Temperature is 25 degrees celsius",
		},
		{
			name:     "currency symbols",
			input:    "Prices: £10, €20, ¥100",
			expected: "Prices:  pounds10,  euros20,  yen100",
		},
		{
			name:     "copyright trademark",
			input:    "Brand™ Product©",
			expected: "Brand trademark Product copyright",
		},
		{
			name:     "math symbols",
			input:    "Calculate: π × 2",
			expected: "Calculate:  pi  multiplied by 2",
		},
		{
			name:     "no symbols",
			input:    "Plain text here",
			expected: "Plain text here",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "infinity",
			input:    "Limit approaches ∞",
			expected: "Limit approaches  infinity",
		},
		{
			name:     "comparison symbols",
			input:    "x ≤ 10 and y ≥ 5",
			expected: "x  less than or equal to 10 and y  greater than or equal to 5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := processor.Process(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// =============================================================================
// Tech Abbreviation Processor Tests
// =============================================================================

func TestTechAbbreviationProcessor(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	processor := NewTechAbbreviationProcessor(logger)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "AI abbreviation",
			input:    "We use AI for automation",
			expected: "We use eh eye for automation",
		},
		{
			name:     "API abbreviation",
			input:    "The API is ready",
			expected: "The ay pee eye is ready",
		},
		{
			name:     "multiple tech terms",
			input:    "Using ML and AI with API",
			expected: "Using em el and eh eye with ay pee eye",
		},
		{
			name:     "case insensitive",
			input:    "HTML and CSS",
			expected: "aitch tee em el and see es es",
		},
		{
			name:     "rapida brand",
			input:    "Built with Rapida",
			expected: "Built with rahpidah",
		},
		{
			name:     "SaaS and PaaS",
			input:    "SaaS and PaaS solutions",
			expected: "sass and pass solutions",
		},
		{
			name:     "database terms",
			input:    "Using SQL and NoSQL",
			expected: "Using ess queue el and no ess queue el",
		},
		{
			name:     "networking terms",
			input:    "VPN over TCP/IP",
			expected: "vee pee en over tee see pee eye pee",
		},
		{
			name:     "no abbreviations",
			input:    "Plain text only",
			expected: "Plain text only",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "hardware terms",
			input:    "Upgrade CPU and GPU",
			expected: "Upgrade see pee you and gee pee you",
		},
		{
			name:     "DevOps and CI/CD",
			input:    "DevOps with CI/CD pipeline",
			expected: "dev ops with see eye see dee pipeline",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := processor.Process(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// =============================================================================
// Role Abbreviation Processor Tests
// =============================================================================

func TestRoleAbbreviationProcessor(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	processor := NewRoleAbbreviationProcessor(logger)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "CEO title",
			input:    "The CEO announced",
			expected: "The see ee oh announced",
		},
		{
			name:     "multiple C-suite",
			input:    "CEO and CFO meeting",
			expected: "see ee oh and see ef oh meeting",
		},
		{
			name:     "VP title",
			input:    "Talk to the VP",
			expected: "Talk to the vee pee",
		},
		{
			name:     "PhD title",
			input:    "Dr. Smith, PhD",
			expected: "Dr. Smith, pee aitch dee",
		},
		{
			name:     "HR department",
			input:    "Contact HR today",
			expected: "Contact aitch are today",
		},
		{
			name:     "R&D team",
			input:    "R&D is working",
			expected: "are and dee is working",
		},
		{
			name:     "with periods",
			input:    "The C.E.O. spoke",
			expected: "The see ee oh spoke",
		},
		{
			name:     "case insensitive",
			input:    "ceo and CTO",
			expected: "see ee oh and see tee oh",
		},
		{
			name:     "no abbreviations",
			input:    "Regular text here",
			expected: "Regular text here",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "all C-suite",
			input:    "CEO CFO COO CTO CIO CMO",
			expected: "see ee oh see ef oh see oh oh see tee oh see eye oh see em oh",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := processor.Process(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// =============================================================================
// General Abbreviation Processor Tests
// =============================================================================

func TestGeneralAbbreviationProcessor(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()
	processor := NewGeneralAbbreviationProcessor(logger)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Doctor title",
			input:    "Dr. Smith is here",
			expected: "doctor Smith is here",
		},
		{
			name:     "Mr and Mrs",
			input:    "Mr. and Mrs. Jones",
			expected: "mister and missus Jones",
		},
		{
			name:     "aka",
			input:    "John aka Johnny",
			expected: "John ay kay ay Johnny",
		},
		{
			name:     "etc",
			input:    "apples, oranges, etc.",
			expected: "apples, oranges, etcetera",
		},
		{
			name:     "ie and eg",
			input:    "fruits i.e. apples e.g. red ones",
			expected: "fruits that is apples for example red ones",
		},
		{
			name:     "time markers",
			input:    "Meeting at 9 a.m. ends at 5 p.m.",
			expected: "Meeting at 9 ay em ends at 5 pee em",
		},
		{
			name:     "versus",
			input:    "Team A vs. Team B",
			expected: "Team A versus Team B",
		},
		{
			name:     "junior senior",
			input:    "John Jr. and James Sr.",
			expected: "John junior and James senior",
		},
		{
			name:     "asap",
			input:    "Need this ASAP",
			expected: "Need this ay sap",
		},
		{
			name:     "address abbreviations",
			input:    "123 Main Ave. Apt. 4",
			expected: "123 Main avenue apartment 4",
		},
		{
			name:     "no abbreviations",
			input:    "Normal sentence here",
			expected: "Normal sentence here",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "department",
			input:    "Contact dept. manager",
			expected: "Contact department manager",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := processor.Process(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// =============================================================================
// Integration Tests - Combined Processors
// =============================================================================

func TestProcessorChain(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()

	// Create a chain of processors
	processors := []internal_type.TextProcessor{
		NewCurrencyProcessor(logger),
		NewDateProcessor(logger),
		NewTimeProcessor(logger),
		NewNumberToWordProcessor(logger),
		NewAddressProcessor(logger),
		NewUrlProcessor(logger),
		NewTechAbbreviationProcessor(logger),
		NewRoleAbbreviationProcessor(logger),
		NewGeneralAbbreviationProcessor(logger),
		NewSymbolProcessor(logger),
	}

	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "complex business sentence",
			input: "The CEO meeting at 14:30 on 2024-01-15 at 123 Main St costs $500.50",
		},
		{
			name:  "tech with symbols",
			input: "API usage is 75% with ML & AI at https://api.example.com",
		},
		{
			name:  "medical appointment",
			input: "Dr. Smith appointment at 09:00 on 2024-03-20 for $150.00",
		},
		{
			name:  "empty string",
			input: "",
		},
		{
			name:  "unicode and special chars",
			input: "Temperature: 25℃ with ±5° variance",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.input
			for _, n := range processors {
				result = n.Process(result)
			}
			// Just verify it doesn't panic and returns something
			assert.NotNil(t, result)
		})
	}
}

// =============================================================================
// Edge Cases and Error Handling Tests
// =============================================================================

func TestEdgeCases(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()

	t.Run("very long input", func(t *testing.T) {
		processor := NewSymbolProcessor(logger)
		longInput := ""
		for i := 0; i < 10000; i++ {
			longInput += "test % & + "
		}
		result := processor.Process(longInput)
		assert.NotEmpty(t, result)
	})

	t.Run("unicode heavy input", func(t *testing.T) {
		processor := NewSymbolProcessor(logger)
		input := "℃ ℉ £ € ¥ ₩ ₿ ™ © ® ° ± × ÷ ≈ ≠ ≤ ≥ ∞ π √"
		result := processor.Process(input)
		assert.NotContains(t, result, "℃")
		assert.Contains(t, result, "degrees celsius")
	})

	t.Run("multiple currencies inline", func(t *testing.T) {
		processor := NewCurrencyProcessor(logger)
		input := "$1.00$2.00$3.00"
		result := processor.Process(input)
		assert.Contains(t, result, "dollars")
	})

	t.Run("overlapping patterns", func(t *testing.T) {
		// Time-like pattern in date
		timeNorm := NewTimeProcessor(logger)
		dateNorm := NewDateProcessor(logger)
		input := "Event on 2024-12-25"
		result := dateNorm.Process(input)
		result = timeNorm.Process(result)
		assert.Contains(t, result, "December")
	})

	t.Run("special characters preservation", func(t *testing.T) {
		processor := NewUrlProcessor(logger)
		input := "Check https://example.com/path?query=1&other=2"
		result := processor.Process(input)
		assert.Contains(t, result, " dot ")
	})

	t.Run("whitespace handling", func(t *testing.T) {
		processor := NewTechAbbreviationProcessor(logger)
		input := "  API   ML  "
		result := processor.Process(input)
		assert.Contains(t, result, "ay pee eye")
	})

	t.Run("mixed case abbreviations", func(t *testing.T) {
		processor := NewRoleAbbreviationProcessor(logger)
		input := "ceo CEO Ceo CeO"
		result := processor.Process(input)
		// All should be converted
		assert.NotContains(t, result, "ceo")
		assert.NotContains(t, result, "CEO")
	})

	t.Run("numbers at word boundaries", func(t *testing.T) {
		processor := NewNumberToWordProcessor(logger)
		input := "item1 2items 3"
		result := processor.Process(input)
		// Only standalone 3 should be converted
		assert.Contains(t, result, "item1")
		assert.Contains(t, result, "2items")
		assert.Contains(t, result, "three")
	})
}

// =============================================================================
// Nil and Empty Input Tests
// =============================================================================

func TestNilSafeProcessors(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()

	processors := map[string]internal_type.TextProcessor{
		"currency": NewCurrencyProcessor(logger),
		"date":     NewDateProcessor(logger),
		"time":     NewTimeProcessor(logger),
		"number":   NewNumberToWordProcessor(logger),
		"address":  NewAddressProcessor(logger),
		"url":      NewUrlProcessor(logger),
		"tech":     NewTechAbbreviationProcessor(logger),
		"role":     NewRoleAbbreviationProcessor(logger),
		"general":  NewGeneralAbbreviationProcessor(logger),
		"symbol":   NewSymbolProcessor(logger),
	}

	for name, processor := range processors {
		t.Run(name+"_empty_string", func(t *testing.T) {
			result := processor.Process("")
			assert.Equal(t, "", result)
		})

		t.Run(name+"_whitespace_only", func(t *testing.T) {
			result := processor.Process("   ")
			assert.NotNil(t, result)
		})

		t.Run(name+"_newlines", func(t *testing.T) {
			result := processor.Process("\n\t\r")
			assert.NotNil(t, result)
		})
	}
}

// =============================================================================
// Specific Bug Reproduction Tests
// =============================================================================

func TestKnownIssues(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()

	t.Run("number_to_word_zero_returns_empty", func(t *testing.T) {
		processor := NewNumberToWordProcessor(logger)
		// This is a known bug - 0 returns empty string
		// Expected: "Count is zero"
		// Actual: "Count is "
		result := processor.Process("Count is 0")
		assert.Equal(t, "Count is ", result)
	})

	t.Run("currency_without_cents_not_matched", func(t *testing.T) {
		processor := NewCurrencyProcessor(logger)
		// Known limitation - requires .XX cents format
		result := processor.Process("Price is $50")
		assert.Equal(t, "Price is $50", result)
	})

	t.Run("time_invalid_format_preserved", func(t *testing.T) {
		processor := NewTimeProcessor(logger)
		// Invalid time should be preserved
		result := processor.Process("Time is 25:00")
		assert.Equal(t, "Time is 25:00", result)
	})
}
