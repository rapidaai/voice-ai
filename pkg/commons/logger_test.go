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

// TestNewApplicationLogger tests default logger initialization
func TestNewApplicationLogger(t *testing.T) {
	logger := NewApplicationLogger()
	if logger == nil {
		t.Fatal("Expected logger, got nil")
	}
	if logger.opts.name != defaultLoggerOptions.name {
		t.Errorf("Expected name %s, got %s", defaultLoggerOptions.name, logger.opts.name)
	}
	if logger.opts.level != defaultLoggerOptions.level {
		t.Errorf("Expected level %s, got %s", defaultLoggerOptions.level, logger.opts.level)
	}
}

// TestNewApplicationLoggerWithOptions tests logger with custom options
func TestNewApplicationLoggerWithOptions(t *testing.T) {
	testName := "test-app"
	testLevel := "debug"

	logger := NewApplicationLoggerWithOptions(
		Name(testName),
		Level(testLevel),
	)

	if logger.opts.name != testName {
		t.Errorf("Expected name %s, got %s", testName, logger.opts.name)
	}
	if logger.opts.level != testLevel {
		t.Errorf("Expected level %s, got %s", testLevel, logger.opts.level)
	}
}

// TestLoggerInitialization tests logger initialization with temp directory
func TestLoggerInitialization(t *testing.T) {
	tmpDir := t.TempDir()

	logger := NewApplicationLoggerWithOptions(
		Name("test-logger"),
		Path(tmpDir),
		Level("info"),
	)

	err := logger.InitLogger()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if logger.sugarLogger == nil {
		t.Fatal("Expected sugarLogger to be initialized")
	}
}

// TestDebugLogging tests debug level logging
func TestDebugLogging(t *testing.T) {
	tmpDir := t.TempDir()
	logger := NewApplicationLoggerWithOptions(
		Name("test-debug"),
		Path(tmpDir),
		Level("debug"),
	)

	if err := logger.InitLogger(); err != nil {
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
	logger := NewApplicationLoggerWithOptions(
		Name("test-info"),
		Path(tmpDir),
		Level("info"),
	)

	if err := logger.InitLogger(); err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	logger.Info("Test info message")
	logger.Infof("Test info message with format: %d", 42)
}

// TestWarnLogging tests warn level logging
func TestWarnLogging(t *testing.T) {
	tmpDir := t.TempDir()
	logger := NewApplicationLoggerWithOptions(
		Name("test-warn"),
		Path(tmpDir),
		Level("warn"),
	)

	if err := logger.InitLogger(); err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	logger.Warn("Test warn message")
	logger.Warnf("Test warn message with format: %v", []string{"a", "b"})
}

// TestErrorLogging tests error level logging
func TestErrorLogging(t *testing.T) {
	tmpDir := t.TempDir()
	logger := NewApplicationLoggerWithOptions(
		Name("test-error"),
		Path(tmpDir),
		Level("error"),
	)

	if err := logger.InitLogger(); err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	logger.Error("Test error message")
	logger.Errorf("Test error message with format: %s", "error detail")
}

// TestDPanicLogging tests DPanic level logging (only in development)
func TestDPanicLogging(t *testing.T) {
	tmpDir := t.TempDir()
	logger := NewApplicationLoggerWithOptions(
		Name("test-dpanic"),
		Path(tmpDir),
		Level("debug"),
	)

	if err := logger.InitLogger(); err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	logger.DPanic("Test dpanic message")
	logger.DPanicf("Test dpanic message with format: %s", "detail")
}

// TestBenchmarkLogging tests benchmark logging with different durations
func TestBenchmarkLogging(t *testing.T) {
	tmpDir := t.TempDir()
	logger := NewApplicationLoggerWithOptions(
		Name("test-benchmark"),
		Path(tmpDir),
		Level("info"),
	)

	if err := logger.InitLogger(); err != nil {
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
	logger := NewApplicationLoggerWithOptions(
		Name("test-trace"),
		Path(tmpDir),
		Level("info"),
	)

	if err := logger.InitLogger(); err != nil {
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
		logger := NewApplicationLoggerWithOptions(Level(tt.levelStr))
		level := logger.getLoggerLevel()

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

// TestLoggerOptions tests various configuration options
func TestLoggerOptions(t *testing.T) {
	tmpDir := t.TempDir()

	logger := NewApplicationLoggerWithOptions(
		Name("test-options"),
		Path(tmpDir),
		Level("debug"),
		EnableConsole(true),
		EnableFile(true),
		MaxSize(100),
		MaxBackups(5),
		MaxAge(14),
	)

	if logger.opts.name != "test-options" {
		t.Errorf("Expected name test-options, got %s", logger.opts.name)
	}
	if logger.opts.path != tmpDir {
		t.Errorf("Expected path %s, got %s", tmpDir, logger.opts.path)
	}
	if logger.opts.level != "debug" {
		t.Errorf("Expected level debug, got %s", logger.opts.level)
	}
	if !logger.opts.enableConsole {
		t.Error("Expected enableConsole to be true")
	}
	if !logger.opts.enableFile {
		t.Error("Expected enableFile to be true")
	}
	if logger.opts.maxSize != 100 {
		t.Errorf("Expected maxSize 100, got %d", logger.opts.maxSize)
	}
	if logger.opts.maxBackups != 5 {
		t.Errorf("Expected maxBackups 5, got %d", logger.opts.maxBackups)
	}
	if logger.opts.maxAge != 14 {
		t.Errorf("Expected maxAge 14, got %d", logger.opts.maxAge)
	}
}

// TestLoggerSync tests logger sync functionality
func TestLoggerSync(t *testing.T) {
	tmpDir := t.TempDir()
	logger := NewApplicationLoggerWithOptions(
		Name("test-sync"),
		Path(tmpDir),
		Level("info"),
	)

	if err := logger.InitLogger(); err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}

	logger.Info("Test message before sync")

	err := logger.Sync()
	if err != nil {
		t.Errorf("Expected no error on sync, got %v", err)
	}
}

// TestConsoleOnlyLogger tests logger with only console output
func TestConsoleOnlyLogger(t *testing.T) {
	logger := NewApplicationLoggerWithOptions(
		Name("test-console-only"),
		EnableConsole(true),
		EnableFile(false),
	)

	if err := logger.InitLogger(); err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	logger.Info("Console only message")
}

// TestFileOnlyLogger tests logger with only file output
func TestFileOnlyLogger(t *testing.T) {
	tmpDir := t.TempDir()
	logger := NewApplicationLoggerWithOptions(
		Name("test-file-only"),
		Path(tmpDir),
		EnableConsole(false),
		EnableFile(true),
	)

	if err := logger.InitLogger(); err != nil {
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

	logger1 := NewApplicationLoggerWithOptions(
		Name("app1"),
		Path(tmpDir),
		Level("info"),
	)

	logger2 := NewApplicationLoggerWithOptions(
		Name("app2"),
		Path(tmpDir),
		Level("debug"),
	)

	if err := logger1.InitLogger(); err != nil {
		t.Fatalf("Failed to initialize logger1: %v", err)
	}
	if err := logger2.InitLogger(); err != nil {
		t.Fatalf("Failed to initialize logger2: %v", err)
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
