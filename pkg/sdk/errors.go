package sdk

import "fmt"

// SDKError is the base error type for all SDK errors
type SDKError struct {
	Code    string
	Message string
	Cause   error
}

func (e *SDKError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *SDKError) Unwrap() error {
	return e.Cause
}

// Error codes
const (
	ErrCodeProvisionFailed    = "PROVISION_FAILED"
	ErrCodeDeprovisionFailed  = "DEPROVISION_FAILED"
	ErrCodeStatusCheckFailed  = "STATUS_CHECK_FAILED"
	ErrCodeInvalidConfig      = "INVALID_CONFIG"
	ErrCodeInvalidResource    = "INVALID_RESOURCE"
	ErrCodeInvalidPlatformErr = "INVALID_PLATFORM"
	ErrCodeNotFound           = "NOT_FOUND"
	ErrCodeAlreadyExists      = "ALREADY_EXISTS"
	ErrCodeTimeout            = "TIMEOUT"
	ErrCodeUnauthorized       = "UNAUTHORIZED"
)

// ErrProvisionFailed creates a provision failure error
func ErrProvisionFailed(message string, args ...interface{}) *SDKError {
	return &SDKError{
		Code:    ErrCodeProvisionFailed,
		Message: fmt.Sprintf(message, args...),
	}
}

// ErrDeprovisionFailed creates a deprovision failure error
func ErrDeprovisionFailed(message string, args ...interface{}) *SDKError {
	return &SDKError{
		Code:    ErrCodeDeprovisionFailed,
		Message: fmt.Sprintf(message, args...),
	}
}

// ErrStatusCheckFailed creates a status check failure error
func ErrStatusCheckFailed(message string, args ...interface{}) *SDKError {
	return &SDKError{
		Code:    ErrCodeStatusCheckFailed,
		Message: fmt.Sprintf(message, args...),
	}
}

// ErrInvalidConfig creates an invalid configuration error
func ErrInvalidConfig(message string, args ...interface{}) *SDKError {
	return &SDKError{
		Code:    ErrCodeInvalidConfig,
		Message: fmt.Sprintf(message, args...),
	}
}

// ErrInvalidResource creates an invalid resource error
func ErrInvalidResource(message string, args ...interface{}) *SDKError {
	return &SDKError{
		Code:    ErrCodeInvalidResource,
		Message: fmt.Sprintf(message, args...),
	}
}

// ErrInvalidPlatform creates an invalid platform error (legacy, use ErrInvalidProvider)
func ErrInvalidPlatform(message string, args ...interface{}) *SDKError {
	return &SDKError{
		Code:    ErrCodeInvalidPlatformErr,
		Message: fmt.Sprintf(message, args...),
	}
}

// ErrInvalidProvider creates an invalid provider error
func ErrInvalidProvider(message string, args ...interface{}) *SDKError {
	return &SDKError{
		Code:    ErrCodeInvalidPlatformErr, // Same code for backward compatibility
		Message: fmt.Sprintf(message, args...),
	}
}

// ErrNotFound creates a not found error
func ErrNotFound(message string, args ...interface{}) *SDKError {
	return &SDKError{
		Code:    ErrCodeNotFound,
		Message: fmt.Sprintf(message, args...),
	}
}

// ErrAlreadyExists creates an already exists error
func ErrAlreadyExists(message string, args ...interface{}) *SDKError {
	return &SDKError{
		Code:    ErrCodeAlreadyExists,
		Message: fmt.Sprintf(message, args...),
	}
}

// ErrTimeout creates a timeout error
func ErrTimeout(message string, args ...interface{}) *SDKError {
	return &SDKError{
		Code:    ErrCodeTimeout,
		Message: fmt.Sprintf(message, args...),
	}
}

// ErrUnauthorized creates an unauthorized error
func ErrUnauthorized(message string, args ...interface{}) *SDKError {
	return &SDKError{
		Code:    ErrCodeUnauthorized,
		Message: fmt.Sprintf(message, args...),
	}
}
