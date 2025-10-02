package errors

import (
	"fmt"
	"strings"
)

// SuggestionEngine provides intelligent error suggestions
type SuggestionEngine struct {
	patterns map[string][]string
}

// NewSuggestionEngine creates a new suggestion engine
func NewSuggestionEngine() *SuggestionEngine {
	engine := &SuggestionEngine{
		patterns: make(map[string][]string),
	}
	engine.initializePatterns()
	return engine
}

// initializePatterns sets up common error patterns and their suggestions
func (se *SuggestionEngine) initializePatterns() {
	// Validation errors
	se.patterns["missing required field"] = []string{
		"Check the Score specification format: https://score.dev",
		"Ensure all required fields (apiVersion, metadata, containers) are present",
		"Run `innominatus-ctl validate --explain <file>` for detailed validation",
	}

	se.patterns["invalid yaml"] = []string{
		"Check for proper YAML indentation (use spaces, not tabs)",
		"Ensure all strings with special characters are quoted",
		"Validate YAML syntax at https://www.yamllint.com/",
	}

	se.patterns["resource not found"] = []string{
		"Verify the resource name matches the definition in your Score spec",
		"Check if the resource was provisioned successfully",
		"Run `innominatus-ctl list-resources` to see available resources",
	}

	se.patterns["workflow failed"] = []string{
		"Check workflow logs: `innominatus-ctl logs <workflow-id>`",
		"Verify all workflow prerequisites are met",
		"Try running with --verbose flag for more details",
	}

	se.patterns["connection refused"] = []string{
		"Ensure the innominatus server is running: `innominatus`",
		"Check if the server port (8081) is accessible",
		"Verify network connectivity and firewall settings",
	}

	se.patterns["authentication failed"] = []string{
		"Check your credentials in users.yaml",
		"Verify your API key is valid: echo $IDP_API_KEY",
		"Try logging in again with correct credentials",
	}

	se.patterns["timeout"] = []string{
		"Increase the timeout value in your configuration",
		"Check if the target service is responsive",
		"Retry the operation - network issues may be transient",
	}

	se.patterns["database connection"] = []string{
		"Verify PostgreSQL is running and accessible",
		"Check database connection settings (DB_USER, DB_NAME environment variables)",
		"Try starting with --disable-db flag for in-memory mode",
	}

	se.patterns["file not found"] = []string{
		"Verify the file path is correct and the file exists",
		"Check file permissions - ensure the file is readable",
		"Use absolute paths or ensure you're in the correct directory",
	}

	se.patterns["circular dependency"] = []string{
		"Review your workflow steps for circular references",
		"Use `innominatus-ctl analyze <file>` to visualize dependencies",
		"Reorganize workflow steps to break the dependency cycle",
	}

	se.patterns["resource conflict"] = []string{
		"Another workflow may be modifying the same resource",
		"Wait for the conflicting operation to complete",
		"Check active workflows: `innominatus-ctl list-workflows`",
	}

	se.patterns["permission denied"] = []string{
		"Check your user role and permissions",
		"Ensure you have the required access level for this operation",
		"Contact your platform administrator for access",
	}

	se.patterns["kubernetes"] = []string{
		"Verify kubectl is configured and can access the cluster",
		"Check cluster connectivity: `kubectl cluster-info`",
		"Ensure you have the correct context: `kubectl config current-context`",
	}

	se.patterns["docker"] = []string{
		"Ensure Docker Desktop is running",
		"Check Docker daemon status: `docker ps`",
		"Verify Docker has sufficient resources (memory, disk)",
	}

	se.patterns["port already in use"] = []string{
		"Another process is using the port",
		"Stop the conflicting process or use a different port",
		"Find the process: `lsof -i :<port>` or `netstat -tulpn | grep <port>`",
	}
}

// GetSuggestions returns suggestions for an error message
func (se *SuggestionEngine) GetSuggestions(errorMessage string) []string {
	lowerMsg := strings.ToLower(errorMessage)

	var suggestions []string
	for pattern, patternSuggestions := range se.patterns {
		if strings.Contains(lowerMsg, pattern) {
			suggestions = append(suggestions, patternSuggestions...)
		}
	}

	// If no specific suggestions, provide general ones
	if len(suggestions) == 0 {
		suggestions = []string{
			"Run with --verbose flag for more details",
			"Check the documentation: https://docs.innominatus.dev",
			"Review recent changes that might have caused this issue",
		}
	}

	return suggestions
}

// EnrichError adds intelligent suggestions to a RichError
func (se *SuggestionEngine) EnrichError(err *RichError) *RichError {
	suggestions := se.GetSuggestions(err.Message)
	for _, suggestion := range suggestions {
		// Only add if not already present
		exists := false
		for _, existing := range err.Suggestions {
			if existing == suggestion {
				exists = true
				break
			}
		}
		if !exists {
			err.Suggestions = append(err.Suggestions, suggestion)
		}
	}
	return err
}

// ValidationSuggestion provides suggestions for validation errors
type ValidationSuggestion struct {
	Field       string
	ActualValue interface{}
	Expected    string
	Example     string
}

// Format formats a validation suggestion
func (vs *ValidationSuggestion) Format() string {
	return fmt.Sprintf(
		"Field '%s' has value '%v', but expected: %s\nExample: %s",
		vs.Field,
		vs.ActualValue,
		vs.Expected,
		vs.Example,
	)
}

// CommonValidationSuggestions provides suggestions for common validation errors
func CommonValidationSuggestions(field string, constraint string) []string {
	suggestions := []string{}

	switch constraint {
	case "required":
		suggestions = append(suggestions, fmt.Sprintf("Add the required field '%s' to your configuration", field))
		suggestions = append(suggestions, "Check the Score specification for required fields")

	case "format":
		suggestions = append(suggestions, fmt.Sprintf("Ensure field '%s' follows the correct format", field))
		suggestions = append(suggestions, "Check for typos or incorrect syntax")

	case "type":
		suggestions = append(suggestions, fmt.Sprintf("Field '%s' must be the correct data type", field))
		suggestions = append(suggestions, "Verify you're using the right type (string, number, array, object)")

	case "enum":
		suggestions = append(suggestions, fmt.Sprintf("Field '%s' must be one of the allowed values", field))
		suggestions = append(suggestions, "Check the documentation for valid values")

	case "pattern":
		suggestions = append(suggestions, fmt.Sprintf("Field '%s' doesn't match the required pattern", field))
		suggestions = append(suggestions, "Verify the format matches the expected pattern (e.g., naming conventions)")
	}

	return suggestions
}

// WorkflowSuggestion provides suggestions for workflow errors
type WorkflowSuggestion struct {
	StepName    string
	StepType    string
	FailureType string
}

// GetWorkflowSuggestions returns suggestions based on workflow failure
func GetWorkflowSuggestions(stepType, failureType string) []string {
	suggestions := []string{}

	switch stepType {
	case "kubernetes":
		suggestions = append(suggestions, "Verify Kubernetes cluster is accessible")
		suggestions = append(suggestions, "Check namespace exists and you have permissions")
		suggestions = append(suggestions, "Review kubectl configuration")

	case "terraform":
		suggestions = append(suggestions, "Verify Terraform configuration is valid")
		suggestions = append(suggestions, "Check provider credentials and permissions")
		suggestions = append(suggestions, "Review Terraform state for conflicts")

	case "ansible":
		suggestions = append(suggestions, "Verify Ansible playbook syntax")
		suggestions = append(suggestions, "Check SSH connectivity to target hosts")
		suggestions = append(suggestions, "Review inventory configuration")

	case "gitea-repo":
		suggestions = append(suggestions, "Verify Gitea service is running")
		suggestions = append(suggestions, "Check Gitea credentials and permissions")
		suggestions = append(suggestions, "Ensure repository doesn't already exist")

	case "argocd-app":
		suggestions = append(suggestions, "Verify ArgoCD is installed and accessible")
		suggestions = append(suggestions, "Check ArgoCD credentials")
		suggestions = append(suggestions, "Ensure repository is accessible by ArgoCD")
	}

	// Add failure-type specific suggestions
	switch failureType {
	case "timeout":
		suggestions = append(suggestions, "Increase timeout value in step configuration")
		suggestions = append(suggestions, "Check if the operation is taking longer than expected")

	case "unauthorized":
		suggestions = append(suggestions, "Verify authentication credentials")
		suggestions = append(suggestions, "Check service permissions and access policies")

	case "not_found":
		suggestions = append(suggestions, "Verify the resource exists")
		suggestions = append(suggestions, "Check for typos in resource names")
	}

	return suggestions
}

// ResourceConflictSuggestion provides suggestions for resource conflicts
func ResourceConflictSuggestion(resourceName, operation string) []string {
	return []string{
		fmt.Sprintf("Resource '%s' is being modified by another operation", resourceName),
		"Wait for the conflicting operation to complete",
		"Check active workflows: `innominatus-ctl list-workflows`",
		"If the conflict persists, check resource locks in the database",
	}
}
