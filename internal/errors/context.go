package errors

import (
	"context"
	"fmt"
	"time"
)

// ExecutionContext captures the context of an operation for error reporting
type ExecutionContext struct {
	OperationType string
	OperationID   string
	UserID        string
	RequestID     string
	StartTime     time.Time
	Metadata      map[string]interface{}
}

// NewExecutionContext creates a new execution context
func NewExecutionContext(operationType, operationID string) *ExecutionContext {
	return &ExecutionContext{
		OperationType: operationType,
		OperationID:   operationID,
		StartTime:     time.Now(),
		Metadata:      make(map[string]interface{}),
	}
}

// WithUser adds user information to the context
func (ec *ExecutionContext) WithUser(userID string) *ExecutionContext {
	ec.UserID = userID
	return ec
}

// WithRequest adds request ID for tracing
func (ec *ExecutionContext) WithRequest(requestID string) *ExecutionContext {
	ec.RequestID = requestID
	return ec
}

// WithMetadata adds metadata to the context
func (ec *ExecutionContext) WithMetadata(key string, value interface{}) *ExecutionContext {
	ec.Metadata[key] = value
	return ec
}

// Duration returns how long the operation has been running
func (ec *ExecutionContext) Duration() time.Duration {
	return time.Since(ec.StartTime)
}

// ToMap converts execution context to a map for error context
func (ec *ExecutionContext) ToMap() map[string]interface{} {
	m := map[string]interface{}{
		"operation_type": ec.OperationType,
		"operation_id":   ec.OperationID,
		"duration_ms":    ec.Duration().Milliseconds(),
		"start_time":     ec.StartTime.Format(time.RFC3339),
	}

	if ec.UserID != "" {
		m["user_id"] = ec.UserID
	}

	if ec.RequestID != "" {
		m["request_id"] = ec.RequestID
	}

	for k, v := range ec.Metadata {
		m[k] = v
	}

	return m
}

// WrapError wraps an error with execution context
func (ec *ExecutionContext) WrapError(err error, message string) *RichError {
	richErr := NewRichError(CategorySystem, SeverityError, message).
		WithCause(err)

	// Add execution context
	for k, v := range ec.ToMap() {
		richErr = richErr.WithContext(k, v)
	}

	return richErr
}

// contextKey is the type for context keys
type contextKey string

const (
	executionContextKey contextKey = "execution_context"
	traceIDKey          contextKey = "trace_id"
)

// WithExecutionContext adds execution context to a Go context
func WithExecutionContext(ctx context.Context, execCtx *ExecutionContext) context.Context {
	return context.WithValue(ctx, executionContextKey, execCtx)
}

// GetExecutionContext retrieves execution context from a Go context
func GetExecutionContext(ctx context.Context) (*ExecutionContext, bool) {
	execCtx, ok := ctx.Value(executionContextKey).(*ExecutionContext)
	return execCtx, ok
}

// WithTraceID adds a trace ID to context for distributed tracing
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceIDKey, traceID)
}

// GetTraceID retrieves trace ID from context
func GetTraceID(ctx context.Context) string {
	traceID, ok := ctx.Value(traceIDKey).(string)
	if !ok {
		return ""
	}
	return traceID
}

// ContextualError captures an error with its execution context
type ContextualError struct {
	Error            error
	ExecutionContext *ExecutionContext
	Timestamp        time.Time
	Recovered        bool // Whether this was recovered from a panic
}

// NewContextualError creates a contextual error
func NewContextualError(err error, execCtx *ExecutionContext) *ContextualError {
	return &ContextualError{
		Error:            err,
		ExecutionContext: execCtx,
		Timestamp:        time.Now(),
	}
}

// Format formats the contextual error with all details
func (ce *ContextualError) Format() string {
	if ce.ExecutionContext == nil {
		return fmt.Sprintf("Error: %v", ce.Error)
	}

	return fmt.Sprintf(`
Error Details:
  Message: %v
  Operation: %s (ID: %s)
  Duration: %v
  Timestamp: %s
  User: %s
  Request ID: %s
`,
		ce.Error,
		ce.ExecutionContext.OperationType,
		ce.ExecutionContext.OperationID,
		ce.ExecutionContext.Duration(),
		ce.Timestamp.Format(time.RFC3339),
		ce.ExecutionContext.UserID,
		ce.ExecutionContext.RequestID,
	)
}

// ErrorWithContext wraps an error with context information from Go context
func ErrorWithContext(ctx context.Context, err error, message string) error {
	execCtx, ok := GetExecutionContext(ctx)
	if !ok {
		// No execution context, return simple rich error
		return NewRichError(CategorySystem, SeverityError, message).WithCause(err)
	}

	// Create rich error with execution context
	richErr := execCtx.WrapError(err, message)

	// Add trace ID if available
	if traceID := GetTraceID(ctx); traceID != "" {
		richErr = richErr.WithContext("trace_id", traceID)
	}

	return richErr
}

// RecoverWithContext recovers from panics and converts them to errors with context
func RecoverWithContext(ctx context.Context) error {
	if r := recover(); r != nil {
		err := fmt.Errorf("panic recovered: %v", r)

		execCtx, ok := GetExecutionContext(ctx)
		if !ok {
			return err
		}

		richErr := execCtx.WrapError(err, "Operation panicked")
		richErr.Severity = SeverityFatal
		richErr = richErr.WithContext("panic_value", r)
		richErr = richErr.WithSuggestion("This is a critical error - please report this bug")
		richErr = richErr.WithSuggestion("Include the full error details and stack trace")

		return richErr
	}
	return nil
}