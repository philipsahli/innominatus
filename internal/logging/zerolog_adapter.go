package logging

import (
	"io"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

// LogFormat represents the output format for logs
type LogFormat string

const (
	FormatJSON    LogFormat = "json"
	FormatConsole LogFormat = "console"
	FormatPretty  LogFormat = "pretty" // Human-readable with colors (existing format)
)

// ZerologAdapter wraps zerolog to provide structured JSON logging while maintaining
// backward compatibility with the existing Logger interface
type ZerologAdapter struct {
	zlogger   zerolog.Logger
	component string
	format    LogFormat
	minLevel  LogLevel
	fields    map[string]interface{}
}

// NewZerologLogger creates a new zerolog-based logger with configurable format
func NewZerologLogger(component string) *ZerologAdapter {
	format := getLogFormatFromEnv()
	level := getLogLevelFromEnv()

	var writer io.Writer = os.Stdout
	var zlog zerolog.Logger

	switch format {
	case FormatJSON:
		// JSON output for production
		zlog = zerolog.New(writer).With().Timestamp().Logger()
	case FormatConsole:
		// Console output without colors
		writer = zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: "2006-01-02 15:04:05.000",
			NoColor:    true,
		}
		zlog = zerolog.New(writer).With().Timestamp().Logger()
	case FormatPretty:
		// Pretty console output with colors (existing format)
		writer = zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: "2006-01-02 15:04:05.000",
			NoColor:    false,
		}
		zlog = zerolog.New(writer).With().Timestamp().Logger()
	default:
		// Default to JSON for production
		zlog = zerolog.New(writer).With().Timestamp().Logger()
	}

	// Set log level
	zlog = zlog.Level(mapLogLevelToZerolog(level))

	// Add component if provided
	if component != "" {
		zlog = zlog.With().Str("component", component).Logger()
	}

	return &ZerologAdapter{
		zlogger:   zlog,
		component: component,
		format:    format,
		minLevel:  level,
		fields:    make(map[string]interface{}),
	}
}

// getLogFormatFromEnv reads LOG_FORMAT environment variable
func getLogFormatFromEnv() LogFormat {
	format := os.Getenv("LOG_FORMAT")
	switch strings.ToLower(format) {
	case "json":
		return FormatJSON
	case "console":
		return FormatConsole
	case "pretty":
		return FormatPretty
	default:
		// Default to pretty for development, json for production
		if os.Getenv("ENV") == "production" {
			return FormatJSON
		}
		return FormatPretty
	}
}

// getLogLevelFromEnv reads LOG_LEVEL environment variable
func getLogLevelFromEnv() LogLevel {
	level := os.Getenv("LOG_LEVEL")
	switch strings.ToUpper(level) {
	case "DEBUG":
		return DEBUG
	case "INFO":
		return INFO
	case "WARN", "WARNING":
		return WARN
	case "ERROR":
		return ERROR
	case "FATAL":
		return FATAL
	default:
		return INFO // Default to INFO
	}
}

// mapLogLevelToZerolog converts our LogLevel to zerolog.Level
func mapLogLevelToZerolog(level LogLevel) zerolog.Level {
	switch level {
	case DEBUG:
		return zerolog.DebugLevel
	case INFO:
		return zerolog.InfoLevel
	case WARN:
		return zerolog.WarnLevel
	case ERROR:
		return zerolog.ErrorLevel
	case FATAL:
		return zerolog.FatalLevel
	default:
		return zerolog.InfoLevel
	}
}

// WithLevel sets the minimum log level
func (z *ZerologAdapter) WithLevel(level LogLevel) *ZerologAdapter {
	z.minLevel = level
	z.zlogger = z.zlogger.Level(mapLogLevelToZerolog(level))
	return z
}

// WithOutput sets the output writer
func (z *ZerologAdapter) WithOutput(output io.Writer) *ZerologAdapter {
	z.zlogger = z.zlogger.Output(output)
	return z
}

// WithColor is a no-op for zerolog adapter (format is set at initialization)
func (z *ZerologAdapter) WithColor(enabled bool) *ZerologAdapter {
	// Color is determined by format, not a separate flag
	return z
}

// WithField adds a field to all log messages
func (z *ZerologAdapter) WithField(key string, value interface{}) *ZerologAdapter {
	z.fields[key] = value
	z.zlogger = z.zlogger.With().Interface(key, value).Logger()
	return z
}

// WithFields adds multiple fields
func (z *ZerologAdapter) WithFields(fields map[string]interface{}) *ZerologAdapter {
	for k, v := range fields {
		z.fields[k] = v
		z.zlogger = z.zlogger.With().Interface(k, v).Logger()
	}
	return z
}

// buildEvent creates a zerolog event with merged fields
func (z *ZerologAdapter) buildEvent(level zerolog.Level, fields map[string]interface{}) *zerolog.Event {
	var event *zerolog.Event
	switch level {
	case zerolog.DebugLevel:
		event = z.zlogger.Debug()
	case zerolog.InfoLevel:
		event = z.zlogger.Info()
	case zerolog.WarnLevel:
		event = z.zlogger.Warn()
	case zerolog.ErrorLevel:
		event = z.zlogger.Error()
	case zerolog.FatalLevel:
		event = z.zlogger.Fatal()
	default:
		event = z.zlogger.Info()
	}

	// Add one-time fields
	for k, v := range fields {
		event = event.Interface(k, v)
	}

	return event
}

// Debug logs a debug message
func (z *ZerologAdapter) Debug(message string) {
	z.buildEvent(zerolog.DebugLevel, nil).Msg(message)
}

// DebugWithFields logs a debug message with fields
func (z *ZerologAdapter) DebugWithFields(message string, fields map[string]interface{}) {
	z.buildEvent(zerolog.DebugLevel, fields).Msg(message)
}

// Info logs an info message
func (z *ZerologAdapter) Info(message string) {
	z.buildEvent(zerolog.InfoLevel, nil).Msg(message)
}

// InfoWithFields logs an info message with fields
func (z *ZerologAdapter) InfoWithFields(message string, fields map[string]interface{}) {
	z.buildEvent(zerolog.InfoLevel, fields).Msg(message)
}

// Warn logs a warning message
func (z *ZerologAdapter) Warn(message string) {
	z.buildEvent(zerolog.WarnLevel, nil).Msg(message)
}

// WarnWithFields logs a warning message with fields
func (z *ZerologAdapter) WarnWithFields(message string, fields map[string]interface{}) {
	z.buildEvent(zerolog.WarnLevel, fields).Msg(message)
}

// Error logs an error message
func (z *ZerologAdapter) Error(message string) {
	z.buildEvent(zerolog.ErrorLevel, nil).Msg(message)
}

// ErrorWithFields logs an error message with fields
func (z *ZerologAdapter) ErrorWithFields(message string, fields map[string]interface{}) {
	z.buildEvent(zerolog.ErrorLevel, fields).Msg(message)
}

// ErrorWithError logs an error with the error object
func (z *ZerologAdapter) ErrorWithError(message string, err error) {
	z.zlogger.Error().Err(err).Msg(message)
}

// Fatal logs a fatal message and exits
func (z *ZerologAdapter) Fatal(message string) {
	z.buildEvent(zerolog.FatalLevel, nil).Msg(message)
}

// FatalWithError logs a fatal message with error and exits
func (z *ZerologAdapter) FatalWithError(message string, err error) {
	z.zlogger.Fatal().Err(err).Msg(message)
}

// Performance logs a performance metric
func (z *ZerologAdapter) Performance(operation string, duration time.Duration) {
	z.zlogger.Info().
		Str("operation", operation).
		Int64("duration_ms", duration.Milliseconds()).
		Str("duration_str", duration.String()).
		Msg("Performance measurement")
}

// WithTimer returns a function that logs the duration when called
func (z *ZerologAdapter) WithTimer(operation string) func() {
	start := time.Now()
	return func() {
		z.Performance(operation, time.Since(start))
	}
}

// LogWithCaller logs a message with caller information
func (z *ZerologAdapter) LogWithCaller(level LogLevel, message string) {
	event := z.buildEvent(mapLogLevelToZerolog(level), nil)
	event.Caller(2).Msg(message)
}

// NewStructuredLogger creates a production-ready structured logger
// This is the recommended logger for new code
func NewStructuredLogger(component string) *ZerologAdapter {
	return NewZerologLogger(component)
}
