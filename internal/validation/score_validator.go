package validation

import (
	"fmt"
	"innominatus/internal/errors"
	"innominatus/internal/types"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// ScoreValidator validates Score specifications with detailed error reporting
type ScoreValidator struct {
	filePath string
	content  []byte
	lines    []string
	spec     *types.ScoreSpec
}

// NewScoreValidator creates a new Score validator
func NewScoreValidator(filePath string) (*ScoreValidator, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(content), "\n")

	return &ScoreValidator{
		filePath: filePath,
		content:  content,
		lines:    lines,
	}, nil
}

// Validate performs comprehensive validation with detailed error reporting
func (sv *ScoreValidator) Validate() ([]*errors.RichError, error) {
	var validationErrors []*errors.RichError

	// Step 1: Parse YAML structure
	var rawSpec map[string]interface{}
	if err := yaml.Unmarshal(sv.content, &rawSpec); err != nil {
		// Try to extract line number from YAML error
		lineNum, colNum := extractYAMLErrorLocation(err.Error())
		richErr := errors.NewRichError(errors.CategoryValidation, errors.SeverityError, "Invalid YAML syntax")
		richErr.WithLocation(sv.filePath, lineNum, colNum, sv.getLine(lineNum))
		richErr.WithSuggestion("Check for proper YAML indentation (use spaces, not tabs)")
		richErr.WithSuggestion("Ensure all strings with special characters are quoted")
		richErr.WithSuggestion("Validate YAML syntax at https://www.yamllint.com/")
		validationErrors = append(validationErrors, richErr)
		return validationErrors, err
	}

	// Step 2: Parse into Score spec structure
	if err := yaml.Unmarshal(sv.content, &sv.spec); err != nil {
		richErr := errors.NewRichError(errors.CategoryValidation, errors.SeverityError, "Failed to parse Score specification")
		richErr.WithCause(err)
		richErr.WithSuggestion("Check the Score specification format: https://score.dev")
		richErr.WithSuggestion("Ensure all required fields are present")
		validationErrors = append(validationErrors, richErr)
		return validationErrors, err
	}

	// Step 3: Validate required fields
	validationErrors = append(validationErrors, sv.validateRequiredFields()...)

	// Step 4: Validate field formats
	validationErrors = append(validationErrors, sv.validateFieldFormats()...)

	// Step 5: Validate resources
	validationErrors = append(validationErrors, sv.validateResources()...)

	// Step 6: Validate workflows
	validationErrors = append(validationErrors, sv.validateWorkflows()...)

	// Step 7: Validate containers
	validationErrors = append(validationErrors, sv.validateContainers()...)

	// Step 8: Check for best practices
	validationErrors = append(validationErrors, sv.checkBestPractices()...)

	return validationErrors, nil
}

// validateRequiredFields checks for required Score spec fields
func (sv *ScoreValidator) validateRequiredFields() []*errors.RichError {
	var errs []*errors.RichError

	// Check apiVersion
	if sv.spec.APIVersion == "" {
		lineNum := sv.findFieldLine("apiVersion")
		err := errors.NewRichError(errors.CategoryValidation, errors.SeverityError, "Missing required field: apiVersion").
			WithLocation(sv.filePath, lineNum, 0, sv.getLine(lineNum))
		err.WithSuggestion("Add 'apiVersion: score.dev/v1b1' to your Score spec")
		err.WithSuggestion("Check the Score specification: https://score.dev")
		errs = append(errs, err)
	} else if !isValidAPIVersion(sv.spec.APIVersion) {
		lineNum := sv.findFieldLine("apiVersion")
		err := errors.NewRichError(errors.CategoryValidation, errors.SeverityError, fmt.Sprintf("Invalid apiVersion: %s", sv.spec.APIVersion)).
			WithLocation(sv.filePath, lineNum, 0, sv.getLine(lineNum))
		err.WithSuggestion("Use 'score.dev/v1b1' as the apiVersion")
		errs = append(errs, err)
	}

	// Check metadata
	if sv.spec.Metadata.Name == "" {
		lineNum := sv.findFieldLine("name")
		err := errors.NewRichError(errors.CategoryValidation, errors.SeverityError, "Missing required field: metadata.name").
			WithLocation(sv.filePath, lineNum, 0, sv.getLine(lineNum))
		err.WithSuggestion("Add a name to your application metadata")
		err.WithSuggestion("Example: metadata:\n  name: my-app")
		errs = append(errs, err)
	}

	// Check containers
	if len(sv.spec.Containers) == 0 {
		lineNum := sv.findFieldLine("containers")
		err := errors.NewRichError(errors.CategoryValidation, errors.SeverityError, "At least one container is required").
			WithLocation(sv.filePath, lineNum, 0, sv.getLine(lineNum))
		err.WithSuggestion("Add at least one container definition")
		err.WithSuggestion("Example: containers:\n  web:\n    image: nginx:latest")
		errs = append(errs, err)
	}

	return errs
}

// validateFieldFormats validates field format constraints
func (sv *ScoreValidator) validateFieldFormats() []*errors.RichError {
	var errs []*errors.RichError

	// Validate metadata.name format
	if sv.spec.Metadata.Name != "" {
		if !isValidKubernetesName(sv.spec.Metadata.Name) {
			lineNum := sv.findFieldLine("name")
			err := errors.NewRichError(errors.CategoryValidation, errors.SeverityError, fmt.Sprintf("Invalid name format: %s", sv.spec.Metadata.Name)).
				WithLocation(sv.filePath, lineNum, 0, sv.getLine(lineNum))
			err.WithSuggestion("Name must be lowercase alphanumeric with hyphens")
			err.WithSuggestion("Must start and end with alphanumeric character")
			err.WithSuggestion("Example: my-app, web-service, api-v1")
			errs = append(errs, err)
		}
	}

	return errs
}

// validateResources validates resource definitions
func (sv *ScoreValidator) validateResources() []*errors.RichError {
	var errs []*errors.RichError

	for resourceName, resource := range sv.spec.Resources {
		// Check if resource has a type
		if resource.Type == "" {
			lineNum := sv.findFieldLineInSection("resources", resourceName)
			err := errors.NewRichError(errors.CategoryValidation, errors.SeverityError, fmt.Sprintf("Resource '%s' missing type", resourceName)).
				WithLocation(sv.filePath, lineNum, 0, sv.getLine(lineNum))
			err.WithSuggestion("Add a type to the resource definition")
			err.WithSuggestion("Example: type: postgres")
			errs = append(errs, err)
		}

		// Validate common resource types
		if err := sv.validateResourceType(resourceName, resource); err != nil {
			errs = append(errs, err)
		}
	}

	return errs
}

// validateResourceType validates specific resource type requirements
func (sv *ScoreValidator) validateResourceType(name string, resource types.Resource) *errors.RichError {
	switch resource.Type {
	case "postgres", "mysql", "mongodb":
		// Database resources should have required params
		if resource.Params == nil || len(resource.Params) == 0 {
			lineNum := sv.findFieldLineInSection("resources", name)
			err := errors.NewRichError(errors.CategoryValidation, errors.SeverityError, fmt.Sprintf("Database resource '%s' should have parameters", name)).WithLocation(sv.filePath, lineNum, 0, sv.getLine(lineNum))
			err.Severity = errors.SeverityWarning
			err.WithSuggestion("Consider adding database version, size, or other configuration")
			return err
		}
	}
	return nil
}

// validateWorkflows validates workflow definitions
func (sv *ScoreValidator) validateWorkflows() []*errors.RichError {
	var errs []*errors.RichError

	for workflowName, workflow := range sv.spec.Workflows {
		if len(workflow.Steps) == 0 {
			lineNum := sv.findFieldLineInSection("workflows", workflowName)
			err := errors.NewRichError(errors.CategoryValidation, errors.SeverityError, fmt.Sprintf("Workflow '%s' has no steps", workflowName)).WithLocation(sv.filePath, lineNum, 0, sv.getLine(lineNum))
			err.WithSuggestion("Add at least one step to the workflow")
			err.WithSuggestion("Example: steps:\n  - name: deploy\n    type: kubernetes")
			errs = append(errs, err)
		}

		// Validate each step
		for i, step := range workflow.Steps {
			if step.Name == "" {
				lineNum := sv.findFieldLineInSection("workflows", fmt.Sprintf("%s.steps[%d]", workflowName, i))
				err := errors.NewRichError(errors.CategoryValidation, errors.SeverityError, "Step missing required 'name' field").WithLocation(sv.filePath, lineNum, 0, sv.getLine(lineNum))
				errs = append(errs, err)
			}

			if step.Type == "" {
				lineNum := sv.findFieldLineInSection("workflows", fmt.Sprintf("%s.steps[%d]", workflowName, i))
				err := errors.NewRichError(errors.CategoryValidation, errors.SeverityError, "Step missing required 'type' field").WithLocation(sv.filePath, lineNum, 0, sv.getLine(lineNum))
				errs = append(errs, err)
			}
		}
	}

	return errs
}

// validateContainers validates container definitions
func (sv *ScoreValidator) validateContainers() []*errors.RichError {
	var errs []*errors.RichError

	for containerName, container := range sv.spec.Containers {
		if container.Image == "" {
			lineNum := sv.findFieldLineInSection("containers", containerName)
			err := errors.NewRichError(errors.CategoryValidation, errors.SeverityError, fmt.Sprintf("Container '%s' missing image", containerName)).WithLocation(sv.filePath, lineNum, 0, sv.getLine(lineNum))
			err.WithSuggestion("Add an image to the container definition")
			errs = append(errs, err)
		}
	}

	return errs
}

// checkBestPractices provides suggestions for best practices
func (sv *ScoreValidator) checkBestPractices() []*errors.RichError {
	var errs []*errors.RichError

	// Check for image tags
	for containerName, container := range sv.spec.Containers {
		if container.Image != "" && strings.Contains(container.Image, ":latest") {
			lineNum := sv.findFieldLineInSection("containers", containerName)
			err := errors.NewRichError(errors.CategoryValidation, errors.SeverityError, fmt.Sprintf("Container '%s' uses 'latest' tag", containerName)).WithLocation(sv.filePath, lineNum, 0, sv.getLine(lineNum))
			err.Severity = errors.SeverityWarning
			err.WithSuggestion("Use specific version tags instead of 'latest' for reproducibility")
			errs = append(errs, err)
		}
	}

	return errs
}

// Helper functions

func (sv *ScoreValidator) getLine(lineNum int) string {
	if lineNum <= 0 || lineNum > len(sv.lines) {
		return ""
	}
	return sv.lines[lineNum-1]
}

func (sv *ScoreValidator) findFieldLine(fieldName string) int {
	for i, line := range sv.lines {
		if strings.Contains(line, fieldName+":") {
			return i + 1
		}
	}
	return 1
}

func (sv *ScoreValidator) findFieldLineInSection(section, field string) int {
	inSection := false
	for i, line := range sv.lines {
		if strings.Contains(line, section+":") {
			inSection = true
			continue
		}
		if inSection && strings.Contains(line, field) {
			return i + 1
		}
	}
	return 1
}

func extractYAMLErrorLocation(errMsg string) (int, int) {
	// Try to extract line and column from YAML error message
	// Format: "yaml: line X: message" or "yaml: line X, column Y: message"
	lineRegex := regexp.MustCompile(`line (\d+)`)
	colRegex := regexp.MustCompile(`column (\d+)`)

	lineMatch := lineRegex.FindStringSubmatch(errMsg)
	colMatch := colRegex.FindStringSubmatch(errMsg)

	line := 1
	col := 0

	if len(lineMatch) > 1 {
		fmt.Sscanf(lineMatch[1], "%d", &line)
	}
	if len(colMatch) > 1 {
		fmt.Sscanf(colMatch[1], "%d", &col)
	}

	return line, col
}

func isValidAPIVersion(version string) bool {
	validVersions := []string{"score.dev/v1b1", "score.dev/v1"}
	for _, v := range validVersions {
		if version == v {
			return true
		}
	}
	return false
}

func isValidKubernetesName(name string) bool {
	// Kubernetes name must be lowercase alphanumeric with hyphens
	// Must start and end with alphanumeric
	pattern := `^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`
	matched, _ := regexp.MatchString(pattern, name)
	return matched && len(name) <= 253
}