package workflow

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOutputParser_ParseJSON(t *testing.T) {
	parser := NewOutputParser()

	tests := []struct {
		name     string
		content  string
		expected map[string]string
		wantErr  bool
	}{
		{
			name:    "simple JSON",
			content: `{"version": "1.0.0", "build_id": "12345"}`,
			expected: map[string]string{
				"version":  "1.0.0",
				"build_id": "12345",
			},
			wantErr: false,
		},
		{
			name:    "JSON with numbers",
			content: `{"count": 42, "price": 99.99}`,
			expected: map[string]string{
				"count": "42",
				"price": "99.99",
			},
			wantErr: false,
		},
		{
			name:    "JSON with boolean",
			content: `{"enabled": true, "debug": false}`,
			expected: map[string]string{
				"enabled": "true",
				"debug":   "false",
			},
			wantErr: false,
		},
		{
			name:     "empty JSON",
			content:  `{}`,
			expected: map[string]string{},
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.parseJSON([]byte(tt.content))

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestOutputParser_ParseKeyValue(t *testing.T) {
	parser := NewOutputParser()

	tests := []struct {
		name     string
		content  string
		expected map[string]string
		wantErr  bool
	}{
		{
			name: "simple key=value",
			content: `VERSION=1.0.0
BUILD_ID=12345`,
			expected: map[string]string{
				"VERSION":  "1.0.0",
				"BUILD_ID": "12345",
			},
			wantErr: false,
		},
		{
			name: "with quotes",
			content: `MESSAGE="Hello World"
PATH='/usr/local/bin'`,
			expected: map[string]string{
				"MESSAGE": "Hello World",
				"PATH":    "/usr/local/bin",
			},
			wantErr: false,
		},
		{
			name: "with comments and empty lines",
			content: `# This is a comment
VERSION=1.0.0

# Another comment
BUILD_ID=12345`,
			expected: map[string]string{
				"VERSION":  "1.0.0",
				"BUILD_ID": "12345",
			},
			wantErr: false,
		},
		{
			name: "with spaces",
			content: `VERSION = 1.0.0
BUILD_ID  =  12345 `,
			expected: map[string]string{
				"VERSION":  "1.0.0",
				"BUILD_ID": "12345",
			},
			wantErr: false,
		},
		{
			name:     "invalid format",
			content:  `INVALID LINE WITHOUT EQUALS`,
			expected: nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.parseKeyValue([]byte(tt.content))

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestOutputParser_ParseOutputFile(t *testing.T) {
	parser := NewOutputParser()

	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "output-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	t.Run("JSON file", func(t *testing.T) {
		jsonFile := filepath.Join(tmpDir, "output.json")
		content := `{"version": "1.0.0", "build_id": "12345"}`
		err := os.WriteFile(jsonFile, []byte(content), 0644)
		require.NoError(t, err)

		result, err := parser.ParseOutputFile(jsonFile)
		require.NoError(t, err)

		expected := map[string]string{
			"version":  "1.0.0",
			"build_id": "12345",
		}
		assert.Equal(t, expected, result)
	})

	t.Run("key=value file", func(t *testing.T) {
		kvFile := filepath.Join(tmpDir, "output.env")
		content := `VERSION=1.0.0
BUILD_ID=12345`
		err := os.WriteFile(kvFile, []byte(content), 0644)
		require.NoError(t, err)

		result, err := parser.ParseOutputFile(kvFile)
		require.NoError(t, err)

		expected := map[string]string{
			"VERSION":  "1.0.0",
			"BUILD_ID": "12345",
		}
		assert.Equal(t, expected, result)
	})

	t.Run("file not found", func(t *testing.T) {
		result, err := parser.ParseOutputFile("/nonexistent/file.json")
		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestOutputParser_ParseStdout(t *testing.T) {
	parser := NewOutputParser()

	t.Run("GitHub Actions style", func(t *testing.T) {
		stdout := `Building application...
::set-output name=version::1.0.0
::set-output name=build_id::12345
Build complete`

		result := parser.ParseStdout(stdout, nil)

		expected := map[string]string{
			"version":  "1.0.0",
			"build_id": "12345",
		}
		assert.Equal(t, expected, result)
	})

	t.Run("environment style", func(t *testing.T) {
		stdout := `Building application...
OUTPUT_VERSION=1.0.0
OUTPUT_BUILD_ID=12345
Build complete`

		result := parser.ParseStdout(stdout, nil)

		expected := map[string]string{
			"VERSION":  "1.0.0",
			"BUILD_ID": "12345",
		}
		assert.Equal(t, expected, result)
	})

	t.Run("last line for single output", func(t *testing.T) {
		stdout := `Building application...
Compiling files...
1.0.0-rc1`

		result := parser.ParseStdout(stdout, []string{"version"})

		expected := map[string]string{
			"version": "1.0.0-rc1",
		}
		assert.Equal(t, expected, result)
	})

	t.Run("named outputs", func(t *testing.T) {
		stdout := `version=1.0.0
build_id: 12345
status=success`

		result := parser.ParseStdout(stdout, []string{"version", "build_id", "status"})

		expected := map[string]string{
			"version":  "1.0.0",
			"build_id": "12345",
			"status":   "success",
		}
		assert.Equal(t, expected, result)
	})

	t.Run("empty stdout", func(t *testing.T) {
		result := parser.ParseStdout("", nil)
		assert.Empty(t, result)
	})
}

func TestOutputParser_GitHubActionsOutput(t *testing.T) {
	parser := NewOutputParser()

	tests := []struct {
		name     string
		line     string
		expected map[string]string
	}{
		{
			name:     "valid output",
			line:     "::set-output name=version::1.0.0",
			expected: map[string]string{"version": "1.0.0"},
		},
		{
			name:     "with spaces",
			line:     "::set-output name=message::Hello World",
			expected: map[string]string{"message": "Hello World"},
		},
		{
			name:     "invalid format",
			line:     "::set-output invalid",
			expected: nil,
		},
		{
			name:     "missing name",
			line:     "::set-output ::value",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.parseGitHubActionsOutput(tt.line)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestOutputParser_EnvOutput(t *testing.T) {
	parser := NewOutputParser()

	tests := []struct {
		name     string
		line     string
		expected map[string]string
	}{
		{
			name:     "valid output",
			line:     "OUTPUT_VERSION=1.0.0",
			expected: map[string]string{"VERSION": "1.0.0"},
		},
		{
			name:     "with quotes",
			line:     `OUTPUT_MESSAGE="Hello World"`,
			expected: map[string]string{"MESSAGE": "Hello World"},
		},
		{
			name:     "invalid format",
			line:     "OUTPUT_INVALID",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.parseEnvOutput(tt.line)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExecutionContext_WorkflowVariables(t *testing.T) {
	ctx := NewExecutionContext()

	t.Run("set and get workflow variables", func(t *testing.T) {
		ctx.SetVariable("ENVIRONMENT", "production")
		ctx.SetVariable("VERSION", "1.0.0")

		env, exists := ctx.GetVariable("ENVIRONMENT")
		assert.True(t, exists)
		assert.Equal(t, "production", env)

		version, exists := ctx.GetVariable("VERSION")
		assert.True(t, exists)
		assert.Equal(t, "1.0.0", version)
	})

	t.Run("get non-existent variable", func(t *testing.T) {
		_, exists := ctx.GetVariable("NONEXISTENT")
		assert.False(t, exists)
	})

	t.Run("set workflow variables bulk", func(t *testing.T) {
		vars := map[string]string{
			"APP_NAME": "myapp",
			"REGION":   "us-east-1",
		}

		ctx.SetWorkflowVariables(vars)

		appName, exists := ctx.GetVariable("APP_NAME")
		assert.True(t, exists)
		assert.Equal(t, "myapp", appName)

		region, exists := ctx.GetVariable("REGION")
		assert.True(t, exists)
		assert.Equal(t, "us-east-1", region)
	})
}

func TestExecutionContext_StepOutputs(t *testing.T) {
	ctx := NewExecutionContext()

	t.Run("set and get step outputs", func(t *testing.T) {
		outputs := map[string]string{
			"version":  "1.0.0",
			"build_id": "12345",
		}

		ctx.SetStepOutputs("build", outputs)

		version, found := ctx.GetStepOutput("build", "version")
		assert.True(t, found)
		assert.Equal(t, "1.0.0", version)

		buildID, found := ctx.GetStepOutput("build", "build_id")
		assert.True(t, found)
		assert.Equal(t, "12345", buildID)
	})

	t.Run("get non-existent step output", func(t *testing.T) {
		_, found := ctx.GetStepOutput("nonexistent", "key")
		assert.False(t, found)
	})

	t.Run("get all step outputs", func(t *testing.T) {
		outputs := map[string]string{
			"version":  "1.0.0",
			"build_id": "12345",
		}

		ctx.SetStepOutputs("build", outputs)

		allOutputs, exists := ctx.GetAllStepOutputs("build")
		assert.True(t, exists)
		assert.Equal(t, outputs, allOutputs)
	})

	t.Run("set single output", func(t *testing.T) {
		ctx.SetStepOutput("deploy", "url", "https://example.com")

		url, found := ctx.GetStepOutput("deploy", "url")
		assert.True(t, found)
		assert.Equal(t, "https://example.com", url)
	})
}

func TestExecutionContext_VariableReplacement_WithOutputs(t *testing.T) {
	ctx := NewExecutionContext()

	// Setup context
	ctx.SetVariable("ENVIRONMENT", "production")
	ctx.SetStepOutputs("build", map[string]string{
		"version":  "1.0.0",
		"build_id": "12345",
	})

	tests := []struct {
		name     string
		input    string
		env      map[string]string
		expected string
	}{
		{
			name:     "workflow variable",
			input:    "${workflow.ENVIRONMENT}",
			env:      map[string]string{},
			expected: "production",
		},
		{
			name:     "step output",
			input:    "${build.version}",
			env:      map[string]string{},
			expected: "1.0.0",
		},
		{
			name:     "multiple references",
			input:    "Deploying ${build.version} to ${workflow.ENVIRONMENT}",
			env:      map[string]string{},
			expected: "Deploying 1.0.0 to production",
		},
		{
			name:     "step output with $ syntax",
			input:    "$build.version",
			env:      map[string]string{},
			expected: "1.0.0",
		},
		{
			name:     "step env variable",
			input:    "${APP_NAME}",
			env:      map[string]string{"APP_NAME": "myapp"},
			expected: "myapp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ctx.replaceVariables(tt.input, tt.env)
			assert.Equal(t, tt.expected, result)
		})
	}
}
