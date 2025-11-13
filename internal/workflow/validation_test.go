package workflow

import (
	"os"
	"strings"
	"testing"

	"innominatus/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractVariableReferences(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "single braced variable",
			input:    "Hello ${NAME}",
			expected: []string{"${NAME}"},
		},
		{
			name:     "multiple braced variables",
			input:    "postgresql://${DB_HOST}:${DB_PORT}/myapp",
			expected: []string{"${DB_HOST}", "${DB_PORT}"},
		},
		{
			name:     "workflow variable",
			input:    "namespace: ${workflow.ENVIRONMENT}",
			expected: []string{"${workflow.ENVIRONMENT}"},
		},
		{
			name:     "step output reference",
			input:    "version: ${build.version}",
			expected: []string{"${build.version}"},
		},
		{
			name:     "resource output reference",
			input:    "endpoint: ${resources.db.host}",
			expected: []string{"${resources.db.host}"},
		},
		{
			name:     "unbraced variable",
			input:    "Hello $NAME",
			expected: []string{"$NAME"},
		},
		{
			name:     "mixed braced and unbraced",
			input:    "Image: $REGISTRY/${IMAGE_NAME}:${VERSION}",
			expected: []string{"$REGISTRY", "${IMAGE_NAME}", "${VERSION}"},
		},
		{
			name:     "no variables",
			input:    "static text with no variables",
			expected: nil, // No matches returns nil, not empty slice
		},
		{
			name:     "duplicate variables",
			input:    "test ${VAR} and ${VAR} again",
			expected: []string{"${VAR}", "${VAR}"}, // Extract all occurrences
		},
		{
			name:     "multiline with variables",
			input:    "image: registry.com/app:${VERSION}\nnamespace: ${NAMESPACE}\nreplicas: ${REPLICAS}",
			expected: []string{"${VERSION}", "${NAMESPACE}", "${REPLICAS}"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractVariableReferences(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateVariableExists(t *testing.T) {
	execContext := &ExecutionContext{
		WorkflowVariables: map[string]string{
			"ENVIRONMENT": "production",
			"REGION":      "us-east-1",
		},
		PreviousStepOutputs: map[string]map[string]string{
			"build": {
				"version": "v1.2.3",
			},
			"provision": {
				"db_url": "postgresql://db:5432",
			},
		},
		ResourceOutputs: map[string]map[string]string{
			"db": {
				"host": "db.example.com",
				"port": "5432",
			},
		},
	}

	env := map[string]string{
		"ENV_VAR": "test-value",
	}

	tests := []struct {
		name        string
		varRef      string
		expectError bool
	}{
		{
			name:        "workflow variable exists",
			varRef:      "${workflow.ENVIRONMENT}",
			expectError: false,
		},
		{
			name:        "workflow variable missing",
			varRef:      "${workflow.MISSING}",
			expectError: true,
		},
		{
			name:        "step output exists",
			varRef:      "${build.version}",
			expectError: false,
		},
		{
			name:        "step output missing",
			varRef:      "${deploy.result}",
			expectError: true,
		},
		{
			name:        "resource output exists",
			varRef:      "${resources.db.host}",
			expectError: false,
		},
		{
			name:        "resource output missing",
			varRef:      "${resources.cache.endpoint}",
			expectError: true,
		},
		{
			name:        "env variable exists",
			varRef:      "${ENV_VAR}",
			expectError: false,
		},
		{
			name:        "env variable missing",
			varRef:      "${MISSING_ENV}",
			expectError: true,
		},
		{
			name:        "unbraced variable exists",
			varRef:      "$ENV_VAR",
			expectError: false,
		},
		{
			name:        "unbraced variable missing",
			varRef:      "$MISSING",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := execContext.ValidateVariableExists(tt.varRef, env)
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "undefined variable")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateStepVariables_Success(t *testing.T) {
	execContext := &ExecutionContext{
		WorkflowVariables: map[string]string{
			"NAMESPACE": "production",
		},
		PreviousStepOutputs: map[string]map[string]string{
			"build": {
				"version": "v1.2.3",
			},
		},
	}

	step := types.Step{
		Name: "deploy",
		Type: "kubernetes",
		Config: map[string]interface{}{
			"namespace": "${workflow.NAMESPACE}",
			"image":     "registry.com/app:${build.version}",
		},
		Env: map[string]string{
			"DEPLOY_ENV": "prod",
		},
	}

	err := execContext.ValidateStepVariables(step, step.Env)
	assert.NoError(t, err)
}

func TestValidateStepVariables_FailFast(t *testing.T) {
	execContext := &ExecutionContext{
		WorkflowVariables:   map[string]string{},
		PreviousStepOutputs: map[string]map[string]string{},
	}

	step := types.Step{
		Name: "deploy",
		Type: "kubernetes",
		Config: map[string]interface{}{
			"namespace": "${workflow.NAMESPACE}", // Missing
			"image":     "registry.com/app:${build.version}", // Also missing
			"replicas":  "${workflow.REPLICAS}", // Also missing
		},
	}

	// Should fail-fast on first missing variable
	err := execContext.ValidateStepVariables(step, step.Env)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "undefined variable")
	// Should only report ONE variable (fail-fast)
	// Note: Map iteration order is non-deterministic, so we can't predict which variable fails first
	// Just verify that it's one of the expected missing variables
	errorMsg := err.Error()
	hasMissingVar := strings.Contains(errorMsg, "${workflow.NAMESPACE}") ||
		strings.Contains(errorMsg, "${build.version}") ||
		strings.Contains(errorMsg, "${workflow.REPLICAS}")
	assert.True(t, hasMissingVar, "Error should contain one of the missing variables")
}

func TestValidateStepVariables_NestedConfig(t *testing.T) {
	execContext := &ExecutionContext{
		WorkflowVariables:   map[string]string{},
		PreviousStepOutputs: map[string]map[string]string{},
	}

	step := types.Step{
		Name: "terraform",
		Type: "terraform",
		Config: map[string]interface{}{
			"variables": map[string]interface{}{
				"region":      "${workflow.REGION}", // Missing
				"environment": "production",
			},
			"backend_config": map[string]interface{}{
				"bucket": "${workflow.TERRAFORM_BUCKET}", // Missing
			},
		},
	}

	err := execContext.ValidateStepVariables(step, step.Env)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "undefined variable")
}

func TestValidateStepVariables_ArrayValues(t *testing.T) {
	execContext := &ExecutionContext{
		WorkflowVariables: map[string]string{
			"SUBNET_1": "subnet-abc",
		},
		PreviousStepOutputs: map[string]map[string]string{},
	}

	step := types.Step{
		Name: "configure",
		Type: "terraform",
		Config: map[string]interface{}{
			"subnets": []interface{}{
				"${workflow.SUBNET_1}",
				"${workflow.SUBNET_2}", // Missing
			},
		},
	}

	err := execContext.ValidateStepVariables(step, step.Env)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "${workflow.SUBNET_2}")
}

func TestValidateWorkflowVariables(t *testing.T) {
	execContext := &ExecutionContext{
		WorkflowVariables: map[string]string{
			"ENVIRONMENT": "production",
		},
		PreviousStepOutputs: map[string]map[string]string{},
	}

	workflow := types.Workflow{
		Variables: map[string]string{
			"ENVIRONMENT": "production",
		},
		Steps: []types.Step{
			{
				Name: "build",
				Type: "shell",
				Config: map[string]interface{}{
					"command": "echo Building for ${workflow.ENVIRONMENT}",
				},
			},
			{
				Name: "deploy",
				Type: "kubernetes",
				Config: map[string]interface{}{
					"namespace": "${workflow.ENVIRONMENT}",
					"image":     "app:${build.version}", // OK - will be available after build step
				},
			},
		},
	}

	err := execContext.ValidateWorkflowVariables(workflow)
	assert.NoError(t, err)
}

func TestValidateWorkflowVariables_MissingVariable(t *testing.T) {
	execContext := &ExecutionContext{
		WorkflowVariables:   map[string]string{},
		PreviousStepOutputs: map[string]map[string]string{},
	}

	workflow := types.Workflow{
		Variables: map[string]string{},
		Steps: []types.Step{
			{
				Name: "deploy",
				Type: "kubernetes",
				Config: map[string]interface{}{
					"namespace": "${workflow.REGION}", // Missing - not in workflow.Variables
				},
			},
		},
	}

	err := execContext.ValidateWorkflowVariables(workflow)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "undefined variable")
	assert.Contains(t, err.Error(), "${workflow.REGION}")
}

func TestIsStrictMode(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		expected bool
	}{
		{
			name:     "strict mode enabled explicitly",
			envValue: "true",
			expected: true,
		},
		{
			name:     "strict mode disabled",
			envValue: "false",
			expected: false,
		},
		{
			name:     "strict mode enabled with 1",
			envValue: "1",
			expected: true,
		},
		{
			name:     "strict mode default (not set)",
			envValue: "",
			expected: true, // Default is strict
		},
		{
			name:     "strict mode case insensitive TRUE",
			envValue: "TRUE",
			expected: true,
		},
		{
			name:     "strict mode case insensitive FALSE",
			envValue: "FALSE",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean environment
			os.Unsetenv("STRICT_VALIDATION")

			if tt.envValue != "" {
				os.Setenv("STRICT_VALIDATION", tt.envValue)
				defer os.Unsetenv("STRICT_VALIDATION")
			}

			result := IsStrictMode()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateStepVariables_StrictMode(t *testing.T) {
	// Save original env value
	originalEnv := os.Getenv("STRICT_VALIDATION")
	defer func() {
		if originalEnv != "" {
			os.Setenv("STRICT_VALIDATION", originalEnv)
		} else {
			os.Unsetenv("STRICT_VALIDATION")
		}
	}()

	execContext := &ExecutionContext{
		WorkflowVariables:   map[string]string{},
		PreviousStepOutputs: map[string]map[string]string{},
	}

	step := types.Step{
		Name: "deploy",
		Type: "kubernetes",
		Config: map[string]interface{}{
			"namespace": "${workflow.MISSING}",
		},
	}

	t.Run("strict mode fails", func(t *testing.T) {
		os.Setenv("STRICT_VALIDATION", "true")

		err := execContext.ValidateStepVariables(step, step.Env)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "undefined variable")
	})

	t.Run("lenient mode succeeds with warning", func(t *testing.T) {
		os.Setenv("STRICT_VALIDATION", "false")

		// In lenient mode, validation should NOT error
		// (warnings will be logged but function returns nil)
		err := execContext.ValidateStepVariables(step, step.Env)
		// Based on fail-fast + lenient design, should return nil
		assert.NoError(t, err)
	})
}

func TestValidateVariableExists_SystemEnv(t *testing.T) {
	// Set a system environment variable
	os.Setenv("TEST_SYSTEM_VAR", "system-value")
	defer os.Unsetenv("TEST_SYSTEM_VAR")

	execContext := &ExecutionContext{
		WorkflowVariables:   map[string]string{},
		PreviousStepOutputs: map[string]map[string]string{},
	}

	// Should find system environment variable
	err := execContext.ValidateVariableExists("${TEST_SYSTEM_VAR}", nil)
	assert.NoError(t, err)

	// Should fail for non-existent system variable
	err = execContext.ValidateVariableExists("${NONEXISTENT_SYSTEM_VAR}", nil)
	assert.Error(t, err)
}

func TestExtractVariableReferences_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: nil, // Empty result for empty input
		},
		{
			name:     "only braces no variable",
			input:    "${}",
			expected: nil, // Regex requires at least one char inside braces, ${} is not valid syntax
		},
		{
			name:     "incomplete variable",
			input:    "${INCOMPLETE",
			expected: nil, // Regex won't match incomplete pattern
		},
		{
			name:     "escaped dollar",
			input:    "\\$NOT_A_VAR",
			expected: []string{"$NOT_A_VAR"}, // Regex doesn't understand backslash escaping (YAML context handles this)
		},
		{
			name:     "variable in quotes",
			input:    "\"${VAR}\"",
			expected: []string{"${VAR}"}, // Variables in quotes are still valid
		},
		{
			name:     "complex nested structure",
			input:    "postgresql://${resources.db.host}:${resources.db.port}/${workflow.DATABASE}",
			expected: []string{"${resources.db.host}", "${resources.db.port}", "${workflow.DATABASE}"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractVariableReferences(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
