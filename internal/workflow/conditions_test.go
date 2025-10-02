package workflow

import (
	"innominatus/internal/types"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecutionContext_InterpolateResourceParams(t *testing.T) {
	ctx := NewExecutionContext()
	ctx.SetVariable("ENVIRONMENT", "production")
	ctx.SetVariable("REGION", "us-east-1")
	ctx.SetStepOutputs("build", map[string]string{
		"version":  "1.0.0",
		"build_id": "12345",
	})

	tests := []struct {
		name     string
		params   map[string]interface{}
		env      map[string]string
		expected map[string]interface{}
	}{
		{
			name: "simple string interpolation",
			params: map[string]interface{}{
				"name":    "myapp-${workflow.ENVIRONMENT}",
				"version": "${build.version}",
			},
			env: map[string]string{},
			expected: map[string]interface{}{
				"name":    "myapp-production",
				"version": "1.0.0",
			},
		},
		{
			name: "nested map interpolation",
			params: map[string]interface{}{
				"database": map[string]interface{}{
					"host": "db-${workflow.ENVIRONMENT}.example.com",
					"port": 5432,
					"name": "app_${workflow.REGION}",
				},
			},
			env: map[string]string{},
			expected: map[string]interface{}{
				"database": map[string]interface{}{
					"host": "db-production.example.com",
					"port": 5432,
					"name": "app_us-east-1",
				},
			},
		},
		{
			name: "array interpolation",
			params: map[string]interface{}{
				"tags": []interface{}{
					"env:${workflow.ENVIRONMENT}",
					"version:${build.version}",
					"build:${build.build_id}",
				},
			},
			env: map[string]string{},
			expected: map[string]interface{}{
				"tags": []interface{}{
					"env:production",
					"version:1.0.0",
					"build:12345",
				},
			},
		},
		{
			name: "mixed types interpolation",
			params: map[string]interface{}{
				"name":     "myapp",
				"replicas": 3,
				"enabled":  true,
				"image":    "registry.example.com/myapp:${build.version}",
				"metadata": map[string]interface{}{
					"labels": map[string]interface{}{
						"app":     "myapp",
						"env":     "${workflow.ENVIRONMENT}",
						"version": "${build.version}",
					},
				},
			},
			env: map[string]string{},
			expected: map[string]interface{}{
				"name":     "myapp",
				"replicas": 3,
				"enabled":  true,
				"image":    "registry.example.com/myapp:1.0.0",
				"metadata": map[string]interface{}{
					"labels": map[string]interface{}{
						"app":     "myapp",
						"env":     "production",
						"version": "1.0.0",
					},
				},
			},
		},
		{
			name: "step env override",
			params: map[string]interface{}{
				"environment": "${ENV}",
				"custom":      "${CUSTOM_VAR}",
			},
			env: map[string]string{
				"ENV":        "staging", // Override workflow variable
				"CUSTOM_VAR": "custom_value",
			},
			expected: map[string]interface{}{
				"environment": "staging",
				"custom":      "custom_value",
			},
		},
		{
			name: "multiple variable references in one string",
			params: map[string]interface{}{
				"url": "https://api-${workflow.ENVIRONMENT}.example.com/v${build.version}",
			},
			env: map[string]string{},
			expected: map[string]interface{}{
				"url": "https://api-production.example.com/v1.0.0",
			},
		},
		{
			name: "no interpolation needed",
			params: map[string]interface{}{
				"static":  "value",
				"number":  42,
				"boolean": false,
			},
			env: map[string]string{},
			expected: map[string]interface{}{
				"static":  "value",
				"number":  42,
				"boolean": false,
			},
		},
		{
			name:     "nil params",
			params:   nil,
			env:      map[string]string{},
			expected: nil,
		},
		{
			name:     "empty params",
			params:   map[string]interface{}{},
			env:      map[string]string{},
			expected: map[string]interface{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ctx.InterpolateResourceParams(tt.params, tt.env)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExecutionContext_ResourceOutputs(t *testing.T) {
	ctx := NewExecutionContext()

	t.Run("set and get resource outputs", func(t *testing.T) {
		outputs := map[string]string{
			"host": "db.example.com",
			"port": "5432",
			"name": "myapp_db",
		}

		ctx.SetResourceOutputs("database", outputs)

		host, found := ctx.GetResourceOutput("database", "host")
		assert.True(t, found)
		assert.Equal(t, "db.example.com", host)

		port, found := ctx.GetResourceOutput("database", "port")
		assert.True(t, found)
		assert.Equal(t, "5432", port)
	})

	t.Run("get non-existent resource output", func(t *testing.T) {
		_, found := ctx.GetResourceOutput("nonexistent", "key")
		assert.False(t, found)
	})

	t.Run("get all resource outputs", func(t *testing.T) {
		outputs := map[string]string{
			"endpoint": "cache.example.com",
			"port":     "6379",
		}

		ctx.SetResourceOutputs("cache", outputs)

		allOutputs, exists := ctx.GetAllResourceOutputs("cache")
		assert.True(t, exists)
		assert.Equal(t, outputs, allOutputs)
	})

	t.Run("set single resource output", func(t *testing.T) {
		ctx.SetResourceOutput("storage", "bucket_name", "my-bucket")

		bucket, found := ctx.GetResourceOutput("storage", "bucket_name")
		assert.True(t, found)
		assert.Equal(t, "my-bucket", bucket)
	})
}

func TestExecutionContext_ResourceReferences(t *testing.T) {
	ctx := NewExecutionContext()
	ctx.SetVariable("ENVIRONMENT", "production")
	ctx.SetResourceOutputs("database", map[string]string{
		"host": "db-prod.example.com",
		"port": "5432",
		"name": "myapp_production",
	})
	ctx.SetResourceOutputs("cache", map[string]string{
		"endpoint": "cache-prod.example.com",
		"port":     "6379",
	})

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "resources with braces",
			input:    "${resources.database.host}",
			expected: "db-prod.example.com",
		},
		{
			name:     "resources without braces",
			input:    "$resources.database.port",
			expected: "5432",
		},
		{
			name:     "multiple resource references",
			input:    "postgresql://${resources.database.host}:${resources.database.port}/${resources.database.name}",
			expected: "postgresql://db-prod.example.com:5432/myapp_production",
		},
		{
			name:     "mixed resources and workflow variables",
			input:    "env=${workflow.ENVIRONMENT},db=${resources.database.host}",
			expected: "env=production,db=db-prod.example.com",
		},
		{
			name:     "cache resource reference",
			input:    "redis://${resources.cache.endpoint}:${resources.cache.port}",
			expected: "redis://cache-prod.example.com:6379",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ctx.replaceVariables(tt.input, map[string]string{})
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExecutionContext_AllSyntaxSupport(t *testing.T) {
	// Test that all three syntaxes work together
	ctx := NewExecutionContext()

	// Set workflow variables
	ctx.SetVariable("APP_NAME", "myapp")
	ctx.SetVariable("ENVIRONMENT", "production")

	// Set step outputs
	ctx.SetStepOutputs("build", map[string]string{
		"version": "2.5.0",
		"image":   "registry.example.com/myapp:2.5.0",
	})

	// Set resource outputs
	ctx.SetResourceOutputs("database", map[string]string{
		"host": "db-prod.example.com",
		"port": "5432",
	})

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "workflow variable",
			input:    "${workflow.APP_NAME}",
			expected: "myapp",
		},
		{
			name:     "step output",
			input:    "${build.version}",
			expected: "2.5.0",
		},
		{
			name:     "resource output",
			input:    "${resources.database.host}",
			expected: "db-prod.example.com",
		},
		{
			name:     "all three combined",
			input:    "app=${workflow.APP_NAME},version=${build.version},db=${resources.database.host}",
			expected: "app=myapp,version=2.5.0,db=db-prod.example.com",
		},
		{
			name:     "connection string with all three",
			input:    "postgresql://${resources.database.host}:${resources.database.port}/${workflow.APP_NAME}_${workflow.ENVIRONMENT}?version=${build.version}",
			expected: "postgresql://db-prod.example.com:5432/myapp_production?version=2.5.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ctx.replaceVariables(tt.input, map[string]string{})
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExecutionContext_InterpolateResourceParams_WithResources(t *testing.T) {
	ctx := NewExecutionContext()
	ctx.SetVariable("ENVIRONMENT", "production")
	ctx.SetResourceOutputs("database", map[string]string{
		"host": "db.example.com",
		"port": "5432",
	})

	params := map[string]interface{}{
		"application": map[string]interface{}{
			"name": "myapp",
			"environment": map[string]interface{}{
				"DATABASE_URL": "postgresql://${resources.database.host}:${resources.database.port}/myapp",
				"ENV":          "${workflow.ENVIRONMENT}",
			},
		},
	}

	result := ctx.InterpolateResourceParams(params, map[string]string{})

	app := result["application"].(map[string]interface{})
	env := app["environment"].(map[string]interface{})
	assert.Equal(t, "postgresql://db.example.com:5432/myapp", env["DATABASE_URL"])
	assert.Equal(t, "production", env["ENV"])
}

func TestExecutionContext_InterpolateResourceParams_Integration(t *testing.T) {
	// Simulate a real workflow scenario
	ctx := NewExecutionContext()

	// Set workflow variables
	ctx.SetWorkflowVariables(map[string]string{
		"APP_NAME":    "myapp",
		"ENVIRONMENT": "production",
		"REGION":      "us-west-2",
	})

	// Simulate build step outputs
	ctx.SetStepOutputs("build", map[string]string{
		"version":      "2.5.0",
		"commit_sha":   "abc123def",
		"artifact_url": "https://artifacts.example.com/myapp-2.5.0.tar.gz",
	})

	// Simulate database provisioning outputs
	ctx.SetStepOutputs("provision-db", map[string]string{
		"db_host": "postgres-prod.example.com",
		"db_port": "5432",
		"db_name": "myapp_production",
	})

	// Resource params that need interpolation
	params := map[string]interface{}{
		"application": map[string]interface{}{
			"name":    "${workflow.APP_NAME}",
			"version": "${build.version}",
			"image":   "registry.example.com/${workflow.APP_NAME}:${build.version}",
			"environment": map[string]interface{}{
				"DATABASE_URL": "postgresql://${provision-db.db_host}:${provision-db.db_port}/${provision-db.db_name}",
				"APP_VERSION":  "${build.version}",
				"COMMIT_SHA":   "${build.commit_sha}",
				"REGION":       "${workflow.REGION}",
			},
			"tags": []interface{}{
				"app:${workflow.APP_NAME}",
				"env:${workflow.ENVIRONMENT}",
				"version:${build.version}",
			},
		},
		"replicas": 3,
		"monitoring": map[string]interface{}{
			"enabled": true,
			"labels": map[string]interface{}{
				"application": "${workflow.APP_NAME}",
				"environment": "${workflow.ENVIRONMENT}",
			},
		},
	}

	result := ctx.InterpolateResourceParams(params, map[string]string{})

	// Verify application settings
	app := result["application"].(map[string]interface{})
	assert.Equal(t, "myapp", app["name"])
	assert.Equal(t, "2.5.0", app["version"])
	assert.Equal(t, "registry.example.com/myapp:2.5.0", app["image"])

	// Verify environment variables
	env := app["environment"].(map[string]interface{})
	assert.Equal(t, "postgresql://postgres-prod.example.com:5432/myapp_production", env["DATABASE_URL"])
	assert.Equal(t, "2.5.0", env["APP_VERSION"])
	assert.Equal(t, "abc123def", env["COMMIT_SHA"])
	assert.Equal(t, "us-west-2", env["REGION"])

	// Verify tags
	tags := app["tags"].([]interface{})
	assert.Equal(t, "app:myapp", tags[0])
	assert.Equal(t, "env:production", tags[1])
	assert.Equal(t, "version:2.5.0", tags[2])

	// Verify non-interpolated values
	assert.Equal(t, 3, result["replicas"])

	// Verify monitoring
	monitoring := result["monitoring"].(map[string]interface{})
	assert.Equal(t, true, monitoring["enabled"])
	labels := monitoring["labels"].(map[string]interface{})
	assert.Equal(t, "myapp", labels["application"])
	assert.Equal(t, "production", labels["environment"])
}

func TestExecutionContext_WhenConditions(t *testing.T) {
	tests := []struct {
		name           string
		when           string
		previousStatus map[string]string
		expected       bool
		description    string
	}{
		{
			name:        "always runs",
			when:        "always",
			expected:    true,
			description: "Step with when=always should always execute",
		},
		{
			name:           "on_success with all success",
			when:           "on_success",
			previousStatus: map[string]string{"step1": "success", "step2": "success"},
			expected:       true,
			description:    "Should run when all previous steps succeeded",
		},
		{
			name:           "on_success with one failure",
			when:           "on_success",
			previousStatus: map[string]string{"step1": "success", "step2": "failed"},
			expected:       false,
			description:    "Should not run when any previous step failed",
		},
		{
			name:           "on_failure with failures",
			when:           "on_failure",
			previousStatus: map[string]string{"step1": "failed"},
			expected:       true,
			description:    "Should run when there are failures",
		},
		{
			name:           "on_failure with no failures",
			when:           "on_failure",
			previousStatus: map[string]string{"step1": "success", "step2": "success"},
			expected:       false,
			description:    "Should not run when no steps have failed",
		},
		{
			name:        "manual requires approval",
			when:        "manual",
			expected:    false,
			description: "Manual steps should be skipped",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := NewExecutionContext()
			for name, status := range tt.previousStatus {
				ctx.SetStepStatus(name, status)
			}

			step := types.Step{When: tt.when}
			shouldRun, _ := ctx.ShouldExecuteStep(step)

			assert.Equal(t, tt.expected, shouldRun, tt.description)
		})
	}
}

func TestExecutionContext_IfConditions(t *testing.T) {
	tests := []struct {
		name        string
		ifCondition string
		env         map[string]string
		expected    bool
		description string
	}{
		{
			name:        "simple true",
			ifCondition: "true",
			expected:    true,
			description: "if=true should execute",
		},
		{
			name:        "simple false",
			ifCondition: "false",
			expected:    false,
			description: "if=false should not execute",
		},
		{
			name:        "equality check - equal",
			ifCondition: "foo == bar",
			env:         map[string]string{"foo": "bar"},
			expected:    false, // "foo" != "bar" as strings
			description: "String equality check",
		},
		{
			name:        "equality check with variables",
			ifCondition: "$ENV == production",
			env:         map[string]string{"ENV": "production"},
			expected:    true,
			description: "Variable substitution in equality",
		},
		{
			name:        "numeric comparison",
			ifCondition: "10 > 5",
			expected:    true,
			description: "Numeric greater than",
		},
		{
			name:        "numeric comparison with variable",
			ifCondition: "$COUNT >= 3",
			env:         map[string]string{"COUNT": "5"},
			expected:    true,
			description: "Variable in numeric comparison",
		},
		{
			name:        "variable exists",
			ifCondition: "DEPLOY_ENV",
			env:         map[string]string{"DEPLOY_ENV": "staging"},
			expected:    true,
			description: "Variable exists and is non-empty",
		},
		{
			name:        "variable doesn't exist",
			ifCondition: "MISSING_VAR",
			env:         map[string]string{},
			expected:    false,
			description: "Variable doesn't exist",
		},
		{
			name:        "contains check",
			ifCondition: "hello world contains world",
			expected:    true,
			description: "String contains substring",
		},
		{
			name:        "startsWith check",
			ifCondition: "production startsWith prod",
			expected:    true,
			description: "String starts with prefix",
		},
		{
			name:        "endsWith check",
			ifCondition: "filename.txt endsWith .txt",
			expected:    true,
			description: "String ends with suffix",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := NewExecutionContext()
			step := types.Step{If: tt.ifCondition, Env: tt.env}

			shouldRun, _ := ctx.ShouldExecuteStep(step)

			assert.Equal(t, tt.expected, shouldRun, tt.description)
		})
	}
}

func TestExecutionContext_UnlessConditions(t *testing.T) {
	tests := []struct {
		name            string
		unlessCondition string
		env             map[string]string
		expected        bool
		description     string
	}{
		{
			name:            "unless true",
			unlessCondition: "true",
			expected:        false,
			description:     "unless=true should not execute",
		},
		{
			name:            "unless false",
			unlessCondition: "false",
			expected:        true,
			description:     "unless=false should execute",
		},
		{
			name:            "unless variable set",
			unlessCondition: "SKIP_TESTS",
			env:             map[string]string{"SKIP_TESTS": "true"},
			expected:        false,
			description:     "Should not run when SKIP_TESTS is set",
		},
		{
			name:            "unless variable not set",
			unlessCondition: "SKIP_TESTS",
			env:             map[string]string{},
			expected:        true,
			description:     "Should run when SKIP_TESTS is not set",
		},
		{
			name:            "unless equality",
			unlessCondition: "$ENV == production",
			env:             map[string]string{"ENV": "development"},
			expected:        true,
			description:     "Should run when ENV is not production",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := NewExecutionContext()
			step := types.Step{Unless: tt.unlessCondition, Env: tt.env}

			shouldRun, _ := ctx.ShouldExecuteStep(step)

			assert.Equal(t, tt.expected, shouldRun, tt.description)
		})
	}
}

func TestExecutionContext_StepStatusReference(t *testing.T) {
	ctx := NewExecutionContext()
	ctx.SetStepStatus("build", "success")
	ctx.SetStepStatus("test", "failed")

	tests := []struct {
		name        string
		ifCondition string
		expected    bool
		description string
	}{
		{
			name:        "check step success",
			ifCondition: "build.success",
			expected:    true,
			description: "Should detect successful step",
		},
		{
			name:        "check step failed",
			ifCondition: "test.failed",
			expected:    true,
			description: "Should detect failed step",
		},
		{
			name:        "check step not failed",
			ifCondition: "build.failed",
			expected:    false,
			description: "Should return false for non-failed step",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step := types.Step{If: tt.ifCondition}
			shouldRun, _ := ctx.ShouldExecuteStep(step)

			assert.Equal(t, tt.expected, shouldRun, tt.description)
		})
	}
}

func TestExecutionContext_CombinedConditions(t *testing.T) {
	tests := []struct {
		name        string
		step        types.Step
		setupCtx    func(*ExecutionContext)
		expected    bool
		description string
	}{
		{
			name: "when=always with if=true",
			step: types.Step{
				When: "always",
				If:   "true",
			},
			expected:    true,
			description: "Both conditions should pass",
		},
		{
			name: "when=on_success with if checking variable",
			step: types.Step{
				When: "on_success",
				If:   "$DEPLOY == true",
				Env:  map[string]string{"DEPLOY": "true"},
			},
			setupCtx: func(ctx *ExecutionContext) {
				ctx.SetStepStatus("previous", "success")
			},
			expected:    true,
			description: "Should run when both conditions are met",
		},
		{
			name: "when=on_success but previous failed",
			step: types.Step{
				When: "on_success",
				If:   "true",
			},
			setupCtx: func(ctx *ExecutionContext) {
				ctx.SetStepStatus("previous", "failed")
			},
			expected:    false,
			description: "Should not run when 'when' condition fails",
		},
		{
			name: "if and unless both present",
			step: types.Step{
				If:     "$DEPLOY == true",
				Unless: "$SKIP_DEPLOY == true",
				Env: map[string]string{
					"DEPLOY": "true",
					// SKIP_DEPLOY not set
				},
			},
			expected:    true,
			description: "Should run when if=true and unless=false",
		},
		{
			name: "if true but unless true",
			step: types.Step{
				If:     "$DEPLOY == true",
				Unless: "$SKIP_DEPLOY == true",
				Env: map[string]string{
					"DEPLOY":      "true",
					"SKIP_DEPLOY": "true",
				},
			},
			expected:    false,
			description: "Should not run when unless condition is true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := NewExecutionContext()
			if tt.setupCtx != nil {
				tt.setupCtx(ctx)
			}

			shouldRun, _ := ctx.ShouldExecuteStep(tt.step)

			assert.Equal(t, tt.expected, shouldRun, tt.description)
		})
	}
}

func TestExecutionContext_VariableReplacement(t *testing.T) {
	ctx := NewExecutionContext()

	tests := []struct {
		name     string
		input    string
		env      map[string]string
		expected string
	}{
		{
			name:     "replace ${VAR}",
			input:    "${APP_NAME}",
			env:      map[string]string{"APP_NAME": "myapp"},
			expected: "myapp",
		},
		{
			name:     "replace $VAR",
			input:    "$APP_NAME",
			env:      map[string]string{"APP_NAME": "myapp"},
			expected: "myapp",
		},
		{
			name:     "replace multiple variables",
			input:    "${APP}-${ENV}",
			env:      map[string]string{"APP": "myapp", "ENV": "prod"},
			expected: "myapp-prod",
		},
		{
			name:     "no replacement needed",
			input:    "static-value",
			env:      map[string]string{},
			expected: "static-value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ctx.replaceVariables(tt.input, tt.env)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExecutionContext_CompareValues(t *testing.T) {
	ctx := NewExecutionContext()

	tests := []struct {
		name     string
		left     string
		right    string
		operator string
		expected bool
	}{
		{"numeric equals", "10", "10", "==", true},
		{"numeric not equals", "10", "20", "!=", true},
		{"numeric less than", "5", "10", "<", true},
		{"numeric greater than", "10", "5", ">", true},
		{"numeric less or equal", "5", "5", "<=", true},
		{"numeric greater or equal", "10", "10", ">=", true},
		{"string equals", "hello", "hello", "==", true},
		{"string not equals", "hello", "world", "!=", true},
		{"string less than", "abc", "xyz", "<", true},
		{"string greater than", "xyz", "abc", ">", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ctx.compareValues(tt.left, tt.right, tt.operator)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExecutionContext_StringOperations(t *testing.T) {
	ctx := NewExecutionContext()

	t.Run("contains", func(t *testing.T) {
		result, err := ctx.evaluateCondition("hello world contains world", map[string]string{})
		require.NoError(t, err)
		assert.True(t, result)

		result, err = ctx.evaluateCondition("hello world contains missing", map[string]string{})
		require.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("startsWith", func(t *testing.T) {
		result, err := ctx.evaluateCondition("production startsWith prod", map[string]string{})
		require.NoError(t, err)
		assert.True(t, result)

		result, err = ctx.evaluateCondition("development startsWith prod", map[string]string{})
		require.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("endsWith", func(t *testing.T) {
		result, err := ctx.evaluateCondition("file.txt endsWith .txt", map[string]string{})
		require.NoError(t, err)
		assert.True(t, result)

		result, err = ctx.evaluateCondition("file.txt endsWith .md", map[string]string{})
		require.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("matches regex", func(t *testing.T) {
		result, err := ctx.evaluateCondition("test123 matches test[0-9]+", map[string]string{})
		require.NoError(t, err)
		assert.True(t, result)

		result, err = ctx.evaluateCondition("test matches ^[0-9]+$", map[string]string{})
		require.NoError(t, err)
		assert.False(t, result)
	})
}

func TestExecutionContext_ErrorHandling(t *testing.T) {
	ctx := NewExecutionContext()

	tests := []struct {
		name        string
		condition   string
		shouldError bool
		description string
	}{
		{
			name:        "invalid operator",
			condition:   "5 === 5",
			shouldError: false, // Will try to evaluate as string
			description: "Unknown operators should be handled",
		},
		{
			name:        "step not found",
			condition:   "nonexistent_step.success",
			shouldError: true,
			description: "Referencing non-existent step should error",
		},
		{
			name:        "invalid regex",
			condition:   "test matches [",
			shouldError: true,
			description: "Invalid regex should error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step := types.Step{If: tt.condition}
			_, reason := ctx.ShouldExecuteStep(step)

			if tt.shouldError {
				assert.NotEmpty(t, reason, tt.description)
			}
		})
	}
}
