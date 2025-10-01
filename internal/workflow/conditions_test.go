package workflow

import (
	"innominatus/internal/types"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
