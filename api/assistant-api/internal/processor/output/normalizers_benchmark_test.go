// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_output_processors

import (
	"strings"
	"testing"

	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
)

// =============================================================================
// Benchmark Mock Logger (minimal implementation for benchmarks)
// =============================================================================

// =============================================================================
// Sample Input Data for Benchmarks
// =============================================================================

var (
	shortSentence     = "Hello world"
	mediumSentence    = "The CEO meeting at 14:30 on 2024-01-15 at 123 Main St costs $500.50 with 25% discount"
	longSentence      = strings.Repeat("The CEO meeting at 14:30 on 2024-01-15 at 123 Main St costs $500.50 with API and ML. ", 10)
	currencyInput     = "Total: $1,234.56 plus $99.99 equals $1,334.55"
	dateInput         = "Events on 2024-01-15, 2024-06-30, and 2024-12-25"
	timeInput         = "Meetings at 09:00, 14:30, and 17:45"
	numberInput       = "We have 5 apples, 12 oranges, and 42 bananas"
	addressInput      = "Visit 123 Main St, 456 Park Ave, and 789 Oak Rd"
	urlInput          = "Check https://example.com, www.google.com, and api.test.org"
	symbolInput       = "Growth is 25% with ±5% variance, temperature 25℃"
	techInput         = "Using AI, ML, API, DevOps, and CI/CD for automation"
	roleInput         = "CEO, CFO, CTO, and VP discussed R&D plans"
	generalInput      = "Dr. Smith aka Johnny Jr. said etc. i.e. examples"
	mixedComplexInput = "The CEO Dr. Smith announced at 14:30 on 2024-01-15 that our API costs $500.50 with 25% growth at https://rapida.ai using AI & ML"
	unicodeHeavyInput = "℃ ℉ £ € ¥ ₩ ₿ ™ © ® ° ± × ÷ ≈ ≠ ≤ ≥ ∞ π √ ∑ ∫"
	noMatchInput      = "This is a plain sentence without any special patterns to match"
)

// =============================================================================
// Individual Processor Benchmarks
// =============================================================================

func benchLogger() commons.Logger {
	lgr, _ := commons.NewApplicationLogger()
	return lgr
}

func BenchmarkCurrencyProcessor(b *testing.B) {
	processor := NewCurrencyProcessor(benchLogger())

	b.Run("short", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			processor.Process(shortSentence)
		}
	})

	b.Run("currency_input", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			processor.Process(currencyInput)
		}
	})

	b.Run("no_match", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			processor.Process(noMatchInput)
		}
	})

	b.Run("long", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			processor.Process(longSentence)
		}
	})
}

func BenchmarkDateProcessor(b *testing.B) {
	processor := NewDateProcessor(benchLogger())

	b.Run("short", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			processor.Process(shortSentence)
		}
	})

	b.Run("date_input", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			processor.Process(dateInput)
		}
	})

	b.Run("no_match", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			processor.Process(noMatchInput)
		}
	})

	b.Run("long", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			processor.Process(longSentence)
		}
	})
}

func BenchmarkTimeProcessor(b *testing.B) {
	processor := NewTimeProcessor(benchLogger())

	b.Run("short", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			processor.Process(shortSentence)
		}
	})

	b.Run("time_input", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			processor.Process(timeInput)
		}
	})

	b.Run("no_match", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			processor.Process(noMatchInput)
		}
	})

	b.Run("long", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			processor.Process(longSentence)
		}
	})
}

func BenchmarkNumberToWordProcessor(b *testing.B) {
	processor := NewNumberToWordProcessor(benchLogger())

	b.Run("short", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			processor.Process(shortSentence)
		}
	})

	b.Run("number_input", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			processor.Process(numberInput)
		}
	})

	b.Run("no_match", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			processor.Process(noMatchInput)
		}
	})

	b.Run("long", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			processor.Process(longSentence)
		}
	})
}

func BenchmarkAddressProcessor(b *testing.B) {
	processor := NewAddressProcessor(benchLogger())

	b.Run("short", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			processor.Process(shortSentence)
		}
	})

	b.Run("address_input", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			processor.Process(addressInput)
		}
	})

	b.Run("no_match", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			processor.Process(noMatchInput)
		}
	})

	b.Run("long", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			processor.Process(longSentence)
		}
	})
}

func BenchmarkUrlProcessor(b *testing.B) {
	processor := NewUrlProcessor(benchLogger())

	b.Run("short", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			processor.Process(shortSentence)
		}
	})

	b.Run("url_input", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			processor.Process(urlInput)
		}
	})

	b.Run("no_match", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			processor.Process(noMatchInput)
		}
	})

	b.Run("long", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			processor.Process(longSentence)
		}
	})
}

func BenchmarkSymbolProcessor(b *testing.B) {
	processor := NewSymbolProcessor(benchLogger())

	b.Run("short", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			processor.Process(shortSentence)
		}
	})

	b.Run("symbol_input", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			processor.Process(symbolInput)
		}
	})

	b.Run("unicode_heavy", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			processor.Process(unicodeHeavyInput)
		}
	})

	b.Run("no_match", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			processor.Process(noMatchInput)
		}
	})

	b.Run("long", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			processor.Process(longSentence)
		}
	})
}

func BenchmarkTechAbbreviationProcessor(b *testing.B) {
	processor := NewTechAbbreviationProcessor(benchLogger())

	b.Run("short", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			processor.Process(shortSentence)
		}
	})

	b.Run("tech_input", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			processor.Process(techInput)
		}
	})

	b.Run("no_match", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			processor.Process(noMatchInput)
		}
	})

	b.Run("long", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			processor.Process(longSentence)
		}
	})
}

func BenchmarkRoleAbbreviationProcessor(b *testing.B) {
	processor := NewRoleAbbreviationProcessor(benchLogger())

	b.Run("short", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			processor.Process(shortSentence)
		}
	})

	b.Run("role_input", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			processor.Process(roleInput)
		}
	})

	b.Run("no_match", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			processor.Process(noMatchInput)
		}
	})

	b.Run("long", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			processor.Process(longSentence)
		}
	})
}

func BenchmarkGeneralAbbreviationProcessor(b *testing.B) {
	processor := NewGeneralAbbreviationProcessor(benchLogger())

	b.Run("short", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			processor.Process(shortSentence)
		}
	})

	b.Run("general_input", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			processor.Process(generalInput)
		}
	})

	b.Run("no_match", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			processor.Process(noMatchInput)
		}
	})

	b.Run("long", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			processor.Process(longSentence)
		}
	})
}

// =============================================================================
// Processor Chain Benchmarks (simulating real-world pipeline)
// =============================================================================

func BenchmarkProcessorChain(b *testing.B) {
	processors := []internal_type.TextProcessor{
		NewCurrencyProcessor(benchLogger()),
		NewDateProcessor(benchLogger()),
		NewTimeProcessor(benchLogger()),
		NewNumberToWordProcessor(benchLogger()),
		NewAddressProcessor(benchLogger()),
		NewUrlProcessor(benchLogger()),
		NewTechAbbreviationProcessor(benchLogger()),
		NewRoleAbbreviationProcessor(benchLogger()),
		NewGeneralAbbreviationProcessor(benchLogger()),
		NewSymbolProcessor(benchLogger()),
	}

	applyAll := func(input string) string {
		result := input
		for _, n := range processors {
			result = n.Process(result)
		}
		return result
	}

	b.Run("short", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			applyAll(shortSentence)
		}
	})

	b.Run("medium", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			applyAll(mediumSentence)
		}
	})

	b.Run("long", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			applyAll(longSentence)
		}
	})

	b.Run("mixed_complex", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			applyAll(mixedComplexInput)
		}
	})

	b.Run("no_match", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			applyAll(noMatchInput)
		}
	})
}

// =============================================================================
// Memory Allocation Benchmarks
// =============================================================================

func BenchmarkProcessorAllocations(b *testing.B) {
	processors := map[string]internal_type.TextProcessor{
		"currency": NewCurrencyProcessor(benchLogger()),
		"date":     NewDateProcessor(benchLogger()),
		"time":     NewTimeProcessor(benchLogger()),
		"number":   NewNumberToWordProcessor(benchLogger()),
		"address":  NewAddressProcessor(benchLogger()),
		"url":      NewUrlProcessor(benchLogger()),
		"symbol":   NewSymbolProcessor(benchLogger()),
		"tech":     NewTechAbbreviationProcessor(benchLogger()),
		"role":     NewRoleAbbreviationProcessor(benchLogger()),
		"general":  NewGeneralAbbreviationProcessor(benchLogger()),
	}

	for name, processor := range processors {
		b.Run(name+"_allocs", func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				processor.Process(mediumSentence)
			}
		})
	}
}

// =============================================================================
// Scaling Benchmarks (input size scaling)
// =============================================================================

func BenchmarkInputSizeScaling(b *testing.B) {
	processor := NewSymbolProcessor(benchLogger())

	sizes := []int{10, 100, 1000, 10000}
	baseText := "Hello 25% world & test + more = result @ place # tag "

	for _, size := range sizes {
		input := strings.Repeat(baseText, size/len(baseText)+1)[:size]
		b.Run(string(rune('0'+size/1000))+"k_chars", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				processor.Process(input)
			}
		})
	}
}

func BenchmarkChainInputSizeScaling(b *testing.B) {
	processors := []internal_type.TextProcessor{
		NewCurrencyProcessor(benchLogger()),
		NewDateProcessor(benchLogger()),
		NewTimeProcessor(benchLogger()),
		NewSymbolProcessor(benchLogger()),
	}

	applyAll := func(input string) string {
		result := input
		for _, n := range processors {
			result = n.Process(result)
		}
		return result
	}

	sizes := []int{50, 200, 500, 1000}
	baseText := "Meeting at 14:30 costs $50.00 with 25% off "

	for _, size := range sizes {
		input := strings.Repeat(baseText, size/len(baseText)+1)[:size]
		b.Run(string(rune('0'+size/100))+"00_chars", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				applyAll(input)
			}
		})
	}
}

// =============================================================================
// Concurrent Access Benchmarks
// =============================================================================

func BenchmarkConcurrentProcessing(b *testing.B) {
	processor := NewSymbolProcessor(benchLogger())

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			processor.Process(symbolInput)
		}
	})
}

func BenchmarkConcurrentChain(b *testing.B) {
	processors := []internal_type.TextProcessor{
		NewCurrencyProcessor(benchLogger()),
		NewDateProcessor(benchLogger()),
		NewTimeProcessor(benchLogger()),
		NewSymbolProcessor(benchLogger()),
	}

	applyAll := func(input string) string {
		result := input
		for _, n := range processors {
			result = n.Process(result)
		}
		return result
	}

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			applyAll(mediumSentence)
		}
	})
}

// =============================================================================
// Worst Case Benchmarks (many matches)
// =============================================================================

func BenchmarkWorstCaseCurrency(b *testing.B) {
	processor := NewCurrencyProcessor(benchLogger())
	// Many currency values in one string
	input := strings.Repeat("$1.00 ", 100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		processor.Process(input)
	}
}

func BenchmarkWorstCaseSymbol(b *testing.B) {
	processor := NewSymbolProcessor(benchLogger())
	// Many symbols in one string
	input := strings.Repeat("% & + = @ # ", 100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		processor.Process(input)
	}
}

func BenchmarkWorstCaseAddress(b *testing.B) {
	processor := NewAddressProcessor(benchLogger())
	// Many address abbreviations
	input := strings.Repeat("123 Main St 456 Park Ave 789 Oak Rd ", 50)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		processor.Process(input)
	}
}

// =============================================================================
// Processor Creation Benchmarks
// =============================================================================

func BenchmarkProcessorCreation(b *testing.B) {
	b.Run("currency", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			NewCurrencyProcessor(benchLogger())
		}
	})

	b.Run("date", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			NewDateProcessor(benchLogger())
		}
	})

	b.Run("time", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			NewTimeProcessor(benchLogger())
		}
	})

	b.Run("number", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			NewNumberToWordProcessor(benchLogger())
		}
	})

	b.Run("address", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			NewAddressProcessor(benchLogger())
		}
	})

	b.Run("url", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			NewUrlProcessor(benchLogger())
		}
	})

	b.Run("symbol", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			NewSymbolProcessor(benchLogger())
		}
	})

	b.Run("tech", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			NewTechAbbreviationProcessor(benchLogger())
		}
	})

	b.Run("role", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			NewRoleAbbreviationProcessor(benchLogger())
		}
	})

	b.Run("general", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			NewGeneralAbbreviationProcessor(benchLogger())
		}
	})

	b.Run("all_processors", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			NewCurrencyProcessor(benchLogger())
			NewDateProcessor(benchLogger())
			NewTimeProcessor(benchLogger())
			NewNumberToWordProcessor(benchLogger())
			NewAddressProcessor(benchLogger())
			NewUrlProcessor(benchLogger())
			NewTechAbbreviationProcessor(benchLogger())
			NewRoleAbbreviationProcessor(benchLogger())
			NewGeneralAbbreviationProcessor(benchLogger())
			NewSymbolProcessor(benchLogger())
		}
	})
}

// =============================================================================
// Real-World TTS Input Benchmarks
// =============================================================================

func BenchmarkRealWorldTTSInputs(b *testing.B) {
	processors := []internal_type.TextProcessor{
		NewCurrencyProcessor(benchLogger()),
		NewDateProcessor(benchLogger()),
		NewTimeProcessor(benchLogger()),
		NewNumberToWordProcessor(benchLogger()),
		NewAddressProcessor(benchLogger()),
		NewUrlProcessor(benchLogger()),
		NewTechAbbreviationProcessor(benchLogger()),
		NewRoleAbbreviationProcessor(benchLogger()),
		NewGeneralAbbreviationProcessor(benchLogger()),
		NewSymbolProcessor(benchLogger()),
	}

	applyAll := func(input string) string {
		result := input
		for _, n := range processors {
			result = n.Process(result)
		}
		return result
	}

	realWorldInputs := map[string]string{
		"customer_service":  "Hello! Your order #12345 for $99.99 will arrive on 2024-01-20 between 14:00 and 17:00. Visit https://track.example.com for updates.",
		"appointment":       "Dr. Smith will see you at 10:30 a.m. on 2024-03-15. The consultation costs $150.00. Our address is 123 Main St.",
		"tech_announcement": "The new AI & ML features in our API are launching on 2024-06-01. CEO John Smith said this represents 25% improvement.",
		"financial_report":  "Q4 revenue: $1,234,567.89 with 15% YoY growth. CFO meeting at 09:00 on 2024-01-30.",
		"simple_greeting":   "Hello, how can I help you today?",
		"numbers_heavy":     "You have 5 items, 12 messages, and 42 notifications. Total: 59 updates.",
	}

	for name, input := range realWorldInputs {
		b.Run(name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				applyAll(input)
			}
		})
	}
}
