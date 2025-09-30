package validation

import (
	"fmt"
	"net/http"
	"innominatus/internal/admin"
	"strings"
	"time"
)

// AdminConfigValidator validates admin configuration
type AdminConfigValidator struct {
	config *admin.AdminConfig
}

// NewAdminConfigValidator creates a new admin config validator
func NewAdminConfigValidator(configPath string) (*AdminConfigValidator, error) {
	config, err := admin.LoadAdminConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load admin config: %w", err)
	}

	return &AdminConfigValidator{config: config}, nil
}

// Validate validates the admin configuration
func (v *AdminConfigValidator) Validate() *ValidationResult {
	result := &ValidationResult{
		Valid:     true,
		Errors:    make([]string, 0),
		Warnings:  make([]string, 0),
		Component: "Admin Configuration",
	}

	// Validate admin section
	v.validateAdminSection(result)

	// Validate resource definitions
	v.validateResourceDefinitions(result)

	// Validate policies
	v.validatePolicies(result)

	// Validate Gitea configuration
	v.validateGiteaConfig(result)

	// Validate ArgoCD configuration
	v.validateArgoCDConfig(result)

	// Overall validity
	result.Valid = len(result.Errors) == 0

	return result
}

// GetComponent returns the component name
func (v *AdminConfigValidator) GetComponent() string {
	return "Admin Configuration"
}

func (v *AdminConfigValidator) validateAdminSection(result *ValidationResult) {
	admin := v.config.Admin

	// Validate default cost center
	if err := ValidateRequired("defaultCostCenter", admin.DefaultCostCenter); err != nil {
		result.Errors = append(result.Errors, err.Error())
	} else if err := ValidateRegex("defaultCostCenter", admin.DefaultCostCenter,
		`^[a-z][a-z0-9\-]*[a-z0-9]$`, "lowercase alphanumeric with hyphens"); err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("Cost center format: %s", err.Error()))
	}

	// Validate default runtime
	allowedRuntimes := []string{"kubernetes", "docker", "nomad", "openshift"}
	if err := ValidateEnum("defaultRuntime", admin.DefaultRuntime, allowedRuntimes); err != nil {
		result.Errors = append(result.Errors, err.Error())
	}

	// Validate Splunk index (if provided)
	if admin.SplunkIndex != "" {
		if err := ValidateRegex("splunkIndex", admin.SplunkIndex,
			`^[a-z][a-z0-9_\-]*[a-z0-9]$`, "lowercase alphanumeric with underscores and hyphens"); err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Splunk index format: %s", err.Error()))
		}
	}
}

func (v *AdminConfigValidator) validateResourceDefinitions(result *ValidationResult) {
	resourceDefs := v.config.ResourceDefinitions

	requiredResources := []string{"postgres", "redis", "volume", "route"}
	for _, resource := range requiredResources {
		if definition, exists := resourceDefs[resource]; !exists {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Missing resource definition for '%s'", resource))
		} else if definition == "" {
			result.Errors = append(result.Errors, fmt.Sprintf("Resource definition for '%s' cannot be empty", resource))
		} else {
			// Validate resource definition format (should be kebab-case)
			if err := ValidateRegex(fmt.Sprintf("resourceDefinitions.%s", resource), definition,
				`^[a-z][a-z0-9\-]*[a-z0-9]$`, "kebab-case format"); err != nil {
				result.Warnings = append(result.Warnings, err.Error())
			}
		}
	}
}

func (v *AdminConfigValidator) validatePolicies(result *ValidationResult) {
	policies := v.config.Policies

	// Validate allowed environments
	if len(policies.AllowedEnvironments) == 0 {
		result.Warnings = append(result.Warnings, "No allowed environments defined - this may restrict deployments")
	} else {
		standardEnvs := []string{"development", "staging", "production"}
		for _, env := range policies.AllowedEnvironments {
			if err := ValidateRegex("allowedEnvironment", env,
				`^[a-z][a-z0-9\-]*[a-z0-9]$`, "lowercase alphanumeric with hyphens"); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("Environment name '%s': %s", env, err.Error()))
			}

			// Warn if using non-standard environment names
			isStandard := false
			for _, std := range standardEnvs {
				if env == std {
					isStandard = true
					break
				}
			}
			if !isStandard {
				result.Warnings = append(result.Warnings, fmt.Sprintf("Non-standard environment name '%s' - consider using: %v", env, standardEnvs))
			}
		}
	}

	// Validate that production environment has enforceBackups enabled
	if contains(policies.AllowedEnvironments, "production") && !policies.EnforceBackups {
		result.Warnings = append(result.Warnings, "Production environment detected but backups are not enforced - consider enabling enforceBackups")
	}
}

func (v *AdminConfigValidator) validateGiteaConfig(result *ValidationResult) {
	gitea := v.config.Gitea

	// Validate Gitea URL
	if err := ValidateRequired("gitea.url", gitea.URL); err != nil {
		result.Errors = append(result.Errors, err.Error())
		return // Can't validate further without URL
	}

	if err := ValidateURL(gitea.URL, []string{"http", "https"}); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("gitea.url: %s", err.Error()))
	} else {
		// Test connectivity to Gitea (with timeout)
		if err := v.testGiteaConnectivity(); err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Gitea connectivity test failed: %s", err.Error()))
		}
	}

	// Validate credentials
	if err := ValidateRequired("gitea.username", gitea.Username); err != nil {
		result.Errors = append(result.Errors, err.Error())
	}

	if err := ValidateRequired("gitea.password", gitea.Password); err != nil {
		result.Errors = append(result.Errors, err.Error())
	} else if gitea.Password == "admin" || gitea.Password == "password" || gitea.Password == "123456" {
		result.Warnings = append(result.Warnings, "Gitea password appears to be a default/weak password - consider using a stronger password")
	}

	// Validate org name format
	if gitea.OrgName != "" {
		if err := ValidateRegex("gitea.orgName", gitea.OrgName,
			`^[a-zA-Z0-9][a-zA-Z0-9\-]*[a-zA-Z0-9]$`, "alphanumeric with hyphens"); err != nil {
			result.Warnings = append(result.Warnings, err.Error())
		}
	}
}

func (v *AdminConfigValidator) validateArgoCDConfig(result *ValidationResult) {
	argocd := v.config.ArgoCD

	// Validate ArgoCD URL
	if err := ValidateRequired("argocd.url", argocd.URL); err != nil {
		result.Errors = append(result.Errors, err.Error())
		return // Can't validate further without URL
	}

	if err := ValidateURL(argocd.URL, []string{"http", "https"}); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("argocd.url: %s", err.Error()))
	} else {
		// Test connectivity to ArgoCD (with timeout)
		if err := v.testArgoCDConnectivity(); err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("ArgoCD connectivity test failed: %s", err.Error()))
		}
	}

	// Validate credentials
	if err := ValidateRequired("argocd.username", argocd.Username); err != nil {
		result.Errors = append(result.Errors, err.Error())
	}

	if err := ValidateRequired("argocd.password", argocd.Password); err != nil {
		result.Errors = append(result.Errors, err.Error())
	} else if argocd.Password == "admin" || argocd.Password == "password" || len(argocd.Password) < 8 {
		result.Warnings = append(result.Warnings, "ArgoCD password appears to be weak - consider using a stronger password (8+ characters)")
	}
}

func (v *AdminConfigValidator) testGiteaConnectivity() error {
	client := &http.Client{Timeout: 5 * time.Second}

	// Test basic connectivity
	resp, err := client.Get(v.config.Gitea.URL + "/api/v1/version")
	if err != nil {
		return fmt.Errorf("cannot reach Gitea API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("Gitea API returned status %d", resp.StatusCode)
	}

	return nil
}

func (v *AdminConfigValidator) testArgoCDConnectivity() error {
	client := &http.Client{Timeout: 5 * time.Second}

	// Test basic connectivity (ArgoCD often redirects, so allow redirects)
	resp, err := client.Get(v.config.ArgoCD.URL + "/api/version")
	if err != nil {
		return fmt.Errorf("cannot reach ArgoCD API: %w", err)
	}
	defer resp.Body.Close()

	// ArgoCD might return 401 for unauthenticated version endpoint, which is fine
	if resp.StatusCode >= 500 {
		return fmt.Errorf("ArgoCD API returned status %d", resp.StatusCode)
	}

	return nil
}

// Helper function to check if slice contains string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if strings.EqualFold(s, item) {
			return true
		}
	}
	return false
}