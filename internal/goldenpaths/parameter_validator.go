package goldenpaths

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ParameterValidationError represents a parameter validation error with rich context
type ParameterValidationError struct {
	ParameterName string
	ProvidedValue string
	ExpectedType  string
	Constraint    string
	Suggestion    string
}

// Error implements the error interface
func (e *ParameterValidationError) Error() string {
	msg := fmt.Sprintf("parameter '%s' validation failed", e.ParameterName)
	if e.ProvidedValue != "" {
		msg += fmt.Sprintf(": provided value '%s'", e.ProvidedValue)
	}
	if e.Constraint != "" {
		msg += fmt.Sprintf(", %s", e.Constraint)
	}
	if e.Suggestion != "" {
		msg += fmt.Sprintf(". Suggestion: %s", e.Suggestion)
	}
	return msg
}

// ValidateParameterValue validates a parameter value against its schema
func ValidateParameterValue(paramName string, value string, schema *ParameterSchema) error {
	if schema == nil {
		// No schema means no validation (backward compatibility)
		return nil
	}

	// Handle empty values
	if value == "" {
		if schema.Required {
			return &ParameterValidationError{
				ParameterName: paramName,
				ExpectedType:  schema.Type,
				Constraint:    "parameter is required",
				Suggestion:    fmt.Sprintf("provide a value of type '%s'", schema.Type),
			}
		}
		// Empty value for optional parameter is OK
		return nil
	}

	// Validate based on type
	switch schema.Type {
	case "string", "":
		return validateString(paramName, value, schema)
	case "int", "integer":
		return validateInt(paramName, value, schema)
	case "bool", "boolean":
		return validateBool(paramName, value, schema)
	case "duration":
		return validateDuration(paramName, value, schema)
	case "enum":
		return validateEnum(paramName, value, schema)
	default:
		return &ParameterValidationError{
			ParameterName: paramName,
			ProvidedValue: value,
			Constraint:    fmt.Sprintf("unsupported parameter type '%s'", schema.Type),
			Suggestion:    "use one of: string, int, bool, duration, enum",
		}
	}
}

// validateString validates a string parameter
func validateString(paramName string, value string, schema *ParameterSchema) error {
	// Check pattern if specified
	if schema.Pattern != "" {
		matched, err := regexp.MatchString(schema.Pattern, value)
		if err != nil {
			return &ParameterValidationError{
				ParameterName: paramName,
				ProvidedValue: value,
				Constraint:    fmt.Sprintf("invalid regex pattern: %s", err.Error()),
			}
		}
		if !matched {
			return &ParameterValidationError{
				ParameterName: paramName,
				ProvidedValue: value,
				ExpectedType:  "string",
				Constraint:    fmt.Sprintf("value must match pattern '%s'", schema.Pattern),
				Suggestion:    schema.Description,
			}
		}
	}

	// Check allowed values if specified (can be used for string enums)
	if len(schema.AllowedValues) > 0 {
		for _, allowed := range schema.AllowedValues {
			if value == allowed {
				return nil
			}
		}
		return &ParameterValidationError{
			ParameterName: paramName,
			ProvidedValue: value,
			ExpectedType:  "string",
			Constraint:    fmt.Sprintf("value must be one of: %s", strings.Join(schema.AllowedValues, ", ")),
		}
	}

	return nil
}

// validateInt validates an integer parameter
func validateInt(paramName string, value string, schema *ParameterSchema) error {
	intValue, err := strconv.Atoi(value)
	if err != nil {
		return &ParameterValidationError{
			ParameterName: paramName,
			ProvidedValue: value,
			ExpectedType:  "int",
			Constraint:    "value must be a valid integer",
			Suggestion:    "provide a numeric value like: 1, 42, 100",
		}
	}

	// Check min constraint
	if schema.Min != nil && intValue < *schema.Min {
		return &ParameterValidationError{
			ParameterName: paramName,
			ProvidedValue: value,
			ExpectedType:  "int",
			Constraint:    fmt.Sprintf("value must be >= %d", *schema.Min),
		}
	}

	// Check max constraint
	if schema.Max != nil && intValue > *schema.Max {
		return &ParameterValidationError{
			ParameterName: paramName,
			ProvidedValue: value,
			ExpectedType:  "int",
			Constraint:    fmt.Sprintf("value must be <= %d", *schema.Max),
		}
	}

	return nil
}

// validateBool validates a boolean parameter
func validateBool(paramName string, value string, schema *ParameterSchema) error {
	lowerValue := strings.ToLower(value)
	validBoolValues := []string{"true", "false", "yes", "no", "1", "0", "on", "off"}

	for _, valid := range validBoolValues {
		if lowerValue == valid {
			return nil
		}
	}

	return &ParameterValidationError{
		ParameterName: paramName,
		ProvidedValue: value,
		ExpectedType:  "bool",
		Constraint:    "value must be a boolean",
		Suggestion:    "use: true, false, yes, no, 1, 0, on, or off",
	}
}

// validateDuration validates a duration parameter (e.g., "2h", "30m", "7d")
func validateDuration(paramName string, value string, schema *ParameterSchema) error {
	// Support common duration formats
	// Go's time.ParseDuration supports: h, m, s, ms, us, ns
	// We extend with: d (days), w (weeks)

	// Convert days and weeks to hours for validation
	normalizedValue := value
	if strings.HasSuffix(value, "d") {
		days := strings.TrimSuffix(value, "d")
		daysInt, err := strconv.Atoi(days)
		if err != nil {
			return &ParameterValidationError{
				ParameterName: paramName,
				ProvidedValue: value,
				ExpectedType:  "duration",
				Constraint:    "invalid duration format",
				Suggestion:    "use format like: 2h, 30m, 7d (h=hours, m=minutes, d=days)",
			}
		}
		normalizedValue = fmt.Sprintf("%dh", daysInt*24)
	} else if strings.HasSuffix(value, "w") {
		weeks := strings.TrimSuffix(value, "w")
		weeksInt, err := strconv.Atoi(weeks)
		if err != nil {
			return &ParameterValidationError{
				ParameterName: paramName,
				ProvidedValue: value,
				ExpectedType:  "duration",
				Constraint:    "invalid duration format",
				Suggestion:    "use format like: 2h, 30m, 1w (h=hours, m=minutes, w=weeks)",
			}
		}
		normalizedValue = fmt.Sprintf("%dh", weeksInt*24*7)
	}

	// Validate using Go's time.ParseDuration
	_, err := time.ParseDuration(normalizedValue)
	if err != nil {
		return &ParameterValidationError{
			ParameterName: paramName,
			ProvidedValue: value,
			ExpectedType:  "duration",
			Constraint:    "invalid duration format",
			Suggestion:    "use format like: 2h, 30m, 90s, 7d",
		}
	}

	// Check pattern if specified
	if schema.Pattern != "" {
		matched, err := regexp.MatchString(schema.Pattern, value)
		if err == nil && !matched {
			return &ParameterValidationError{
				ParameterName: paramName,
				ProvidedValue: value,
				ExpectedType:  "duration",
				Constraint:    fmt.Sprintf("value must match pattern '%s'", schema.Pattern),
				Suggestion:    schema.Description,
			}
		}
	}

	return nil
}

// validateEnum validates an enum parameter
func validateEnum(paramName string, value string, schema *ParameterSchema) error {
	if len(schema.AllowedValues) == 0 {
		return &ParameterValidationError{
			ParameterName: paramName,
			ProvidedValue: value,
			ExpectedType:  "enum",
			Constraint:    "no allowed values defined for enum parameter",
			Suggestion:    "contact administrator to fix parameter schema",
		}
	}

	for _, allowed := range schema.AllowedValues {
		if value == allowed {
			return nil
		}
	}

	return &ParameterValidationError{
		ParameterName: paramName,
		ProvidedValue: value,
		ExpectedType:  "enum",
		Constraint:    fmt.Sprintf("value must be one of: %s", strings.Join(schema.AllowedValues, ", ")),
	}
}

// NormalizeBoolValue converts various boolean representations to "true" or "false"
func NormalizeBoolValue(value string) string {
	lowerValue := strings.ToLower(value)
	switch lowerValue {
	case "true", "yes", "1", "on":
		return "true"
	case "false", "no", "0", "off":
		return "false"
	default:
		return value
	}
}
