package demo

import (
	"fmt"
	"os/exec"
	"strings"
)

// DemoReset handles resetting the demo environment database
type DemoReset struct {
	kubeContext string
}

// NewDemoReset creates a new demo reset instance
func NewDemoReset(kubeContext string) *DemoReset {
	return &DemoReset{
		kubeContext: kubeContext,
	}
}

// CheckDemoInstalled verifies that demo-time has been run by checking for demo namespaces
func (r *DemoReset) CheckDemoInstalled() (bool, error) {
	// Check if any of the key demo namespaces exist
	demoNamespaces := []string{
		"gitea",
		"argocd",
		"vault",
		"minio-system",
	}

	// #nosec G204 -- kubectl command with controlled arguments
	cmd := exec.Command("kubectl", "--context", r.kubeContext, "get", "namespaces", "-o", "json")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("failed to list namespaces: %w\nOutput: %s", err, string(output))
	}

	// Check if at least 2 demo namespaces exist
	found := 0
	for _, ns := range demoNamespaces {
		if strings.Contains(string(output), fmt.Sprintf(`"namespace":"%s"`, ns)) ||
			strings.Contains(string(output), fmt.Sprintf(`"name":"%s"`, ns)) {
			found++
		}
	}

	return found >= 2, nil
}

// ResetStatistics contains statistics about what was reset
type ResetStatistics struct {
	TablesTruncated int
	TasksStopped    int
}

// String returns a formatted string of reset statistics
func (s *ResetStatistics) String() string {
	return fmt.Sprintf("Tables truncated: %d, Tasks stopped: %d", s.TablesTruncated, s.TasksStopped)
}
