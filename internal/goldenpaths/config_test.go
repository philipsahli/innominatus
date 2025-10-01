package goldenpaths

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bxcodec/faker/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper functions for test setup

// createTempGoldenPathsFile creates a temporary goldenpaths.yaml file with given content
func createTempGoldenPathsFile(t *testing.T, content string) string {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "goldenpaths.yaml")
	err := os.WriteFile(configFile, []byte(content), 0644)
	require.NoError(t, err, "Failed to create temp goldenpaths.yaml")
	return configFile
}

// createTempWorkflowFile creates a temporary workflow file
func createTempWorkflowFile(t *testing.T, dir, name string) string {
	workflowFile := filepath.Join(dir, name)
	err := os.MkdirAll(filepath.Dir(workflowFile), 0755)
	require.NoError(t, err)
	err = os.WriteFile(workflowFile, []byte("apiVersion: workflow.dev/v1\nkind: Workflow"), 0644)
	require.NoError(t, err)
	return workflowFile
}

// changeToTempDir changes working directory to temp dir and restores it after test
func changeToTempDir(t *testing.T) string {
	originalDir, err := os.Getwd()
	require.NoError(t, err)

	tmpDir := t.TempDir()
	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	t.Cleanup(func() {
		os.Chdir(originalDir)
	})

	return tmpDir
}

func TestLoadGoldenPaths(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func(t *testing.T, tmpDir string)
		expectError bool
		errorMsg    string
		validate    func(t *testing.T, config *GoldenPathsConfig)
	}{
		{
			name: "valid goldenpaths with full metadata",
			setupFunc: func(t *testing.T, tmpDir string) {
				content := `goldenpaths:
  deploy-app:
    workflow: ./workflows/deploy-app.yaml
    description: Deploy application with GitOps
    category: deployment
    tags: [deployment, gitops]
    estimated_duration: 5-10 minutes
    required_params: [app_name]
    optional_params:
      sync_policy: auto
      replicas: "3"
`
				os.WriteFile(filepath.Join(tmpDir, "goldenpaths.yaml"), []byte(content), 0644)
				os.MkdirAll(filepath.Join(tmpDir, "workflows"), 0755)
				os.WriteFile(filepath.Join(tmpDir, "workflows/deploy-app.yaml"), []byte("workflow"), 0644)
			},
			expectError: false,
			validate: func(t *testing.T, config *GoldenPathsConfig) {
				assert.Len(t, config.GoldenPaths, 1)
				assert.Contains(t, config.GoldenPaths, "deploy-app")

				metadata, err := config.GetMetadata("deploy-app")
				require.NoError(t, err)
				assert.Equal(t, "./workflows/deploy-app.yaml", metadata.WorkflowFile)
				assert.Equal(t, "Deploy application with GitOps", metadata.Description)
				assert.Equal(t, "deployment", metadata.Category)
				assert.Equal(t, []string{"deployment", "gitops"}, metadata.Tags)
				assert.Equal(t, "5-10 minutes", metadata.EstimatedDuration)
				assert.Equal(t, []string{"app_name"}, metadata.RequiredParams)
				assert.Equal(t, "auto", metadata.OptionalParams["sync_policy"])
			},
		},
		{
			name: "valid goldenpaths with simple string format",
			setupFunc: func(t *testing.T, tmpDir string) {
				content := `goldenpaths:
  simple-path: ./workflows/simple.yaml
  another-path: ./workflows/another.yaml
`
				os.WriteFile(filepath.Join(tmpDir, "goldenpaths.yaml"), []byte(content), 0644)
			},
			expectError: false,
			validate: func(t *testing.T, config *GoldenPathsConfig) {
				assert.Len(t, config.GoldenPaths, 2)

				workflow, err := config.GetWorkflowFile("simple-path")
				require.NoError(t, err)
				assert.Equal(t, "./workflows/simple.yaml", workflow)

				metadata, err := config.GetMetadata("another-path")
				require.NoError(t, err)
				assert.Equal(t, "./workflows/another.yaml", metadata.WorkflowFile)
				assert.Empty(t, metadata.Description)
				assert.Empty(t, metadata.Tags)
			},
		},
		{
			name: "valid goldenpaths with mixed formats",
			setupFunc: func(t *testing.T, tmpDir string) {
				content := `goldenpaths:
  full-metadata:
    workflow: ./workflows/full.yaml
    description: Full metadata example
    category: deployment
    tags: [test]
  simple-path: ./workflows/simple.yaml
`
				os.WriteFile(filepath.Join(tmpDir, "goldenpaths.yaml"), []byte(content), 0644)
			},
			expectError: false,
			validate: func(t *testing.T, config *GoldenPathsConfig) {
				assert.Len(t, config.GoldenPaths, 2)

				fullMeta, err := config.GetMetadata("full-metadata")
				require.NoError(t, err)
				assert.Equal(t, "Full metadata example", fullMeta.Description)

				simpleMeta, err := config.GetMetadata("simple-path")
				require.NoError(t, err)
				assert.Empty(t, simpleMeta.Description)
			},
		},
		{
			name: "file not found error",
			setupFunc: func(t *testing.T, tmpDir string) {
				// Don't create goldenpaths.yaml
			},
			expectError: true,
			errorMsg:    "failed to read goldenpaths.yaml",
		},
		{
			name: "invalid YAML syntax error",
			setupFunc: func(t *testing.T, tmpDir string) {
				content := `goldenpaths:
  invalid syntax: [unclosed bracket
    description: missing closing
`
				os.WriteFile(filepath.Join(tmpDir, "goldenpaths.yaml"), []byte(content), 0644)
			},
			expectError: true,
			errorMsg:    "failed to parse goldenpaths.yaml",
		},
		{
			name: "empty goldenpaths map",
			setupFunc: func(t *testing.T, tmpDir string) {
				content := `goldenpaths: {}
`
				os.WriteFile(filepath.Join(tmpDir, "goldenpaths.yaml"), []byte(content), 0644)
			},
			expectError: false,
			validate: func(t *testing.T, config *GoldenPathsConfig) {
				assert.Len(t, config.GoldenPaths, 0)
				assert.Empty(t, config.ListPaths())
			},
		},
		{
			name: "invalid metadata format - missing workflow",
			setupFunc: func(t *testing.T, tmpDir string) {
				content := `goldenpaths:
  invalid-path:
    description: Missing workflow field
    category: deployment
`
				os.WriteFile(filepath.Join(tmpDir, "goldenpaths.yaml"), []byte(content), 0644)
			},
			expectError: true,
			errorMsg:    "workflow file is required",
		},
		{
			name: "golden paths with faker generated data",
			setupFunc: func(t *testing.T, tmpDir string) {
				description := faker.Sentence()
				category := faker.Word()
				content := `goldenpaths:
  faker-path:
    workflow: ./workflows/faker.yaml
    description: ` + description + `
    category: ` + category + `
    tags: [auto-generated, test]
`
				os.WriteFile(filepath.Join(tmpDir, "goldenpaths.yaml"), []byte(content), 0644)
			},
			expectError: false,
			validate: func(t *testing.T, config *GoldenPathsConfig) {
				assert.Len(t, config.GoldenPaths, 1)
				metadata, err := config.GetMetadata("faker-path")
				require.NoError(t, err)
				assert.NotEmpty(t, metadata.Description)
				assert.NotEmpty(t, metadata.Category)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := changeToTempDir(t)
			tt.setupFunc(t, tmpDir)

			config, err := LoadGoldenPaths()

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				require.NoError(t, err)
				assert.NotNil(t, config)
				if tt.validate != nil {
					tt.validate(t, config)
				}
			}
		})
	}
}

func TestGoldenPathsConfig_GetWorkflowFile(t *testing.T) {
	tests := []struct {
		name         string
		config       *GoldenPathsConfig
		pathName     string
		expectError  bool
		expectedFile string
	}{
		{
			name: "get workflow file for existing path - simple format",
			config: &GoldenPathsConfig{
				GoldenPaths: map[string]interface{}{
					"test-path": "./workflows/test.yaml",
				},
				paths: map[string]*GoldenPathMetadata{
					"test-path": {
						WorkflowFile: "./workflows/test.yaml",
					},
				},
			},
			pathName:     "test-path",
			expectError:  false,
			expectedFile: "./workflows/test.yaml",
		},
		{
			name: "get workflow file for existing path - metadata format",
			config: &GoldenPathsConfig{
				GoldenPaths: map[string]interface{}{
					"deploy-app": map[string]interface{}{
						"workflow":    "./workflows/deploy.yaml",
						"description": "Deploy application",
					},
				},
				paths: map[string]*GoldenPathMetadata{
					"deploy-app": {
						WorkflowFile: "./workflows/deploy.yaml",
						Description:  "Deploy application",
					},
				},
			},
			pathName:     "deploy-app",
			expectError:  false,
			expectedFile: "./workflows/deploy.yaml",
		},
		{
			name: "path does not exist",
			config: &GoldenPathsConfig{
				GoldenPaths: map[string]interface{}{},
				paths:       map[string]*GoldenPathMetadata{},
			},
			pathName:    "non-existent",
			expectError: true,
		},
		{
			name: "multiple paths - correct file returned",
			config: &GoldenPathsConfig{
				GoldenPaths: map[string]interface{}{
					"path-a": "./workflows/a.yaml",
					"path-b": "./workflows/b.yaml",
					"path-c": "./workflows/c.yaml",
				},
				paths: map[string]*GoldenPathMetadata{
					"path-a": {WorkflowFile: "./workflows/a.yaml"},
					"path-b": {WorkflowFile: "./workflows/b.yaml"},
					"path-c": {WorkflowFile: "./workflows/c.yaml"},
				},
			},
			pathName:     "path-b",
			expectError:  false,
			expectedFile: "./workflows/b.yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, err := tt.config.GetWorkflowFile(tt.pathName)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "not found")
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedFile, file)
			}
		})
	}
}

func TestGoldenPathsConfig_GetMetadata(t *testing.T) {
	tests := []struct {
		name        string
		config      *GoldenPathsConfig
		pathName    string
		expectError bool
		validate    func(t *testing.T, metadata *GoldenPathMetadata)
	}{
		{
			name: "get metadata for existing path with full metadata",
			config: &GoldenPathsConfig{
				paths: map[string]*GoldenPathMetadata{
					"full-path": {
						WorkflowFile:      "./workflows/full.yaml",
						Description:       "Full metadata example",
						Category:          "deployment",
						Tags:              []string{"tag1", "tag2"},
						EstimatedDuration: "5-10 minutes",
						RequiredParams:    []string{"param1", "param2"},
						OptionalParams:    map[string]string{"opt1": "default1"},
					},
				},
			},
			pathName:    "full-path",
			expectError: false,
			validate: func(t *testing.T, metadata *GoldenPathMetadata) {
				assert.Equal(t, "./workflows/full.yaml", metadata.WorkflowFile)
				assert.Equal(t, "Full metadata example", metadata.Description)
				assert.Equal(t, "deployment", metadata.Category)
				assert.Equal(t, []string{"tag1", "tag2"}, metadata.Tags)
				assert.Equal(t, "5-10 minutes", metadata.EstimatedDuration)
				assert.Equal(t, []string{"param1", "param2"}, metadata.RequiredParams)
				assert.Equal(t, "default1", metadata.OptionalParams["opt1"])
			},
		},
		{
			name: "get metadata for path with minimal metadata",
			config: &GoldenPathsConfig{
				paths: map[string]*GoldenPathMetadata{
					"minimal-path": {
						WorkflowFile: "./workflows/minimal.yaml",
					},
				},
			},
			pathName:    "minimal-path",
			expectError: false,
			validate: func(t *testing.T, metadata *GoldenPathMetadata) {
				assert.Equal(t, "./workflows/minimal.yaml", metadata.WorkflowFile)
				assert.Empty(t, metadata.Description)
				assert.Empty(t, metadata.Category)
				assert.Empty(t, metadata.Tags)
				assert.Empty(t, metadata.RequiredParams)
				assert.Nil(t, metadata.OptionalParams)
			},
		},
		{
			name: "path does not exist",
			config: &GoldenPathsConfig{
				paths: map[string]*GoldenPathMetadata{},
			},
			pathName:    "non-existent",
			expectError: true,
		},
		{
			name: "verify all metadata fields populated correctly",
			config: &GoldenPathsConfig{
				paths: map[string]*GoldenPathMetadata{
					"test-path": {
						WorkflowFile:      faker.URL(),
						Description:       faker.Sentence(),
						Category:          faker.Word(),
						Tags:              []string{faker.Word(), faker.Word()},
						EstimatedDuration: "10-15 minutes",
						RequiredParams:    []string{faker.Word()},
						OptionalParams:    map[string]string{faker.Word(): faker.Word()},
					},
				},
			},
			pathName:    "test-path",
			expectError: false,
			validate: func(t *testing.T, metadata *GoldenPathMetadata) {
				assert.NotEmpty(t, metadata.WorkflowFile)
				assert.NotEmpty(t, metadata.Description)
				assert.NotEmpty(t, metadata.Category)
				assert.NotEmpty(t, metadata.Tags)
				assert.NotEmpty(t, metadata.EstimatedDuration)
				assert.NotEmpty(t, metadata.RequiredParams)
				assert.NotEmpty(t, metadata.OptionalParams)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metadata, err := tt.config.GetMetadata(tt.pathName)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "not found")
			} else {
				require.NoError(t, err)
				assert.NotNil(t, metadata)
				if tt.validate != nil {
					tt.validate(t, metadata)
				}
			}
		})
	}
}

func TestGoldenPathsConfig_ListPaths(t *testing.T) {
	tests := []struct {
		name     string
		config   *GoldenPathsConfig
		expected []string
	}{
		{
			name: "list paths returns all paths",
			config: &GoldenPathsConfig{
				GoldenPaths: map[string]interface{}{
					"deploy-app":   "./workflows/deploy.yaml",
					"undeploy-app": "./workflows/undeploy.yaml",
					"backup-db":    "./workflows/backup.yaml",
				},
			},
			expected: []string{"backup-db", "deploy-app", "undeploy-app"},
		},
		{
			name: "paths are sorted alphabetically",
			config: &GoldenPathsConfig{
				GoldenPaths: map[string]interface{}{
					"zebra":  "./workflows/z.yaml",
					"alpha":  "./workflows/a.yaml",
					"middle": "./workflows/m.yaml",
				},
			},
			expected: []string{"alpha", "middle", "zebra"},
		},
		{
			name: "empty config returns empty list",
			config: &GoldenPathsConfig{
				GoldenPaths: map[string]interface{}{},
			},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			paths := tt.config.ListPaths()
			assert.Equal(t, tt.expected, paths)
		})
	}
}

func TestGoldenPathsConfig_ValidatePaths(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func(t *testing.T, tmpDir string) *GoldenPathsConfig
		expectError bool
		errorMsg    string
	}{
		{
			name: "all workflow files exist",
			setupFunc: func(t *testing.T, tmpDir string) *GoldenPathsConfig {
				os.MkdirAll(filepath.Join(tmpDir, "workflows"), 0755)
				os.WriteFile(filepath.Join(tmpDir, "workflows/deploy.yaml"), []byte("workflow"), 0644)
				os.WriteFile(filepath.Join(tmpDir, "workflows/undeploy.yaml"), []byte("workflow"), 0644)

				return &GoldenPathsConfig{
					paths: map[string]*GoldenPathMetadata{
						"deploy":   {WorkflowFile: filepath.Join(tmpDir, "workflows/deploy.yaml")},
						"undeploy": {WorkflowFile: filepath.Join(tmpDir, "workflows/undeploy.yaml")},
					},
				}
			},
			expectError: false,
		},
		{
			name: "one workflow file missing",
			setupFunc: func(t *testing.T, tmpDir string) *GoldenPathsConfig {
				os.MkdirAll(filepath.Join(tmpDir, "workflows"), 0755)
				os.WriteFile(filepath.Join(tmpDir, "workflows/exists.yaml"), []byte("workflow"), 0644)

				return &GoldenPathsConfig{
					paths: map[string]*GoldenPathMetadata{
						"exists":  {WorkflowFile: filepath.Join(tmpDir, "workflows/exists.yaml")},
						"missing": {WorkflowFile: filepath.Join(tmpDir, "workflows/missing.yaml")},
					},
				}
			},
			expectError: true,
			errorMsg:    "not found",
		},
		{
			name: "multiple workflow files missing",
			setupFunc: func(t *testing.T, tmpDir string) *GoldenPathsConfig {
				return &GoldenPathsConfig{
					paths: map[string]*GoldenPathMetadata{
						"missing1": {WorkflowFile: filepath.Join(tmpDir, "workflows/missing1.yaml")},
						"missing2": {WorkflowFile: filepath.Join(tmpDir, "workflows/missing2.yaml")},
					},
				}
			},
			expectError: true,
			errorMsg:    "not found",
		},
		{
			name: "empty config validation passes",
			setupFunc: func(t *testing.T, tmpDir string) *GoldenPathsConfig {
				return &GoldenPathsConfig{
					paths: map[string]*GoldenPathMetadata{},
				}
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			config := tt.setupFunc(t, tmpDir)

			err := config.ValidatePaths()

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

func TestGoldenPathsConfig_ValidateParameters(t *testing.T) {
	tests := []struct {
		name        string
		config      *GoldenPathsConfig
		pathName    string
		params      map[string]string
		expectError bool
		errorMsg    string
	}{
		{
			name: "all required params provided",
			config: &GoldenPathsConfig{
				paths: map[string]*GoldenPathMetadata{
					"test-path": {
						RequiredParams: []string{"app_name", "environment"},
					},
				},
			},
			pathName: "test-path",
			params: map[string]string{
				"app_name":    "myapp",
				"environment": "production",
			},
			expectError: false,
		},
		{
			name: "one required param missing",
			config: &GoldenPathsConfig{
				paths: map[string]*GoldenPathMetadata{
					"test-path": {
						RequiredParams: []string{"app_name", "environment"},
					},
				},
			},
			pathName: "test-path",
			params: map[string]string{
				"app_name": "myapp",
			},
			expectError: true,
			errorMsg:    "required parameter 'environment' is missing",
		},
		{
			name: "multiple required params missing",
			config: &GoldenPathsConfig{
				paths: map[string]*GoldenPathMetadata{
					"test-path": {
						RequiredParams: []string{"app_name", "environment", "region"},
					},
				},
			},
			pathName:    "test-path",
			params:      map[string]string{},
			expectError: true,
			errorMsg:    "required parameter",
		},
		{
			name: "no required params - valid",
			config: &GoldenPathsConfig{
				paths: map[string]*GoldenPathMetadata{
					"test-path": {
						RequiredParams: []string{},
					},
				},
			},
			pathName:    "test-path",
			params:      map[string]string{},
			expectError: false,
		},
		{
			name: "extra params provided - valid",
			config: &GoldenPathsConfig{
				paths: map[string]*GoldenPathMetadata{
					"test-path": {
						RequiredParams: []string{"app_name"},
					},
				},
			},
			pathName: "test-path",
			params: map[string]string{
				"app_name":    "myapp",
				"extra_param": "extra_value",
			},
			expectError: false,
		},
		{
			name: "path does not exist",
			config: &GoldenPathsConfig{
				paths: map[string]*GoldenPathMetadata{},
			},
			pathName:    "non-existent",
			params:      map[string]string{},
			expectError: true,
			errorMsg:    "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.ValidateParameters(tt.pathName, tt.params)

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

func TestGoldenPathsConfig_GetParametersWithDefaults(t *testing.T) {
	tests := []struct {
		name        string
		config      *GoldenPathsConfig
		pathName    string
		params      map[string]string
		expectError bool
		expected    map[string]string
	}{
		{
			name: "user params override defaults",
			config: &GoldenPathsConfig{
				paths: map[string]*GoldenPathMetadata{
					"test-path": {
						OptionalParams: map[string]string{
							"sync_policy": "auto",
							"replicas":    "3",
						},
					},
				},
			},
			pathName: "test-path",
			params: map[string]string{
				"sync_policy": "manual",
			},
			expectError: false,
			expected: map[string]string{
				"sync_policy": "manual",
				"replicas":    "3",
			},
		},
		{
			name: "defaults used for missing params",
			config: &GoldenPathsConfig{
				paths: map[string]*GoldenPathMetadata{
					"test-path": {
						OptionalParams: map[string]string{
							"ttl":     "2h",
							"env":     "dev",
							"verbose": "false",
						},
					},
				},
			},
			pathName:    "test-path",
			params:      map[string]string{},
			expectError: false,
			expected: map[string]string{
				"ttl":     "2h",
				"env":     "dev",
				"verbose": "false",
			},
		},
		{
			name: "no optional params defined",
			config: &GoldenPathsConfig{
				paths: map[string]*GoldenPathMetadata{
					"test-path": {
						OptionalParams: map[string]string{},
					},
				},
			},
			pathName: "test-path",
			params: map[string]string{
				"custom": "value",
			},
			expectError: false,
			expected: map[string]string{
				"custom": "value",
			},
		},
		{
			name: "mix of user params and defaults",
			config: &GoldenPathsConfig{
				paths: map[string]*GoldenPathMetadata{
					"test-path": {
						OptionalParams: map[string]string{
							"param1": "default1",
							"param2": "default2",
							"param3": "default3",
						},
					},
				},
			},
			pathName: "test-path",
			params: map[string]string{
				"param1":       "user1",
				"param3":       "user3",
				"extra_param": "extra",
			},
			expectError: false,
			expected: map[string]string{
				"param1":      "user1",
				"param2":      "default2",
				"param3":      "user3",
				"extra_param": "extra",
			},
		},
		{
			name: "path does not exist",
			config: &GoldenPathsConfig{
				paths: map[string]*GoldenPathMetadata{},
			},
			pathName:    "non-existent",
			params:      map[string]string{},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.config.GetParametersWithDefaults(tt.pathName, tt.params)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestGoldenPathsConfig_parsePathMetadata(t *testing.T) {
	tests := []struct {
		name        string
		pathName    string
		value       interface{}
		expectError bool
		errorMsg    string
		validate    func(t *testing.T, metadata *GoldenPathMetadata)
	}{
		{
			name:        "parse simple string format",
			pathName:    "simple-path",
			value:       "./workflows/simple.yaml",
			expectError: false,
			validate: func(t *testing.T, metadata *GoldenPathMetadata) {
				assert.Equal(t, "./workflows/simple.yaml", metadata.WorkflowFile)
				assert.Empty(t, metadata.Description)
				assert.Empty(t, metadata.Tags)
			},
		},
		{
			name:     "parse full metadata format",
			pathName: "full-path",
			value: map[string]interface{}{
				"workflow":            "./workflows/full.yaml",
				"description":         "Full metadata",
				"category":            "deployment",
				"tags":                []interface{}{"tag1", "tag2"},
				"estimated_duration":  "5 minutes",
				"required_params":     []interface{}{"param1"},
				"optional_params":     map[string]interface{}{"opt1": "default1"},
			},
			expectError: false,
			validate: func(t *testing.T, metadata *GoldenPathMetadata) {
				assert.Equal(t, "./workflows/full.yaml", metadata.WorkflowFile)
				assert.Equal(t, "Full metadata", metadata.Description)
				assert.Equal(t, "deployment", metadata.Category)
				assert.Contains(t, metadata.Tags, "tag1")
				assert.Equal(t, "5 minutes", metadata.EstimatedDuration)
				assert.Contains(t, metadata.RequiredParams, "param1")
				assert.Equal(t, "default1", metadata.OptionalParams["opt1"])
			},
		},
		{
			name:     "parse metadata with all optional fields",
			pathName: "all-fields",
			value: map[string]interface{}{
				"workflow":            faker.URL(),
				"description":         faker.Sentence(),
				"category":            faker.Word(),
				"tags":                []interface{}{faker.Word(), faker.Word()},
				"estimated_duration":  "10-15 minutes",
				"required_params":     []interface{}{faker.Word()},
				"optional_params":     map[string]interface{}{faker.Word(): faker.Word()},
			},
			expectError: false,
			validate: func(t *testing.T, metadata *GoldenPathMetadata) {
				assert.NotEmpty(t, metadata.WorkflowFile)
				assert.NotEmpty(t, metadata.Description)
				assert.NotEmpty(t, metadata.Category)
				assert.NotEmpty(t, metadata.Tags)
				assert.NotEmpty(t, metadata.EstimatedDuration)
				assert.NotEmpty(t, metadata.RequiredParams)
				assert.NotEmpty(t, metadata.OptionalParams)
			},
		},
		{
			name:     "parse metadata with minimal fields - only workflow",
			pathName: "minimal",
			value: map[string]interface{}{
				"workflow": "./workflows/minimal.yaml",
			},
			expectError: false,
			validate: func(t *testing.T, metadata *GoldenPathMetadata) {
				assert.Equal(t, "./workflows/minimal.yaml", metadata.WorkflowFile)
			},
		},
		{
			name:        "invalid type - not string or map",
			pathName:    "invalid",
			value:       12345,
			expectError: true,
			errorMsg:    "invalid golden path value type",
		},
		{
			name:     "missing workflow field in metadata",
			pathName: "no-workflow",
			value: map[string]interface{}{
				"description": "Missing workflow field",
				"category":    "deployment",
			},
			expectError: true,
			errorMsg:    "workflow file is required",
		},
		{
			name:     "parse metadata with empty workflow",
			pathName: "empty-workflow",
			value: map[string]interface{}{
				"workflow":    "",
				"description": "Empty workflow path",
			},
			expectError: true,
			errorMsg:    "workflow file is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &GoldenPathsConfig{}
			metadata, err := config.parsePathMetadata(tt.pathName, tt.value)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				require.NoError(t, err)
				assert.NotNil(t, metadata)
				if tt.validate != nil {
					tt.validate(t, metadata)
				}
			}
		})
	}
}
