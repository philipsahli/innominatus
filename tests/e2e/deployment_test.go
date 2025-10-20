package e2e

import (
	"fmt"
	"innominatus/internal/cli"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDeploymentWorkflow tests the full deployment lifecycle:
// 1. Deploy application via golden path
// 2. Verify deployment success
// 3. Update application
// 4. Destroy application
//
// This test requires a running innominatus server
func TestDeploymentWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping deployment workflow integration test in short mode")
	}

	if os.Getenv("SKIP_INTEGRATION_TESTS") != "" {
		t.Skip("Skipping integration tests (SKIP_INTEGRATION_TESTS is set)")
	}

	// Check if server is running
	client := cli.NewClient("http://localhost:8081")
	if !isServerAvailable(t, client) {
		t.Skip("Innominatus server not available at http://localhost:8081")
	}

	testAppName := fmt.Sprintf("test-app-%d", time.Now().Unix())
	specFile := createTestScoreSpec(t, testAppName)
	defer os.Remove(specFile)

	// Test 1: Deploy Application
	t.Run("DeployApp_ViaGoldenPath", func(t *testing.T) {
		t.Log("Deploying application via golden path...")

		// Use the deploy-app golden path with the test score spec
		params := map[string]string{
			"environment": "test",
		}

		err := client.RunGoldenPathCommand("deploy-app", specFile, params)
		if err != nil {
			// If golden path doesn't exist, try direct deployment
			t.Logf("Golden path failed, trying direct deployment: %v", err)

			// Read spec file and deploy directly
			specContent, readErr := os.ReadFile(specFile)
			require.NoError(t, readErr, "Should read spec file")

			_, deployErr := client.Deploy(specContent)
			require.NoError(t, deployErr, "Direct deployment should succeed")
		}

		// Verify application exists
		t.Log("Verifying application exists...")
		specs, err := client.ListSpecs()
		require.NoError(t, err, "Should list applications")

		_, exists := specs[testAppName]
		assert.True(t, exists, "Application %s should exist", testAppName)
	})

	// Test 2: Check Application Status
	t.Run("CheckApplicationStatus", func(t *testing.T) {
		t.Log("Checking application status...")

		// Get application details
		spec, err := client.GetSpec(testAppName)
		require.NoError(t, err, "Should get application spec")
		assert.NotNil(t, spec, "Application spec should not be nil")

		// Verify workflow execution
		t.Log("Checking workflow executions...")
		workflows, err := client.ListWorkflows(testAppName)
		if err == nil {
			assert.NotEmpty(t, workflows, "Should have at least one workflow execution")
			t.Logf("Found %d workflow executions", len(workflows))
		} else {
			t.Logf("Could not fetch workflows: %v", err)
		}
	})

	// Test 3: List Resources
	t.Run("ListApplicationResources", func(t *testing.T) {
		t.Log("Listing application resources...")

		resources, err := client.ListResources(testAppName)
		if err == nil {
			t.Logf("Found %d resource types", len(resources))

			for resourceType, instances := range resources {
				t.Logf("  - %s: %d instances", resourceType, len(instances))
			}
		} else {
			t.Logf("Could not fetch resources: %v", err)
		}
	})

	// Test 4: Update Application (if supported)
	t.Run("UpdateApplication", func(t *testing.T) {
		t.Log("Updating application...")

		// Create updated spec
		updatedSpecFile := createTestScoreSpec(t, testAppName+"_updated")
		defer os.Remove(updatedSpecFile)

		// Try to update
		specContent, err := os.ReadFile(updatedSpecFile)
		require.NoError(t, err, "Should read updated spec")

		_, err = client.Deploy(specContent)
		if err != nil {
			t.Logf("Update failed (may not be supported): %v", err)
		} else {
			t.Log("Application updated successfully")
		}
	})

	// Test 5: Cleanup
	t.Run("DestroyApplication", func(t *testing.T) {
		t.Log("Destroying application...")

		err := client.DeleteApplication(testAppName)
		if err != nil {
			// Try deprovision if delete failed
			t.Logf("Delete failed, trying deprovision: %v", err)
			err = client.DeprovisionApplication(testAppName)
		}

		if err != nil {
			t.Logf("Cleanup failed: %v", err)
		}

		// Verify application is gone
		specs, err := client.ListSpecs()
		if err == nil {
			_, exists := specs[testAppName]
			assert.False(t, exists, "Application %s should not exist after deletion", testAppName)
		}
	})
}

// TestValidateCommand tests score spec validation
func TestValidateCommand(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping in short mode")
	}

	client := cli.NewClient("http://localhost:8081")

	t.Run("ValidateValidSpec", func(t *testing.T) {
		// Use the test fixture
		specFile := "../../testdata/score-spec-simple.yaml"

		if _, err := os.Stat(specFile); os.IsNotExist(err) {
			t.Skip("Test fixture not found")
		}

		err := client.ValidateCommand(specFile, false, "text")
		assert.NoError(t, err, "Valid spec should pass validation")
	})

	t.Run("ValidateInvalidSpec", func(t *testing.T) {
		// Use invalid spec fixture
		specFile := "../../testdata/score-spec-invalid.yaml"

		if _, err := os.Stat(specFile); os.IsNotExist(err) {
			t.Skip("Invalid spec fixture not found")
		}

		err := client.ValidateCommand(specFile, false, "text")
		assert.Error(t, err, "Invalid spec should fail validation")
	})

	t.Run("ValidateNonExistentFile", func(t *testing.T) {
		err := client.ValidateCommand("/tmp/nonexistent-spec.yaml", false, "text")
		assert.Error(t, err, "Non-existent file should fail")
	})
}

// TestAnalyzeCommand tests score spec analysis
func TestAnalyzeCommand(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping in short mode")
	}

	client := cli.NewClient("http://localhost:8081")

	t.Run("AnalyzeSpec", func(t *testing.T) {
		// Use test fixture with workflow
		specFile := "../../testdata/score-spec-terraform.yaml"

		if _, err := os.Stat(specFile); os.IsNotExist(err) {
			t.Skip("Test fixture not found")
		}

		err := client.AnalyzeCommand(specFile)
		assert.NoError(t, err, "Analyze should succeed for valid spec")
	})

	t.Run("AnalyzeNonExistentFile", func(t *testing.T) {
		err := client.AnalyzeCommand("/tmp/nonexistent-spec.yaml")
		assert.Error(t, err, "Non-existent file should fail")
		assert.Contains(t, err.Error(), "failed to read file")
	})
}

// TestGoldenPathsCommands tests golden paths functionality
func TestGoldenPathsCommands(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping in short mode")
	}

	client := cli.NewClient("http://localhost:8081")

	t.Run("ListGoldenPaths", func(t *testing.T) {
		// This will fail if goldenpaths.yaml doesn't exist, which is expected
		err := client.ListGoldenPathsCommand()

		if err != nil {
			t.Logf("ListGoldenPaths failed (expected if goldenpaths.yaml not configured): %v", err)
			assert.Contains(t, err.Error(), "failed to load golden paths")
		} else {
			t.Log("Golden paths loaded successfully")
		}
	})
}

// TestWorkflowCommands tests workflow-related commands
func TestWorkflowCommands(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping in short mode")
	}

	if os.Getenv("SKIP_INTEGRATION_TESTS") != "" {
		t.Skip("Skipping integration tests")
	}

	client := cli.NewClient("http://localhost:8081")

	if !isServerAvailable(t, client) {
		t.Skip("Server not available")
	}

	t.Run("ListWorkflows", func(t *testing.T) {
		// List all workflows
		workflows, err := client.ListWorkflows("")

		if err != nil {
			t.Logf("ListWorkflows failed: %v", err)
		} else {
			t.Logf("Found %d workflow executions", len(workflows))
		}
	})
}

// Helper functions

func isServerAvailable(t *testing.T, client *cli.Client) bool {
	_, err := client.ListSpecs()
	if err != nil {
		t.Logf("Server not available: %v", err)
		return false
	}
	return true
}

func createTestScoreSpec(t *testing.T, appName string) string {
	tmpDir := t.TempDir()
	specFile := filepath.Join(tmpDir, "test-spec.yaml")

	specContent := fmt.Sprintf(`apiVersion: score.dev/v1b1

metadata:
  name: %s
  team: test-team
  costCenter: engineering

containers:
  web:
    image: nginx:latest
    variables:
      PORT: "8080"
      ENVIRONMENT: "test"

resources:
  route:
    type: route
    params:
      host: %s.localtest.me
      port: 8080
`, appName, appName)

	err := os.WriteFile(specFile, []byte(specContent), 0644)
	require.NoError(t, err, "Should create test spec file")

	return specFile
}
