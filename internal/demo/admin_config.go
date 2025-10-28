package demo

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// AdminConfig represents the admin-config.yaml structure
type AdminConfig struct {
	Admin               AdminSettings     `yaml:"admin"`
	Providers           []ProviderSource  `yaml:"providers"`
	ResourceDefinitions map[string]string `yaml:"resourceDefinitions"`
	Policies            Policies          `yaml:"policies"`
	WorkflowPolicies    WorkflowPolicies  `yaml:"workflowPolicies"`
	Gitea               GiteaConfig       `yaml:"gitea"`
	ArgoCD              ArgoCDConfig      `yaml:"argocd"`
	Vault               VaultConfig       `yaml:"vault"`
}

type AdminSettings struct {
	DefaultCostCenter string `yaml:"defaultCostCenter"`
	DefaultRuntime    string `yaml:"defaultRuntime"`
	SplunkIndex       string `yaml:"splunkIndex,omitempty"`
}

type ProviderSource struct {
	Name       string `yaml:"name"`
	Type       string `yaml:"type"`
	Path       string `yaml:"path,omitempty"`
	Repository string `yaml:"repository,omitempty"`
	Ref        string `yaml:"ref,omitempty"`
	Category   string `yaml:"category,omitempty"`
	Enabled    bool   `yaml:"enabled"`
}

type Policies struct {
	EnforceBackups      bool     `yaml:"enforceBackups"`
	AllowedEnvironments []string `yaml:"allowedEnvironments"`
}

type WorkflowPolicies struct {
	WorkflowsRoot             string           `yaml:"workflowsRoot"`
	RequiredPlatformWorkflows []string         `yaml:"requiredPlatformWorkflows"`
	AllowedProductWorkflows   []string         `yaml:"allowedProductWorkflows"`
	WorkflowOverrides         map[string]bool  `yaml:"workflowOverrides"`
	MaxWorkflowDuration       string           `yaml:"maxWorkflowDuration"`
	MaxConcurrentWorkflows    int              `yaml:"maxConcurrentWorkflows"`
	MaxStepsPerWorkflow       int              `yaml:"maxStepsPerWorkflow"`
	Security                  WorkflowSecurity `yaml:"security"`
	AllowedStepTypes          []string         `yaml:"allowedStepTypes"`
}

type WorkflowSecurity struct {
	RequireApproval  []string          `yaml:"requireApproval"`
	AllowedExecutors []string          `yaml:"allowedExecutors"`
	SecretsAccess    map[string]string `yaml:"secretsAccess"`
}

type GiteaConfig struct {
	URL         string `yaml:"url"`
	InternalURL string `yaml:"internalURL"`
	Username    string `yaml:"username"`
	Password    string `yaml:"password"`
	OrgName     string `yaml:"orgName"`
}

type ArgoCDConfig struct {
	URL      string `yaml:"url"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type VaultConfig struct {
	URL       string `yaml:"url"`
	Token     string `yaml:"token"`
	Namespace string `yaml:"namespace"`
}

// CreateAdminConfig generates an admin-config.yaml file for the demo environment
func CreateAdminConfig(outputPath string) error {
	config := AdminConfig{
		Admin: AdminSettings{
			DefaultCostCenter: "engineering",
			DefaultRuntime:    "kubernetes",
			SplunkIndex:       "orchestrator-logs",
		},
		Providers: []ProviderSource{
			{
				Name:    "builtin",
				Type:    "filesystem",
				Path:    "./providers/builtin",
				Enabled: true,
			},
			// Git-based provider example (disabled by default)
			// Uncomment to load from Gitea repository
			// {
			// 	Name:       "builtin-git",
			// 	Type:       "git",
			// 	Repository: "http://gitea.localtest.me/giteaadmin/platform-config",
			// 	Ref:        "main",
			// 	Category:   "infrastructure",
			// 	Enabled:    false,
			// },
		},
		ResourceDefinitions: map[string]string{
			"postgres":    "managed-postgres-cluster",
			"redis":       "redis-cluster",
			"volume":      "persistent-volume-claim",
			"route":       "ingress-route",
			"vault-space": "vault-application-namespace",
		},
		Policies: Policies{
			EnforceBackups: true,
			AllowedEnvironments: []string{
				"development",
				"staging",
				"production",
			},
		},
		WorkflowPolicies: WorkflowPolicies{
			WorkflowsRoot: "./workflows",
			RequiredPlatformWorkflows: []string{
				"security-scan",
				"cost-monitoring",
			},
			AllowedProductWorkflows: []string{
				"ecommerce/database-setup",
				"ecommerce/payment-integration",
				"analytics/data-pipeline",
			},
			WorkflowOverrides: map[string]bool{
				"platform": true,
				"product":  true,
			},
			MaxWorkflowDuration:    "30m",
			MaxConcurrentWorkflows: 10,
			MaxStepsPerWorkflow:    50,
			Security: WorkflowSecurity{
				RequireApproval: []string{"production"},
				AllowedExecutors: []string{
					"platform-team",
					"infrastructure-teams",
				},
				SecretsAccess: map[string]string{
					"vault":      "read-only",
					"kubernetes": "namespace-scoped",
				},
			},
			AllowedStepTypes: []string{
				"terraform",
				"kubernetes",
				"ansible",
				"database-migration",
				"vault-setup",
				"monitoring",
				"validation",
				"security",
				"policy",
				"tagging",
				"cost-analysis",
				"resource-provisioning",
			},
		},
		Gitea: GiteaConfig{
			URL:         "http://gitea.localtest.me",
			InternalURL: "http://gitea-http.gitea.svc.cluster.local:3000",
			Username:    "giteaadmin",
			Password:    "admin123",
			OrgName:     "platform-team",
		},
		ArgoCD: ArgoCDConfig{
			URL:      "http://argocd.localtest.me",
			Username: "admin",
			Password: "admin123",
		},
		Vault: VaultConfig{
			URL:       "http://vault.localtest.me",
			Token:     "root",
			Namespace: "",
		},
	}

	// Marshal to YAML
	data, err := yaml.Marshal(&config)
	if err != nil {
		return fmt.Errorf("failed to marshal admin config: %w", err)
	}

	// Add header comment
	header := `# Innominatus Admin Configuration
# Generated by demo-time for local development
#
# This file configures:
# - Provider sources (filesystem and git-based)
# - Resource definitions
# - Workflow policies
# - Integration credentials (Gitea, ArgoCD, Vault)
#
# For production use, copy this file and:
# 1. Change passwords and tokens
# 2. Enable git-based providers instead of filesystem
# 3. Adjust policies to match your organization
#
# Documentation: https://docs.example.com/admin-config

`

	content := header + string(data)

	// Write to file
	// #nosec G306 -- admin-config.yaml needs to be readable by server process
	if err := os.WriteFile(outputPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write admin config: %w", err)
	}

	return nil
}
