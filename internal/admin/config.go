package admin

import (
	"fmt"
	"innominatus/internal/security"
	"os"

	"gopkg.in/yaml.v3"
)

type AdminConfig struct {
	Admin struct {
		DefaultCostCenter string `yaml:"defaultCostCenter"`
		DefaultRuntime    string `yaml:"defaultRuntime"`
		SplunkIndex       string `yaml:"splunkIndex"`
	} `yaml:"admin"`
	ResourceDefinitions map[string]string `yaml:"resourceDefinitions"`
	Policies            struct {
		EnforceBackups      bool     `yaml:"enforceBackups"`
		AllowedEnvironments []string `yaml:"allowedEnvironments"`
	} `yaml:"policies"`
	Gitea struct {
		URL         string `yaml:"url"`
		InternalURL string `yaml:"internalURL"`
		Username    string `yaml:"username"`
		Password    string `yaml:"password"`
		OrgName     string `yaml:"orgName"`
	} `yaml:"gitea"`
	ArgoCD struct {
		URL      string `yaml:"url"`
		Username string `yaml:"username"`
		Password string `yaml:"password"`
	} `yaml:"argocd"`
	Vault struct {
		URL       string `yaml:"url"`
		Token     string `yaml:"token"`
		Namespace string `yaml:"namespace"`
	} `yaml:"vault"`
	Keycloak struct {
		URL          string `yaml:"url"`
		AdminUser    string `yaml:"adminUser"`
		AdminPassword string `yaml:"adminPassword"`
		Realm        string `yaml:"realm"`
	} `yaml:"keycloak"`
	Minio struct {
		URL             string `yaml:"url"`
		ConsoleURL      string `yaml:"consoleURL"`
		AccessKey       string `yaml:"accessKey"`
		SecretKey       string `yaml:"secretKey"`
	} `yaml:"minio"`
	Prometheus struct {
		URL string `yaml:"url"`
	} `yaml:"prometheus"`
	Grafana struct {
		URL      string `yaml:"url"`
		Username string `yaml:"username"`
		Password string `yaml:"password"`
	} `yaml:"grafana"`
	KubernetesDashboard struct {
		URL string `yaml:"url"`
	} `yaml:"kubernetesDashboard"`
	WorkflowPolicies struct {
		WorkflowsRoot             string   `yaml:"workflowsRoot"`
		RequiredPlatformWorkflows []string `yaml:"requiredPlatformWorkflows"`
		AllowedProductWorkflows   []string `yaml:"allowedProductWorkflows"`
		MaxWorkflowDuration       string   `yaml:"maxWorkflowDuration"`
		MaxConcurrentWorkflows    int      `yaml:"maxConcurrentWorkflows"`
		MaxStepsPerWorkflow       int      `yaml:"maxStepsPerWorkflow"`
		AllowedStepTypes          []string `yaml:"allowedStepTypes"`
		WorkflowOverrides         struct {
			Platform bool `yaml:"platform"`
			Product  bool `yaml:"product"`
		} `yaml:"workflowOverrides"`
		Security struct {
			RequireApproval  []string          `yaml:"requireApproval"`
			AllowedExecutors []string          `yaml:"allowedExecutors"`
			SecretsAccess    map[string]string `yaml:"secretsAccess"`
		} `yaml:"security"`
	} `yaml:"workflowPolicies"`
}

func LoadAdminConfig(configPath string) (*AdminConfig, error) {
	// Validate config path to prevent path traversal
	if err := security.ValidateConfigPath(configPath); err != nil {
		return nil, fmt.Errorf("invalid config path: %w", err)
	}

	// Check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to read config file: %s", configPath)
	}

	data, err := os.ReadFile(configPath) // #nosec G304 - path validated above
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config AdminConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Initialize maps if nil
	if config.ResourceDefinitions == nil {
		config.ResourceDefinitions = make(map[string]string)
	}

	return &config, nil
}

func (c *AdminConfig) PrintConfig() {
	fmt.Println("Admin Configuration:")
	fmt.Printf("  Default Cost Center: %s\n", c.Admin.DefaultCostCenter)
	fmt.Printf("  Default Runtime: %s\n", c.Admin.DefaultRuntime)
	fmt.Printf("  Splunk Index: %s\n", c.Admin.SplunkIndex)

	fmt.Println("Resource Definitions:")
	for name, definition := range c.ResourceDefinitions {
		fmt.Printf("  %s: %s\n", name, definition)
	}

	fmt.Println("Policies:")
	fmt.Printf("  Enforce Backups: %t\n", c.Policies.EnforceBackups)
	fmt.Printf("  Allowed Environments: %v\n", c.Policies.AllowedEnvironments)

	fmt.Println("Gitea Configuration:")
	fmt.Printf("  URL: %s\n", c.Gitea.URL)
	fmt.Printf("  Username: %s\n", c.Gitea.Username)
	fmt.Printf("  Password: %s\n", "***")
	fmt.Printf("  Organization: %s\n", c.Gitea.OrgName)

	fmt.Println("ArgoCD Configuration:")
	fmt.Printf("  URL: %s\n", c.ArgoCD.URL)
	fmt.Printf("  Username: %s\n", c.ArgoCD.Username)
	fmt.Printf("  Password: %s\n", "***")

	fmt.Println("Workflow Policies:")
	fmt.Printf("  Workflows Root: %s\n", c.WorkflowPolicies.WorkflowsRoot)
	fmt.Printf("  Required Platform Workflows: %v\n", c.WorkflowPolicies.RequiredPlatformWorkflows)
	fmt.Printf("  Allowed Product Workflows: %v\n", c.WorkflowPolicies.AllowedProductWorkflows)
	fmt.Printf("  Max Workflow Duration: %s\n", c.WorkflowPolicies.MaxWorkflowDuration)
	fmt.Printf("  Max Concurrent Workflows: %d\n", c.WorkflowPolicies.MaxConcurrentWorkflows)
	fmt.Printf("  Max Steps Per Workflow: %d\n", c.WorkflowPolicies.MaxStepsPerWorkflow)
	fmt.Printf("  Allowed Step Types: %v\n", c.WorkflowPolicies.AllowedStepTypes)
}

func (c *AdminConfig) GetResourceDefinition(resourceType string) (string, bool) {
	definition, exists := c.ResourceDefinitions[resourceType]
	return definition, exists
}

// MaskedAdminConfig is a JSON-serializable version with sensitive data masked
type MaskedAdminConfig struct {
	Admin struct {
		DefaultCostCenter string `json:"defaultCostCenter"`
		DefaultRuntime    string `json:"defaultRuntime"`
		SplunkIndex       string `json:"splunkIndex"`
	} `json:"admin"`
	ResourceDefinitions map[string]string `json:"resourceDefinitions"`
	Policies            struct {
		EnforceBackups      bool     `json:"enforceBackups"`
		AllowedEnvironments []string `json:"allowedEnvironments"`
	} `json:"policies"`
	Gitea struct {
		URL         string `json:"url"`
		InternalURL string `json:"internalURL"`
		Username    string `json:"username"`
		Password    string `json:"password"` // Will be "****"
		OrgName     string `json:"orgName"`
	} `json:"gitea"`
	ArgoCD struct {
		URL      string `json:"url"`
		Username string `json:"username"`
		Password string `json:"password"` // Will be "****"
	} `json:"argocd"`
	Vault struct {
		URL       string `json:"url"`
		Token     string `json:"token"` // Will be "****"
		Namespace string `json:"namespace"`
	} `json:"vault"`
	Keycloak struct {
		URL          string `json:"url"`
		AdminUser    string `json:"adminUser"`
		AdminPassword string `json:"adminPassword"` // Will be "****"
		Realm        string `json:"realm"`
	} `json:"keycloak"`
	Minio struct {
		URL             string `json:"url"`
		ConsoleURL      string `json:"consoleURL"`
		AccessKey       string `json:"accessKey"`
		SecretKey       string `json:"secretKey"` // Will be "****"
	} `json:"minio"`
	Prometheus struct {
		URL string `json:"url"`
	} `json:"prometheus"`
	Grafana struct {
		URL      string `json:"url"`
		Username string `json:"username"`
		Password string `json:"password"` // Will be "****"
	} `json:"grafana"`
	KubernetesDashboard struct {
		URL string `json:"url"`
	} `json:"kubernetesDashboard"`
	WorkflowPolicies struct {
		WorkflowsRoot             string   `json:"workflowsRoot"`
		RequiredPlatformWorkflows []string `json:"requiredPlatformWorkflows"`
		AllowedProductWorkflows   []string `json:"allowedProductWorkflows"`
		MaxWorkflowDuration       string   `json:"maxWorkflowDuration"`
		MaxConcurrentWorkflows    int      `json:"maxConcurrentWorkflows"`
		MaxStepsPerWorkflow       int      `json:"maxStepsPerWorkflow"`
		AllowedStepTypes          []string `json:"allowedStepTypes"`
		WorkflowOverrides         struct {
			Platform bool `json:"platform"`
			Product  bool `json:"product"`
		} `json:"workflowOverrides"`
		Security struct {
			RequireApproval  []string          `json:"requireApproval"`
			AllowedExecutors []string          `json:"allowedExecutors"`
			SecretsAccess    map[string]string `json:"secretsAccess"`
		} `json:"security"`
	} `json:"workflowPolicies"`
}

// ToMaskedJSON returns a JSON-serializable version with sensitive data masked
func (c *AdminConfig) ToMaskedJSON() *MaskedAdminConfig {
	masked := &MaskedAdminConfig{
		ResourceDefinitions: c.ResourceDefinitions,
	}

	// Copy admin settings
	masked.Admin.DefaultCostCenter = c.Admin.DefaultCostCenter
	masked.Admin.DefaultRuntime = c.Admin.DefaultRuntime
	masked.Admin.SplunkIndex = c.Admin.SplunkIndex

	// Copy policies
	masked.Policies.EnforceBackups = c.Policies.EnforceBackups
	masked.Policies.AllowedEnvironments = c.Policies.AllowedEnvironments

	// Copy Gitea config with masked password
	masked.Gitea.URL = c.Gitea.URL
	masked.Gitea.InternalURL = c.Gitea.InternalURL
	masked.Gitea.Username = c.Gitea.Username
	masked.Gitea.Password = "****"
	masked.Gitea.OrgName = c.Gitea.OrgName

	// Copy ArgoCD config with masked password
	masked.ArgoCD.URL = c.ArgoCD.URL
	masked.ArgoCD.Username = c.ArgoCD.Username
	masked.ArgoCD.Password = "****"

	// Copy Vault config with masked token
	masked.Vault.URL = c.Vault.URL
	masked.Vault.Token = "****"
	masked.Vault.Namespace = c.Vault.Namespace

	// Copy Keycloak config with masked password
	masked.Keycloak.URL = c.Keycloak.URL
	masked.Keycloak.AdminUser = c.Keycloak.AdminUser
	masked.Keycloak.AdminPassword = "****"
	masked.Keycloak.Realm = c.Keycloak.Realm

	// Copy Minio config with masked secret key
	masked.Minio.URL = c.Minio.URL
	masked.Minio.ConsoleURL = c.Minio.ConsoleURL
	masked.Minio.AccessKey = c.Minio.AccessKey
	masked.Minio.SecretKey = "****"

	// Copy Prometheus config
	masked.Prometheus.URL = c.Prometheus.URL

	// Copy Grafana config with masked password
	masked.Grafana.URL = c.Grafana.URL
	masked.Grafana.Username = c.Grafana.Username
	masked.Grafana.Password = "****"

	// Copy Kubernetes Dashboard config
	masked.KubernetesDashboard.URL = c.KubernetesDashboard.URL

	// Copy workflow policies
	masked.WorkflowPolicies.WorkflowsRoot = c.WorkflowPolicies.WorkflowsRoot
	masked.WorkflowPolicies.RequiredPlatformWorkflows = c.WorkflowPolicies.RequiredPlatformWorkflows
	masked.WorkflowPolicies.AllowedProductWorkflows = c.WorkflowPolicies.AllowedProductWorkflows
	masked.WorkflowPolicies.MaxWorkflowDuration = c.WorkflowPolicies.MaxWorkflowDuration
	masked.WorkflowPolicies.MaxConcurrentWorkflows = c.WorkflowPolicies.MaxConcurrentWorkflows
	masked.WorkflowPolicies.MaxStepsPerWorkflow = c.WorkflowPolicies.MaxStepsPerWorkflow
	masked.WorkflowPolicies.AllowedStepTypes = c.WorkflowPolicies.AllowedStepTypes

	// Copy workflow overrides
	masked.WorkflowPolicies.WorkflowOverrides.Platform = c.WorkflowPolicies.WorkflowOverrides.Platform
	masked.WorkflowPolicies.WorkflowOverrides.Product = c.WorkflowPolicies.WorkflowOverrides.Product

	// Copy security settings
	masked.WorkflowPolicies.Security.RequireApproval = c.WorkflowPolicies.Security.RequireApproval
	masked.WorkflowPolicies.Security.AllowedExecutors = c.WorkflowPolicies.Security.AllowedExecutors
	masked.WorkflowPolicies.Security.SecretsAccess = c.WorkflowPolicies.Security.SecretsAccess

	return masked
}
