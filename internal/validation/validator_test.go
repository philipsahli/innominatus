package validation

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ===== Mock Validator for Testing =====

type MockValidator struct {
	component string
	valid     bool
	errors    []string
	warnings  []string
}

func (m *MockValidator) GetComponent() string {
	return m.component
}

func (m *MockValidator) Validate() *ValidationResult {
	return &ValidationResult{
		Valid:     m.valid,
		Errors:    m.errors,
		Warnings:  m.warnings,
		Component: m.component,
	}
}

// ===== Validation Suite Tests =====

func TestNewValidationSuite(t *testing.T) {
	suite := NewValidationSuite("test-suite")

	if suite == nil {
		t.Fatal("NewValidationSuite() returned nil")
	}

	if suite.name != "test-suite" {
		t.Errorf("NewValidationSuite() name = %v, want test-suite", suite.name)
	}

	if suite.validators == nil {
		t.Error("NewValidationSuite() validators is nil")
	}
}

func TestValidationSuite_AddValidator(t *testing.T) {
	suite := NewValidationSuite("test-suite")
	validator := &MockValidator{component: "mock", valid: true}

	suite.AddValidator(validator)

	if len(suite.validators) != 1 {
		t.Errorf("AddValidator() count = %v, want 1", len(suite.validators))
	}
}

func TestValidationSuite_ValidateAllSuccess(t *testing.T) {
	suite := NewValidationSuite("test-suite")

	suite.AddValidator(&MockValidator{
		component: "comp1",
		valid:     true,
		errors:    nil,
		warnings:  []string{"warning1"},
	})

	suite.AddValidator(&MockValidator{
		component: "comp2",
		valid:     true,
		errors:    nil,
		warnings:  nil,
	})

	summary := suite.ValidateAll()

	if !summary.Valid {
		t.Error("ValidateAll() Valid = false, want true")
	}

	if summary.ErrorCount != 0 {
		t.Errorf("ValidateAll() ErrorCount = %v, want 0", summary.ErrorCount)
	}

	if summary.WarningCount != 1 {
		t.Errorf("ValidateAll() WarningCount = %v, want 1", summary.WarningCount)
	}

	if len(summary.Results) != 2 {
		t.Errorf("ValidateAll() Results count = %v, want 2", len(summary.Results))
	}
}

func TestValidationSuite_ValidateAllWithErrors(t *testing.T) {
	suite := NewValidationSuite("test-suite")

	suite.AddValidator(&MockValidator{
		component: "comp1",
		valid:     false,
		errors:    []string{"error1", "error2"},
		warnings:  []string{"warning1"},
	})

	suite.AddValidator(&MockValidator{
		component: "comp2",
		valid:     true,
	})

	summary := suite.ValidateAll()

	if summary.Valid {
		t.Error("ValidateAll() Valid = true, want false")
	}

	if summary.ErrorCount != 2 {
		t.Errorf("ValidateAll() ErrorCount = %v, want 2", summary.ErrorCount)
	}

	if summary.WarningCount != 1 {
		t.Errorf("ValidateAll() WarningCount = %v, want 1", summary.WarningCount)
	}
}

// ===== URL Validation Tests =====

func TestValidateURL(t *testing.T) {
	tests := []struct {
		name           string
		url            string
		allowedSchemes []string
		expectError    bool
	}{
		{
			name:        "valid HTTPS URL",
			url:         "https://example.com",
			expectError: false,
		},
		{
			name:        "valid HTTP URL",
			url:         "http://example.com",
			expectError: false,
		},
		{
			name:           "valid URL with allowed scheme",
			url:            "https://example.com",
			allowedSchemes: []string{"https"},
			expectError:    false,
		},
		{
			name:           "URL with disallowed scheme",
			url:            "ftp://example.com",
			allowedSchemes: []string{"http", "https"},
			expectError:    true,
		},
		{
			name:        "empty URL",
			url:         "",
			expectError: true,
		},
		{
			name:        "URL without scheme",
			url:         "example.com",
			expectError: true,
		},
		{
			name:        "URL without host",
			url:         "https://",
			expectError: true,
		},
		{
			name:        "malformed URL",
			url:         "://invalid",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateURL(tt.url, tt.allowedSchemes)
			if (err != nil) != tt.expectError {
				t.Errorf("ValidateURL() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}

// ===== File Validation Tests =====

func TestValidateFileExists(t *testing.T) {
	// Create a temporary test file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(tmpFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tests := []struct {
		name        string
		filePath    string
		expectError bool
		setup       func() string
	}{
		{
			name:        "valid existing file",
			filePath:    tmpFile,
			expectError: false,
		},
		{
			name:        "non-existent file",
			filePath:    filepath.Join(tmpDir, "nonexistent.txt"),
			expectError: true,
		},
		{
			name:        "empty path",
			filePath:    "",
			expectError: true,
		},
		{
			name:        "directory instead of file",
			filePath:    tmpDir,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFileExists(tt.filePath)
			if (err != nil) != tt.expectError {
				t.Errorf("ValidateFileExists() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}

func TestValidateDirectoryExists(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "file.txt")
	_ = os.WriteFile(tmpFile, []byte("test"), 0644)

	tests := []struct {
		name        string
		dirPath     string
		expectError bool
	}{
		{
			name:        "valid existing directory",
			dirPath:     tmpDir,
			expectError: false,
		},
		{
			name:        "non-existent directory",
			dirPath:     filepath.Join(tmpDir, "nonexistent"),
			expectError: true,
		},
		{
			name:        "empty path",
			dirPath:     "",
			expectError: true,
		},
		{
			name:        "file instead of directory",
			dirPath:     tmpFile,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDirectoryExists(tt.dirPath)
			if (err != nil) != tt.expectError {
				t.Errorf("ValidateDirectoryExists() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}

// ===== Field Validation Tests =====

func TestValidateRequired(t *testing.T) {
	tests := []struct {
		name        string
		fieldName   string
		value       string
		expectError bool
	}{
		{"valid value", "username", "john", false},
		{"empty value", "username", "", true},
		{"whitespace only", "username", "   ", true},
		{"value with spaces", "name", "John Doe", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRequired(tt.fieldName, tt.value)
			if (err != nil) != tt.expectError {
				t.Errorf("ValidateRequired() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}

func TestValidateEnum(t *testing.T) {
	allowedValues := []string{"dev", "staging", "prod"}

	tests := []struct {
		name        string
		value       string
		expectError bool
	}{
		{"valid value dev", "dev", false},
		{"valid value prod", "prod", false},
		{"invalid value", "test", true},
		{"empty value", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEnum("environment", tt.value, allowedValues)
			if (err != nil) != tt.expectError {
				t.Errorf("ValidateEnum() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}

func TestValidateRegex(t *testing.T) {
	tests := []struct {
		name        string
		value       string
		pattern     string
		description string
		expectError bool
	}{
		{
			name:        "valid email",
			value:       "user@example.com",
			pattern:     `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`,
			description: "email format",
			expectError: false,
		},
		{
			name:        "invalid email",
			value:       "not-an-email",
			pattern:     `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`,
			description: "email format",
			expectError: true,
		},
		{
			name:        "empty value",
			value:       "",
			pattern:     `^[a-z]+$`,
			description: "lowercase letters",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRegex("email", tt.value, tt.pattern, tt.description)
			if (err != nil) != tt.expectError {
				t.Errorf("ValidateRegex() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}

// ===== Password Validation Tests =====

func TestValidatePasswordStrength(t *testing.T) {
	tests := []struct {
		name        string
		password    string
		expectError bool
	}{
		{"strong password", "Passw0rd!", false},
		{"too short", "Pass1!", true},
		{"no uppercase", "password123!", true},
		{"no lowercase", "PASSWORD123!", true},
		{"no number", "Password!", true},
		{"no special char", "Password123", true},
		{"minimum valid", "Abcd123!", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePasswordStrength(tt.password)
			if (err != nil) != tt.expectError {
				t.Errorf("ValidatePasswordStrength() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}

// ===== API Key Validation Tests =====

func TestValidateAPIKeyFormat(t *testing.T) {
	tests := []struct {
		name        string
		apiKey      string
		expectError bool
	}{
		{
			name:        "valid 64-char hex key",
			apiKey:      strings.Repeat("a", 64),
			expectError: false,
		},
		{
			name:        "valid base64 key",
			apiKey:      "dGhpc2lzYXRlc3RrZXl0aGF0aXNsb25nZW5vdWdo",
			expectError: false,
		},
		{
			name:        "too short",
			apiKey:      "short",
			expectError: true,
		},
		{
			name:        "invalid characters",
			apiKey:      strings.Repeat("#", 64),
			expectError: true,
		},
		{
			name:        "32 chars but invalid format",
			apiKey:      "this-is-a-test-key-32-chars!",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAPIKeyFormat(tt.apiKey)
			if (err != nil) != tt.expectError {
				t.Errorf("ValidateAPIKeyFormat() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}

// ===== Validation Result Tests =====

func TestValidationResult(t *testing.T) {
	result := &ValidationResult{
		Valid:     false,
		Errors:    []string{"error1", "error2"},
		Warnings:  []string{"warning1"},
		Component: "test-component",
	}

	if result.Valid {
		t.Error("ValidationResult.Valid should be false")
	}

	if len(result.Errors) != 2 {
		t.Errorf("ValidationResult.Errors count = %v, want 2", len(result.Errors))
	}

	if len(result.Warnings) != 1 {
		t.Errorf("ValidationResult.Warnings count = %v, want 1", len(result.Warnings))
	}

	if result.Component != "test-component" {
		t.Errorf("ValidationResult.Component = %v, want test-component", result.Component)
	}
}

// ===== Validation Summary Tests =====

func TestValidationSummary_EmptySuite(t *testing.T) {
	suite := NewValidationSuite("empty-suite")
	summary := suite.ValidateAll()

	if !summary.Valid {
		t.Error("Empty suite should be valid")
	}

	if summary.ErrorCount != 0 {
		t.Errorf("Empty suite ErrorCount = %v, want 0", summary.ErrorCount)
	}

	if summary.WarningCount != 0 {
		t.Errorf("Empty suite WarningCount = %v, want 0", summary.WarningCount)
	}

	if len(summary.Results) != 0 {
		t.Errorf("Empty suite Results count = %v, want 0", len(summary.Results))
	}
}

func TestValidationSummary_MixedResults(t *testing.T) {
	suite := NewValidationSuite("mixed-suite")

	// Add valid validator
	suite.AddValidator(&MockValidator{
		component: "valid-comp",
		valid:     true,
		warnings:  []string{"warning"},
	})

	// Add invalid validator
	suite.AddValidator(&MockValidator{
		component: "invalid-comp",
		valid:     false,
		errors:    []string{"error1", "error2"},
	})

	summary := suite.ValidateAll()

	if summary.Valid {
		t.Error("Suite with errors should not be valid")
	}

	if summary.ErrorCount != 2 {
		t.Errorf("ErrorCount = %v, want 2", summary.ErrorCount)
	}

	if summary.WarningCount != 1 {
		t.Errorf("WarningCount = %v, want 1", summary.WarningCount)
	}

	if summary.SuiteName != "mixed-suite" {
		t.Errorf("SuiteName = %v, want mixed-suite", summary.SuiteName)
	}
}
