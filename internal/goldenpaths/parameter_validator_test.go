package goldenpaths

import (
	"testing"
)

func TestValidateParameterValue(t *testing.T) {
	tests := []struct {
		name      string
		paramName string
		value     string
		schema    *ParameterSchema
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "nil schema allows any value",
			paramName: "test",
			value:     "anything",
			schema:    nil,
			wantErr:   false,
		},
		{
			name:      "required parameter with empty value fails",
			paramName: "app-name",
			value:     "",
			schema: &ParameterSchema{
				Type:     "string",
				Required: true,
			},
			wantErr: true,
			errMsg:  "parameter is required",
		},
		{
			name:      "optional parameter with empty value passes",
			paramName: "description",
			value:     "",
			schema: &ParameterSchema{
				Type:     "string",
				Required: false,
			},
			wantErr: false,
		},
		{
			name:      "unsupported type fails",
			paramName: "test",
			value:     "value",
			schema: &ParameterSchema{
				Type: "unsupported",
			},
			wantErr: true,
			errMsg:  "unsupported parameter type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateParameterValue(tt.paramName, tt.value, tt.schema)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateParameterValue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.errMsg != "" {
				if _, ok := err.(*ParameterValidationError); !ok {
					t.Errorf("Expected ParameterValidationError, got %T", err)
				}
			}
		})
	}
}

func TestValidateString(t *testing.T) {
	tests := []struct {
		name      string
		paramName string
		value     string
		schema    *ParameterSchema
		wantErr   bool
	}{
		{
			name:      "valid string passes",
			paramName: "app-name",
			value:     "my-app",
			schema:    &ParameterSchema{Type: "string"},
			wantErr:   false,
		},
		{
			name:      "string matching pattern passes",
			paramName: "app-name",
			value:     "my-app",
			schema: &ParameterSchema{
				Type:    "string",
				Pattern: "^[a-z-]+$",
			},
			wantErr: false,
		},
		{
			name:      "string not matching pattern fails",
			paramName: "app-name",
			value:     "My App!",
			schema: &ParameterSchema{
				Type:    "string",
				Pattern: "^[a-z-]+$",
			},
			wantErr: true,
		},
		{
			name:      "string in allowed values passes",
			paramName: "environment",
			value:     "production",
			schema: &ParameterSchema{
				Type:          "string",
				AllowedValues: []string{"dev", "staging", "production"},
			},
			wantErr: false,
		},
		{
			name:      "string not in allowed values fails",
			paramName: "environment",
			value:     "test",
			schema: &ParameterSchema{
				Type:          "string",
				AllowedValues: []string{"dev", "staging", "production"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateString(tt.paramName, tt.value, tt.schema)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateString() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateInt(t *testing.T) {
	min5 := 5
	max100 := 100

	tests := []struct {
		name      string
		paramName string
		value     string
		schema    *ParameterSchema
		wantErr   bool
	}{
		{
			name:      "valid integer passes",
			paramName: "replicas",
			value:     "3",
			schema:    &ParameterSchema{Type: "int"},
			wantErr:   false,
		},
		{
			name:      "non-integer string fails",
			paramName: "replicas",
			value:     "abc",
			schema:    &ParameterSchema{Type: "int"},
			wantErr:   true,
		},
		{
			name:      "integer below min fails",
			paramName: "replicas",
			value:     "2",
			schema: &ParameterSchema{
				Type: "int",
				Min:  &min5,
			},
			wantErr: true,
		},
		{
			name:      "integer at min passes",
			paramName: "replicas",
			value:     "5",
			schema: &ParameterSchema{
				Type: "int",
				Min:  &min5,
			},
			wantErr: false,
		},
		{
			name:      "integer above max fails",
			paramName: "replicas",
			value:     "200",
			schema: &ParameterSchema{
				Type: "int",
				Max:  &max100,
			},
			wantErr: true,
		},
		{
			name:      "integer at max passes",
			paramName: "replicas",
			value:     "100",
			schema: &ParameterSchema{
				Type: "int",
				Max:  &max100,
			},
			wantErr: false,
		},
		{
			name:      "integer in range passes",
			paramName: "replicas",
			value:     "50",
			schema: &ParameterSchema{
				Type: "int",
				Min:  &min5,
				Max:  &max100,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateInt(tt.paramName, tt.value, tt.schema)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateInt() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateBool(t *testing.T) {
	tests := []struct {
		name      string
		paramName string
		value     string
		wantErr   bool
	}{
		{name: "true passes", paramName: "enabled", value: "true", wantErr: false},
		{name: "false passes", paramName: "enabled", value: "false", wantErr: false},
		{name: "yes passes", paramName: "enabled", value: "yes", wantErr: false},
		{name: "no passes", paramName: "enabled", value: "no", wantErr: false},
		{name: "1 passes", paramName: "enabled", value: "1", wantErr: false},
		{name: "0 passes", paramName: "enabled", value: "0", wantErr: false},
		{name: "on passes", paramName: "enabled", value: "on", wantErr: false},
		{name: "off passes", paramName: "enabled", value: "off", wantErr: false},
		{name: "True passes (case insensitive)", paramName: "enabled", value: "True", wantErr: false},
		{name: "YES passes (case insensitive)", paramName: "enabled", value: "YES", wantErr: false},
		{name: "invalid value fails", paramName: "enabled", value: "maybe", wantErr: true},
		{name: "random string fails", paramName: "enabled", value: "abc", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateBool(tt.paramName, tt.value, &ParameterSchema{Type: "bool"})
			if (err != nil) != tt.wantErr {
				t.Errorf("validateBool() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateDuration(t *testing.T) {
	tests := []struct {
		name      string
		paramName string
		value     string
		wantErr   bool
	}{
		{name: "hours format passes", paramName: "timeout", value: "2h", wantErr: false},
		{name: "minutes format passes", paramName: "timeout", value: "30m", wantErr: false},
		{name: "seconds format passes", paramName: "timeout", value: "90s", wantErr: false},
		{name: "days format passes", paramName: "retention", value: "7d", wantErr: false},
		{name: "weeks format passes", paramName: "retention", value: "2w", wantErr: false},
		{name: "combined format passes", paramName: "timeout", value: "1h30m", wantErr: false},
		{name: "milliseconds passes", paramName: "timeout", value: "500ms", wantErr: false},
		{name: "invalid format fails", paramName: "timeout", value: "abc", wantErr: true},
		{name: "invalid days format fails", paramName: "timeout", value: "xd", wantErr: true},
		{name: "invalid weeks format fails", paramName: "timeout", value: "yw", wantErr: true},
		{name: "negative duration passes (allowed by time.ParseDuration)", paramName: "timeout", value: "-1h", wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateDuration(tt.paramName, tt.value, &ParameterSchema{Type: "duration"})
			if (err != nil) != tt.wantErr {
				t.Errorf("validateDuration() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateEnum(t *testing.T) {
	tests := []struct {
		name      string
		paramName string
		value     string
		schema    *ParameterSchema
		wantErr   bool
	}{
		{
			name:      "value in enum passes",
			paramName: "size",
			value:     "medium",
			schema: &ParameterSchema{
				Type:          "enum",
				AllowedValues: []string{"small", "medium", "large"},
			},
			wantErr: false,
		},
		{
			name:      "value not in enum fails",
			paramName: "size",
			value:     "xlarge",
			schema: &ParameterSchema{
				Type:          "enum",
				AllowedValues: []string{"small", "medium", "large"},
			},
			wantErr: true,
		},
		{
			name:      "empty allowed values fails",
			paramName: "size",
			value:     "medium",
			schema: &ParameterSchema{
				Type:          "enum",
				AllowedValues: []string{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateEnum(tt.paramName, tt.value, tt.schema)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateEnum() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNormalizeBoolValue(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{name: "true", value: "true", want: "true"},
		{name: "True", value: "True", want: "true"},
		{name: "TRUE", value: "TRUE", want: "true"},
		{name: "yes", value: "yes", want: "true"},
		{name: "Yes", value: "Yes", want: "true"},
		{name: "1", value: "1", want: "true"},
		{name: "on", value: "on", want: "true"},
		{name: "ON", value: "ON", want: "true"},
		{name: "false", value: "false", want: "false"},
		{name: "False", value: "False", want: "false"},
		{name: "FALSE", value: "FALSE", want: "false"},
		{name: "no", value: "no", want: "false"},
		{name: "No", value: "No", want: "false"},
		{name: "0", value: "0", want: "false"},
		{name: "off", value: "off", want: "false"},
		{name: "OFF", value: "OFF", want: "false"},
		{name: "invalid returns unchanged", value: "maybe", want: "maybe"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeBoolValue(tt.value)
			if got != tt.want {
				t.Errorf("NormalizeBoolValue() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParameterValidationError_Error(t *testing.T) {
	tests := []struct {
		name string
		err  *ParameterValidationError
		want string
	}{
		{
			name: "full error message",
			err: &ParameterValidationError{
				ParameterName: "replicas",
				ProvidedValue: "abc",
				ExpectedType:  "int",
				Constraint:    "value must be a valid integer",
				Suggestion:    "provide a numeric value like: 1, 42, 100",
			},
			want: "parameter 'replicas' validation failed: provided value 'abc', value must be a valid integer. Suggestion: provide a numeric value like: 1, 42, 100",
		},
		{
			name: "minimal error message",
			err: &ParameterValidationError{
				ParameterName: "test",
			},
			want: "parameter 'test' validation failed",
		},
		{
			name: "error with constraint only",
			err: &ParameterValidationError{
				ParameterName: "app-name",
				Constraint:    "parameter is required",
			},
			want: "parameter 'app-name' validation failed, parameter is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			if got != tt.want {
				t.Errorf("ParameterValidationError.Error() = %v, want %v", got, tt.want)
			}
		})
	}
}
