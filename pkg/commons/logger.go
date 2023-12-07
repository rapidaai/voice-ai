package commons

import (
	"fmt"
	"io"
	"os"

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
}

type logOptions struct {
	level string
	name  string
}

var defaultLoggerOptions = logOptions{
	level: "info",
	name:  "go-template-service",
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
type applicatoinLogger struct {
	opts        logOptions
	sugarLogger *zap.SugaredLogger
}

// Applicaiton Logger constructor level will be default info in production
func NewApplicationLogger() *applicatoinLogger {
	opts := defaultLoggerOptions
	return &applicatoinLogger{opts: opts}
}

// NewApplicationLoggerWithOtptions returns a ptr application logger instance which impliment logger interface
//
// # This will also override default option for logger
//
// This function is provided for advanced uses; prefer to provide sepcific option when initializing the logger
// using common.logger.NewApplicaitonLoggerWithOptions
func NewApplicationLoggerWithOptions(opt ...LoggerOption) *applicatoinLogger {
	opts := defaultLoggerOptions
	for _, o := range extraLoggerOptions {
		o.apply(&opts)
	}

	for _, o := range opt {
		o.apply(&opts)
	}
	return &applicatoinLogger{opts: opts}

}

// For mapping config logger to app logger levels
func (l *applicatoinLogger) getLoggerLevel() zapcore.Level {
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

func getWriteSyncer(name string) zapcore.WriteSyncer {
	var ioWriter = &lumberjack.Logger{
		Filename:   fmt.Sprintf("/var/log/go-app/%s.log", name),
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
func (l *applicatoinLogger) InitLogger() {
	logLevel := l.getLoggerLevel()
	// logWriter := zapcore.AddSync(os.Stderr)
	syncer := zap.CombineWriteSyncers(os.Stdout, getWriteSyncer(l.opts.name))
	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.LevelKey = "LEVEL"
	encoderCfg.CallerKey = "CALLER"
	encoderCfg.TimeKey = "TIME"
	encoderCfg.NameKey = "NAME"
	encoderCfg.MessageKey = "MESSAGE"
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder
	encoder := zapcore.NewConsoleEncoder(encoderCfg)
	core := zapcore.NewCore(encoder, syncer, zap.NewAtomicLevelAt(logLevel))
	logger := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
	l.sugarLogger = logger.Sugar()
	if err := l.sugarLogger.Sync(); err != nil {
		l.sugarLogger.Error(err)
	}
}

func (l *applicatoinLogger) Debug(args ...interface{}) {
	l.sugarLogger.Debug(args...)
}

func (l *applicatoinLogger) Debugf(msg string, args ...interface{}) {
	l.sugarLogger.Debugf(msg, args...)
}

func (l *applicatoinLogger) Info(args ...interface{}) {
	l.sugarLogger.Info(args...)
}

func (l *applicatoinLogger) Infof(msg string, args ...interface{}) {
	l.sugarLogger.Infof(msg, args...)
}

func (l *applicatoinLogger) Warn(args ...interface{}) {
	l.sugarLogger.Warn(args...)
}

func (l *applicatoinLogger) Warnf(msg string, args ...interface{}) {
	l.sugarLogger.Warnf(msg, args...)
}

func (l *applicatoinLogger) Error(args ...interface{}) {
	l.sugarLogger.Error(args...)
}

func (l *applicatoinLogger) Errorf(msg string, args ...interface{}) {
	l.sugarLogger.Errorf(msg, args...)
}

func (l *applicatoinLogger) DPanic(args ...interface{}) {
	l.sugarLogger.DPanic(args...)
}

func (l *applicatoinLogger) DPanicf(msg string, args ...interface{}) {
	l.sugarLogger.DPanicf(msg, args...)
}

func (l *applicatoinLogger) Panic(args ...interface{}) {
	l.sugarLogger.Panic(args...)
}

func (l *applicatoinLogger) Panicf(msg string, args ...interface{}) {
	l.sugarLogger.Panicf(msg, args...)
}

func (l *applicatoinLogger) Fatal(args ...interface{}) {
	l.sugarLogger.Fatal(args...)
}

func (l *applicatoinLogger) Fatalf(msg string, args ...interface{}) {
	l.sugarLogger.Fatalf(msg, args...)
}
