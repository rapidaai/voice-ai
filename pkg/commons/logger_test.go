package commons

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type mockWriteSyncer struct {
	output bytes.Buffer
}

func (m *mockWriteSyncer) Write(p []byte) (n int, err error) {
	return m.output.Write(p)
}

func (m *mockWriteSyncer) Sync() error {
	return nil
}

func TestLoggerInitialization(t *testing.T) {
	logger := NewApplicationLogger()
	logger.InitLogger()
	assert.NotNil(t, logger, "Logger instance should not be nil")
	assert.NotNil(t, logger.sugarLogger, "Sugared logger instance should not be nil")
}

func TestInfoLog(t *testing.T) {
	mockSyncer := &mockWriteSyncer{}
	logger := NewApplicationLogger()

	// Redirect logger output to the mock writer
	logger.init(mockSyncer)

	logger.Info("This is a test info log")
	logOutput := mockSyncer.output.String()

	assert.Contains(t, logOutput, "This is a test info log", "Log output should contain the info message")
}

func TestLogLevel(t *testing.T) {
	mockSyncer := &mockWriteSyncer{}
	logger := NewApplicationLoggerWithOptions(Level("warn"))
	logger.init(mockSyncer)

	logger.Debug("This debug message should not appear")
	logger.Warn("This is a warning message")

	logOutput := mockSyncer.output.String()
	assert.NotContains(t, logOutput, "This debug message should not appear", "Debug message should not appear in log output")
	assert.Contains(t, logOutput, "This is a warning message", "Warning message should appear in log output")
}

func TestBenchmarkLog(t *testing.T) {
	mockSyncer := &mockWriteSyncer{}
	logger := NewApplicationLogger()
	logger.init(mockSyncer)

	start := time.Now()
	time.Sleep(2 * time.Millisecond)
	duration := time.Since(start)

	logger.Benchmark("TestFunction", duration)
	logOutput := mockSyncer.output.String()

	assert.Contains(t, logOutput, "TestFunction", "Log output should contain the benchmark function name")
	assert.Contains(t, logOutput, "Benchmark", "Log output should indicate it's a benchmark")
}

func TestErrorLog(t *testing.T) {
	mockSyncer := &mockWriteSyncer{}
	logger := NewApplicationLogger()
	logger.init(mockSyncer)

	logger.Error("This is a test error")
	logOutput := mockSyncer.output.String()

	assert.Contains(t, logOutput, "This is a test error", "Error log message should appear in log output")
}

func TestLoggerWithCustomName(t *testing.T) {
	mockSyncer := &mockWriteSyncer{}
	logger := NewApplicationLoggerWithOptions(Name("custom-service"))
	logger.init(mockSyncer)

	logger.Info("Test custom logger name")
	logOutput := mockSyncer.output.String()

	assert.Contains(t, logOutput, "custom-service", "Log output should contain the custom logger name")
	assert.Contains(t, logOutput, "Test custom logger name", "Log output should contain the log message")
}
