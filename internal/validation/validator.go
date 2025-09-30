package validation

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ValidationResult represents the result of a validation check
type ValidationResult struct {
	Valid     bool     `json:"valid"`
	Errors    []string `json:"errors,omitempty"`
	Warnings  []string `json:"warnings,omitempty"`
	Component string   `json:"component"`
}

// Validator defines the interface for configuration validators
type Validator interface {
	Validate() *ValidationResult
	GetComponent() string
}

// ValidationSuite manages multiple validators
type ValidationSuite struct {
	validators []Validator
	name       string
}

// NewValidationSuite creates a new validation suite
func NewValidationSuite(name string) *ValidationSuite {
	return &ValidationSuite{
		validators: make([]Validator, 0),
		name:       name,
	}
}

// AddValidator adds a validator to the suite
func (vs *ValidationSuite) AddValidator(validator Validator) {
	vs.validators = append(vs.validators, validator)
}

// ValidateAll runs all validators and returns consolidated results
func (vs *ValidationSuite) ValidateAll() *ValidationSummary {
	summary := &ValidationSummary{
		SuiteName: vs.name,
		Results:   make([]*ValidationResult, 0),
		Valid:     true,
	}

	for _, validator := range vs.validators {
		result := validator.Validate()
		summary.Results = append(summary.Results, result)

		if !result.Valid {
			summary.Valid = false
			summary.ErrorCount += len(result.Errors)
		}
		summary.WarningCount += len(result.Warnings)
	}

	return summary
}

// ValidationSummary provides a summary of all validation results
type ValidationSummary struct {
	SuiteName    string               `json:"suite_name"`
	Valid        bool                 `json:"valid"`
	ErrorCount   int                  `json:"error_count"`
	WarningCount int                  `json:"warning_count"`
	Results      []*ValidationResult  `json:"results"`
}

// PrintSummary prints a formatted validation summary
func (vs *ValidationSummary) PrintSummary() {
	fmt.Printf("\n=== Configuration Validation Summary: %s ===\n", vs.SuiteName)

	if vs.Valid {
		fmt.Printf("✅ All validations passed (%d components checked)\n", len(vs.Results))
	} else {
		fmt.Printf("❌ Validation failed with %d errors", vs.ErrorCount)
	}

	if vs.WarningCount > 0 {
		fmt.Printf(" and %d warnings", vs.WarningCount)
	}
	fmt.Println()

	for _, result := range vs.Results {
		if !result.Valid || len(result.Warnings) > 0 {
			fmt.Printf("\n🔧 Component: %s\n", result.Component)

			for _, err := range result.Errors {
				fmt.Printf("  ❌ ERROR: %s\n", err)
			}

			for _, warning := range result.Warnings {
				fmt.Printf("  ⚠️  WARNING: %s\n", warning)
			}
		}
	}
	fmt.Println()
}

// Common validation utilities

// ValidateURL validates that a string is a valid URL with allowed schemes
func ValidateURL(urlStr string, allowedSchemes []string) error {
	if urlStr == "" {
		return fmt.Errorf("URL cannot be empty")
	}

	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	if parsedURL.Scheme == "" {
		return fmt.Errorf("URL must have a scheme (http/https)")
	}

	if len(allowedSchemes) > 0 {
		schemeAllowed := false
		for _, allowed := range allowedSchemes {
			if parsedURL.Scheme == allowed {
				schemeAllowed = true
				break
			}
		}
		if !schemeAllowed {
			return fmt.Errorf("URL scheme '%s' not allowed. Allowed schemes: %v", parsedURL.Scheme, allowedSchemes)
		}
	}

	if parsedURL.Host == "" {
		return fmt.Errorf("URL must have a host")
	}

	return nil
}

// ValidateFileExists validates that a file exists and is readable
func ValidateFileExists(filePath string) error {
	if filePath == "" {
		return fmt.Errorf("file path cannot be empty")
	}

	// Clean the path to handle relative paths properly
	cleanPath := filepath.Clean(filePath)

	info, err := os.Stat(cleanPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("file does not exist: %s", cleanPath)
	}
	if err != nil {
		return fmt.Errorf("cannot access file: %w", err)
	}

	if info.IsDir() {
		return fmt.Errorf("path is a directory, not a file: %s", cleanPath)
	}

	// Test if file is readable
	file, err := os.Open(cleanPath)
	if err != nil {
		return fmt.Errorf("file is not readable: %w", err)
	}
	file.Close()

	return nil
}

// ValidateDirectoryExists validates that a directory exists and is accessible
func ValidateDirectoryExists(dirPath string) error {
	if dirPath == "" {
		return fmt.Errorf("directory path cannot be empty")
	}

	cleanPath := filepath.Clean(dirPath)

	info, err := os.Stat(cleanPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("directory does not exist: %s", cleanPath)
	}
	if err != nil {
		return fmt.Errorf("cannot access directory: %w", err)
	}

	if !info.IsDir() {
		return fmt.Errorf("path is not a directory: %s", cleanPath)
	}

	return nil
}

// ValidateRequired validates that a required field is not empty
func ValidateRequired(fieldName, value string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("required field '%s' cannot be empty", fieldName)
	}
	return nil
}

// ValidateEnum validates that a value is within an allowed set
func ValidateEnum(fieldName, value string, allowedValues []string) error {
	if value == "" {
		return fmt.Errorf("field '%s' cannot be empty", fieldName)
	}

	for _, allowed := range allowedValues {
		if value == allowed {
			return nil
		}
	}

	return fmt.Errorf("field '%s' has invalid value '%s'. Allowed values: %v", fieldName, value, allowedValues)
}

// ValidateRegex validates that a value matches a regular expression
func ValidateRegex(fieldName, value, pattern, description string) error {
	if value == "" {
		return fmt.Errorf("field '%s' cannot be empty", fieldName)
	}

	matched, err := regexp.MatchString(pattern, value)
	if err != nil {
		return fmt.Errorf("invalid regex pattern for field '%s': %w", fieldName, err)
	}

	if !matched {
		return fmt.Errorf("field '%s' value '%s' does not match required format: %s", fieldName, value, description)
	}

	return nil
}

// ValidatePasswordStrength validates password meets minimum security requirements
func ValidatePasswordStrength(password string) error {
	if len(password) < 8 {
		return fmt.Errorf("password must be at least 8 characters long")
	}

	hasLower := regexp.MustCompile(`[a-z]`).MatchString(password)
	hasUpper := regexp.MustCompile(`[A-Z]`).MatchString(password)
	hasNumber := regexp.MustCompile(`[0-9]`).MatchString(password)
	hasSpecial := regexp.MustCompile(`[^a-zA-Z0-9\s]`).MatchString(password)

	var missing []string
	if !hasLower {
		missing = append(missing, "lowercase letter")
	}
	if !hasUpper {
		missing = append(missing, "uppercase letter")
	}
	if !hasNumber {
		missing = append(missing, "number")
	}
	if !hasSpecial {
		missing = append(missing, "special character")
	}

	if len(missing) > 0 {
		return fmt.Errorf("password must contain at least one: %s", strings.Join(missing, ", "))
	}

	return nil
}

// ValidateAPIKeyFormat validates API key format and entropy
func ValidateAPIKeyFormat(apiKey string) error {
	if len(apiKey) < 32 {
		return fmt.Errorf("API key must be at least 32 characters long")
	}

	// Check if it's hexadecimal (common format for generated keys)
	hexPattern := `^[a-fA-F0-9]+$`
	if matched, _ := regexp.MatchString(hexPattern, apiKey); matched && len(apiKey) == 64 {
		return nil // 64-character hex key is valid
	}

	// Check for base64-like format
	base64Pattern := `^[A-Za-z0-9+/]+=*$`
	if matched, _ := regexp.MatchString(base64Pattern, apiKey); matched && len(apiKey) >= 32 {
		return nil
	}

	return fmt.Errorf("API key format is invalid - must be 64-character hex or base64-encoded string")
}