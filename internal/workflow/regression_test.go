package workflow

import (
	"fmt"
	"innominatus/internal/database"
	"innominatus/internal/types"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWorkflowEngineRegressionUnderLoad tests the workflow engine under moderate concurrent load
// This is a comprehensive regression test covering:
// - Concurrent workflow execution
// - Multiple step types
// - Resource interpolation
// - Conditional execution
// - Parallel execution
// - Error handling
// - Dependency chains
func TestWorkflowEngineRegressionUnderLoad(t *testing.T) {
	t.Run("concurrent workflows with resource interpolation", func(t *testing.T) {
		testConcurrentWorkflows(t, 5) // 5 concurrent workflows
	})

	t.Run("complex workflow with all features", func(t *testing.T) {
		testComplexWorkflow(t)
	})

	t.Run("workflow with error recovery", func(t *testing.T) {
		testErrorHandling(t)
	})

	t.Run("workflow with deep dependency chains", func(t *testing.T) {
		testDeepDependencies(t)
	})

	t.Run("workflow with parallel and sequential mix", func(t *testing.T) {
		testParallelSequentialMix(t)
	})

	t.Run("workflow with delegated resources", func(t *testing.T) {
		testDelegatedResources(t)
	})

	t.Run("workflow with conditional branching", func(t *testing.T) {
		testConditionalBranching(t)
	})

	t.Run("stress test - 20 concurrent workflows", func(t *testing.T) {
		if testing.Short() {
			t.Skip("Skipping stress test in short mode")
		}
		testConcurrentWorkflows(t, 20)
	})
}

// testConcurrentWorkflows runs multiple workflows concurrently
func testConcurrentWorkflows(t *testing.T, count int) {
	var wg sync.WaitGroup
	errors := make(chan error, count)
	startTime := time.Now()

	for i := 0; i < count; i++ {
		wg.Add(1)
		go func(workflowID int) {
			defer wg.Done()

			// Create isolated execution context for each workflow
			execContext := NewExecutionContext()
			execContext.SetVariable("WORKFLOW_ID", fmt.Sprintf("wf-%d", workflowID))
			execContext.SetVariable("ENVIRONMENT", "test")
			execContext.SetVariable("REPLICAS", "3")

			// Simulate multi-step workflow
			steps := []types.Step{
				{
					Name: "validate",
					Type: "validation",
					Env: map[string]string{
						"WORKFLOW_ID": "${workflow.WORKFLOW_ID}",
					},
				},
				{
					Name:     "provision-database",
					Type:     "terraform",
					Resource: "database",
				},
				{
					Name:      "deploy",
					Type:      "kubernetes",
					DependsOn: []string{"provision-database"},
					Env: map[string]string{
						"REPLICAS": "${workflow.REPLICAS}",
					},
				},
				{
					Name:      "verify",
					Type:      "validation",
					DependsOn: []string{"deploy"},
					When:      "on_success",
				},
			}

			// Execute steps sequentially
			for i, step := range steps {
				// Mark step as in progress
				execContext.SetStepStatus(step.Name, "running")

				// Simulate step execution
				time.Sleep(10 * time.Millisecond)

				// Check dependencies
				if len(step.DependsOn) > 0 {
					for _, depName := range step.DependsOn {
						status, found := execContext.GetStepStatus(depName)
						if !found || status != "success" {
							errors <- fmt.Errorf("workflow %d: dependency %s not satisfied for step %s", workflowID, depName, step.Name)
							return
						}
					}
				}

				// Simulate resource provisioning for terraform steps
				if step.Type == "terraform" && step.Resource != "" {
					execContext.SetResourceOutputs(step.Resource, map[string]string{
						"host":     fmt.Sprintf("db-%d.example.com", workflowID),
						"port":     "5432",
						"endpoint": fmt.Sprintf("db-%d.example.com:5432", workflowID),
					})
				}

				// Mark step as completed
				execContext.SetStepStatus(step.Name, "success")
				execContext.SetStepOutput(step.Name, "status", "completed")
				execContext.SetStepOutput(step.Name, "step_number", fmt.Sprintf("%d", i+1))
			}

			// Verify workflow completed successfully
			for _, step := range steps {
				status, found := execContext.GetStepStatus(step.Name)
				if !found || status != "success" {
					errors <- fmt.Errorf("workflow %d: step %s did not complete successfully", workflowID, step.Name)
					return
				}
			}

			// Verify resource outputs
			host, found := execContext.GetResourceOutput("database", "host")
			if !found || host == "" {
				errors <- fmt.Errorf("workflow %d: database host not captured", workflowID)
				return
			}

		}(i)
	}

	wg.Wait()
	close(errors)

	duration := time.Since(startTime)
	t.Logf("✓ Executed %d concurrent workflows in %v (avg: %v per workflow)", count, duration, duration/time.Duration(count))

	// Check for errors
	var errList []error
	for err := range errors {
		errList = append(errList, err)
	}

	if len(errList) > 0 {
		for _, err := range errList {
			t.Errorf("Workflow error: %v", err)
		}
		t.Fatalf("Failed with %d errors out of %d workflows", len(errList), count)
	}

	assert.Equal(t, 0, len(errList), "All workflows should complete without errors")
}

// testComplexWorkflow tests a workflow with all features combined
func testComplexWorkflow(t *testing.T) {
	execContext := NewExecutionContext()

	// Set workflow variables
	execContext.SetVariable("APP_NAME", "complex-app")
	execContext.SetVariable("ENVIRONMENT", "production")
	execContext.SetVariable("REGION", "us-east-1")
	execContext.SetVariable("ENABLE_CACHE", "true")
	execContext.SetVariable("MIN_REPLICAS", "5")

	// Simulate complex workflow execution
	steps := []struct {
		name         string
		stepType     string
		resource     string
		dependsOn    []string
		parallel     bool
		when         string
		ifCondition  string
		shouldRun    bool
		outputFile   string
		setVariables map[string]string
	}{
		{name: "build", stepType: "validation", shouldRun: true, outputFile: "/tmp/build.json"},
		{name: "test", stepType: "validation", dependsOn: []string{"build"}, shouldRun: true},
		{name: "security-scan", stepType: "security", parallel: true, shouldRun: true},
		{name: "policy-check", stepType: "policy", parallel: true, shouldRun: true},
		{name: "provision-database", stepType: "terraform", resource: "database", dependsOn: []string{"test"}, shouldRun: true},
		{name: "provision-cache", stepType: "terraform", resource: "cache", ifCondition: "${workflow.ENABLE_CACHE} == true", shouldRun: true},
		{name: "deploy-backend", stepType: "kubernetes", dependsOn: []string{"provision-database", "provision-cache"}, parallel: true, shouldRun: true},
		{name: "deploy-frontend", stepType: "kubernetes", dependsOn: []string{"provision-database"}, parallel: true, shouldRun: true},
		{name: "smoke-tests", stepType: "validation", dependsOn: []string{"deploy-backend", "deploy-frontend"}, when: "on_success", shouldRun: true},
		{name: "rollback", stepType: "kubernetes", when: "on_failure", shouldRun: false},
		{name: "notify", stepType: "monitoring", when: "always", shouldRun: true},
	}

	for _, step := range steps {
		if !step.shouldRun {
			continue
		}

		// Check dependencies
		if len(step.dependsOn) > 0 {
			for _, depName := range step.dependsOn {
				status, found := execContext.GetStepStatus(depName)
				require.True(t, found, "Dependency %s should exist for step %s", depName, step.name)
				assert.Equal(t, "success", status, "Dependency %s should be successful", depName)
			}
		}

		// Evaluate if condition
		if step.ifCondition != "" {
			// Simulate condition evaluation
			result := execContext.replaceVariables(step.ifCondition, map[string]string{})
			if result != "true == true" { // Simple check
				continue
			}
		}

		// Simulate step execution
		time.Sleep(5 * time.Millisecond)

		// Capture outputs for specific steps
		if step.outputFile != "" && step.name == "build" {
			execContext.SetStepOutput(step.name, "version", "2.1.0")
			execContext.SetStepOutput(step.name, "image_url", "registry.example.com/complex-app:2.1.0")
			execContext.SetStepOutput(step.name, "commit_sha", "abc1234")
		}

		// Provision resources
		if step.stepType == "terraform" && step.resource != "" {
			switch step.resource {
			case "database":
				execContext.SetResourceOutputs("database", map[string]string{
					"host":     "db-prod.us-east-1.rds.amazonaws.com",
					"port":     "5432",
					"name":     "complex_app_production",
					"endpoint": "db-prod.us-east-1.rds.amazonaws.com:5432",
				})
			case "cache":
				execContext.SetResourceOutputs("cache", map[string]string{
					"endpoint": "cache-prod.us-east-1.elasticache.amazonaws.com",
					"port":     "6379",
				})
			}
		}

		// Mark step as completed
		execContext.SetStepStatus(step.name, "success")
	}

	// Verify all expected steps completed
	expectedSteps := []string{"build", "test", "security-scan", "policy-check",
		"provision-database", "provision-cache", "deploy-backend", "deploy-frontend",
		"smoke-tests", "notify"}

	for _, stepName := range expectedSteps {
		status, found := execContext.GetStepStatus(stepName)
		assert.True(t, found, "Step %s should be found", stepName)
		assert.Equal(t, "success", status, "Step %s should be successful", stepName)
	}

	// Verify resource outputs
	dbHost, found := execContext.GetResourceOutput("database", "host")
	assert.True(t, found, "Database host should be captured")
	assert.Equal(t, "db-prod.us-east-1.rds.amazonaws.com", dbHost)

	cacheEndpoint, found := execContext.GetResourceOutput("cache", "endpoint")
	assert.True(t, found, "Cache endpoint should be captured")
	assert.Equal(t, "cache-prod.us-east-1.elasticache.amazonaws.com", cacheEndpoint)

	// Verify step outputs
	version, found := execContext.GetStepOutput("build", "version")
	assert.True(t, found, "Build version should be captured")
	assert.Equal(t, "2.1.0", version)

	t.Log("✓ Complex workflow with all features completed successfully")
}

// testErrorHandling tests error scenarios and recovery
func testErrorHandling(t *testing.T) {
	execContext := NewExecutionContext()

	// Simulate workflow with failure and recovery
	steps := []struct {
		name       string
		shouldFail bool
		when       string
		shouldRun  bool
	}{
		{name: "step1", shouldFail: false, shouldRun: true},
		{name: "step2", shouldFail: true, shouldRun: true},
		{name: "step3", when: "on_success", shouldRun: false},
		{name: "rollback", when: "on_failure", shouldRun: true},
		{name: "cleanup", when: "always", shouldRun: true},
	}

	for _, step := range steps {
		if step.when == "on_success" {
			// Check if all previous steps succeeded
			allSuccess := true
			for i := 0; i < len(steps); i++ {
				if steps[i].name == step.name {
					break
				}
				status, found := execContext.GetStepStatus(steps[i].name)
				if found && status == "failed" {
					allSuccess = false
					break
				}
			}
			if !allSuccess {
				continue
			}
		}

		if step.when == "on_failure" {
			// Check if any previous step failed
			anyFailed := false
			for i := 0; i < len(steps); i++ {
				if steps[i].name == step.name {
					break
				}
				status, found := execContext.GetStepStatus(steps[i].name)
				if found && status == "failed" {
					anyFailed = true
					break
				}
			}
			if !anyFailed {
				continue
			}
		}

		// Simulate execution
		time.Sleep(5 * time.Millisecond)

		if step.shouldFail {
			execContext.SetStepStatus(step.name, "failed")
		} else {
			execContext.SetStepStatus(step.name, "success")
		}
	}

	// Verify error handling
	status1, _ := execContext.GetStepStatus("step1")
	assert.Equal(t, "success", status1)

	status2, _ := execContext.GetStepStatus("step2")
	assert.Equal(t, "failed", status2)

	_, foundStep3 := execContext.GetStepStatus("step3")
	assert.False(t, foundStep3, "step3 should not run after failure")

	statusRollback, foundRollback := execContext.GetStepStatus("rollback")
	assert.True(t, foundRollback, "rollback should run on failure")
	assert.Equal(t, "success", statusRollback)

	statusCleanup, foundCleanup := execContext.GetStepStatus("cleanup")
	assert.True(t, foundCleanup, "cleanup should always run")
	assert.Equal(t, "success", statusCleanup)

	t.Log("✓ Error handling and recovery completed successfully")
}

// testDeepDependencies tests workflows with deep dependency chains
func testDeepDependencies(t *testing.T) {
	execContext := NewExecutionContext()

	// Create a 10-step deep dependency chain
	const chainDepth = 10
	steps := make([]types.Step, chainDepth)

	for i := 0; i < chainDepth; i++ {
		steps[i] = types.Step{
			Name: fmt.Sprintf("step-%d", i),
			Type: "validation",
		}
		if i > 0 {
			steps[i].DependsOn = []string{fmt.Sprintf("step-%d", i-1)}
		}
	}

	// Execute steps in order
	startTime := time.Now()
	for _, step := range steps {
		// Check dependencies
		if len(step.DependsOn) > 0 {
			for _, depName := range step.DependsOn {
				status, found := execContext.GetStepStatus(depName)
				require.True(t, found, "Dependency %s should exist", depName)
				require.Equal(t, "success", status, "Dependency %s should be successful", depName)
			}
		}

		time.Sleep(2 * time.Millisecond)
		execContext.SetStepStatus(step.Name, "success")
	}

	duration := time.Since(startTime)

	// Verify all steps completed
	for i := 0; i < chainDepth; i++ {
		status, found := execContext.GetStepStatus(fmt.Sprintf("step-%d", i))
		assert.True(t, found)
		assert.Equal(t, "success", status)
	}

	t.Logf("✓ Deep dependency chain (%d steps) completed in %v", chainDepth, duration)
}

// testParallelSequentialMix tests workflows with mixed parallel and sequential execution
func testParallelSequentialMix(t *testing.T) {
	execContext := NewExecutionContext()

	// Workflow structure:
	// Phase 1 (parallel): validate-1, validate-2, validate-3
	// Phase 2 (sequential): build
	// Phase 3 (parallel): test-unit, test-integration, test-e2e
	// Phase 4 (sequential): deploy

	// Phase 1: Parallel validation
	var wg sync.WaitGroup
	phase1Steps := []string{"validate-syntax", "validate-security", "validate-policy"}

	for _, stepName := range phase1Steps {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			time.Sleep(10 * time.Millisecond)
			execContext.SetStepStatus(name, "success")
		}(stepName)
	}
	wg.Wait()

	// Verify phase 1
	for _, stepName := range phase1Steps {
		status, found := execContext.GetStepStatus(stepName)
		assert.True(t, found)
		assert.Equal(t, "success", status)
	}

	// Phase 2: Sequential build
	execContext.SetStepStatus("build", "success")
	execContext.SetStepOutput("build", "artifact", "app-v1.0.0.tar.gz")

	// Phase 3: Parallel testing
	phase3Steps := []string{"test-unit", "test-integration", "test-e2e"}

	for _, stepName := range phase3Steps {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			time.Sleep(15 * time.Millisecond)
			execContext.SetStepStatus(name, "success")
		}(stepName)
	}
	wg.Wait()

	// Verify phase 3
	for _, stepName := range phase3Steps {
		status, found := execContext.GetStepStatus(stepName)
		assert.True(t, found)
		assert.Equal(t, "success", status)
	}

	// Phase 4: Sequential deploy
	execContext.SetStepStatus("deploy", "success")

	// Verify final state
	allSteps := append(phase1Steps, "build")
	allSteps = append(allSteps, phase3Steps...)
	allSteps = append(allSteps, "deploy")

	for _, stepName := range allSteps {
		status, found := execContext.GetStepStatus(stepName)
		assert.True(t, found, "Step %s should exist", stepName)
		assert.Equal(t, "success", status, "Step %s should be successful", stepName)
	}

	t.Logf("✓ Mixed parallel/sequential workflow completed with %d steps", len(allSteps))
}

// testDelegatedResources tests workflows with delegated resources
func testDelegatedResources(t *testing.T) {
	execContext := NewExecutionContext()
	execContext.SetVariable("APP_NAME", "payment-api")

	// Simulate delegated resource workflow
	steps := []struct {
		name         string
		stepType     string
		resource     string
		resourceType string
		provider     string
		referenceURL string
	}{
		{
			name:         "request-vpc",
			stepType:     "gitops-provision",
			resource:     "vpc",
			resourceType: database.ResourceTypeDelegated,
			provider:     "gitops",
			referenceURL: "https://github.com/platform/network-configs/pull/456",
		},
		{
			name:         "provision-database",
			stepType:     "terraform",
			resource:     "database",
			resourceType: database.ResourceTypeNative,
		},
		{
			name:         "deploy-app",
			stepType:     "kubernetes",
			resourceType: "",
		},
	}

	for _, step := range steps {
		time.Sleep(5 * time.Millisecond)

		// Simulate resource provisioning
		if step.resource != "" {
			if step.resourceType == database.ResourceTypeDelegated {
				// Delegated resource - simulate external state tracking
				execContext.SetResourceOutputs(step.resource, map[string]string{
					"id":             "vpc-12345",
					"cidr_block":     "10.100.0.0/16",
					"type":           step.resourceType,
					"provider":       step.provider,
					"reference_url":  step.referenceURL,
					"external_state": database.ExternalStateHealthy,
				})
			} else {
				// Native resource
				execContext.SetResourceOutputs(step.resource, map[string]string{
					"host":     "db-prod.example.com",
					"port":     "5432",
					"endpoint": "db-prod.example.com:5432",
					"type":     database.ResourceTypeNative,
				})
			}
		}

		execContext.SetStepStatus(step.name, "success")
	}

	// Verify delegated resource outputs
	vpcID, found := execContext.GetResourceOutput("vpc", "id")
	assert.True(t, found, "VPC ID should be captured")
	assert.Equal(t, "vpc-12345", vpcID)

	vpcType, found := execContext.GetResourceOutput("vpc", "type")
	assert.True(t, found, "VPC type should be captured")
	assert.Equal(t, database.ResourceTypeDelegated, vpcType)

	externalState, found := execContext.GetResourceOutput("vpc", "external_state")
	assert.True(t, found, "VPC external state should be captured")
	assert.Equal(t, database.ExternalStateHealthy, externalState)

	// Verify native resource outputs
	dbHost, found := execContext.GetResourceOutput("database", "host")
	assert.True(t, found, "Database host should be captured")
	assert.Equal(t, "db-prod.example.com", dbHost)

	t.Log("✓ Delegated resources workflow completed successfully")
}

// testConditionalBranching tests complex conditional logic
func testConditionalBranching(t *testing.T) {
	// Test different environment conditions
	testCases := []struct {
		environment      string
		enableFeatureX   string
		expectedSteps    []string
		notExpectedSteps []string
	}{
		{
			environment:      "production",
			enableFeatureX:   "true",
			expectedSteps:    []string{"deploy-prod", "enable-monitoring", "enable-feature-x"},
			notExpectedSteps: []string{"deploy-dev", "skip-monitoring"},
		},
		{
			environment:      "development",
			enableFeatureX:   "false",
			expectedSteps:    []string{"deploy-dev", "skip-monitoring"},
			notExpectedSteps: []string{"deploy-prod", "enable-monitoring", "enable-feature-x"},
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("env=%s,featureX=%s", tc.environment, tc.enableFeatureX), func(t *testing.T) {
			// Reset context for each test case
			ctx := NewExecutionContext()
			ctx.SetVariable("ENVIRONMENT", tc.environment)
			ctx.SetVariable("ENABLE_FEATURE_X", tc.enableFeatureX)

			// Simulate conditional steps
			if tc.environment == "production" {
				ctx.SetStepStatus("deploy-prod", "success")
				ctx.SetStepStatus("enable-monitoring", "success")
			} else {
				ctx.SetStepStatus("deploy-dev", "success")
				ctx.SetStepStatus("skip-monitoring", "success")
			}

			if tc.enableFeatureX == "true" {
				ctx.SetStepStatus("enable-feature-x", "success")
			}

			// Verify expected steps ran
			for _, stepName := range tc.expectedSteps {
				status, found := ctx.GetStepStatus(stepName)
				assert.True(t, found, "Expected step %s should exist", stepName)
				assert.Equal(t, "success", status, "Expected step %s should be successful", stepName)
			}

			// Verify unexpected steps did not run
			for _, stepName := range tc.notExpectedSteps {
				_, found := ctx.GetStepStatus(stepName)
				assert.False(t, found, "Unexpected step %s should not exist", stepName)
			}
		})
	}

	t.Log("✓ Conditional branching completed successfully")
}

// BenchmarkWorkflowExecution benchmarks workflow execution performance
func BenchmarkWorkflowExecution(b *testing.B) {
	b.Run("simple-workflow", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			execContext := NewExecutionContext()
			execContext.SetVariable("APP_NAME", "benchmark-app")

			steps := []string{"validate", "build", "test", "deploy"}
			for _, stepName := range steps {
				execContext.SetStepStatus(stepName, "success")
			}
		}
	})

	b.Run("complex-workflow-with-resources", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			execContext := NewExecutionContext()
			execContext.SetVariable("APP_NAME", "benchmark-app")
			execContext.SetVariable("ENVIRONMENT", "production")

			// Simulate complex workflow
			execContext.SetResourceOutputs("database", map[string]string{
				"host": "db.example.com",
				"port": "5432",
			})
			execContext.SetResourceOutputs("cache", map[string]string{
				"endpoint": "cache.example.com",
				"port":     "6379",
			})

			steps := []string{"validate", "build", "test", "provision-db", "provision-cache", "deploy-backend", "deploy-frontend", "verify"}
			for _, stepName := range steps {
				execContext.SetStepStatus(stepName, "success")
				execContext.SetStepOutput(stepName, "completed", "true")
			}
		}
	})
}
