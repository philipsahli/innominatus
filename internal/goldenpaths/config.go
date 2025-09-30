package goldenpaths

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// GoldenPathsConfig defines the configuration for available golden paths
type GoldenPathsConfig struct {
	GoldenPaths map[string]string `yaml:"goldenpaths"`
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

	return &config, nil
}

// GetWorkflowFile returns the workflow file path for a given golden path name
func (c *GoldenPathsConfig) GetWorkflowFile(pathName string) (string, error) {
	workflowFile, exists := c.GoldenPaths[pathName]
	if !exists {
		return "", fmt.Errorf("golden path '%s' not found", pathName)
	}
	return workflowFile, nil
}

// ListPaths returns a list of available golden path names
func (c *GoldenPathsConfig) ListPaths() []string {
	paths := make([]string, 0, len(c.GoldenPaths))
	for pathName := range c.GoldenPaths {
		paths = append(paths, pathName)
	}
	return paths
}

// ValidatePaths checks if all workflow files exist
func (c *GoldenPathsConfig) ValidatePaths() error {
	for pathName, workflowFile := range c.GoldenPaths {
		if _, err := os.Stat(workflowFile); os.IsNotExist(err) {
			return fmt.Errorf("workflow file for golden path '%s' not found: %s", pathName, workflowFile)
		}
	}
	return nil
}