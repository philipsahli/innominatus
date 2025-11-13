package workflow

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"innominatus/internal/types"

	"github.com/sirupsen/logrus"
)

var (
	//varPattern extracts both ${VAR} and $VAR patterns
	varPattern = regexp.MustCompile(`\$\{[^}]+\}|\$[A-Za-z_][A-Za-z0-9_]*`)
)

// ExtractVariableReferences finds all variable references in a string
// Returns both ${VAR} and $VAR patterns
func ExtractVariableReferences(text string) []string {
	return varPattern.FindAllString(text, -1)
}

// IsStrictMode checks if STRICT_VALIDATION environment variable is enabled
// Default: true (strict mode)
// Set to "false", "0", or "FALSE" to enable lenient mode
func IsStrictMode() bool {
	value := strings.ToLower(os.Getenv("STRICT_VALIDATION"))
	if value == "" {
		return true // Default is strict
	}
	return value != "false" && value != "0"
}

// ValidateVariableExists checks if a variable reference can be resolved
// Returns error if variable is not found and strict mode is enabled
// Returns nil in lenient mode (logs warning instead)
func (e *ExecutionContext) ValidateVariableExists(varRef string, env map[string]string) error {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// Strip ${ } or $ prefix
	varName := strings.TrimPrefix(varRef, "$")
	if strings.HasPrefix(varName, "{") && strings.HasSuffix(varName, "}") {
		varName = strings.TrimSuffix(strings.TrimPrefix(varName, "{"), "}")
	}

	// Check if empty (${} case)
	if varName == "" {
		err := fmt.Errorf("undefined variable: empty variable reference")
		if IsStrictMode() {
			return err
		}
		logrus.Warnf("Validation warning: %v", err)
		return nil
	}

	// Try workflow variables (workflow.VAR)
	if strings.HasPrefix(varName, "workflow.") {
		key := strings.TrimPrefix(varName, "workflow.")
		if _, found := e.WorkflowVariables[key]; found {
			return nil
		}
		err := fmt.Errorf("undefined variable: %s (workflow variable '%s' not found)", varRef, key)
		if IsStrictMode() {
			return err
		}
		logrus.Warnf("Validation warning: %v", err)
		return nil
	}

	// Try step outputs (step.output)
	if strings.Contains(varName, ".") && !strings.HasPrefix(varName, "resources.") {
		parts := strings.SplitN(varName, ".", 2)
		if len(parts) == 2 {
			stepName := parts[0]
			outputName := parts[1]
			if stepOutputs, found := e.PreviousStepOutputs[stepName]; found {
				if _, found := stepOutputs[outputName]; found {
					return nil
				}
				err := fmt.Errorf("undefined variable: %s (step '%s' has no output '%s')", varRef, stepName, outputName)
				if IsStrictMode() {
					return err
				}
				logrus.Warnf("Validation warning: %v", err)
				return nil
			}
			err := fmt.Errorf("undefined variable: %s (step '%s' outputs not available)", varRef, stepName)
			if IsStrictMode() {
				return err
			}
			logrus.Warnf("Validation warning: %v", err)
			return nil
		}
	}

	// Try resource outputs (resources.name.attr)
	if strings.HasPrefix(varName, "resources.") {
		parts := strings.SplitN(strings.TrimPrefix(varName, "resources."), ".", 2)
		if len(parts) == 2 {
			resourceName := parts[0]
			attrName := parts[1]
			if resourceOutputs, found := e.ResourceOutputs[resourceName]; found {
				if _, found := resourceOutputs[attrName]; found {
					return nil
				}
				err := fmt.Errorf("undefined variable: %s (resource '%s' has no attribute '%s')", varRef, resourceName, attrName)
				if IsStrictMode() {
					return err
				}
				logrus.Warnf("Validation warning: %v", err)
				return nil
			}
			err := fmt.Errorf("undefined variable: %s (resource '%s' outputs not available)", varRef, resourceName)
			if IsStrictMode() {
				return err
			}
			logrus.Warnf("Validation warning: %v", err)
			return nil
		}
	}

	// Try step environment variables (passed as parameter)
	if env != nil {
		if _, found := env[varName]; found {
			return nil
		}
	}

	// Try system environment variables
	if os.Getenv(varName) != "" {
		return nil
	}

	// Variable not found
	err := fmt.Errorf("undefined variable: %s", varRef)
	if IsStrictMode() {
		return err
	}
	logrus.Warnf("Validation warning: %v", err)
	return nil
}

// ValidateStepVariables validates all variable references in a step
// Fails fast - returns error on first undefined variable (in strict mode)
// In lenient mode, logs warnings but continues
func (e *ExecutionContext) ValidateStepVariables(step types.Step, env map[string]string) error {
	// Extract all variable references from step configuration
	refs := e.extractStepVariableReferences(step)

	// Validate each reference (fail-fast)
	for _, ref := range refs {
		if err := e.ValidateVariableExists(ref, env); err != nil {
			// In strict mode, error is returned
			// In lenient mode, ValidateVariableExists logs warning and returns nil
			if err != nil {
				return fmt.Errorf("step '%s' validation failed: %w", step.Name, err)
			}
		}
	}

	return nil
}

// extractStepVariableReferences extracts all variable references from step config
// Recursively traverses maps, arrays, and string values
func (e *ExecutionContext) extractStepVariableReferences(step types.Step) []string {
	refs := []string{}

	// Extract from Config map
	if step.Config != nil {
		refs = append(refs, e.extractReferencesFromValue(step.Config)...)
	}

	// Extract from Env map
	for _, value := range step.Env {
		refs = append(refs, ExtractVariableReferences(value)...)
	}

	// Extract from SetVariables map
	for _, value := range step.SetVariables {
		refs = append(refs, ExtractVariableReferences(value)...)
	}

	// Extract from other string fields
	for _, field := range []string{
		step.Path,
		step.Playbook,
		step.Namespace,
		step.Resource,
		step.OutputDir,
		step.Repo,
		step.Branch,
		step.CommitMessage,
		step.Workspace,
		step.RepoName,
		step.Description,
		step.Owner,
		step.AppName,
		step.RepoURL,
		step.TargetPath,
		step.Project,
		step.SyncPolicy,
		step.ManifestPath,
		step.GitBranch,
		step.When,
		step.If,
		step.Unless,
		step.OutputFile,
		step.Operation,
		step.WorkingDir,
	} {
		refs = append(refs, ExtractVariableReferences(field)...)
	}

	// Extract from Outputs array
	for _, output := range step.Outputs {
		refs = append(refs, ExtractVariableReferences(output)...)
	}

	// Extract from DependsOn array
	for _, dep := range step.DependsOn {
		refs = append(refs, ExtractVariableReferences(dep)...)
	}

	// Extract from Variables map
	if step.Variables != nil {
		refs = append(refs, e.extractReferencesFromValue(step.Variables)...)
	}

	return refs
}

// extractReferencesFromValue recursively extracts variable references from any value
func (e *ExecutionContext) extractReferencesFromValue(value interface{}) []string {
	refs := []string{}

	switch v := value.(type) {
	case string:
		refs = append(refs, ExtractVariableReferences(v)...)
	case map[string]interface{}:
		for _, val := range v {
			refs = append(refs, e.extractReferencesFromValue(val)...)
		}
	case map[string]string:
		for _, val := range v {
			refs = append(refs, ExtractVariableReferences(val)...)
		}
	case []interface{}:
		for _, item := range v {
			refs = append(refs, e.extractReferencesFromValue(item)...)
		}
	case []string:
		for _, item := range v {
			refs = append(refs, ExtractVariableReferences(item)...)
		}
	}

	return refs
}

// ValidateWorkflowVariables validates all variable references in a workflow
// Checks all steps for undefined workflow variables
// Note: Step outputs validation is deferred to runtime (executed steps provide outputs)
func (e *ExecutionContext) ValidateWorkflowVariables(workflow types.Workflow) error {
	// Initialize workflow variables in context
	e.mu.Lock()
	if e.WorkflowVariables == nil {
		e.WorkflowVariables = make(map[string]string)
	}
	for k, v := range workflow.Variables {
		e.WorkflowVariables[k] = v
	}
	e.mu.Unlock()

	// Validate each step
	for _, step := range workflow.Steps {
		// Only validate workflow variables at this stage
		// Step outputs will be validated at runtime (during execution)
		refs := e.extractStepVariableReferences(step)

		for _, ref := range refs {
			// Only validate workflow.* variables here
			if strings.Contains(ref, "workflow.") {
				if err := e.ValidateVariableExists(ref, step.Env); err != nil {
					if err != nil {
						return fmt.Errorf("workflow validation failed: %w", err)
					}
				}
			}
		}
	}

	return nil
}
