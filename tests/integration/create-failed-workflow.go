//go:build ignore
// +build ignore

package main

import (
	"fmt"
	"log"
	"os"

	"innominatus/internal/database"
	"innominatus/internal/types"
	"innominatus/internal/workflow"
)

func main() {
	// Connect to database
	db, err := database.NewDatabase()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	repo := database.NewWorkflowRepository(db)
	executor := workflow.NewWorkflowExecutor(repo)

	// Create a workflow that will fail at step 2
	appName := "test-app-with-failure"

	failingWorkflow := types.Workflow{
		Steps: []types.Step{
			{
				Name: "step1-success",
				Type: "kubernetes",
				Config: map[string]interface{}{
					"operation": "create-namespace",
					"namespace": "test-success",
				},
			},
			{
				Name: "step2-intentional-failure",
				Type: "invalid-type-that-does-not-exist",
				Config: map[string]interface{}{
					"this": "will-fail",
				},
			},
			{
				Name: "step3-never-executes",
				Type: "kubernetes",
				Config: map[string]interface{}{
					"operation": "create-namespace",
					"namespace": "test-never-runs",
				},
			},
		},
	}

	fmt.Printf("Creating failing workflow for app: %s\n", appName)

	// Execute the workflow (it will fail at step 2)
	err = executor.ExecuteWorkflow(appName, failingWorkflow)
	if err != nil {
		fmt.Printf("✓ Workflow failed as expected: %v\n", err)
		os.Exit(0)
	}

	fmt.Printf("✗ Workflow should have failed but didn't\n")
	os.Exit(1)
}
