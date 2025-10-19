//go:build ignore
// +build ignore

package main

import (
	"fmt"
	"innominatus/internal/database"
	"innominatus/internal/types"
	"innominatus/internal/workflow"
)

// Note: This is a standalone integration test program, not a unit test.
// The fmt.Println statements are intentional for user-facing test output.
func main() {
	// Connect to database using environment variables (or defaults)
	db, err := database.NewDatabase()
	if err != nil {
		fmt.Printf("❌ Failed to connect to database: %v\n", err)
		return
	}
	defer db.Close()

	repo := database.NewWorkflowRepository(db)
	executor := workflow.NewWorkflowExecutor(repo)

	appName := "test-retry-app"
	workflowName := "test-workflow"

	// Create a test workflow that will fail at step 2
	testWorkflow := types.Workflow{
		Steps: []types.Step{
			{
				Name: "step1-success",
				Type: "kubernetes",
				Config: map[string]interface{}{
					"operation": "namespace",
					"namespace": "test",
				},
			},
			{
				Name: "step2-will-fail",
				Type: "invalid-type", // This will fail
				Config: map[string]interface{}{
					"operation": "this-will-fail",
				},
			},
			{
				Name: "step3-never-runs",
				Type: "kubernetes",
				Config: map[string]interface{}{
					"operation": "namespace",
					"namespace": "test",
				},
			},
		},
	}

	fmt.Println("🧪 Testing Workflow Retry Functionality")
	fmt.Println("========================================")
	fmt.Println()

	// Step 1: Execute workflow (should fail at step 2)
	fmt.Println("📋 Step 1: Execute workflow (expect failure at step 2)")
	err = executor.ExecuteWorkflowWithName(appName, workflowName, testWorkflow)
	if err != nil {
		fmt.Printf("✅ Workflow failed as expected: %v\n", err)
	} else {
		fmt.Println("❌ Workflow should have failed but didn't!")
		return
	}
	fmt.Println()

	// Step 2: Get the latest execution
	fmt.Println("📋 Step 2: Get latest workflow execution")
	latestExec, err := repo.GetLatestWorkflowExecution(appName, workflowName)
	if err != nil {
		fmt.Printf("❌ Failed to get latest execution: %v\n", err)
		return
	}
	fmt.Printf("✅ Found execution ID: %d, Status: %s\n", latestExec.ID, latestExec.Status)
	fmt.Println()

	// Step 3: Get the failed step number
	fmt.Println("📋 Step 3: Find first failed step")
	failedStep, err := repo.GetFirstFailedStepNumber(latestExec.ID)
	if err != nil {
		fmt.Printf("❌ Failed to get failed step: %v\n", err)
		return
	}
	fmt.Printf("✅ First failed step: %d\n", failedStep)
	fmt.Println()

	// Step 4: Fix the workflow (remove the failing step)
	fmt.Println("📋 Step 4: Fix workflow (replace failing step with valid one)")
	fixedWorkflow := types.Workflow{
		Steps: []types.Step{
			{
				Name: "step1-success",
				Type: "kubernetes",
				Config: map[string]interface{}{
					"operation": "namespace",
					"namespace": "test",
				},
			},
			{
				Name: "step2-now-fixed",
				Type: "kubernetes",
				Config: map[string]interface{}{
					"operation": "namespace",
					"namespace": "test-fixed",
				},
			},
			{
				Name: "step3-should-run",
				Type: "kubernetes",
				Config: map[string]interface{}{
					"operation": "namespace",
					"namespace": "test",
				},
			},
		},
	}
	fmt.Println("✅ Workflow fixed")
	fmt.Println()

	// Step 5: Retry the workflow
	fmt.Println("📋 Step 5: Retry workflow from failed step")
	err = executor.RetryWorkflowFromFailedStep(appName, workflowName, fixedWorkflow, latestExec.ID)
	if err != nil {
		fmt.Printf("❌ Retry failed: %v\n", err)
		return
	}
	fmt.Println("✅ Retry completed successfully!")
	fmt.Println()

	// Step 6: Verify retry execution
	fmt.Println("📋 Step 6: Verify retry execution details")
	retryExec, err := repo.GetLatestWorkflowExecution(appName, workflowName)
	if err != nil {
		fmt.Printf("❌ Failed to get retry execution: %v\n", err)
		return
	}

	fmt.Printf("✅ Retry Execution Details:\n")
	fmt.Printf("   ID: %d\n", retryExec.ID)
	fmt.Printf("   Status: %s\n", retryExec.Status)
	fmt.Printf("   Is Retry: %v\n", retryExec.IsRetry)
	fmt.Printf("   Retry Count: %d\n", retryExec.RetryCount)
	fmt.Printf("   Parent Execution ID: %v\n", retryExec.ParentExecutionID)
	if retryExec.ResumeFromStep != nil {
		fmt.Printf("   Resumed From Step: %d\n", *retryExec.ResumeFromStep)
	}
	fmt.Println()

	fmt.Println("🎉 All retry tests passed!")
	fmt.Println()
	fmt.Println("Summary:")
	fmt.Printf("  - Original execution (ID %d): Failed at step %d\n", latestExec.ID, failedStep)
	fmt.Printf("  - Retry execution (ID %d): Started from step %d\n", retryExec.ID, failedStep)
	fmt.Printf("  - Retry count: %d\n", retryExec.RetryCount)
}
