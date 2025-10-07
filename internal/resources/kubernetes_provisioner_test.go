package resources

import (
	"innominatus/internal/types"
	"strings"
	"testing"
)

func TestGenerateDeploymentWithEnvironmentVariables(t *testing.T) {
	kp := &KubernetesProvisioner{}

	scoreSpec := &types.ScoreSpec{
		Metadata: types.Metadata{
			Name: "test-app",
		},
		Containers: map[string]types.Container{
			"web": {
				Image: "nginx:1.25",
				Variables: map[string]string{
					"S3_BUCKET_ENDPOINT": "http://minio.minio-system.svc.cluster.local:9000",
					"S3_BUCKET_NAME":     "test-storage",
					"S3_ACCESS_KEY":      "minioadmin",
				},
			},
		},
	}

	manifest := kp.generateDeployment("test-app", "test-namespace", scoreSpec)

	// Check that env section exists
	if !strings.Contains(manifest, "env:") {
		t.Error("Expected manifest to contain 'env:' section")
	}

	// Check that all environment variables are present
	expectedVars := []string{
		"S3_BUCKET_ENDPOINT",
		"http://minio.minio-system.svc.cluster.local:9000",
		"S3_BUCKET_NAME",
		"test-storage",
		"S3_ACCESS_KEY",
		"minioadmin",
	}

	for _, expected := range expectedVars {
		if !strings.Contains(manifest, expected) {
			t.Errorf("Expected manifest to contain '%s'", expected)
		}
	}

	// Print manifest for visual inspection
	t.Logf("Generated manifest:\n%s", manifest)
}

func TestGenerateDeploymentWithoutEnvironmentVariables(t *testing.T) {
	kp := &KubernetesProvisioner{}

	scoreSpec := &types.ScoreSpec{
		Metadata: types.Metadata{
			Name: "test-app",
		},
		Containers: map[string]types.Container{
			"web": {
				Image: "nginx:1.25",
				// No variables
			},
		},
	}

	manifest := kp.generateDeployment("test-app", "test-namespace", scoreSpec)

	// Check that env section does NOT exist when no variables
	if strings.Contains(manifest, "env:") {
		t.Error("Expected manifest to NOT contain 'env:' section when no variables defined")
	}

	// Print manifest for visual inspection
	t.Logf("Generated manifest:\n%s", manifest)
}

func TestGenerateEnvSection(t *testing.T) {
	kp := &KubernetesProvisioner{}

	tests := []struct {
		name      string
		variables map[string]string
		wantEmpty bool
	}{
		{
			name:      "empty variables",
			variables: map[string]string{},
			wantEmpty: true,
		},
		{
			name:      "nil variables",
			variables: nil,
			wantEmpty: true,
		},
		{
			name: "single variable",
			variables: map[string]string{
				"TEST_VAR": "test_value",
			},
			wantEmpty: false,
		},
		{
			name: "multiple variables",
			variables: map[string]string{
				"VAR1": "value1",
				"VAR2": "value2",
				"VAR3": "value3",
			},
			wantEmpty: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := kp.generateEnvSection(tt.variables)

			if tt.wantEmpty {
				if result != "" {
					t.Errorf("Expected empty result, got: %s", result)
				}
			} else {
				if result == "" {
					t.Error("Expected non-empty result")
				}
				if !strings.Contains(result, "env:") {
					t.Error("Expected result to contain 'env:'")
				}
				for key, value := range tt.variables {
					if !strings.Contains(result, key) {
						t.Errorf("Expected result to contain key '%s'", key)
					}
					if !strings.Contains(result, value) {
						t.Errorf("Expected result to contain value '%s'", value)
					}
				}
			}

			t.Logf("Result:\n%s", result)
		})
	}
}
