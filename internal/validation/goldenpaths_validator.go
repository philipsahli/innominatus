package validation

import (
	"fmt"
	"os"
	"innominatus/internal/goldenpaths"
	"innominatus/internal/types"

	"gopkg.in/yaml.v3"
)

// GoldenPathsValidator validates golden paths configuration
type GoldenPathsValidator struct {
	config *goldenpaths.GoldenPathsConfig
}

// NewGoldenPathsValidator creates a new golden paths validator
func NewGoldenPathsValidator(configPath string) (*GoldenPathsValidator, error) {
	// Use default path if not provided
	_ = configPath // Parameter kept for interface compatibility but not used

	config, err := goldenpaths.LoadGoldenPaths()
	if err != nil {
		return nil, fmt.Errorf("failed to load golden paths config: %w", err)
	}

	return &GoldenPathsValidator{config: config}, nil
}

// Validate validates the golden paths configuration
func (v *GoldenPathsValidator) Validate() *ValidationResult {
	result := &ValidationResult{
		Valid:     true,
		Errors:    make([]string, 0),
		Warnings:  make([]string, 0),
		Component: "Golden Paths Configuration",
	}

	// Check if any golden paths are defined
	if len(v.config.GoldenPaths) == 0 {
		result.Warnings = append(result.Warnings, "No golden paths defined - consider adding at least one golden path")
		result.Valid = len(result.Errors) == 0
		return result
	}

	// Validate each golden path
	v.validateGoldenPaths(result)

	// Check for recommended golden paths
	v.checkRecommendedPaths(result)

	result.Valid = len(result.Errors) == 0
	return result
}

// GetComponent returns the component name
func (v *GoldenPathsValidator) GetComponent() string {
	return "Golden Paths Configuration"
}

func (v *GoldenPathsValidator) validateGoldenPaths(result *ValidationResult) {
	for pathName := range v.config.GoldenPaths {
		// Validate path name format
		if err := ValidateRegex("goldenPath.name", pathName,
			`^[a-z][a-z0-9\-]*[a-z0-9]$`, "lowercase alphanumeric with hyphens"); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Golden path '%s': %s", pathName, err.Error()))
			continue
		}

		// Get metadata which contains the workflow file
		metadata, err := v.config.GetMetadata(pathName)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Golden path '%s': failed to get metadata: %s", pathName, err.Error()))
			continue
		}

		// Validate workflow file exists
		if err := ValidateFileExists(metadata.WorkflowFile); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Golden path '%s': %s", pathName, err.Error()))
			continue
		}

		// Validate workflow file content
		if err := v.validateWorkflowFile(pathName, metadata.WorkflowFile, result); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Golden path '%s': %s", pathName, err.Error()))
		}
	}
}

func (v *GoldenPathsValidator) validateWorkflowFile(pathName, workflowFile string, result *ValidationResult) error {
	// Read and parse workflow file
	data, err := os.ReadFile(workflowFile)
	if err != nil {
		return fmt.Errorf("cannot read workflow file: %w", err)
	}

	// Parse as generic map first to check structure
	var rawWorkflow map[string]interface{}
	if err := yaml.Unmarshal(data, &rawWorkflow); err != nil {
		return fmt.Errorf("invalid YAML syntax in workflow file: %w", err)
	}

	// Validate workflow structure
	if err := v.validateWorkflowStructure(pathName, rawWorkflow, result); err != nil {
		return err
	}

	// Parse into types.Workflow for step validation
	var workflow types.Workflow
	if err := yaml.Unmarshal(data, &workflow); err != nil {
		return fmt.Errorf("cannot parse workflow steps: %w", err)
	}

	// Validate workflow steps
	if err := v.validateWorkflowSteps(pathName, workflow.Steps, result); err != nil {
		return err
	}

	return nil
}

func (v *GoldenPathsValidator) validateWorkflowStructure(pathName string, rawWorkflow map[string]interface{}, result *ValidationResult) error {
	// Check for name in various possible locations
	hasName := false
	if name, ok := rawWorkflow["name"].(string); ok && name != "" {
		hasName = true
	}
	if metadata, ok := rawWorkflow["metadata"].(map[string]interface{}); ok {
		if name, ok := metadata["name"].(string); ok && name != "" {
			hasName = true
		}
	}

	if !hasName {
		result.Warnings = append(result.Warnings, fmt.Sprintf("Golden path '%s': workflow should have a name field", pathName))
	}

	// Validate API version if present
	if apiVersion, ok := rawWorkflow["apiVersion"].(string); ok && apiVersion != "" {
		if apiVersion != "workflow.dev/v1" {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Golden path '%s': Unexpected API version '%s', expected 'workflow.dev/v1'", pathName, apiVersion))
		}
	}

	// Validate kind if present
	if kind, ok := rawWorkflow["kind"].(string); ok && kind != "" && kind != "Workflow" {
		result.Warnings = append(result.Warnings, fmt.Sprintf("Golden path '%s': Unexpected kind '%s', expected 'Workflow'", pathName, kind))
	}

	// Check for spec.steps or steps
	var steps []interface{}
	if spec, ok := rawWorkflow["spec"].(map[string]interface{}); ok {
		if specSteps, ok := spec["steps"].([]interface{}); ok {
			steps = specSteps
		}
	} else if directSteps, ok := rawWorkflow["steps"].([]interface{}); ok {
		steps = directSteps
	}

	if len(steps) == 0 {
		return fmt.Errorf("workflow must have at least one step")
	}

	return nil
}

func (v *GoldenPathsValidator) validateWorkflowSteps(pathName string, steps []types.Step, result *ValidationResult) error {
	supportedStepTypes := []string{
		"terraform", "ansible", "kubernetes",
		"gitea-repo", "argocd-app", "git-commit-manifests",
	}

	stepNames := make(map[string]bool)

	for i, step := range steps {
		stepContext := fmt.Sprintf("Golden path '%s', step %d", pathName, i+1)

		// Validate step name
		if step.Name == "" {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: step name cannot be empty", stepContext))
			continue
		}

		// Check for duplicate step names
		if stepNames[step.Name] {
			result.Warnings = append(result.Warnings, fmt.Sprintf("%s: duplicate step name '%s'", stepContext, step.Name))
		}
		stepNames[step.Name] = true

		// Validate step type
		if err := ValidateEnum("step.type", step.Type, supportedStepTypes); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s (%s): %s", stepContext, step.Name, err.Error()))
			continue
		}

		// Validate step-specific configuration
		if err := v.validateStepConfiguration(stepContext, &step, result); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s (%s): %s", stepContext, step.Name, err.Error()))
		}
	}

	return nil
}

func (v *GoldenPathsValidator) validateStepConfiguration(stepContext string, step *types.Step, result *ValidationResult) error {
	switch step.Type {
	case "terraform":
		if step.Path == "" {
			return fmt.Errorf("terraform step requires 'path' field")
		}
		// Check if terraform directory exists (optional warning)
		if err := ValidateDirectoryExists(step.Path); err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("%s: terraform directory validation: %s", stepContext, err.Error()))
		}

	case "ansible":
		if step.Playbook == "" {
			return fmt.Errorf("ansible step requires 'playbook' field")
		}
		// Check if playbook file exists (optional warning)
		playbookPath := step.Playbook
		if step.Path != "" {
			playbookPath = step.Path + "/" + step.Playbook
		}
		if err := ValidateFileExists(playbookPath); err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("%s: ansible playbook validation: %s", stepContext, err.Error()))
		}

	case "kubernetes":
		if step.Namespace == "" {
			result.Warnings = append(result.Warnings, fmt.Sprintf("%s: kubernetes step should specify a namespace", stepContext))
		} else {
			// Validate namespace format
			if err := ValidateRegex("namespace", step.Namespace,
				`^[a-z0-9][a-z0-9\-]*[a-z0-9]$|^\${.*}$`, "lowercase alphanumeric with hyphens or template variable"); err != nil {
				result.Warnings = append(result.Warnings, fmt.Sprintf("%s: %s", stepContext, err.Error()))
			}
		}

	case "gitea-repo":
		if step.RepoName == "" {
			return fmt.Errorf("gitea-repo step requires 'repoName' field")
		}
		if err := ValidateRegex("repoName", step.RepoName,
			`^[a-zA-Z0-9][a-zA-Z0-9._\-]*[a-zA-Z0-9]$`, "alphanumeric with dots, underscores, and hyphens"); err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("%s: %s", stepContext, err.Error()))
		}

	case "argocd-app":
		if step.AppName == "" && step.RepoName == "" && step.RepoURL == "" {
			return fmt.Errorf("argocd-app step requires at least one of: appName, repoName, or repoURL")
		}
		if step.SyncPolicy != "" {
			allowedPolicies := []string{"auto", "manual"}
			if err := ValidateEnum("syncPolicy", step.SyncPolicy, allowedPolicies); err != nil {
				result.Warnings = append(result.Warnings, fmt.Sprintf("%s: %s", stepContext, err.Error()))
			}
		}

	case "git-commit-manifests":
		if step.RepoName == "" {
			return fmt.Errorf("git-commit-manifests step requires 'repoName' field")
		}
		if step.GitBranch != "" {
			if err := ValidateRegex("gitBranch", step.GitBranch,
				`^[a-zA-Z0-9][a-zA-Z0-9._/\-]*[a-zA-Z0-9]$`, "valid git branch name"); err != nil {
				result.Warnings = append(result.Warnings, fmt.Sprintf("%s: %s", stepContext, err.Error()))
			}
		}
	}

	return nil
}

func (v *GoldenPathsValidator) checkRecommendedPaths(result *ValidationResult) {
	recommendedPaths := []string{
		"deploy-app",
		"ephemeral-env",
	}

	for _, recommended := range recommendedPaths {
		if _, exists := v.config.GoldenPaths[recommended]; !exists {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Consider adding recommended golden path: '%s'", recommended))
		}
	}

	// Check for common workflow patterns
	hasDeploymentPath := false
	hasEnvironmentPath := false

	for pathName := range v.config.GoldenPaths {
		if contains([]string{"deploy", "deployment", "app-deploy", "deploy-app"}, pathName) {
			hasDeploymentPath = true
		}
		if contains([]string{"env", "environment", "ephemeral", "ephemeral-env"}, pathName) {
			hasEnvironmentPath = true
		}
	}

	if !hasDeploymentPath {
		result.Warnings = append(result.Warnings, "Consider adding a deployment-focused golden path for application deployments")
	}

	if !hasEnvironmentPath {
		result.Warnings = append(result.Warnings, "Consider adding an environment-focused golden path for environment provisioning")
	}
}