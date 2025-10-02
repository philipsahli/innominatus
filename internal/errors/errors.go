package errors

import (
	"fmt"
	"runtime"
	"strings"
)

// ErrorCategory defines the type of error for classification
type ErrorCategory string

const (
	CategoryValidation    ErrorCategory = "validation"
	CategoryWorkflow      ErrorCategory = "workflow"
	CategoryResource      ErrorCategory = "resource"
	CategoryNetwork       ErrorCategory = "network"
	CategoryConfiguration ErrorCategory = "configuration"
	CategorySystem        ErrorCategory = "system"
)

// ErrorSeverity indicates how critical the error is
type ErrorSeverity string

const (
	SeverityFatal   ErrorSeverity = "fatal"
	SeverityError   ErrorSeverity = "error"
	SeverityWarning ErrorSeverity = "warning"
	SeverityInfo    ErrorSeverity = "info"
)

// Icon returns an emoji icon for the severity level
func (s ErrorSeverity) Icon() string {
	switch s {
	case SeverityFatal:
		return "💥"
	case SeverityError:
		return "❌"
	case SeverityWarning:
		return "⚠️"
	case SeverityInfo:
		return "ℹ️"
	default:
		return "•"
	}
}

// RichError represents an error with additional context
type RichError struct {
	Category    ErrorCategory
	Severity    ErrorSeverity
	Message     string
	Cause       error
	Context     map[string]interface{}
	Suggestions []string
	Retriable   bool
	StackTrace  []string
	Location    *ErrorLocation
}

// ErrorLocation provides file/line context for errors
type ErrorLocation struct {
	File   string
	Line   int
	Column int
	Source string // The actual line of source code
}

// Error implements the error interface
func (e *RichError) Error() string {
	return e.Message
}

// Unwrap implements error unwrapping for errors.Is and errors.As
func (e *RichError) Unwrap() error {
	return e.Cause
}

// Format provides a detailed, user-friendly error message
func (e *RichError) Format() string {
	var b strings.Builder

	// Error header with severity
	b.WriteString(fmt.Sprintf("\n%s %s: %s\n",
		severityIcon(e.Severity),
		strings.ToUpper(string(e.Severity)),
		e.Message))

	// Location information if available
	if e.Location != nil {
		b.WriteString(fmt.Sprintf("\n📍 Location: %s:%d:%d\n", e.Location.File, e.Location.Line, e.Location.Column))
		if e.Location.Source != "" {
			b.WriteString(fmt.Sprintf("   │ %s\n", e.Location.Source))
			// Add pointer to error column
			if e.Location.Column > 0 {
				padding := strings.Repeat(" ", e.Location.Column+4)
				b.WriteString(fmt.Sprintf("   │ %s^\n", padding))
			}
		}
	}

	// Context information
	if len(e.Context) > 0 {
		b.WriteString("\n📋 Context:\n")
		for key, value := range e.Context {
			b.WriteString(fmt.Sprintf("   • %s: %v\n", key, value))
		}
	}

	// Cause chain
	if e.Cause != nil {
		b.WriteString(fmt.Sprintf("\n🔗 Caused by: %v\n", e.Cause))
	}

	// Suggestions for fixing
	if len(e.Suggestions) > 0 {
		b.WriteString("\n💡 Suggestions:\n")
		for i, suggestion := range e.Suggestions {
			b.WriteString(fmt.Sprintf("   %d. %s\n", i+1, suggestion))
		}
	}

	// Retry information
	if e.Retriable {
		b.WriteString("\n🔄 This error may be transient. Try running the command again.\n")
	}

	return b.String()
}

// severityIcon returns an emoji icon for the severity level
func severityIcon(severity ErrorSeverity) string {
	switch severity {
	case SeverityFatal:
		return "💥"
	case SeverityError:
		return "❌"
	case SeverityWarning:
		return "⚠️"
	case SeverityInfo:
		return "ℹ️"
	default:
		return "•"
	}
}

// captureStackTrace captures the current stack trace
func captureStackTrace(skip int) []string {
	const maxDepth = 32
	var pcs [maxDepth]uintptr
	n := runtime.Callers(skip, pcs[:])

	var trace []string
	frames := runtime.CallersFrames(pcs[:n])
	for {
		frame, more := frames.Next()
		// Filter out runtime and system frames
		if !strings.Contains(frame.File, "runtime/") {
			trace = append(trace, fmt.Sprintf("%s:%d %s", frame.File, frame.Line, frame.Function))
		}
		if !more {
			break
		}
	}
	return trace
}

// NewRichError creates a new rich error with context
func NewRichError(category ErrorCategory, severity ErrorSeverity, message string) *RichError {
	return &RichError{
		Category:   category,
		Severity:   severity,
		Message:    message,
		Context:    make(map[string]interface{}),
		StackTrace: captureStackTrace(3),
	}
}

// WithCause adds a cause to the error
func (e *RichError) WithCause(cause error) *RichError {
	e.Cause = cause
	return e
}

// WithContext adds context information
func (e *RichError) WithContext(key string, value interface{}) *RichError {
	e.Context[key] = value
	return e
}

// WithLocation adds location information
func (e *RichError) WithLocation(file string, line, column int, source string) *RichError {
	e.Location = &ErrorLocation{
		File:   file,
		Line:   line,
		Column: column,
		Source: source,
	}
	return e
}

// WithSuggestion adds a suggestion for fixing the error
func (e *RichError) WithSuggestion(suggestion string) *RichError {
	e.Suggestions = append(e.Suggestions, suggestion)
	return e
}

// WithRetriable marks the error as potentially retriable
func (e *RichError) WithRetriable(retriable bool) *RichError {
	e.Retriable = retriable
	return e
}

// ValidationError represents a validation failure
type ValidationError struct {
	*RichError
	Field      string
	Value      interface{}
	Constraint string
}

// NewValidationError creates a validation error
func NewValidationError(field, message string) *ValidationError {
	return &ValidationError{
		RichError: NewRichError(CategoryValidation, SeverityError, message),
		Field:     field,
	}
}

// WorkflowError represents a workflow execution failure
type WorkflowError struct {
	*RichError
	WorkflowID string
	StepName   string
	StepIndex  int
}

// NewWorkflowError creates a workflow error
func NewWorkflowError(workflowID, stepName string, stepIndex int, message string) *WorkflowError {
	err := &WorkflowError{
		RichError:  NewRichError(CategoryWorkflow, SeverityError, message),
		WorkflowID: workflowID,
		StepName:   stepName,
		StepIndex:  stepIndex,
	}
	err.RichError = err.WithContext("workflow_id", workflowID)
	err.RichError = err.WithContext("step_name", stepName)
	err.RichError = err.WithContext("step_index", stepIndex)
	return err
}

// ResourceError represents a resource-related failure
type ResourceError struct {
	*RichError
	ResourceType string
	ResourceName string
	Operation    string
	Conflict     bool
}

// NewResourceError creates a resource error
func NewResourceError(resourceType, resourceName, operation, message string) *ResourceError {
	err := &ResourceError{
		RichError:    NewRichError(CategoryResource, SeverityError, message),
		ResourceType: resourceType,
		ResourceName: resourceName,
		Operation:    operation,
	}
	//nolint:staticcheck // Explicit field access improves code clarity - QF1008
	err.RichError = err.RichError.WithContext("resource_type", resourceType)
	err.RichError = err.RichError.WithContext("resource_name", resourceName) //nolint:staticcheck // QF1008
	err.RichError = err.RichError.WithContext("operation", operation)        //nolint:staticcheck // QF1008
	return err
}

// NetworkError represents a network-related failure
type NetworkError struct {
	*RichError
	URL        string
	StatusCode int
	Timeout    bool
}

// NewNetworkError creates a network error
func NewNetworkError(url, message string) *NetworkError {
	err := &NetworkError{
		RichError: NewRichError(CategoryNetwork, SeverityError, message),
		URL:       url,
	}
	err.RichError = err.WithContext("url", url)
	err.RichError = err.WithRetriable(true) // Network errors are often retriable
	return err
}

// ConfigurationError represents a configuration problem
type ConfigurationError struct {
	*RichError
	ConfigFile string
	ConfigKey  string
}

// NewConfigurationError creates a configuration error
func NewConfigurationError(configFile, configKey, message string) *ConfigurationError {
	err := &ConfigurationError{
		RichError:  NewRichError(CategoryConfiguration, SeverityError, message),
		ConfigFile: configFile,
		ConfigKey:  configKey,
	}
	err.RichError = err.RichError.WithContext("config_file", configFile) //nolint:staticcheck // QF1008
	err.RichError = err.RichError.WithContext("config_key", configKey)   //nolint:staticcheck // QF1008
	return err
}
