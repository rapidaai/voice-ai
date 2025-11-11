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

// copied from
type Logger interface {
	InitLogger()
	Debug(args ...interface{})
	Debugf(template string, args ...interface{})
	Info(args ...interface{})
	Infof(template string, args ...interface{})
	Warn(args ...interface{})
	Warnf(template string, args ...interface{})
	Error(args ...interface{})
	Errorf(template string, args ...interface{})
	DPanic(args ...interface{})
	DPanicf(template string, args ...interface{})
	Fatal(args ...interface{})
	Fatalf(template string, args ...interface{})

	//
	Benchmark(functionName string, duration time.Duration)
	Tracef(ctx context.Context, format string, args ...interface{})
}

type logOptions struct {
	level string
	name  string
	path  string
}

var defaultLoggerOptions = logOptions{
	level: "info",
	name:  "go-template-service",
	path:  "/var/log/go-app",
}

var extraLoggerOptions []LoggerOption

type LoggerOption interface {
	apply(*logOptions)
}

type funcloggerOption struct {
	f func(*logOptions)
}

func (fdo *funcloggerOption) apply(do *logOptions) {
	fdo.f(do)
}

func newFuncLoggerOption(f func(*logOptions)) *funcloggerOption {
	return &funcloggerOption{
		f: f,
	}
}

// Name returns a LoggerOptions that sets the name for the logger to represent
// The name will get printed with every logger as stranderd to provide greping capabilites.
// default name (e.g. go-tempate-service)
func Name(name string) LoggerOption {
	return newFuncLoggerOption(func(o *logOptions) {
		o.name = name
	})
}

// Level returns a LoggerOptions that sets the level for the logger.
// The level will get used to identifies what needs to be printed in console/file logs.
// default level (e.g. info)
func Level(level string) LoggerOption {
	return newFuncLoggerOption(func(o *logOptions) {
		o.level = level
	})
}

// Logger
type applicationLogger struct {
	opts        logOptions
	sugarLogger *zap.SugaredLogger
}

// Applicaiton Logger constructor level will be default info in production
func NewApplicationLogger() *applicationLogger {
	opts := defaultLoggerOptions
	return &applicationLogger{opts: opts}
}

// NewApplicationLoggerWithOtptions returns a ptr application logger instance which impliment logger interface
//
// # This will also override default option for logger
//
// This function is provided for advanced uses; prefer to provide sepcific option when initializing the logger
// using common.logger.NewApplicaitonLoggerWithOptions
func NewApplicationLoggerWithOptions(opt ...LoggerOption) *applicationLogger {
	opts := defaultLoggerOptions
	for _, o := range extraLoggerOptions {
		o.apply(&opts)
	}

	for _, o := range opt {
		o.apply(&opts)
	}
	return &applicationLogger{opts: opts}

}

// For mapping config logger to app logger levels
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
		return zapcore.DebugLevel

	}
}

// just to mock don't feel bacd
type WriteSyncer struct {
	io.Writer
}

func (ws WriteSyncer) Sync() error {
	return nil
}

func getWriteSyncer(path, name string) zapcore.WriteSyncer {
	var ioWriter = &lumberjack.Logger{
		Filename:   fmt.Sprintf("%s/%s.log", path, name),
		MaxSize:    500, // megabytes
		MaxBackups: 3,
		MaxAge:     28,   //days
		Compress:   true, // disabled by default
	}
	var sw = WriteSyncer{
		ioWriter,
	}
	return sw
}

// Init logger
func (l *applicationLogger) InitLogger() {
	l.init(os.Stdout, getWriteSyncer(l.opts.path, l.opts.name))
}

func (l *applicationLogger) init(writer ...zapcore.WriteSyncer) {
	l.sugarLogger = zap.New(
		zapcore.NewCore(
			zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig()),
			zap.CombineWriteSyncers(writer...),
			zap.NewAtomicLevelAt(l.getLoggerLevel()),
		),
		zap.AddCaller(),
		zap.AddCallerSkip(1)).Sugar()
	if err := l.sugarLogger.Sync(); err != nil {
		l.sugarLogger.Error(err)
	}
}

func (l *applicationLogger) Debug(args ...interface{}) {
	l.sugarLogger.Debug(args...)
}

func (l *applicationLogger) Debugf(msg string, args ...interface{}) {
	l.sugarLogger.Debugf(msg, args...)
}

func (l *applicationLogger) Benchmark(functionName string, duration time.Duration) {
	// Convert duration to milliseconds
	durationMs := duration.Milliseconds()

	// Format message with or without color based on the duration
	var message string
	if durationMs > 1 {
		// Use red color if duration is more than 1 millisecond
		message = fmt.Sprintf("\033[31mBenchmark: %s [%v]\033[0m", functionName, duration)
	} else {
		message = fmt.Sprintf("Benchmark: %s [%v]", functionName, duration)
	}

	// Log the message
	l.sugarLogger.Infof(message)
}

func (l *applicationLogger) Tracef(ctx context.Context, format string, args ...interface{}) {
	value := ctx.Value("x-request-id")
	if value == nil {
		value = "unknown"
	}
	l.sugarLogger.Info(fmt.Sprintf("[RequestID: %s] %s", value, fmt.Sprintf(format, args...)))
}

func (l *applicationLogger) Info(args ...interface{}) {
	l.sugarLogger.Info(args...)
}

func (l *applicationLogger) Infof(msg string, args ...interface{}) {
	l.sugarLogger.Infof(msg, args...)
}

func (l *applicationLogger) Warn(args ...interface{}) {
	l.sugarLogger.Warn(args...)
}

func (l *applicationLogger) Warnf(msg string, args ...interface{}) {
	l.sugarLogger.Warnf(msg, args...)
}

func (l *applicationLogger) Error(args ...interface{}) {
	l.sugarLogger.Error(args...)
}

func (l *applicationLogger) Errorf(msg string, args ...interface{}) {
	l.sugarLogger.Errorf(msg, args...)
}

func (l *applicationLogger) DPanic(args ...interface{}) {
	l.sugarLogger.DPanic(args...)
}

func (l *applicationLogger) DPanicf(msg string, args ...interface{}) {
	l.sugarLogger.DPanicf(msg, args...)
}

func (l *applicationLogger) Panic(args ...interface{}) {
	l.sugarLogger.Panic(args...)
}

func (l *applicationLogger) Panicf(msg string, args ...interface{}) {
	l.sugarLogger.Panicf(msg, args...)
}

func (l *applicationLogger) Fatal(args ...interface{}) {
	l.sugarLogger.Fatal(args...)
}

func (l *applicationLogger) Fatalf(msg string, args ...interface{}) {
	l.sugarLogger.Fatalf(msg, args...)
}
