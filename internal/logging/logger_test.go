package logging

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"
)

// TestLogLevel tests log level enum
func TestLogLevel(t *testing.T) {
	tests := []struct {
		level    LogLevel
		expected string
	}{
		{DEBUG, "DEBUG"},
		{INFO, "INFO"},
		{WARN, "WARN"},
		{ERROR, "ERROR"},
		{FATAL, "FATAL"},
		{LogLevel(99), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := tt.level.String()
			if result != tt.expected {
				t.Errorf("LogLevel.String() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestLogLevelColor(t *testing.T) {
	tests := []struct {
		level LogLevel
		name  string
	}{
		{DEBUG, "cyan"},
		{INFO, "green"},
		{WARN, "yellow"},
		{ERROR, "red"},
		{FATAL, "magenta"},
		{LogLevel(99), "reset"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			color := tt.level.Color()
			if color == "" {
				t.Errorf("LogLevel.Color() returned empty string")
			}
		})
	}
}

func TestLogLevelIcon(t *testing.T) {
	tests := []struct {
		level        LogLevel
		expectedIcon string
	}{
		{DEBUG, "🔍"},
		{INFO, "ℹ️"},
		{WARN, "⚠️"},
		{ERROR, "❌"},
		{FATAL, "💥"},
		{LogLevel(99), "•"},
	}

	for _, tt := range tests {
		t.Run(tt.level.String(), func(t *testing.T) {
			icon := tt.level.Icon()
			if icon != tt.expectedIcon {
				t.Errorf("LogLevel.Icon() = %v, want %v", icon, tt.expectedIcon)
			}
		})
	}
}

func TestNewLogger(t *testing.T) {
	logger := NewLogger("test")

	if logger.component != "test" {
		t.Errorf("NewLogger() component = %v, want test", logger.component)
	}

	if logger.minLevel != INFO {
		t.Errorf("NewLogger() minLevel = %v, want INFO", logger.minLevel)
	}

	if logger.output == nil {
		t.Error("NewLogger() output is nil")
	}

	if !logger.colorEnabled {
		t.Error("NewLogger() colorEnabled = false, want true")
	}

	if logger.fields == nil {
		t.Error("NewLogger() fields map is nil")
	}
}

func TestLoggerWithLevel(t *testing.T) {
	logger := NewLogger("test").WithLevel(DEBUG)

	if logger.minLevel != DEBUG {
		t.Errorf("WithLevel() minLevel = %v, want DEBUG", logger.minLevel)
	}
}

func TestLoggerWithOutput(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewLogger("test").WithOutput(buf)

	if logger.output != buf {
		t.Error("WithOutput() did not set output correctly")
	}
}

func TestLoggerWithColor(t *testing.T) {
	logger := NewLogger("test").WithColor(false)

	if logger.colorEnabled {
		t.Error("WithColor(false) did not disable color")
	}

	logger = logger.WithColor(true)
	if !logger.colorEnabled {
		t.Error("WithColor(true) did not enable color")
	}
}

func TestLoggerWithField(t *testing.T) {
	logger := NewLogger("test").WithField("user", "alice")

	if logger.fields["user"] != "alice" {
		t.Errorf("WithField() user = %v, want alice", logger.fields["user"])
	}
}

func TestLoggerWithFields(t *testing.T) {
	fields := map[string]interface{}{
		"user":      "alice",
		"requestId": "123",
		"status":    200,
	}

	logger := NewLogger("test").WithFields(fields)

	for k, v := range fields {
		if logger.fields[k] != v {
			t.Errorf("WithFields() %s = %v, want %v", k, logger.fields[k], v)
		}
	}
}

func TestLoggerDebug(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewLogger("test").
		WithOutput(buf).
		WithLevel(DEBUG).
		WithColor(false)

	logger.Debug("debug message")

	output := buf.String()
	if !strings.Contains(output, "DEBUG") {
		t.Error("Debug() output missing DEBUG level")
	}
	if !strings.Contains(output, "debug message") {
		t.Error("Debug() output missing message")
	}
}

func TestLoggerInfo(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewLogger("test").
		WithOutput(buf).
		WithColor(false)

	logger.Info("info message")

	output := buf.String()
	if !strings.Contains(output, "INFO") {
		t.Error("Info() output missing INFO level")
	}
	if !strings.Contains(output, "info message") {
		t.Error("Info() output missing message")
	}
}

func TestLoggerWarn(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewLogger("test").
		WithOutput(buf).
		WithColor(false)

	logger.Warn("warning message")

	output := buf.String()
	if !strings.Contains(output, "WARN") {
		t.Error("Warn() output missing WARN level")
	}
	if !strings.Contains(output, "warning message") {
		t.Error("Warn() output missing message")
	}
}

func TestLoggerError(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewLogger("test").
		WithOutput(buf).
		WithColor(false)

	logger.Error("error message")

	output := buf.String()
	if !strings.Contains(output, "ERROR") {
		t.Error("Error() output missing ERROR level")
	}
	if !strings.Contains(output, "error message") {
		t.Error("Error() output missing message")
	}
}

func TestLoggerWithFieldsInMessage(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewLogger("test").
		WithOutput(buf).
		WithColor(false)

	logger.InfoWithFields("test message", map[string]interface{}{
		"user": "alice",
		"age":  30,
	})

	output := buf.String()
	if !strings.Contains(output, "user=alice") {
		t.Error("InfoWithFields() output missing user field")
	}
	if !strings.Contains(output, "age=30") {
		t.Error("InfoWithFields() output missing age field")
	}
}

func TestLoggerErrorWithError(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewLogger("test").
		WithOutput(buf).
		WithColor(false)

	testErr := errors.New("test error")
	logger.ErrorWithError("operation failed", testErr)

	output := buf.String()
	if !strings.Contains(output, "operation failed") {
		t.Error("ErrorWithError() output missing message")
	}
	if !strings.Contains(output, "test error") {
		t.Error("ErrorWithError() output missing error message")
	}
}

func TestLoggerPerformance(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewLogger("test").
		WithOutput(buf).
		WithColor(false)

	logger.Performance("test_operation", 150*time.Millisecond)

	output := buf.String()
	if !strings.Contains(output, "Performance measurement") {
		t.Error("Performance() output missing message")
	}
	if !strings.Contains(output, "operation=test_operation") {
		t.Error("Performance() output missing operation field")
	}
	if !strings.Contains(output, "duration_ms=150") {
		t.Error("Performance() output missing duration_ms field")
	}
}

func TestLoggerWithTimer(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewLogger("test").
		WithOutput(buf).
		WithColor(false)

	done := logger.WithTimer("test_operation")

	// Simulate some work
	time.Sleep(10 * time.Millisecond)

	done()

	output := buf.String()
	if !strings.Contains(output, "Performance measurement") {
		t.Error("WithTimer() output missing message")
	}
	if !strings.Contains(output, "operation=test_operation") {
		t.Error("WithTimer() output missing operation field")
	}
}

func TestLoggerLogWithCaller(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewLogger("test").
		WithOutput(buf).
		WithColor(false)

	logger.LogWithCaller(INFO, "test message")

	output := buf.String()
	if !strings.Contains(output, "caller=") {
		t.Error("LogWithCaller() output missing caller field")
	}
	if !strings.Contains(output, ".go:") {
		t.Error("LogWithCaller() output missing line number")
	}
}

func TestLoggerMinLevel(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewLogger("test").
		WithOutput(buf).
		WithLevel(ERROR).
		WithColor(false)

	// These should not be logged
	logger.Debug("debug message")
	logger.Info("info message")
	logger.Warn("warn message")

	// This should be logged
	logger.Error("error message")

	output := buf.String()
	if strings.Contains(output, "debug message") {
		t.Error("DEBUG message logged when minLevel is ERROR")
	}
	if strings.Contains(output, "info message") {
		t.Error("INFO message logged when minLevel is ERROR")
	}
	if strings.Contains(output, "warn message") {
		t.Error("WARN message logged when minLevel is ERROR")
	}
	if !strings.Contains(output, "error message") {
		t.Error("ERROR message not logged when minLevel is ERROR")
	}
}

func TestLoggerComponent(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewLogger("mycomponent").
		WithOutput(buf).
		WithColor(false)

	logger.Info("test message")

	output := buf.String()
	if !strings.Contains(output, "[mycomponent]") {
		t.Error("Logger output missing component name")
	}
}

func TestLoggerPersistentFields(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewLogger("test").
		WithOutput(buf).
		WithColor(false).
		WithField("requestId", "123")

	logger.Info("message 1")
	logger.Info("message 2")

	output := buf.String()

	// Both messages should have the persistent field
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 2 {
		t.Fatalf("Expected 2 log lines, got %d", len(lines))
	}

	for i, line := range lines {
		if !strings.Contains(line, "requestId=123") {
			t.Errorf("Log line %d missing persistent field requestId", i+1)
		}
	}
}

func TestGlobalLogger(t *testing.T) {
	// Save original default logger
	original := defaultLogger

	// Create a new logger
	buf := &bytes.Buffer{}
	testLogger := NewLogger("global-test").
		WithOutput(buf).
		WithColor(false)

	SetDefaultLogger(testLogger)

	// Test that GetDefaultLogger returns the same instance
	retrieved := GetDefaultLogger()
	if retrieved != testLogger {
		t.Error("GetDefaultLogger() did not return the set logger")
	}

	// Test global convenience functions
	Info("global info message")

	output := buf.String()
	if !strings.Contains(output, "global info message") {
		t.Error("Global Info() did not use default logger")
	}

	// Restore original
	SetDefaultLogger(original)
}

func TestGlobalConvenienceFunctions(t *testing.T) {
	buf := &bytes.Buffer{}
	testLogger := NewLogger("global-test").
		WithOutput(buf).
		WithLevel(DEBUG).
		WithColor(false)

	original := defaultLogger
	SetDefaultLogger(testLogger)
	defer SetDefaultLogger(original)

	// Test all convenience functions
	Debug("debug msg")
	Info("info msg")
	Warn("warn msg")
	Error("error msg")
	ErrorWithError("error with err", errors.New("test err"))
	Performance("perf test", 100*time.Millisecond)

	output := buf.String()

	expectedStrings := []string{
		"debug msg",
		"info msg",
		"warn msg",
		"error msg",
		"error with err",
		"test err",
		"perf test",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("Global functions output missing: %s", expected)
		}
	}
}

func TestLoggerColorOutput(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewLogger("test").
		WithOutput(buf).
		WithColor(true) // Enable color

	logger.Info("colored message")

	output := buf.String()

	// Should contain ANSI color codes
	if !strings.Contains(output, "\033[") {
		t.Error("Colored output missing ANSI escape codes")
	}
}

func TestLoggerNoColorOutput(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewLogger("test").
		WithOutput(buf).
		WithColor(false) // Disable color

	logger.Info("plain message")

	output := buf.String()

	// Should NOT contain ANSI color codes
	if strings.Contains(output, "\033[") {
		t.Error("Non-colored output contains ANSI escape codes")
	}
}

func TestLoggerConcurrentWrites(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewLogger("test").
		WithOutput(buf).
		WithColor(false)

	// Test concurrent logging (should not panic due to mutex)
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				logger.InfoWithFields("concurrent test", map[string]interface{}{
					"goroutine": id,
					"iteration": j,
				})
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Just verify we got output (exact count may vary due to buffering)
	output := buf.String()
	if len(output) == 0 {
		t.Error("No output from concurrent writes")
	}
}
