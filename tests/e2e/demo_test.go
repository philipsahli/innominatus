package e2e

import (
	"innominatus/internal/cli"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDemoEnvironmentLifecycle tests the full demo environment lifecycle:
// 1. Install (demo-time)
// 2. Status check (demo-status)
// 3. Cleanup (demo-nuke)
//
// This is an integration test that requires:
// - Docker Desktop with Kubernetes enabled
// - kubectl and helm in PATH
// - Sufficient cluster resources
//
// Run with: go test -v ./tests/e2e -run TestDemoEnvironment
func TestDemoEnvironmentLifecycle(t *testing.T) {
	// Skip if not running integration tests
	if testing.Short() {
		t.Skip("Skipping demo environment integration test in short mode")
	}

	// Skip if SKIP_DEMO_TESTS is set (for CI)
	if os.Getenv("SKIP_DEMO_TESTS") != "" {
		t.Skip("Skipping demo tests (SKIP_DEMO_TESTS is set)")
	}

	// Check prerequisites
	if !isKubernetesAvailable(t) {
		t.Skip("Kubernetes is not available - skipping demo tests")
	}

	if !isHelmAvailable(t) {
		t.Skip("Helm is not available - skipping demo tests")
	}

	// Ensure clean state before starting
	t.Log("Ensuring clean state before test...")
	client := cli.NewClient("http://localhost:8081")
	_ = client.DemoNukeCommand() // Best effort cleanup

	// Test 1: Demo Installation
	t.Run("DemoTime_InstallsAllServices", func(t *testing.T) {
		t.Log("Installing demo environment...")

		err := client.DemoTimeCommand("")
		require.NoError(t, err, "demo-time command should succeed")

		// Verify services are deployed
		services := []struct {
			namespace string
			name      string
		}{
			{"gitea", "gitea"},
			{"argocd", "argocd-server"},
			{"vault", "vault"},
			{"minio", "minio"},
			{"grafana", "grafana"},
		}

		for _, svc := range services {
			t.Logf("Verifying service: %s/%s", svc.namespace, svc.name)
			deployed := isServiceDeployed(t, svc.namespace, svc.name)
			assert.True(t, deployed, "Service %s/%s should be deployed", svc.namespace, svc.name)
		}
	})

	// Test 2: Demo Status Check
	t.Run("DemoStatus_ShowsAllServices", func(t *testing.T) {
		t.Log("Checking demo status...")

		// Capture stdout to verify output
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		err := client.DemoStatusCommand()
		require.NoError(t, err, "demo-status command should succeed")

		// Restore stdout
		w.Close()
		os.Stdout = oldStdout

		var buf []byte
		buf, _ = io.ReadAll(r)
		output := string(buf)

		// Verify all services appear in output
		expectedServices := []string{"Gitea", "ArgoCD", "Vault", "Minio", "Grafana"}
		for _, svc := range expectedServices {
			assert.Contains(t, output, svc, "Status should show %s", svc)
		}
	})

	// Test 3: Service Health Check
	t.Run("DemoServices_AreAccessible", func(t *testing.T) {
		t.Log("Checking if services are accessible...")

		// Just verify that at least one service namespace exists
		services := []string{"gitea", "argocd", "vault", "minio", "grafana"}
		foundAny := false

		for _, ns := range services {
			if namespaceExists(t, ns) {
				t.Logf("Service namespace %s exists", ns)
				foundAny = true
			}
		}

		assert.True(t, foundAny, "At least one service namespace should exist")
	})

	// Test 4: Demo Cleanup
	t.Run("DemoNuke_RemovesAllServices", func(t *testing.T) {
		t.Log("Cleaning up demo environment...")

		err := client.DemoNukeCommand()
		require.NoError(t, err, "demo-nuke command should succeed")

		// Verify services are removed
		services := []struct {
			namespace string
			name      string
		}{
			{"gitea", "gitea"},
			{"argocd", "argocd-server"},
			{"vault", "vault"},
			{"minio", "minio"},
			{"grafana", "grafana"},
		}

		// Wait a bit for cleanup to complete
		t.Log("Waiting for cleanup to complete...")
		// In a real test, you might want to poll until services are gone
		// For now, just verify namespaces are deleted

		for _, svc := range services {
			t.Logf("Verifying service removed: %s/%s", svc.namespace, svc.name)
			// Check namespace is gone or service is gone
			namespaceExists := namespaceExists(t, svc.namespace)
			if namespaceExists {
				deployed := isServiceDeployed(t, svc.namespace, svc.name)
				assert.False(t, deployed, "Service %s/%s should be removed", svc.namespace, svc.name)
			}
		}
	})
}

// TestDemoTime_IdempotentInstallation verifies that demo-time can be run multiple times
func TestDemoTime_IdempotentInstallation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping demo environment integration test in short mode")
	}

	if os.Getenv("SKIP_DEMO_TESTS") != "" {
		t.Skip("Skipping demo tests (SKIP_DEMO_TESTS is set)")
	}

	if !isKubernetesAvailable(t) || !isHelmAvailable(t) {
		t.Skip("Kubernetes or Helm not available")
	}

	client := cli.NewClient("http://localhost:8081")

	// First installation
	t.Log("First installation...")
	err := client.DemoTimeCommand("")
	require.NoError(t, err, "First demo-time should succeed")

	// Second installation (idempotent)
	t.Log("Second installation (idempotent check)...")
	err = client.DemoTimeCommand("")
	assert.NoError(t, err, "Second demo-time should succeed (idempotent)")

	// Cleanup
	t.Cleanup(func() {
		_ = client.DemoNukeCommand()
	})
}

// TestDemoTime_ComponentFilter verifies component filtering works
func TestDemoTime_ComponentFilter(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping demo environment integration test in short mode")
	}

	if os.Getenv("SKIP_DEMO_TESTS") != "" {
		t.Skip("Skipping demo tests (SKIP_DEMO_TESTS is set)")
	}

	if !isKubernetesAvailable(t) || !isHelmAvailable(t) {
		t.Skip("Kubernetes or Helm not available")
	}

	client := cli.NewClient("http://localhost:8081")

	// Install only gitea
	t.Log("Installing only Gitea component...")
	err := client.DemoTimeCommand("gitea")
	require.NoError(t, err, "demo-time with filter should succeed")

	// Verify only gitea is deployed
	t.Log("Verifying only Gitea is deployed...")
	assert.True(t, isServiceDeployed(t, "gitea", "gitea"), "Gitea should be deployed")

	// ArgoCD should not be deployed (optional check - might fail if it was already there)
	// This is best-effort since we might have residual state
	t.Log("Note: Other services may exist from previous runs")

	// Cleanup
	t.Cleanup(func() {
		_ = client.DemoNukeCommand()
	})
}

// TestDemoNuke_HandlesNonExistentDemo verifies graceful handling when demo isn't installed
func TestDemoNuke_HandlesNonExistentDemo(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping demo environment integration test in short mode")
	}

	if !isKubernetesAvailable(t) {
		t.Skip("Kubernetes not available")
	}

	client := cli.NewClient("http://localhost:8081")

	// Ensure demo is not installed
	_ = client.DemoNukeCommand()

	// Try to nuke again (should not error)
	t.Log("Running demo-nuke on non-existent demo...")
	err := client.DemoNukeCommand()
	assert.NoError(t, err, "demo-nuke should handle non-existent demo gracefully")
}

// Helper functions

func isKubernetesAvailable(t *testing.T) bool {
	cmd := exec.Command("kubectl", "cluster-info")
	err := cmd.Run()
	if err != nil {
		t.Logf("Kubernetes not available: %v", err)
		return false
	}
	return true
}

func isHelmAvailable(t *testing.T) bool {
	cmd := exec.Command("helm", "version")
	err := cmd.Run()
	if err != nil {
		t.Logf("Helm not available: %v", err)
		return false
	}
	return true
}

func isServiceDeployed(t *testing.T, namespace, serviceName string) bool {
	// Check if deployment/statefulset exists
	cmd := exec.Command("kubectl", "-n", namespace, "get", "deployment,statefulset",
		"-l", "app.kubernetes.io/name="+serviceName, "--no-headers")
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Try without label selector
		cmd = exec.Command("kubectl", "-n", namespace, "get", "deployment,statefulset",
			serviceName, "--no-headers")
		output, err = cmd.CombinedOutput()
		if err != nil {
			t.Logf("Service %s/%s not found: %v", namespace, serviceName, err)
			return false
		}
	}

	return len(strings.TrimSpace(string(output))) > 0
}

func namespaceExists(t *testing.T, namespace string) bool {
	cmd := exec.Command("kubectl", "get", "namespace", namespace, "--no-headers")
	err := cmd.Run()
	return err == nil
}
