package logging

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

// GenerateTraceID generates a unique trace ID for request tracking
func GenerateTraceID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp-based ID if random fails
		return fmt.Sprintf("trace-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(bytes)
}

// contextKey is the type for context keys to avoid collisions
type contextKey string

const (
	traceIDKey  contextKey = "trace_id"
	loggerKey   contextKey = "logger"
	requestKey  contextKey = "request_id"
	userIDKey   contextKey = "user_id"
	operKey     contextKey = "operation"
)

// WithTraceID adds a trace ID to the context
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceIDKey, traceID)
}

// GetTraceID retrieves the trace ID from context
func GetTraceID(ctx context.Context) string {
	if traceID, ok := ctx.Value(traceIDKey).(string); ok {
		return traceID
	}
	return ""
}

// WithLogger attaches a logger to the context
func WithLogger(ctx context.Context, logger *Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

// GetLogger retrieves the logger from context, or returns default
func GetLogger(ctx context.Context) *Logger {
	if logger, ok := ctx.Value(loggerKey).(*Logger); ok {
		return logger
	}
	return defaultLogger
}

// WithRequestID adds a request ID to the context
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestKey, requestID)
}

// GetRequestID retrieves the request ID from context
func GetRequestID(ctx context.Context) string {
	if requestID, ok := ctx.Value(requestKey).(string); ok {
		return requestID
	}
	return ""
}

// WithUserID adds a user ID to the context
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// GetUserID retrieves the user ID from context
func GetUserID(ctx context.Context) string {
	if userID, ok := ctx.Value(userIDKey).(string); ok {
		return userID
	}
	return ""
}

// WithOperation adds an operation name to the context
func WithOperation(ctx context.Context, operation string) context.Context {
	return context.WithValue(ctx, operKey, operation)
}

// GetOperation retrieves the operation name from context
func GetOperation(ctx context.Context) string {
	if operation, ok := ctx.Value(operKey).(string); ok {
		return operation
	}
	return ""
}

// ContextLogger wraps a logger with context information
type ContextLogger struct {
	logger  *Logger
	ctx     context.Context
	fields  map[string]interface{}
}

// NewContextLogger creates a new context-aware logger
func NewContextLogger(ctx context.Context, component string) *ContextLogger {
	logger := GetLogger(ctx)
	if logger == nil {
		logger = NewLogger(component)
	}

	cl := &ContextLogger{
		logger: logger,
		ctx:    ctx,
		fields: make(map[string]interface{}),
	}

	// Auto-populate fields from context
	if traceID := GetTraceID(ctx); traceID != "" {
		cl.fields["trace_id"] = traceID
	}
	if requestID := GetRequestID(ctx); requestID != "" {
		cl.fields["request_id"] = requestID
	}
	if userID := GetUserID(ctx); userID != "" {
		cl.fields["user_id"] = userID
	}
	if operation := GetOperation(ctx); operation != "" {
		cl.fields["operation"] = operation
	}

	return cl
}

// WithField adds a field to the context logger
func (cl *ContextLogger) WithField(key string, value interface{}) *ContextLogger {
	cl.fields[key] = value
	return cl
}

// WithFields adds multiple fields to the context logger
func (cl *ContextLogger) WithFields(fields map[string]interface{}) *ContextLogger {
	for k, v := range fields {
		cl.fields[k] = v
	}
	return cl
}

// mergeFields combines context fields with provided fields
func (cl *ContextLogger) mergeFields(fields map[string]interface{}) map[string]interface{} {
	merged := make(map[string]interface{})
	for k, v := range cl.fields {
		merged[k] = v
	}
	for k, v := range fields {
		merged[k] = v
	}
	return merged
}

// Debug logs a debug message with context
func (cl *ContextLogger) Debug(message string) {
	cl.logger.log(DEBUG, message, cl.fields)
}

// DebugWithFields logs a debug message with additional fields
func (cl *ContextLogger) DebugWithFields(message string, fields map[string]interface{}) {
	cl.logger.log(DEBUG, message, cl.mergeFields(fields))
}

// Info logs an info message with context
func (cl *ContextLogger) Info(message string) {
	cl.logger.log(INFO, message, cl.fields)
}

// InfoWithFields logs an info message with additional fields
func (cl *ContextLogger) InfoWithFields(message string, fields map[string]interface{}) {
	cl.logger.log(INFO, message, cl.mergeFields(fields))
}

// Warn logs a warning message with context
func (cl *ContextLogger) Warn(message string) {
	cl.logger.log(WARN, message, cl.fields)
}

// WarnWithFields logs a warning message with additional fields
func (cl *ContextLogger) WarnWithFields(message string, fields map[string]interface{}) {
	cl.logger.log(WARN, message, cl.mergeFields(fields))
}

// Error logs an error message with context
func (cl *ContextLogger) Error(message string) {
	cl.logger.log(ERROR, message, cl.fields)
}

// ErrorWithFields logs an error message with additional fields
func (cl *ContextLogger) ErrorWithFields(message string, fields map[string]interface{}) {
	cl.logger.log(ERROR, message, cl.mergeFields(fields))
}

// ErrorWithError logs an error with error object
func (cl *ContextLogger) ErrorWithError(message string, err error) {
	fields := map[string]interface{}{
		"error": err.Error(),
	}
	cl.logger.log(ERROR, message, cl.mergeFields(fields))
}

// Performance logs a performance metric with context
func (cl *ContextLogger) Performance(operation string, duration time.Duration) {
	fields := map[string]interface{}{
		"operation":    operation,
		"duration_ms":  duration.Milliseconds(),
		"duration_str": duration.String(),
	}
	cl.logger.log(INFO, "Performance measurement", cl.mergeFields(fields))
}

// WithTimer returns a function that logs duration when called
func (cl *ContextLogger) WithTimer(operation string) func() {
	start := time.Now()
	return func() {
		cl.Performance(operation, time.Since(start))
	}
}

// Convenience functions for context-aware logging

// DebugContext logs a debug message using the logger from context
func DebugContext(ctx context.Context, message string) {
	logger := NewContextLogger(ctx, "")
	logger.Debug(message)
}

// InfoContext logs an info message using the logger from context
func InfoContext(ctx context.Context, message string) {
	logger := NewContextLogger(ctx, "")
	logger.Info(message)
}

// WarnContext logs a warning message using the logger from context
func WarnContext(ctx context.Context, message string) {
	logger := NewContextLogger(ctx, "")
	logger.Warn(message)
}

// ErrorContext logs an error message using the logger from context
func ErrorContext(ctx context.Context, message string) {
	logger := NewContextLogger(ctx, "")
	logger.Error(message)
}

// ErrorWithErrorContext logs an error with error object using context
func ErrorWithErrorContext(ctx context.Context, message string, err error) {
	logger := NewContextLogger(ctx, "")
	logger.ErrorWithError(message, err)
}