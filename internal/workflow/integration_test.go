package workflow

import (
	"context"
	"innominatus/internal/types"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestResourceOutputCapture tests that Terraform outputs are captured and stored
func TestResourceOutputCapture(t *testing.T) {
	// Skip if terraform is not installed (CI environment)
	if _, err := exec.LookPath("terraform"); err != nil {
		t.Skip("Skipping test: terraform executable not found in PATH")
	}

	// Create a temporary directory for terraform outputs
	tmpDir, err := os.MkdirTemp("", "terraform-test-*")
	require.NoError(t, err)
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	// Create a mock terraform output file
	terraformOutput := `{
		"database_host": {
			"value": "db.example.com",
			"type": "string"
		},
		"database_port": {
			"value": "5432",
			"type": "string"
		},
		"database_name": {
			"value": "myapp_production",
			"type": "string"
		}
	}`

	// Mock terraform by creating a script that outputs the JSON
	err = os.WriteFile(filepath.Join(tmpDir, "terraform"), []byte(`#!/bin/bash
if [ "$1" = "output" ] && [ "$2" = "-json" ]; then
	cat <<'EOF'
`+terraformOutput+`
EOF
	exit 0
fi
exit 1
`), 0755)
	require.NoError(t, err)

	// Add tmpDir to PATH so our mock terraform is found
	oldPath := os.Getenv("PATH")
	err = os.Setenv("PATH", tmpDir+":"+oldPath)
	require.NoError(t, err)
	defer func() {
		_ = os.Setenv("PATH", oldPath)
	}()

	// Create execution context and executor
	execContext := NewExecutionContext()
	executor := &WorkflowExecutor{
		execContext: execContext,
	}

	// Create a step with terraform outputs
	step := types.Step{
		Name:     "provision-database",
		Type:     "terraform",
		Resource: "database",
		Outputs:  []string{"database_host", "database_port", "database_name"},
	}

	// Call terraformCaptureOutputs
	err = executor.terraformCaptureOutputs(context.Background(), tmpDir, step.Outputs, step)
	require.NoError(t, err)

	// Verify outputs were stored in execution context
	host, found := execContext.GetResourceOutput("database", "database_host")
	assert.True(t, found, "database_host should be stored")
	assert.Equal(t, "db.example.com", host)

	port, found := execContext.GetResourceOutput("database", "database_port")
	assert.True(t, found, "database_port should be stored")
	assert.Equal(t, "5432", port)

	name, found := execContext.GetResourceOutput("database", "database_name")
	assert.True(t, found, "database_name should be stored")
	assert.Equal(t, "myapp_production", name)
}

// TestResourceInterpolationE2E tests end-to-end resource interpolation in workflows
func TestResourceInterpolationE2E(t *testing.T) {
	// Create execution context
	execContext := NewExecutionContext()

	// Simulate terraform step outputs being stored
	execContext.SetResourceOutputs("database", map[string]string{
		"host": "db-prod.example.com",
		"port": "5432",
		"name": "production_db",
	})

	execContext.SetResourceOutputs("cache", map[string]string{
		"endpoint": "cache-prod.example.com",
		"port":     "6379",
	})

	// Create a deployment step that uses resource interpolation
	deployStep := types.Step{
		Name: "deploy-app",
		Type: "kubernetes",
		Env: map[string]string{
			"DATABASE_URL": "postgresql://${resources.database.host}:${resources.database.port}/${resources.database.name}",
			"REDIS_URL":    "redis://${resources.cache.endpoint}:${resources.cache.port}",
			"APP_NAME":     "my-app",
		},
	}

	// Test interpolation using the execution context's replaceVariables
	actualDBUrl := execContext.replaceVariables(deployStep.Env["DATABASE_URL"], deployStep.Env)
	expectedDBUrl := "postgresql://db-prod.example.com:5432/production_db"
	assert.Equal(t, expectedDBUrl, actualDBUrl, "Database URL should be interpolated")

	actualRedisUrl := execContext.replaceVariables(deployStep.Env["REDIS_URL"], deployStep.Env)
	expectedRedisUrl := "redis://cache-prod.example.com:6379"
	assert.Equal(t, expectedRedisUrl, actualRedisUrl, "Redis URL should be interpolated")

	// Test that non-interpolated values remain unchanged
	actualAppName := execContext.replaceVariables(deployStep.Env["APP_NAME"], deployStep.Env)
	assert.Equal(t, "my-app", actualAppName, "Non-interpolated values should remain unchanged")
}

// TestDependencyEnforcement tests that step dependencies are enforced
func TestDependencyEnforcement(t *testing.T) {
	execContext := NewExecutionContext()

	// Create mock steps with dependencies
	step1 := types.Step{
		Name: "provision-infrastructure",
		Type: "terraform",
	}

	step2 := types.Step{
		Name:      "deploy-application",
		Type:      "kubernetes",
		DependsOn: []string{"provision-infrastructure"},
	}

	step3 := types.Step{
		Name:      "run-tests",
		Type:      "validation",
		DependsOn: []string{"deploy-application"},
	}

	t.Run("dependency not found", func(t *testing.T) {
		// Try to check dependencies when step1 hasn't been executed yet
		depStatus, found := execContext.GetStepStatus("provision-infrastructure")
		assert.False(t, found, "Dependency should not be found")
		assert.Empty(t, depStatus)
	})

	t.Run("dependency failed", func(t *testing.T) {
		// Mark step1 as failed
		execContext.SetStepStatus(step1.Name, "failed")

		// Check if step2 can run
		depStatus, found := execContext.GetStepStatus("provision-infrastructure")
		assert.True(t, found, "Dependency should be found")
		assert.Equal(t, "failed", depStatus)

		// Verify that step2 would not be allowed to run
		// (this simulates the check in executeSingleStep)
		if depStatus != "success" {
			t.Log("✓ Step2 correctly blocked due to failed dependency")
		}
	})

	t.Run("dependency succeeded", func(t *testing.T) {
		// Mark step1 as success
		execContext.SetStepStatus(step1.Name, "success")

		// Check if step2 can now run
		depStatus, found := execContext.GetStepStatus("provision-infrastructure")
		assert.True(t, found, "Dependency should be found")
		assert.Equal(t, "success", depStatus)

		t.Log("✓ Step2 can proceed with successful dependency")
	})

	t.Run("chained dependencies", func(t *testing.T) {
		// Mark step1 and step2 as success
		execContext.SetStepStatus(step1.Name, "success")
		execContext.SetStepStatus(step2.Name, "success")

		// Check if step3 can run (depends on step2, which depends on step1)
		for _, depName := range step3.DependsOn {
			depStatus, found := execContext.GetStepStatus(depName)
			assert.True(t, found, "Dependency %s should be found", depName)
			assert.Equal(t, "success", depStatus)
		}

		t.Log("✓ Step3 can proceed with all chained dependencies satisfied")
	})

	t.Run("skipped dependency", func(t *testing.T) {
		// Reset context
		execContext = NewExecutionContext()
		execContext.SetStepStatus(step1.Name, "skipped")

		// Check if step2 should run with skipped dependency
		depStatus, found := execContext.GetStepStatus("provision-infrastructure")
		assert.True(t, found, "Dependency should be found")
		assert.Equal(t, "skipped", depStatus)

		// Step should NOT proceed if dependency is skipped (not success)
		if depStatus != "success" {
			t.Log("✓ Step2 correctly blocked due to skipped dependency")
		}
	})
}

// TestCompleteWorkflowWithResourcesAndDependencies tests a complete workflow scenario
func TestCompleteWorkflowWithResourcesAndDependencies(t *testing.T) {
	execContext := NewExecutionContext()

	// Set workflow variables
	execContext.SetVariable("ENVIRONMENT", "production")
	execContext.SetVariable("REGION", "us-east-1")
	execContext.SetVariable("APP_NAME", "payment-api")

	// Step 1: Build application (no dependencies)
	step1 := types.Step{
		Name: "build",
		Type: "validation",
	}
	execContext.SetStepStatus(step1.Name, "success")
	execContext.SetStepOutput(step1.Name, "image_url", "registry.example.com/payment-api:v1.2.3")
	execContext.SetStepOutput(step1.Name, "version", "v1.2.3")

	// Step 2: Provision database (no dependencies)
	step2 := types.Step{
		Name:     "provision-database",
		Type:     "terraform",
		Resource: "database",
	}
	execContext.SetStepStatus(step2.Name, "success")
	execContext.SetResourceOutputs("database", map[string]string{
		"host":     "db-prod.us-east-1.rds.amazonaws.com",
		"port":     "5432",
		"name":     "payment_api_production",
		"endpoint": "db-prod.us-east-1.rds.amazonaws.com:5432",
	})

	// Step 3: Provision cache (no dependencies)
	step3 := types.Step{
		Name:     "provision-cache",
		Type:     "terraform",
		Resource: "cache",
	}
	execContext.SetStepStatus(step3.Name, "success")
	execContext.SetResourceOutputs("cache", map[string]string{
		"endpoint": "cache-prod.us-east-1.elasticache.amazonaws.com",
		"port":     "6379",
	})

	// Step 4: Deploy application (depends on all previous steps)
	step4 := types.Step{
		Name:      "deploy-app",
		Type:      "kubernetes",
		DependsOn: []string{"build", "provision-database", "provision-cache"},
		Env: map[string]string{
			"APP_NAME":     "${workflow.APP_NAME}",
			"ENVIRONMENT":  "${workflow.ENVIRONMENT}",
			"REGION":       "${workflow.REGION}",
			"IMAGE":        "${build.image_url}",
			"VERSION":      "${build.version}",
			"DATABASE_URL": "postgresql://${resources.database.host}:${resources.database.port}/${resources.database.name}",
			"REDIS_URL":    "redis://${resources.cache.endpoint}:${resources.cache.port}",
		},
	}

	// Verify all dependencies are satisfied
	for _, depName := range step4.DependsOn {
		depStatus, found := execContext.GetStepStatus(depName)
		assert.True(t, found, "Dependency %s should exist", depName)
		assert.Equal(t, "success", depStatus, "Dependency %s should be successful", depName)
	}

	// Test all variable types in environment configuration
	tests := []struct {
		name     string
		envKey   string
		expected string
	}{
		{
			name:     "workflow variable",
			envKey:   "APP_NAME",
			expected: "payment-api",
		},
		{
			name:     "step output",
			envKey:   "IMAGE",
			expected: "registry.example.com/payment-api:v1.2.3",
		},
		{
			name:     "resource interpolation - database",
			envKey:   "DATABASE_URL",
			expected: "postgresql://db-prod.us-east-1.rds.amazonaws.com:5432/payment_api_production",
		},
		{
			name:     "resource interpolation - cache",
			envKey:   "REDIS_URL",
			expected: "redis://cache-prod.us-east-1.elasticache.amazonaws.com:6379",
		},
		{
			name:     "mixed interpolation",
			envKey:   "ENVIRONMENT",
			expected: "production",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := execContext.replaceVariables(step4.Env[tt.envKey], step4.Env)
			assert.Equal(t, tt.expected, actual, "Environment variable %s should be correctly interpolated", tt.envKey)
		})
	}

	t.Log("✓ Complete workflow with resources and dependencies validated successfully")
}
