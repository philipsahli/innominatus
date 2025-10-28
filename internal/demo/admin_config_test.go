package demo

import (
	"os"
	"strings"
	"testing"
)

func TestCreateAdminConfig(t *testing.T) {
	// Create a temporary file
	tmpFile := "/tmp/test-admin-config-" + t.Name() + ".yaml"
	defer func() {
		_ = os.Remove(tmpFile)
	}()

	// Test admin config creation
	err := CreateAdminConfig(tmpFile)
	if err != nil {
		t.Fatalf("CreateAdminConfig failed: %v", err)
	}

	// Read the file to verify it was created
	content, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("Failed to read created file: %v", err)
	}

	contentStr := string(content)

	// Verify key sections exist
	expectedSections := []string{
		"# Innominatus Admin Configuration",
		"admin:",
		"providers:",
		"builtin",
		"type: filesystem",
		"path: ./providers/builtin",
		"resourceDefinitions:",
		"postgres:",
		"redis:",
		"policies:",
		"workflowPolicies:",
		"gitea:",
		"argocd:",
		"vault:",
	}

	for _, section := range expectedSections {
		if !strings.Contains(contentStr, section) {
			t.Errorf("Expected section '%s' not found in config", section)
		}
	}

	// Verify file size is reasonable (should be > 1KB)
	if len(content) < 1000 {
		t.Errorf("Config file seems too small: %d bytes", len(content))
	}

	t.Logf("✅ Admin config created successfully (%d bytes)", len(content))
}
