package logging

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
)

// LogLevel represents the severity of a log message
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
	FATAL
)

// String returns the string representation of the log level
func (l LogLevel) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	case FATAL:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// Color returns the ANSI color code for the log level
func (l LogLevel) Color() string {
	switch l {
	case DEBUG:
		return "\033[36m" // Cyan
	case INFO:
		return "\033[32m" // Green
	case WARN:
		return "\033[33m" // Yellow
	case ERROR:
		return "\033[31m" // Red
	case FATAL:
		return "\033[35m" // Magenta
	default:
		return "\033[0m" // Reset
	}
}

// Icon returns an emoji icon for the log level
func (l LogLevel) Icon() string {
	switch l {
	case DEBUG:
		return "🔍"
	case INFO:
		return "ℹ️"
	case WARN:
		return "⚠️"
	case ERROR:
		return "❌"
	case FATAL:
		return "💥"
	default:
		return "•"
	}
}

// Logger provides structured logging with context
type Logger struct {
	component    string
	minLevel     LogLevel
	output       io.Writer
	colorEnabled bool
	fields       map[string]interface{}
	mu           sync.Mutex
}

// NewLogger creates a new logger instance
func NewLogger(component string) *Logger {
	return &Logger{
		component:    component,
		minLevel:     INFO,
		output:       os.Stdout,
		colorEnabled: true,
		fields:       make(map[string]interface{}),
	}
}

// WithLevel sets the minimum log level
func (l *Logger) WithLevel(level LogLevel) *Logger {
	l.minLevel = level
	return l
}

// WithOutput sets the output writer
func (l *Logger) WithOutput(output io.Writer) *Logger {
	l.output = output
	return l
}

// WithColor enables or disables colored output
func (l *Logger) WithColor(enabled bool) *Logger {
	l.colorEnabled = enabled
	return l
}

// WithField adds a field to all log messages
func (l *Logger) WithField(key string, value interface{}) *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.fields[key] = value
	return l
}

// WithFields adds multiple fields
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()
	for k, v := range fields {
		l.fields[k] = v
	}
	return l
}

// log is the internal logging function
func (l *Logger) log(level LogLevel, message string, fields map[string]interface{}) {
	if level < l.minLevel {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	timestamp := time.Now().Format("2006-01-02 15:04:05.000")

	// Build the log message
	var b strings.Builder

	// Timestamp
	if l.colorEnabled {
		b.WriteString("\033[90m") // Dark gray
	}
	b.WriteString(timestamp)
	if l.colorEnabled {
		b.WriteString("\033[0m")
	}
	b.WriteString(" ")

	// Level with icon and color
	if l.colorEnabled {
		b.WriteString(level.Color())
	}
	b.WriteString(level.Icon())
	b.WriteString(" ")
	b.WriteString(fmt.Sprintf("%-5s", level.String()))
	if l.colorEnabled {
		b.WriteString("\033[0m")
	}
	b.WriteString(" ")

	// Component
	if l.component != "" {
		if l.colorEnabled {
			b.WriteString("\033[1m") // Bold
		}
		b.WriteString(fmt.Sprintf("[%s]", l.component))
		if l.colorEnabled {
			b.WriteString("\033[0m")
		}
		b.WriteString(" ")
	}

	// Message
	b.WriteString(message)

	// Fields (both persistent and one-time)
	allFields := make(map[string]interface{})
	for k, v := range l.fields {
		allFields[k] = v
	}
	for k, v := range fields {
		allFields[k] = v
	}

	if len(allFields) > 0 {
		b.WriteString(" ")
		if l.colorEnabled {
			b.WriteString("\033[90m") // Dark gray
		}

		first := true
		for k, v := range allFields {
			if !first {
				b.WriteString(", ")
			}
			b.WriteString(fmt.Sprintf("%s=%v", k, v))
			first = false
		}

		if l.colorEnabled {
			b.WriteString("\033[0m")
		}
	}

	b.WriteString("\n")

	// Write to output
	fmt.Fprint(l.output, b.String())
}

// Debug logs a debug message
func (l *Logger) Debug(message string) {
	l.log(DEBUG, message, nil)
}

// DebugWithFields logs a debug message with fields
func (l *Logger) DebugWithFields(message string, fields map[string]interface{}) {
	l.log(DEBUG, message, fields)
}

// Info logs an info message
func (l *Logger) Info(message string) {
	l.log(INFO, message, nil)
}

// InfoWithFields logs an info message with fields
func (l *Logger) InfoWithFields(message string, fields map[string]interface{}) {
	l.log(INFO, message, fields)
}

// Warn logs a warning message
func (l *Logger) Warn(message string) {
	l.log(WARN, message, nil)
}

// WarnWithFields logs a warning message with fields
func (l *Logger) WarnWithFields(message string, fields map[string]interface{}) {
	l.log(WARN, message, fields)
}

// Error logs an error message
func (l *Logger) Error(message string) {
	l.log(ERROR, message, nil)
}

// ErrorWithFields logs an error message with fields
func (l *Logger) ErrorWithFields(message string, fields map[string]interface{}) {
	l.log(ERROR, message, fields)
}

// ErrorWithError logs an error with the error object
func (l *Logger) ErrorWithError(message string, err error) {
	fields := map[string]interface{}{
		"error": err.Error(),
	}
	l.log(ERROR, message, fields)
}

// Fatal logs a fatal message and exits
func (l *Logger) Fatal(message string) {
	l.log(FATAL, message, nil)
	os.Exit(1)
}

// FatalWithError logs a fatal message with error and exits
func (l *Logger) FatalWithError(message string, err error) {
	fields := map[string]interface{}{
		"error": err.Error(),
	}
	l.log(FATAL, message, fields)
	os.Exit(1)
}

// Performance logs a performance metric
func (l *Logger) Performance(operation string, duration time.Duration) {
	fields := map[string]interface{}{
		"operation":    operation,
		"duration_ms":  duration.Milliseconds(),
		"duration_str": duration.String(),
	}
	l.log(INFO, "Performance measurement", fields)
}

// WithTimer returns a function that logs the duration when called
func (l *Logger) WithTimer(operation string) func() {
	start := time.Now()
	return func() {
		l.Performance(operation, time.Since(start))
	}
}

// LogWithCaller logs a message with caller information
func (l *Logger) LogWithCaller(level LogLevel, message string) {
	_, file, line, ok := runtime.Caller(2)
	fields := make(map[string]interface{})
	if ok {
		// Extract just the filename, not the full path
		parts := strings.Split(file, "/")
		filename := parts[len(parts)-1]
		fields["caller"] = fmt.Sprintf("%s:%d", filename, line)
	}
	l.log(level, message, fields)
}

// Global default logger
var defaultLogger = NewLogger("innominatus")

// SetDefaultLogger sets the global default logger
func SetDefaultLogger(logger *Logger) {
	defaultLogger = logger
}

// GetDefaultLogger returns the global default logger
func GetDefaultLogger() *Logger {
	return defaultLogger
}

// Convenience functions using the default logger

// Debug logs a debug message using the default logger
func Debug(message string) {
	defaultLogger.Debug(message)
}

// Info logs an info message using the default logger
func Info(message string) {
	defaultLogger.Info(message)
}

// Warn logs a warning message using the default logger
func Warn(message string) {
	defaultLogger.Warn(message)
}

// Error logs an error message using the default logger
func Error(message string) {
	defaultLogger.Error(message)
}

// ErrorWithError logs an error with error object using the default logger
func ErrorWithError(message string, err error) {
	defaultLogger.ErrorWithError(message, err)
}

// Fatal logs a fatal message and exits using the default logger
func Fatal(message string) {
	defaultLogger.Fatal(message)
}

// Performance logs a performance measurement using the default logger
func Performance(operation string, duration time.Duration) {
	defaultLogger.Performance(operation, duration)
}