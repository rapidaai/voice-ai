// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package commons

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestLoggerInitialization tests logger initialization with temp directory
func TestLoggerInitialization(t *testing.T) {
	tmpDir := t.TempDir()
	_, err := NewApplicationLogger(
		Name("test-logger"),
		Path(tmpDir),
		Level("info"),
	)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

// TestDebugLogging tests debug level logging
func TestDebugLogging(t *testing.T) {
	tmpDir := t.TempDir()
	logger, err := NewApplicationLogger(
		Name("test-debug"),
		Path(tmpDir),
		Level("debug"),
	)

	if err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	logger.Debug("Test debug message")
	logger.Debugf("Test debug message with format: %s", "value")

	logFile := filepath.Join(tmpDir, "test-debug.log")
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		t.Errorf("Expected log file to exist at %s", logFile)
	}
}

// TestInfoLogging tests info level logging
func TestInfoLogging(t *testing.T) {
	tmpDir := t.TempDir()
	logger, err := NewApplicationLogger(
		Name("test-info"),
		Path(tmpDir),
		Level("info"),
	)

	if err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	logger.Info("Test info message")
	logger.Infof("Test info message with format: %d", 42)
}

// TestWarnLogging tests warn level logging
func TestWarnLogging(t *testing.T) {
	tmpDir := t.TempDir()
	logger, err := NewApplicationLogger(
		Name("test-warn"),
		Path(tmpDir),
		Level("warn"),
	)

	if err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	logger.Warn("Test warn message")
	logger.Warnf("Test warn message with format: %v", []string{"a", "b"})
}

// TestErrorLogging tests error level logging
func TestErrorLogging(t *testing.T) {
	tmpDir := t.TempDir()
	logger, err := NewApplicationLogger(
		Name("test-error"),
		Path(tmpDir),
		Level("error"),
	)

	if err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	logger.Error("Test error message")
	logger.Errorf("Test error message with format: %s", "error detail")
}

func TestEscapeLogMessage(t *testing.T) {
	if got := escapeLogMessage("provider response: %s", "first\r\nforged"); got != `provider response: first\r\nforged` {
		t.Fatalf("escapeLogMessage() = %q", got)
	}
	if got := escapeLogMessage("provider response: %d", 42); got != "provider response: 42" {
		t.Fatalf("escapeLogMessage() = %q", got)
	}
}

func TestEscapeLogArguments(t *testing.T) {
	got := escapeLogArguments([]interface{}{"code", "first\r\nforged", "count", 42})
	if got[0] != "code" || got[1] != `first\r\nforged` || got[2] != "count" || got[3] != 42 {
		t.Fatalf("escapeLogArguments() = %#v", got)
	}
}

func TestEscapeLogValue(t *testing.T) {
	if got := EscapeLogValue("first\r\nforged"); got != `first\r\nforged` {
		t.Fatalf("EscapeLogValue() = %q", got)
	}
}

// TestDPanicLogging tests DPanic level logging (only in development)
func TestDPanicLogging(t *testing.T) {
	tmpDir := t.TempDir()
	logger, err := NewApplicationLogger(
		Name("test-dpanic"),
		Path(tmpDir),
		Level("debug"),
	)

	if err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	logger.DPanic("Test dpanic message")
	logger.DPanicf("Test dpanic message with format: %s", "detail")
}

// TestBenchmarkLogging tests benchmark logging with different durations
func TestBenchmarkLogging(t *testing.T) {
	tmpDir := t.TempDir()
	logger, err := NewApplicationLogger(
		Name("test-benchmark"),
		Path(tmpDir),
		Level("info"),
	)

	if err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	// Test with different durations
	logger.Benchmark("fastFunction", 5*time.Millisecond)    // Green
	logger.Benchmark("mediumFunction", 50*time.Millisecond) // Yellow
	logger.Benchmark("slowFunction", 200*time.Millisecond)  // Red
}

// TestTracefLogging tests request tracing with context
func TestTracefLogging(t *testing.T) {
	tmpDir := t.TempDir()
	logger, err := NewApplicationLogger(
		Name("test-trace"),
		Path(tmpDir),
		Level("info"),
	)

	if err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	// Test with request ID in context
	ctx := context.WithValue(context.Background(), "x-request-id", "req-12345")
	logger.Tracef(ctx, "Processing request with payload: %v", map[string]string{"key": "value"})

	// Test without request ID
	ctx2 := context.Background()
	logger.Tracef(ctx2, "Processing request without ID")
}

// TestGetLoggerLevel tests level mapping
func TestGetLoggerLevel(t *testing.T) {
	tests := []struct {
		levelStr string
		expected string
	}{
		{"debug", "debug"},
		{"info", "info"},
		{"warn", "warn"},
		{"error", "error"},
		{"invalid", "info"}, // defaults to info
	}

	for _, tt := range tests {
		logger, _ := NewApplicationLogger(Level(tt.levelStr))
		level := logger.Level()

		switch tt.expected {
		case "debug":
			if level.String() != "debug" {
				t.Errorf("Expected debug level for %s, got %s", tt.levelStr, level.String())
			}
		case "info":
			if level.String() != "info" {
				t.Errorf("Expected info level for %s, got %s", tt.levelStr, level.String())
			}
		case "warn":
			if level.String() != "warn" {
				t.Errorf("Expected warn level for %s, got %s", tt.levelStr, level.String())
			}
		case "error":
			if level.String() != "error" {
				t.Errorf("Expected error level for %s, got %s", tt.levelStr, level.String())
			}
		}
	}
}

// TestConsoleOnlyLogger tests logger with only console output
func TestConsoleOnlyLogger(t *testing.T) {
	logger, err := NewApplicationLogger(
		Name("test-console-only"),
		EnableConsole(true),
		EnableFile(false),
	)

	if err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	logger.Info("Console only message")
}

// TestFileOnlyLogger tests logger with only file output
func TestFileOnlyLogger(t *testing.T) {
	tmpDir := t.TempDir()
	logger, err := NewApplicationLogger(
		Name("test-file-only"),
		Path(tmpDir),
		EnableConsole(false),
		EnableFile(true),
	)

	if err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	logger.Info("File only message")

	logFile := filepath.Join(tmpDir, "test-file-only.log")
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		t.Errorf("Expected log file to exist at %s", logFile)
	}
}

// TestMultipleLoggers tests multiple independent logger instances
func TestMultipleLoggers(t *testing.T) {
	tmpDir := t.TempDir()

	logger1, err := NewApplicationLogger(
		Name("app1"),
		Path(tmpDir),
		Level("info"),
	)

	if err != nil {
		t.Fatalf("Failed to initialize logger2: %v", err)
	}
	logger2, err := NewApplicationLogger(
		Name("app2"),
		Path(tmpDir),
		Level("debug"),
	)

	if err != nil {
		t.Fatalf("Failed to initialize logger1: %v", err)
	}

	defer logger1.Sync()
	defer logger2.Sync()

	logger1.Info("From logger 1")
	logger2.Debug("From logger 2")

	// Check both log files exist
	logFile1 := filepath.Join(tmpDir, "app1.log")
	logFile2 := filepath.Join(tmpDir, "app2.log")

	if _, err := os.Stat(logFile1); os.IsNotExist(err) {
		t.Errorf("Expected log file to exist at %s", logFile1)
	}
	if _, err := os.Stat(logFile2); os.IsNotExist(err) {
		t.Errorf("Expected log file to exist at %s", logFile2)
	}
}
