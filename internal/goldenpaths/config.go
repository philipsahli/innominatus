package goldenpaths

import (
	"fmt"
	"os"
	"sort"

	"gopkg.in/yaml.v3"
)

// ParameterSchema defines the validation schema for a parameter
type ParameterSchema struct {
	Type          string   `yaml:"type"`           // string, int, bool, duration, enum
	Default       string   `yaml:"default"`        // default value as string
	Description   string   `yaml:"description"`    // parameter description
	Required      bool     `yaml:"required"`       // whether parameter is required
	Pattern       string   `yaml:"pattern"`        // regex pattern for string validation
	AllowedValues []string `yaml:"allowed_values"` // for enum type
	Min           *int     `yaml:"min"`            // min value for int type
	Max           *int     `yaml:"max"`            // max value for int type
}

// GoldenPathMetadata defines metadata for a golden path
type GoldenPathMetadata struct {
	Description       string                      `yaml:"description"`
	Tags              []string                    `yaml:"tags"`
	RequiredParams    []string                    `yaml:"required_params"` // DEPRECATED: use Parameters with Required=true
	OptionalParams    map[string]string           `yaml:"optional_params"` // DEPRECATED: use Parameters with Default
	Parameters        map[string]*ParameterSchema `yaml:"parameters"`      // NEW: parameter schemas with validation
	WorkflowFile      string                      `yaml:"workflow"`
	Category          string                      `yaml:"category"`
	EstimatedDuration string                      `yaml:"estimated_duration"`
}

// GoldenPathsConfig defines the configuration for available golden paths
// Supports both simple string format (backward compatible) and metadata format
type GoldenPathsConfig struct {
	GoldenPaths map[string]interface{}         `yaml:"goldenpaths"`
	paths       map[string]*GoldenPathMetadata // Parsed metadata cache
}

// LoadGoldenPaths loads the golden paths configuration from goldenpaths.yaml
func LoadGoldenPaths() (*GoldenPathsConfig, error) {
	const configFile = "goldenpaths.yaml"

	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", configFile, err)
	}

	var config GoldenPathsConfig
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", configFile, err)
	}

	// Parse metadata for all golden paths
	config.paths = make(map[string]*GoldenPathMetadata)
	for pathName, value := range config.GoldenPaths {
		metadata, err := config.parsePathMetadata(pathName, value)
		if err != nil {
			return nil, fmt.Errorf("failed to parse metadata for golden path '%s': %w", pathName, err)
		}
		config.paths[pathName] = metadata
	}

	return &config, nil
}

// parsePathMetadata converts the YAML value to GoldenPathMetadata
// Supports both simple string format and full metadata object
func (c *GoldenPathsConfig) parsePathMetadata(pathName string, value interface{}) (*GoldenPathMetadata, error) {
	switch v := value.(type) {
	case string:
		// Simple format: just a workflow file path (backward compatible)
		return &GoldenPathMetadata{
			WorkflowFile: v,
			Description:  "",
			Tags:         []string{},
		}, nil
	case map[string]interface{}:
		// Full metadata format
		var metadata GoldenPathMetadata

		// Convert to YAML and unmarshal to struct
		yamlBytes, err := yaml.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal metadata: %w", err)
		}

		err = yaml.Unmarshal(yamlBytes, &metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}

		// Validate that workflow file is specified
		if metadata.WorkflowFile == "" {
			return nil, fmt.Errorf("workflow file is required")
		}

		return &metadata, nil
	default:
		return nil, fmt.Errorf("invalid golden path value type: %T", value)
	}
}

// GetWorkflowFile returns the workflow file path for a given golden path name
func (c *GoldenPathsConfig) GetWorkflowFile(pathName string) (string, error) {
	metadata, exists := c.paths[pathName]
	if !exists {
		return "", fmt.Errorf("golden path '%s' not found", pathName)
	}
	return metadata.WorkflowFile, nil
}

// GetMetadata returns the metadata for a given golden path name
func (c *GoldenPathsConfig) GetMetadata(pathName string) (*GoldenPathMetadata, error) {
	metadata, exists := c.paths[pathName]
	if !exists {
		return nil, fmt.Errorf("golden path '%s' not found", pathName)
	}
	return metadata, nil
}

// ListPaths returns a sorted list of available golden path names
func (c *GoldenPathsConfig) ListPaths() []string {
	paths := make([]string, 0, len(c.GoldenPaths))
	for pathName := range c.GoldenPaths {
		paths = append(paths, pathName)
	}
	sort.Strings(paths)
	return paths
}

// ValidatePaths checks if all workflow files exist
func (c *GoldenPathsConfig) ValidatePaths() error {
	for pathName, metadata := range c.paths {
		if _, err := os.Stat(metadata.WorkflowFile); os.IsNotExist(err) {
			return fmt.Errorf("workflow file for golden path '%s' not found: %s", pathName, metadata.WorkflowFile)
		}
	}
	return nil
}

// ValidateParameters validates that all required parameters are provided and validates parameter values
func (c *GoldenPathsConfig) ValidateParameters(pathName string, params map[string]string) error {
	metadata, err := c.GetMetadata(pathName)
	if err != nil {
		return err
	}

	// Use new parameter schema if available
	if len(metadata.Parameters) > 0 {
		return c.validateParametersWithSchema(metadata, params)
	}

	// Fallback to legacy validation for backward compatibility
	return c.validateParametersLegacy(metadata, params)
}

// validateParametersWithSchema validates parameters using the new parameter schema
func (c *GoldenPathsConfig) validateParametersWithSchema(metadata *GoldenPathMetadata, params map[string]string) error {
	// Check required parameters and validate all provided parameters
	for paramName, schema := range metadata.Parameters {
		value, provided := params[paramName]

		// Check if required parameter is provided
		if schema.Required && !provided {
			return &ParameterValidationError{
				ParameterName: paramName,
				ExpectedType:  schema.Type,
				Constraint:    "parameter is required",
				Suggestion:    schema.Description,
			}
		}

		// If parameter was provided (or has default), validate it
		if provided || value != "" {
			if err := ValidateParameterValue(paramName, value, schema); err != nil {
				return err
			}
		}
	}

	return nil
}

// validateParametersLegacy validates parameters using the legacy RequiredParams format
func (c *GoldenPathsConfig) validateParametersLegacy(metadata *GoldenPathMetadata, params map[string]string) error {
	// Check required parameters (backward compatibility)
	for _, requiredParam := range metadata.RequiredParams {
		if _, exists := params[requiredParam]; !exists {
			return fmt.Errorf("required parameter '%s' is missing", requiredParam)
		}
	}

	return nil
}

// GetParametersWithDefaults returns parameters merged with defaults for optional params
func (c *GoldenPathsConfig) GetParametersWithDefaults(pathName string, params map[string]string) (map[string]string, error) {
	metadata, err := c.GetMetadata(pathName)
	if err != nil {
		return nil, err
	}

	// Start with provided params
	result := make(map[string]string)
	for k, v := range params {
		result[k] = v
	}

	// Use new parameter schema if available
	if len(metadata.Parameters) > 0 {
		// Add defaults from parameter schemas
		for paramName, schema := range metadata.Parameters {
			if _, exists := result[paramName]; !exists && schema.Default != "" {
				result[paramName] = schema.Default
			}
		}
	} else {
		// Fallback to legacy optional params for backward compatibility
		for param, defaultValue := range metadata.OptionalParams {
			if _, exists := result[param]; !exists {
				result[param] = defaultValue
			}
		}
	}

	return result, nil
}
