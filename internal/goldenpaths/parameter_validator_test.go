package goldenpaths

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateParameterValue_String(t *testing.T) {
	tests := []struct {
		name        string
		paramName   string
		value       string
		schema      *ParameterSchema
		expectError bool
		errorMsg    string
	}{
		{
			name:      "valid string without constraints",
			paramName: "app_name",
			value:     "myapp",
			schema: &ParameterSchema{
				Type: "string",
			},
			expectError: false,
		},
		{
			name:      "valid string with pattern",
			paramName: "app_name",
			value:     "my-app-123",
			schema: &ParameterSchema{
				Type:    "string",
				Pattern: `^[a-z][a-z0-9\-]*$`,
			},
			expectError: false,
		},
		{
			name:      "invalid string - pattern mismatch",
			paramName: "app_name",
			value:     "My-App-123",
			schema: &ParameterSchema{
				Type:    "string",
				Pattern: `^[a-z][a-z0-9\-]*$`,
			},
			expectError: true,
			errorMsg:    "must match pattern",
		},
		{
			name:      "valid string from allowed values",
			paramName: "environment",
			value:     "production",
			schema: &ParameterSchema{
				Type:          "string",
				AllowedValues: []string{"development", "staging", "production"},
			},
			expectError: false,
		},
		{
			name:      "invalid string - not in allowed values",
			paramName: "environment",
			value:     "testing",
			schema: &ParameterSchema{
				Type:          "string",
				AllowedValues: []string{"development", "staging", "production"},
			},
			expectError: true,
			errorMsg:    "must be one of",
		},
		{
			name:      "empty value for required parameter",
			paramName: "app_name",
			value:     "",
			schema: &ParameterSchema{
				Type:     "string",
				Required: true,
			},
			expectError: true,
			errorMsg:    "required",
		},
		{
			name:      "empty value for optional parameter",
			paramName: "description",
			value:     "",
			schema: &ParameterSchema{
				Type:     "string",
				Required: false,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateParameterValue(tt.paramName, tt.value, tt.schema)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateParameterValue_Int(t *testing.T) {
	min5 := 5
	max100 := 100

	tests := []struct {
		name        string
		paramName   string
		value       string
		schema      *ParameterSchema
		expectError bool
		errorMsg    string
	}{
		{
			name:      "valid integer",
			paramName: "replicas",
			value:     "3",
			schema: &ParameterSchema{
				Type: "int",
			},
			expectError: false,
		},
		{
			name:      "invalid integer - not numeric",
			paramName: "replicas",
			value:     "abc",
			schema: &ParameterSchema{
				Type: "int",
			},
			expectError: true,
			errorMsg:    "valid integer",
		},
		{
			name:      "valid integer within range",
			paramName: "replicas",
			value:     "10",
			schema: &ParameterSchema{
				Type: "int",
				Min:  &min5,
				Max:  &max100,
			},
			expectError: false,
		},
		{
			name:      "integer below minimum",
			paramName: "replicas",
			value:     "3",
			schema: &ParameterSchema{
				Type: "int",
				Min:  &min5,
			},
			expectError: true,
			errorMsg:    "must be >= 5",
		},
		{
			name:      "integer above maximum",
			paramName: "replicas",
			value:     "150",
			schema: &ParameterSchema{
				Type: "int",
				Max:  &max100,
			},
			expectError: true,
			errorMsg:    "must be <= 100",
		},
		{
			name:      "integer at exact minimum",
			paramName: "replicas",
			value:     "5",
			schema: &ParameterSchema{
				Type: "int",
				Min:  &min5,
			},
			expectError: false,
		},
		{
			name:      "integer at exact maximum",
			paramName: "replicas",
			value:     "100",
			schema: &ParameterSchema{
				Type: "int",
				Max:  &max100,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateParameterValue(tt.paramName, tt.value, tt.schema)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateParameterValue_Bool(t *testing.T) {
	tests := []struct {
		name        string
		paramName   string
		value       string
		schema      *ParameterSchema
		expectError bool
	}{
		{name: "true", paramName: "enabled", value: "true", schema: &ParameterSchema{Type: "bool"}, expectError: false},
		{name: "false", paramName: "enabled", value: "false", schema: &ParameterSchema{Type: "bool"}, expectError: false},
		{name: "yes", paramName: "enabled", value: "yes", schema: &ParameterSchema{Type: "bool"}, expectError: false},
		{name: "no", paramName: "enabled", value: "no", schema: &ParameterSchema{Type: "bool"}, expectError: false},
		{name: "1", paramName: "enabled", value: "1", schema: &ParameterSchema{Type: "bool"}, expectError: false},
		{name: "0", paramName: "enabled", value: "0", schema: &ParameterSchema{Type: "bool"}, expectError: false},
		{name: "on", paramName: "enabled", value: "on", schema: &ParameterSchema{Type: "bool"}, expectError: false},
		{name: "off", paramName: "enabled", value: "off", schema: &ParameterSchema{Type: "bool"}, expectError: false},
		{name: "TRUE (uppercase)", paramName: "enabled", value: "TRUE", schema: &ParameterSchema{Type: "bool"}, expectError: false},
		{name: "Yes (mixed case)", paramName: "enabled", value: "Yes", schema: &ParameterSchema{Type: "bool"}, expectError: false},
		{name: "invalid", paramName: "enabled", value: "maybe", schema: &ParameterSchema{Type: "bool"}, expectError: true},
		{name: "invalid number", paramName: "enabled", value: "2", schema: &ParameterSchema{Type: "bool"}, expectError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateParameterValue(tt.paramName, tt.value, tt.schema)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "must be a boolean")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateParameterValue_Duration(t *testing.T) {
	tests := []struct {
		name        string
		paramName   string
		value       string
		schema      *ParameterSchema
		expectError bool
		errorMsg    string
	}{
		{name: "hours", paramName: "ttl", value: "2h", schema: &ParameterSchema{Type: "duration"}, expectError: false},
		{name: "minutes", paramName: "ttl", value: "30m", schema: &ParameterSchema{Type: "duration"}, expectError: false},
		{name: "seconds", paramName: "ttl", value: "90s", schema: &ParameterSchema{Type: "duration"}, expectError: false},
		{name: "days", paramName: "ttl", value: "7d", schema: &ParameterSchema{Type: "duration"}, expectError: false},
		{name: "weeks", paramName: "ttl", value: "2w", schema: &ParameterSchema{Type: "duration"}, expectError: false},
		{name: "combined", paramName: "ttl", value: "1h30m", schema: &ParameterSchema{Type: "duration"}, expectError: false},
		{name: "invalid format", paramName: "ttl", value: "abc", schema: &ParameterSchema{Type: "duration"}, expectError: true, errorMsg: "invalid duration"},
		{name: "invalid unit", paramName: "ttl", value: "2x", schema: &ParameterSchema{Type: "duration"}, expectError: true, errorMsg: "invalid duration"},
		{
			name:      "valid duration matching pattern",
			paramName: "ttl",
			value:     "2h",
			schema: &ParameterSchema{
				Type:    "duration",
				Pattern: `^\d+[hmd]$`,
			},
			expectError: false,
		},
		{
			name:      "duration not matching pattern",
			paramName: "ttl",
			value:     "2w",
			schema: &ParameterSchema{
				Type:    "duration",
				Pattern: `^\d+[hmd]$`, // only h, m, d allowed
			},
			expectError: true,
			errorMsg:    "must match pattern",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateParameterValue(tt.paramName, tt.value, tt.schema)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateParameterValue_Enum(t *testing.T) {
	tests := []struct {
		name        string
		paramName   string
		value       string
		schema      *ParameterSchema
		expectError bool
		errorMsg    string
	}{
		{
			name:      "valid enum value",
			paramName: "environment_type",
			value:     "preview",
			schema: &ParameterSchema{
				Type:          "enum",
				AllowedValues: []string{"preview", "staging", "development"},
			},
			expectError: false,
		},
		{
			name:      "invalid enum value",
			paramName: "environment_type",
			value:     "production",
			schema: &ParameterSchema{
				Type:          "enum",
				AllowedValues: []string{"preview", "staging", "development"},
			},
			expectError: true,
			errorMsg:    "must be one of",
		},
		{
			name:      "enum with no allowed values",
			paramName: "environment_type",
			value:     "preview",
			schema: &ParameterSchema{
				Type:          "enum",
				AllowedValues: []string{},
			},
			expectError: true,
			errorMsg:    "no allowed values defined",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateParameterValue(tt.paramName, tt.value, tt.schema)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateParameterValue_UnsupportedType(t *testing.T) {
	err := ValidateParameterValue("param", "value", &ParameterSchema{Type: "unsupported"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported parameter type")
}

func TestValidateParameterValue_NoSchema(t *testing.T) {
	// No schema means no validation (backward compatibility)
	err := ValidateParameterValue("param", "value", nil)
	assert.NoError(t, err)
}

func TestParameterValidationError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *ParameterValidationError
		expected string
	}{
		{
			name: "full error with all fields",
			err: &ParameterValidationError{
				ParameterName: "replicas",
				ProvidedValue: "150",
				ExpectedType:  "int",
				Constraint:    "value must be <= 100",
				Suggestion:    "reduce the number of replicas",
			},
			expected: "parameter 'replicas' validation failed: provided value '150', value must be <= 100. Suggestion: reduce the number of replicas",
		},
		{
			name: "error without provided value",
			err: &ParameterValidationError{
				ParameterName: "app_name",
				ExpectedType:  "string",
				Constraint:    "parameter is required",
			},
			expected: "parameter 'app_name' validation failed, parameter is required",
		},
		{
			name: "error without suggestion",
			err: &ParameterValidationError{
				ParameterName: "enabled",
				ProvidedValue: "maybe",
				Constraint:    "value must be a boolean",
			},
			expected: "parameter 'enabled' validation failed: provided value 'maybe', value must be a boolean",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.err.Error())
		})
	}
}

func TestNormalizeBoolValue(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"true", "true"},
		{"false", "false"},
		{"yes", "true"},
		{"no", "false"},
		{"1", "true"},
		{"0", "false"},
		{"on", "true"},
		{"off", "false"},
		{"TRUE", "true"},
		{"FALSE", "false"},
		{"Yes", "true"},
		{"No", "false"},
		{"invalid", "invalid"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := NormalizeBoolValue(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
