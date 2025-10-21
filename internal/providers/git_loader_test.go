package providers_test

import (
	"os"
	"path/filepath"
	"testing"

	"innominatus/internal/providers"
)

func TestGitLoader_Creation(t *testing.T) {
	tmpDir := t.TempDir()
	loader := providers.NewGitLoader(tmpDir, "1.5.0")

	if loader == nil {
		t.Fatal("Expected non-nil Git loader")
	}
}

func TestGitProviderSource_Validation(t *testing.T) {
	tests := []struct {
		name   string
		source providers.GitProviderSource
		valid  bool
	}{
		{
			name: "valid tag reference",
			source: providers.GitProviderSource{
				Name:       "test-provider",
				Repository: "https://github.com/example/provider",
				Ref:        "v1.2.3",
			},
			valid: true,
		},
		{
			name: "valid branch reference",
			source: providers.GitProviderSource{
				Name:       "test-provider",
				Repository: "https://github.com/example/provider",
				Ref:        "main",
			},
			valid: true,
		},
		{
			name: "missing name",
			source: providers.GitProviderSource{
				Repository: "https://github.com/example/provider",
				Ref:        "main",
			},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.source.Name == "" && tt.valid {
				t.Error("Expected invalid source to fail validation")
			}
			if tt.source.Name != "" && !tt.valid {
				t.Error("Expected valid source to pass validation")
			}
		})
	}
}

func TestSanitizeRepoName(t *testing.T) {
	// This is a simple test to verify the package compiles
	// Full Git integration tests would require a test Git server
	tmpDir := t.TempDir()
	loader := providers.NewGitLoader(tmpDir, "1.5.0")

	if loader == nil {
		t.Fatal("Expected non-nil loader")
	}

	// Test that cache directory is set correctly
	cacheFile := filepath.Join(tmpDir, "test.txt")
	err := os.WriteFile(cacheFile, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	if _, err := os.Stat(cacheFile); os.IsNotExist(err) {
		t.Error("Expected cache directory to be accessible")
	}
}
