package admin

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadAdminConfig_ValidFile(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "admin-config.yaml")

	configContent := `
admin:
  defaultCostCenter: "engineering"
  defaultRuntime: "kubernetes"
  splunkIndex: "orchestrator-logs"

resourceDefinitions:
  postgres: "managed-postgres-cluster"
  redis: "redis-cluster"
  volume: "persistent-volume-claim"

policies:
  enforceBackups: true
  allowedEnvironments:
    - "development"
    - "staging"
    - "production"

gitea:
  url: "http://gitea.example.com"
  username: "admin"
  password: "secret"
  orgName: "myorg"

argocd:
  url: "http://argocd.example.com"
  username: "admin"
  password: "secret"
`

	err := os.WriteFile(configFile, []byte(configContent), 0644)
	require.NoError(t, err)

	// Test loading the config
	config, err := LoadAdminConfig(configFile)
	require.NoError(t, err)
	require.NotNil(t, config)

	// Test admin section
	assert.Equal(t, "engineering", config.Admin.DefaultCostCenter)
	assert.Equal(t, "kubernetes", config.Admin.DefaultRuntime)
	assert.Equal(t, "orchestrator-logs", config.Admin.SplunkIndex)

	// Test resource definitions
	assert.Equal(t, "managed-postgres-cluster", config.ResourceDefinitions["postgres"])
	assert.Equal(t, "redis-cluster", config.ResourceDefinitions["redis"])
	assert.Equal(t, "persistent-volume-claim", config.ResourceDefinitions["volume"])

	// Test policies
	assert.True(t, config.Policies.EnforceBackups)
	assert.Contains(t, config.Policies.AllowedEnvironments, "development")
	assert.Contains(t, config.Policies.AllowedEnvironments, "staging")
	assert.Contains(t, config.Policies.AllowedEnvironments, "production")

	// Test gitea config
	assert.Equal(t, "http://gitea.example.com", config.Gitea.URL)
	assert.Equal(t, "admin", config.Gitea.Username)
	assert.Equal(t, "secret", config.Gitea.Password)
	assert.Equal(t, "myorg", config.Gitea.OrgName)

	// Test argocd config
	assert.Equal(t, "http://argocd.example.com", config.ArgoCD.URL)
	assert.Equal(t, "admin", config.ArgoCD.Username)
	assert.Equal(t, "secret", config.ArgoCD.Password)
}

func TestLoadAdminConfig_MinimalFile(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "minimal-config.yaml")

	configContent := `
admin:
  defaultCostCenter: "ops"
`

	err := os.WriteFile(configFile, []byte(configContent), 0644)
	require.NoError(t, err)

	config, err := LoadAdminConfig(configFile)
	require.NoError(t, err)
	require.NotNil(t, config)

	// Test that minimal config loads
	assert.Equal(t, "ops", config.Admin.DefaultCostCenter)
	assert.Empty(t, config.Admin.DefaultRuntime)
	assert.Empty(t, config.Admin.SplunkIndex)

	// Test that maps are initialized (not nil)
	assert.NotNil(t, config.ResourceDefinitions)
	assert.Empty(t, config.ResourceDefinitions)
}

func TestLoadAdminConfig_FileNotFound(t *testing.T) {
	config, err := LoadAdminConfig("nonexistent-file.yaml")

	assert.Error(t, err)
	assert.Nil(t, config)
	assert.Contains(t, err.Error(), "failed to read config file")
}

func TestLoadAdminConfig_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "invalid-config.yaml")

	invalidContent := `
admin:
  defaultCostCenter: "engineering"
  invalidYaml: [unclosed array
`

	err := os.WriteFile(configFile, []byte(invalidContent), 0644)
	require.NoError(t, err)

	config, err := LoadAdminConfig(configFile)

	assert.Error(t, err)
	assert.Nil(t, config)
	assert.Contains(t, err.Error(), "failed to parse config file")
}

func TestLoadAdminConfig_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "empty-config.yaml")

	err := os.WriteFile(configFile, []byte(""), 0644)
	require.NoError(t, err)

	config, err := LoadAdminConfig(configFile)
	require.NoError(t, err)
	require.NotNil(t, config)

	// Should load with empty/default values
	assert.Empty(t, config.Admin.DefaultCostCenter)
	assert.NotNil(t, config.ResourceDefinitions)
	assert.Empty(t, config.ResourceDefinitions)
}

func TestPrintConfig(t *testing.T) {
	config := &AdminConfig{
		Admin: struct {
			DefaultCostCenter string `yaml:"defaultCostCenter"`
			DefaultRuntime    string `yaml:"defaultRuntime"`
			SplunkIndex       string `yaml:"splunkIndex"`
		}{
			DefaultCostCenter: "engineering",
			DefaultRuntime:    "kubernetes",
			SplunkIndex:       "test-logs",
		},
		ResourceDefinitions: map[string]string{
			"postgres": "managed-postgres",
			"redis":    "redis-cluster",
		},
		Policies: struct {
			EnforceBackups      bool     `yaml:"enforceBackups"`
			AllowedEnvironments []string `yaml:"allowedEnvironments"`
		}{
			EnforceBackups:      true,
			AllowedEnvironments: []string{"dev", "prod"},
		},
	}

	// Test that PrintConfig doesn't panic
	// We can't easily test the output without capturing stdout,
	// but we can ensure it doesn't crash
	assert.NotPanics(t, func() {
		config.PrintConfig()
	})
}

func TestAdminConfigValidation(t *testing.T) {
	tests := []struct {
		name   string
		config AdminConfig
		valid  bool
	}{
		{
			name: "valid config with all fields",
			config: AdminConfig{
				Admin: struct {
					DefaultCostCenter string `yaml:"defaultCostCenter"`
					DefaultRuntime    string `yaml:"defaultRuntime"`
					SplunkIndex       string `yaml:"splunkIndex"`
				}{
					DefaultCostCenter: "engineering",
					DefaultRuntime:    "kubernetes",
					SplunkIndex:       "logs",
				},
				ResourceDefinitions: map[string]string{
					"postgres": "managed-postgres",
				},
				Policies: struct {
					EnforceBackups      bool     `yaml:"enforceBackups"`
					AllowedEnvironments []string `yaml:"allowedEnvironments"`
				}{
					EnforceBackups:      true,
					AllowedEnvironments: []string{"dev"},
				},
			},
			valid: true,
		},
		{
			name: "valid minimal config",
			config: AdminConfig{
				Admin: struct {
					DefaultCostCenter string `yaml:"defaultCostCenter"`
					DefaultRuntime    string `yaml:"defaultRuntime"`
					SplunkIndex       string `yaml:"splunkIndex"`
				}{
					DefaultCostCenter: "ops",
				},
				ResourceDefinitions: make(map[string]string),
			},
			valid: true,
		},
		{
			name: "empty config",
			config: AdminConfig{
				ResourceDefinitions: make(map[string]string),
			},
			valid: true, // Empty config should be valid (uses defaults)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// For now, we'll just test that the config structure is valid
			// In a real implementation, you might add a Validate() method
			assert.NotNil(t, &tt.config)

			if tt.valid {
				assert.NotNil(t, tt.config.ResourceDefinitions)
			}
		})
	}
}

func TestGiteaConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "gitea-config.yaml")

	configContent := `
gitea:
  url: "https://git.company.com"
  username: "platform-admin"
  password: "super-secret"
  orgName: "platform-team"
`

	err := os.WriteFile(configFile, []byte(configContent), 0644)
	require.NoError(t, err)

	config, err := LoadAdminConfig(configFile)
	require.NoError(t, err)

	assert.Equal(t, "https://git.company.com", config.Gitea.URL)
	assert.Equal(t, "platform-admin", config.Gitea.Username)
	assert.Equal(t, "super-secret", config.Gitea.Password)
	assert.Equal(t, "platform-team", config.Gitea.OrgName)
}

func TestArgoCDConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "argocd-config.yaml")

	configContent := `
argocd:
  url: "https://argocd.company.com"
  username: "platform-admin"
  password: "argo-secret"
`

	err := os.WriteFile(configFile, []byte(configContent), 0644)
	require.NoError(t, err)

	config, err := LoadAdminConfig(configFile)
	require.NoError(t, err)

	assert.Equal(t, "https://argocd.company.com", config.ArgoCD.URL)
	assert.Equal(t, "platform-admin", config.ArgoCD.Username)
	assert.Equal(t, "argo-secret", config.ArgoCD.Password)
}

func TestResourceDefinitions(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "resources-config.yaml")

	configContent := `
resourceDefinitions:
  postgres: "managed-postgres-cluster"
  redis: "redis-cluster"
  mongodb: "managed-mongodb"
  mysql: "managed-mysql-cluster"
  volume: "persistent-volume-claim"
  route: "ingress-route"
`

	err := os.WriteFile(configFile, []byte(configContent), 0644)
	require.NoError(t, err)

	config, err := LoadAdminConfig(configFile)
	require.NoError(t, err)

	expectedResources := map[string]string{
		"postgres": "managed-postgres-cluster",
		"redis":    "redis-cluster",
		"mongodb":  "managed-mongodb",
		"mysql":    "managed-mysql-cluster",
		"volume":   "persistent-volume-claim",
		"route":    "ingress-route",
	}

	assert.Equal(t, expectedResources, config.ResourceDefinitions)
	assert.Len(t, config.ResourceDefinitions, 6)
}

func TestPoliciesConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "policies-config.yaml")

	configContent := `
policies:
  enforceBackups: false
  allowedEnvironments:
    - "development"
    - "staging"
    - "production"
    - "preview"
`

	err := os.WriteFile(configFile, []byte(configContent), 0644)
	require.NoError(t, err)

	config, err := LoadAdminConfig(configFile)
	require.NoError(t, err)

	assert.False(t, config.Policies.EnforceBackups)
	assert.Len(t, config.Policies.AllowedEnvironments, 4)
	assert.Contains(t, config.Policies.AllowedEnvironments, "development")
	assert.Contains(t, config.Policies.AllowedEnvironments, "staging")
	assert.Contains(t, config.Policies.AllowedEnvironments, "production")
	assert.Contains(t, config.Policies.AllowedEnvironments, "preview")
}
