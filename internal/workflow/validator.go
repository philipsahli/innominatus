package workflow

import (
	"fmt"
	"innominatus/internal/types"
)

// WorkflowValidator validates workflow definitions
type WorkflowValidator struct {
	registeredExecutors map[string]bool
}

// NewWorkflowValidator creates a new workflow validator
func NewWorkflowValidator() *WorkflowValidator {
	return &WorkflowValidator{
		registeredExecutors: map[string]bool{
			"terraform":  true,
			"kubernetes": true,
			"ansible":    true,
			"policy":     true,
			"gitea-repo": true,
			"argocd-app": true,
		},
	}
}

// ValidateWorkflow validates a workflow definition and returns all errors found
func (v *WorkflowValidator) ValidateWorkflow(workflow *types.Workflow) []error {
	var errors []error

	// Validate workflow has at least one step
	if len(workflow.Steps) == 0 {
		errors = append(errors, fmt.Errorf("workflow must have at least one step"))
		return errors // No point checking steps if there are none
	}

	// Validate each step
	for i, step := range workflow.Steps {
		stepErrors := v.validateStep(i, step)
		errors = append(errors, stepErrors...)
	}

	return errors
}

// validateStep validates a single workflow step
func (v *WorkflowValidator) validateStep(index int, step types.Step) []error {
	var errors []error

	// Validate step has a name
	if step.Name == "" {
		errors = append(errors, fmt.Errorf("step %d: step must have a name", index+1))
	}

	// Validate step has a type
	if step.Type == "" {
		errors = append(errors, fmt.Errorf("step %d (%s): step must have a type", index+1, step.Name))
		return errors // Can't validate further without a type
	}

	// Validate step type is registered
	if !v.registeredExecutors[step.Type] {
		errors = append(errors, fmt.Errorf(
			"step %d (%s): unknown step type '%s' (valid types: terraform, kubernetes, ansible, policy, gitea-repo, argocd-app)",
			index+1, step.Name, step.Type))
		// Continue validation to catch other errors
	}

	// Validate step has config
	if step.Config == nil {
		errors = append(errors, fmt.Errorf("step %d (%s): step must have a config", index+1, step.Name))
		return errors // Can't validate config fields if config is nil
	}

	// Validate step-specific requirements
	switch step.Type {
	case "policy":
		errors = append(errors, v.validatePolicyStep(index, step)...)
	case "terraform":
		errors = append(errors, v.validateTerraformStep(index, step)...)
	case "kubernetes":
		errors = append(errors, v.validateKubernetesStep(index, step)...)
	case "ansible":
		errors = append(errors, v.validateAnsibleStep(index, step)...)
	}

	return errors
}

// validatePolicyStep validates a policy step configuration
func (v *WorkflowValidator) validatePolicyStep(index int, step types.Step) []error {
	var errors []error

	// Policy steps must have 'script' field, not 'command'
	scriptValue, hasScript := step.Config["script"]
	_, hasCommand := step.Config["command"]

	if !hasScript {
		if hasCommand {
			errors = append(errors, fmt.Errorf(
				"step %d (%s): policy step requires 'script' in config (found 'command' instead - please rename to 'script')",
				index+1, step.Name))
		} else {
			errors = append(errors, fmt.Errorf(
				"step %d (%s): policy step requires 'script' in config",
				index+1, step.Name))
		}
	} else {
		// Validate script is not empty
		if scriptStr, ok := scriptValue.(string); !ok || scriptStr == "" {
			errors = append(errors, fmt.Errorf(
				"step %d (%s): policy step 'script' must be a non-empty string",
				index+1, step.Name))
		}
	}

	return errors
}

// validateTerraformStep validates a terraform step configuration
func (v *WorkflowValidator) validateTerraformStep(index int, step types.Step) []error {
	var errors []error

	// Terraform steps need working_dir (either in WorkingDir field or config)
	hasWorkingDir := step.WorkingDir != ""
	if !hasWorkingDir {
		if workingDirValue, ok := step.Config["working_dir"]; !ok || workingDirValue == nil || workingDirValue == "" {
			errors = append(errors, fmt.Errorf(
				"step %d (%s): terraform step requires 'working_dir' in config or workingDir field",
				index+1, step.Name))
		}
	}

	// Terraform steps must have an operation
	if operation, ok := step.Config["operation"]; !ok || operation == nil || operation == "" {
		errors = append(errors, fmt.Errorf(
			"step %d (%s): terraform step requires 'operation' in config (init, plan, apply, or destroy)",
			index+1, step.Name))
	} else {
		// Validate operation is one of the allowed values
		if opStr, ok := operation.(string); ok {
			validOps := map[string]bool{"init": true, "plan": true, "apply": true, "destroy": true}
			if !validOps[opStr] {
				errors = append(errors, fmt.Errorf(
					"step %d (%s): terraform operation '%s' is invalid (must be: init, plan, apply, or destroy)",
					index+1, step.Name, opStr))
			}
		}
	}

	return errors
}

// validateKubernetesStep validates a kubernetes step configuration
func (v *WorkflowValidator) validateKubernetesStep(index int, step types.Step) []error {
	var errors []error

	// Kubernetes steps must have an operation
	if operation, ok := step.Config["operation"]; !ok || operation == nil || operation == "" {
		errors = append(errors, fmt.Errorf(
			"step %d (%s): kubernetes step requires 'operation' in config (apply, delete, etc.)",
			index+1, step.Name))
	}

	// Kubernetes steps must have a manifest or namespace
	hasManifest := step.Config["manifest"] != nil && step.Config["manifest"] != ""
	hasNamespace := step.Config["namespace"] != nil && step.Config["namespace"] != ""

	if !hasManifest && !hasNamespace {
		errors = append(errors, fmt.Errorf(
			"step %d (%s): kubernetes step requires 'manifest' or 'namespace' in config",
			index+1, step.Name))
	}

	return errors
}

// validateAnsibleStep validates an ansible step configuration
func (v *WorkflowValidator) validateAnsibleStep(index int, step types.Step) []error {
	var errors []error

	// Ansible steps should have a playbook
	if playbook, ok := step.Config["playbook"]; !ok || playbook == nil || playbook == "" {
		errors = append(errors, fmt.Errorf(
			"step %d (%s): ansible step should have 'playbook' in config",
			index+1, step.Name))
	}

	return errors
}

// FormatValidationErrors formats validation errors into a human-readable string
func FormatValidationErrors(workflowName string, errors []error) string {
	if len(errors) == 0 {
		return ""
	}

	result := fmt.Sprintf("Workflow '%s' validation failed with %d error(s):\n", workflowName, len(errors))
	for _, err := range errors {
		result += fmt.Sprintf("  - %s\n", err.Error())
	}

	return result
}
