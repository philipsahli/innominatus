package resources

import (
	"innominatus/internal/database"
	"innominatus/internal/types"
	"os/exec"
	"strings"
	"testing"
)

func TestKubernetesProvisionerIntegration(t *testing.T) {

	t.Skip("Skipping testing")

	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Create a Score spec with environment variables
	scoreSpec := &types.ScoreSpec{
		Metadata: types.Metadata{
			Name: "alice-nginx-excessive",
		},
		Containers: map[string]types.Container{
			"web": {
				Image: "nginx:1.25",
				Variables: map[string]string{
					"S3_BUCKET_ENDPOINT": "http://minio.minio-system.svc.cluster.local:9000",
					"S3_BUCKET_NAME":     "alice-nginx-excessive-storage",
					"S3_ACCESS_KEY":      "minioadmin",
					"S3_SECRET_KEY":      "minioadmin",
					"S3_REGION":          "us-east-1",
				},
			},
		},
	}

	// Create resource instance
	resource := &database.ResourceInstance{
		ApplicationName: "alice-nginx-excessive",
		ResourceName:    "alice-nginx-excessive-default",
		ResourceType:    "kubernetes",
	}

	// Create provisioner
	kp := NewKubernetesProvisioner(&database.ResourceRepository{})

	// Provision
	config := map[string]interface{}{
		"score_spec": scoreSpec,
	}

	t.Log("Provisioning alice-nginx-excessive with environment variables...")
	if err := kp.Provision(resource, config, "integration-test"); err != nil {
		t.Fatalf("Failed to provision: %v", err)
	}

	// Verify deployment has environment variables
	t.Log("Verifying environment variables in deployment...")
	cmd := exec.Command("kubectl", "get", "deployment", "alice-nginx-excessive",
		"-n", "alice-nginx-excessive-default", "-o", "yaml")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to get deployment: %v", err)
	}

	manifestStr := string(output)

	// Check for env section
	if !strings.Contains(manifestStr, "env:") {
		t.Error("Deployment manifest does not contain env section")
	}

	// Check for specific environment variables
	expectedVars := []string{
		"S3_BUCKET_ENDPOINT",
		"S3_BUCKET_NAME",
		"S3_ACCESS_KEY",
		"S3_SECRET_KEY",
		"S3_REGION",
	}

	for _, envVar := range expectedVars {
		if !strings.Contains(manifestStr, envVar) {
			t.Errorf("Deployment manifest does not contain environment variable: %s", envVar)
		}
	}

	// Verify in running pod
	t.Log("Verifying environment variables in running pod...")
	cmd = exec.Command("kubectl", "wait", "--for=condition=ready",
		"pod", "-l", "app=alice-nginx-excessive",
		"-n", "alice-nginx-excessive-default", "--timeout=60s")
	if err := cmd.Run(); err != nil {
		t.Logf("Warning: Pod not ready: %v", err)
	}

	// Get pod name
	cmd = exec.Command("kubectl", "get", "pods",
		"-n", "alice-nginx-excessive-default",
		"-l", "app=alice-nginx-excessive",
		"-o", "jsonpath={.items[0].metadata.name}")
	podName, err := cmd.Output()
	if err != nil {
		t.Fatalf("Failed to get pod name: %v", err)
	}

	// Check environment variables in pod
	cmd = exec.Command("kubectl", "exec",
		"-n", "alice-nginx-excessive-default",
		string(podName), "--", "env")
	envOutput, err := cmd.Output()
	if err != nil {
		t.Fatalf("Failed to get pod environment: %v", err)
	}

	envStr := string(envOutput)
	for _, envVar := range expectedVars {
		if !strings.Contains(envStr, envVar) {
			t.Errorf("Pod environment does not contain: %s", envVar)
		}
	}

	t.Log("✅ Integration test passed - environment variables present in deployment and pod")

	// Cleanup
	t.Log("Cleaning up...")
	if err := kp.Deprovision(resource); err != nil {
		t.Logf("Warning: Cleanup failed: %v", err)
	}
}
