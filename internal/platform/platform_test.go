package platform_test

import (
	"os"
	"path/filepath"
	"testing"

	"innominatus/internal/platform"
	"innominatus/pkg/sdk"
)

func TestLoaderLoadFromFile(t *testing.T) {
	// Create temporary platform.yaml
	tmpDir := t.TempDir()
	platformPath := filepath.Join(tmpDir, "platform.yaml")

	platformYAML := `apiVersion: innominatus.io/v1
kind: Platform
metadata:
  name: test-platform
  version: 1.0.0
  description: Test platform
compatibility:
  minCoreVersion: "1.0.0"
  maxCoreVersion: "2.0.0"
provisioners:
  - name: test-provisioner
    type: postgres
    version: 1.0.0
    description: Test provisioner
`

	if err := os.WriteFile(platformPath, []byte(platformYAML), 0644); err != nil {
		t.Fatalf("Failed to write test platform.yaml: %v", err)
	}

	// Load platform
	loader := platform.NewLoader("1.5.0")
	platform, err := loader.LoadFromFile(platformPath)
	if err != nil {
		t.Fatalf("Failed to load platform: %v", err)
	}

	// Verify platform
	if platform.Metadata.Name != "test-platform" {
		t.Errorf("Expected name='test-platform', got '%s'", platform.Metadata.Name)
	}

	if len(platform.Provisioners) != 1 {
		t.Errorf("Expected 1 provisioner, got %d", len(platform.Provisioners))
	}

	if platform.Provisioners[0].Type != "postgres" {
		t.Errorf("Expected provisioner type='postgres', got '%s'", platform.Provisioners[0].Type)
	}
}

func TestLoaderVersionCompatibility(t *testing.T) {
	tmpDir := t.TempDir()
	platformPath := filepath.Join(tmpDir, "platform.yaml")

	// Platform requires core 2.0.0-3.0.0
	platformYAML := `apiVersion: innominatus.io/v1
kind: Platform
metadata:
  name: future-platform
  version: 1.0.0
compatibility:
  minCoreVersion: "2.0.0"
  maxCoreVersion: "3.0.0"
provisioners:
  - name: test
    type: test
    version: 1.0.0
`

	if err := os.WriteFile(platformPath, []byte(platformYAML), 0644); err != nil {
		t.Fatalf("Failed to write test platform.yaml: %v", err)
	}

	// Test with core version 1.5.0 (too old)
	loader := platform.NewLoader("1.5.0")
	_, err := loader.LoadFromFile(platformPath)
	if err == nil {
		t.Error("Expected error for incompatible core version, got nil")
	}

	// Test with core version 2.5.0 (compatible)
	loader = platform.NewLoader("2.5.0")
	_, err = loader.LoadFromFile(platformPath)
	if err != nil {
		t.Errorf("Expected successful load for compatible version, got error: %v", err)
	}

	// Test with core version 4.0.0 (too new)
	loader = platform.NewLoader("4.0.0")
	_, err = loader.LoadFromFile(platformPath)
	if err == nil {
		t.Error("Expected error for incompatible core version (too new), got nil")
	}
}

func TestLoaderLoadFromDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	// Create multiple platform.yaml files in subdirectories
	platform1Dir := filepath.Join(tmpDir, "platform1")
	platform2Dir := filepath.Join(tmpDir, "platform2")

	if err := os.MkdirAll(platform1Dir, 0755); err != nil {
		t.Fatalf("Failed to create platform1 dir: %v", err)
	}
	if err := os.MkdirAll(platform2Dir, 0755); err != nil {
		t.Fatalf("Failed to create platform2 dir: %v", err)
	}

	platform1YAML := `apiVersion: innominatus.io/v1
kind: Platform
metadata:
  name: platform-one
  version: 1.0.0
compatibility:
  minCoreVersion: "1.0.0"
  maxCoreVersion: "2.0.0"
provisioners:
  - name: p1
    type: test1
    version: 1.0.0
`

	platform2YAML := `apiVersion: innominatus.io/v1
kind: Platform
metadata:
  name: platform-two
  version: 1.0.0
compatibility:
  minCoreVersion: "1.0.0"
  maxCoreVersion: "2.0.0"
provisioners:
  - name: p2
    type: test2
    version: 1.0.0
`

	if err := os.WriteFile(filepath.Join(platform1Dir, "platform.yaml"), []byte(platform1YAML), 0644); err != nil {
		t.Fatalf("Failed to write platform1.yaml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(platform2Dir, "platform.yaml"), []byte(platform2YAML), 0644); err != nil {
		t.Fatalf("Failed to write platform2.yaml: %v", err)
	}

	// Load all platforms
	loader := platform.NewLoader("1.5.0")
	platforms, err := loader.LoadFromDirectory(tmpDir)
	if err != nil {
		t.Fatalf("Failed to load platforms from directory: %v", err)
	}

	if len(platforms) != 2 {
		t.Errorf("Expected 2 platforms, got %d", len(platforms))
	}

	// Verify platform names
	names := make(map[string]bool)
	for _, p := range platforms {
		names[p.Metadata.Name] = true
	}

	if !names["platform-one"] {
		t.Error("Expected to find platform-one")
	}
	if !names["platform-two"] {
		t.Error("Expected to find platform-two")
	}
}

func TestRegistryRegisterPlatform(t *testing.T) {
	registry := platform.NewRegistry()

	platform := &sdk.Platform{
		APIVersion: "innominatus.io/v1",
		Kind:       "Platform",
		Metadata: sdk.PlatformMetadata{
			Name:    "test-platform",
			Version: "1.0.0",
		},
		Compatibility: sdk.PlatformCompatibility{
			MinCoreVersion: "1.0.0",
			MaxCoreVersion: "2.0.0",
		},
		Provisioners: []sdk.ProvisionerMetadata{
			{Name: "test", Type: "postgres", Version: "1.0.0"},
		},
	}

	// Register platform
	if err := registry.RegisterPlatform(platform); err != nil {
		t.Fatalf("Failed to register platform: %v", err)
	}

	// Verify registration
	retrieved, err := registry.GetPlatform("test-platform")
	if err != nil {
		t.Fatalf("Failed to get platform: %v", err)
	}

	if retrieved.Metadata.Name != "test-platform" {
		t.Errorf("Expected name='test-platform', got '%s'", retrieved.Metadata.Name)
	}

	// Try to register duplicate
	err = registry.RegisterPlatform(platform)
	if err == nil {
		t.Error("Expected error when registering duplicate platform, got nil")
	}
}

func TestRegistryListPlatforms(t *testing.T) {
	registry := platform.NewRegistry()

	// Register multiple platforms
	platforms := []*sdk.Platform{
		{
			APIVersion: "innominatus.io/v1",
			Kind:       "Platform",
			Metadata:   sdk.PlatformMetadata{Name: "platform1", Version: "1.0.0"},
			Compatibility: sdk.PlatformCompatibility{
				MinCoreVersion: "1.0.0",
				MaxCoreVersion: "2.0.0",
			},
			Provisioners: []sdk.ProvisionerMetadata{{Name: "p1", Type: "db", Version: "1.0.0"}},
		},
		{
			APIVersion: "innominatus.io/v1",
			Kind:       "Platform",
			Metadata:   sdk.PlatformMetadata{Name: "platform2", Version: "1.0.0"},
			Compatibility: sdk.PlatformCompatibility{
				MinCoreVersion: "1.0.0",
				MaxCoreVersion: "2.0.0",
			},
			Provisioners: []sdk.ProvisionerMetadata{{Name: "p2", Type: "cache", Version: "1.0.0"}},
		},
	}

	for _, p := range platforms {
		if err := registry.RegisterPlatform(p); err != nil {
			t.Fatalf("Failed to register platform: %v", err)
		}
	}

	// List platforms
	listed := registry.ListPlatforms()
	if len(listed) != 2 {
		t.Errorf("Expected 2 platforms, got %d", len(listed))
	}

	// Verify count
	platformCount, provCount := registry.Count()
	if platformCount != 2 {
		t.Errorf("Expected platform count=2, got %d", platformCount)
	}
	if provCount != 0 {
		t.Errorf("Expected provisioner count=0, got %d", provCount)
	}
}

func TestRegistryGetProvisionerTypes(t *testing.T) {
	registry := platform.NewRegistry()

	// Initially empty
	types := registry.GetProvisionerTypes()
	if len(types) != 0 {
		t.Errorf("Expected 0 provisioner types initially, got %d", len(types))
	}

	// HasProvisioner should return false
	if registry.HasProvisioner("postgres") {
		t.Error("Expected HasProvisioner('postgres') = false")
	}
}
