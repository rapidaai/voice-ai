// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package commons

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// ============================================================================
// Logger Interface - Core logging contract
// ============================================================================

type Logger interface {
	// Initialization
	InitLogger() error

	// Standard logging levels
	Debug(args ...interface{})
	Debugf(template string, args ...interface{})

	Info(args ...interface{})
	Infof(template string, args ...interface{})

	Warn(args ...interface{})
	Warnf(template string, args ...interface{})

	Error(args ...interface{})
	Errorf(template string, args ...interface{})

	// Panic levels
	DPanic(args ...interface{})
	DPanicf(template string, args ...interface{})

	Panic(args ...interface{})
	Panicf(template string, args ...interface{})

	Fatal(args ...interface{})
	Fatalf(template string, args ...interface{})

	// Specialized logging
	Benchmark(functionName string, duration time.Duration)
	Tracef(ctx context.Context, format string, args ...interface{})

	// Lifecycle
	Sync() error
}

// ============================================================================
// Color codes for console output
// ============================================================================

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorGreen  = "\033[32m"
	colorCyan   = "\033[36m"
)

// ============================================================================
// Configuration Options
// ============================================================================

type logOptions struct {
	level         string
	name          string
	path          string
	enableConsole bool
	enableFile    bool
	maxSize       int // MB
	maxBackups    int
	maxAge        int // days
	compress      bool
}

var defaultLoggerOptions = logOptions{
	level:         "info",
	name:          "go-app",
	path:          "/var/log/go-app",
	enableConsole: true,
	enableFile:    true,
	maxSize:       500,
	maxBackups:    3,
	maxAge:        28,
	compress:      true,
}

type LoggerOption interface {
	apply(*logOptions)
}

type funcloggerOption struct {
	f func(*logOptions)
}

func (flo *funcloggerOption) apply(opts *logOptions) {
	flo.f(opts)
}

func newFuncLoggerOption(f func(*logOptions)) *funcloggerOption {
	return &funcloggerOption{f: f}
}

// Name sets the logger name for identification
func Name(name string) LoggerOption {
	return newFuncLoggerOption(func(o *logOptions) {
		o.name = name
	})
}

// Level sets the logging level (debug, info, warn, error)
func Level(level string) LoggerOption {
	return newFuncLoggerOption(func(o *logOptions) {
		o.level = level
	})
}

// Path sets the log file directory
func Path(path string) LoggerOption {
	return newFuncLoggerOption(func(o *logOptions) {
		o.path = path
	})
}

// EnableConsole enables/disables console output
func EnableConsole(enable bool) LoggerOption {
	return newFuncLoggerOption(func(o *logOptions) {
		o.enableConsole = enable
	})
}

// EnableFile enables/disables file output
func EnableFile(enable bool) LoggerOption {
	return newFuncLoggerOption(func(o *logOptions) {
		o.enableFile = enable
	})
}

// MaxSize sets max log file size in MB
func MaxSize(size int) LoggerOption {
	return newFuncLoggerOption(func(o *logOptions) {
		o.maxSize = size
	})
}

// MaxBackups sets max number of backup log files
func MaxBackups(backups int) LoggerOption {
	return newFuncLoggerOption(func(o *logOptions) {
		o.maxBackups = backups
	})
}

// MaxAge sets max age of log files in days
func MaxAge(days int) LoggerOption {
	return newFuncLoggerOption(func(o *logOptions) {
		o.maxAge = days
	})
}

// ============================================================================
// Application Logger Implementation
// ============================================================================

type applicationLogger struct {
	opts        logOptions
	sugarLogger *zap.SugaredLogger
	logger      *zap.Logger
}

// NewApplicationLogger returns a logger with default options
func NewApplicationLogger() *applicationLogger {
	return &applicationLogger{opts: defaultLoggerOptions}
}

// NewApplicationLoggerWithOptions returns a logger with custom options
func NewApplicationLoggerWithOptions(opts ...LoggerOption) *applicationLogger {
	options := defaultLoggerOptions
	for _, opt := range opts {
		opt.apply(&options)
	}
	return &applicationLogger{opts: options}
}

// ============================================================================
// Logger Initialization
// ============================================================================

func (l *applicationLogger) getLoggerLevel() zapcore.Level {
	switch l.opts.level {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}

func getConsoleEncoder() zapcore.Encoder {
	config := zap.NewDevelopmentEncoderConfig()
	config.EncodeLevel = func(level zapcore.Level, encoder zapcore.PrimitiveArrayEncoder) {
		switch level {
		case zapcore.DebugLevel:
			encoder.AppendString(colorGreen + "DEBUG" + colorReset)
		case zapcore.InfoLevel:
			encoder.AppendString(colorBlue + "INFO" + colorReset)
		case zapcore.WarnLevel:
			encoder.AppendString(colorYellow + "WARN" + colorReset)
		case zapcore.ErrorLevel, zapcore.DPanicLevel, zapcore.PanicLevel, zapcore.FatalLevel:
			encoder.AppendString(colorRed + level.CapitalString() + colorReset)
		default:
			encoder.AppendString(level.CapitalString())
		}
	}
	config.EncodeTime = zapcore.ISO8601TimeEncoder
	config.EncodeCaller = zapcore.ShortCallerEncoder
	return zapcore.NewConsoleEncoder(config)
}

func getFileEncoder() zapcore.Encoder {
	config := zap.NewDevelopmentEncoderConfig()
	config.EncodeTime = zapcore.ISO8601TimeEncoder
	config.EncodeCaller = zapcore.ShortCallerEncoder
	config.EncodeLevel = zapcore.CapitalLevelEncoder
	return zapcore.NewConsoleEncoder(config)
}

type WriteSyncer struct {
	io.Writer
}

func (ws WriteSyncer) Sync() error {
	return nil
}

func getWriteSyncer(path, name string) zapcore.WriteSyncer {
	ioWriter := &lumberjack.Logger{
		Filename:   fmt.Sprintf("%s/%s.log", path, name),
		MaxSize:    500,
		MaxBackups: 3,
		MaxAge:     28,
		Compress:   true,
	}
	return WriteSyncer{ioWriter}
}

// InitLogger initializes the logger with console and file writers
func (l *applicationLogger) InitLogger() error {
	var cores []zapcore.Core
	level := zap.NewAtomicLevelAt(l.getLoggerLevel())

	// Console output
	if l.opts.enableConsole {
		consoleEncoder := getConsoleEncoder()
		consoleWriter := zapcore.AddSync(os.Stdout)
		cores = append(cores, zapcore.NewCore(consoleEncoder, consoleWriter, level))
	}

	// File output
	if l.opts.enableFile {
		fileEncoder := getFileEncoder()
		fileWriter := getWriteSyncer(l.opts.path, l.opts.name)
		cores = append(cores, zapcore.NewCore(fileEncoder, fileWriter, level))
	}

	if len(cores) == 0 {
		return fmt.Errorf("logger must have at least one output (console or file)")
	}

	var core zapcore.Core
	if len(cores) == 1 {
		core = cores[0]
	} else {
		core = zapcore.NewTee(cores...)
	}

	l.logger = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
	l.sugarLogger = l.logger.Sugar()

	return nil
}

func (l *applicationLogger) Sync() error {
	if l.logger != nil {
		return l.logger.Sync()
	}
	return nil
}

// ============================================================================
// Standard Logging Methods
// ============================================================================

func (l *applicationLogger) Debug(args ...interface{}) {
	l.sugarLogger.Debug(args...)
}

func (l *applicationLogger) Debugf(template string, args ...interface{}) {
	l.sugarLogger.Debugf(template, args...)
}

func (l *applicationLogger) Info(args ...interface{}) {
	l.sugarLogger.Info(args...)
}

func (l *applicationLogger) Infof(template string, args ...interface{}) {
	l.sugarLogger.Infof(template, args...)
}

func (l *applicationLogger) Warn(args ...interface{}) {
	l.sugarLogger.Warn(args...)
}

func (l *applicationLogger) Warnf(template string, args ...interface{}) {
	l.sugarLogger.Warnf(template, args...)
}

func (l *applicationLogger) Error(args ...interface{}) {
	l.sugarLogger.Error(args...)
}

func (l *applicationLogger) Errorf(template string, args ...interface{}) {
	l.sugarLogger.Errorf(template, args...)
}

// ============================================================================
// Panic Logging Methods
// ============================================================================

func (l *applicationLogger) DPanic(args ...interface{}) {
	l.sugarLogger.DPanic(args...)
}

func (l *applicationLogger) DPanicf(template string, args ...interface{}) {
	l.sugarLogger.DPanicf(template, args...)
}

func (l *applicationLogger) Panic(args ...interface{}) {
	l.sugarLogger.Panic(args...)
}

func (l *applicationLogger) Panicf(template string, args ...interface{}) {
	l.sugarLogger.Panicf(template, args...)
}

func (l *applicationLogger) Fatal(args ...interface{}) {
	l.sugarLogger.Fatal(args...)
}

func (l *applicationLogger) Fatalf(template string, args ...interface{}) {
	l.sugarLogger.Fatalf(template, args...)
}

// ============================================================================
// Specialized Logging Methods
// ============================================================================

func (l *applicationLogger) Benchmark(functionName string, duration time.Duration) {
	l.sugarLogger.Infof("Benchmark: %s took %v", functionName, duration)
}

func (l *applicationLogger) Tracef(ctx context.Context, format string, args ...interface{}) {
	requestID := "unknown"
	if value := ctx.Value("x-request-id"); value != nil {
		if id, ok := value.(string); ok {
			requestID = id
		}
	}

	message := fmt.Sprintf(format, args...)
	l.sugarLogger.Infof("%s[RequestID: %s]%s %s", colorCyan, requestID, colorReset, message)
}
