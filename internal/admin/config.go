package admin

import (
	"fmt"
	"io/ioutil"
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
	WorkflowPolicies struct {
		WorkflowsRoot                string   `yaml:"workflowsRoot"`
		RequiredPlatformWorkflows    []string `yaml:"requiredPlatformWorkflows"`
		AllowedProductWorkflows      []string `yaml:"allowedProductWorkflows"`
		MaxWorkflowDuration          string   `yaml:"maxWorkflowDuration"`
		MaxConcurrentWorkflows       int      `yaml:"maxConcurrentWorkflows"`
		MaxStepsPerWorkflow          int      `yaml:"maxStepsPerWorkflow"`
		AllowedStepTypes             []string `yaml:"allowedStepTypes"`
		WorkflowOverrides struct {
			Platform bool `yaml:"platform"`
			Product  bool `yaml:"product"`
		} `yaml:"workflowOverrides"`
		Security struct {
			RequireApproval    []string `yaml:"requireApproval"`
			AllowedExecutors   []string `yaml:"allowedExecutors"`
			SecretsAccess      map[string]string `yaml:"secretsAccess"`
		} `yaml:"security"`
	} `yaml:"workflowPolicies"`
}

func LoadAdminConfig(configPath string) (*AdminConfig, error) {
	// Check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to read config file: %s", configPath)
	}

	data, err := ioutil.ReadFile(configPath)
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