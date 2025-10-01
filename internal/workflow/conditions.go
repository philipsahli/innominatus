package workflow

import (
	"fmt"
	"innominatus/internal/types"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// ExecutionContext holds the context for evaluating conditions and sharing variables
type ExecutionContext struct {
	PreviousStepStatus  map[string]string            // Map of step name -> status ("success", "failed", "skipped")
	PreviousStepOutputs map[string]map[string]string // Map of step name -> map of output variables
	Environment         map[string]string            // Environment variables
	WorkflowVariables   map[string]string            // Workflow-level variables
	WorkflowStatus      string                       // Overall workflow status
}

// NewExecutionContext creates a new execution context
func NewExecutionContext() *ExecutionContext {
	return &ExecutionContext{
		PreviousStepStatus:  make(map[string]string),
		PreviousStepOutputs: make(map[string]map[string]string),
		Environment:         make(map[string]string),
		WorkflowVariables:   make(map[string]string),
		WorkflowStatus:      "running",
	}
}

// SetWorkflowVariables initializes workflow-level variables
func (ctx *ExecutionContext) SetWorkflowVariables(variables map[string]string) {
	for k, v := range variables {
		ctx.WorkflowVariables[k] = v
	}
}

// SetVariable sets a single workflow variable
func (ctx *ExecutionContext) SetVariable(key, value string) {
	ctx.WorkflowVariables[key] = value
}

// GetVariable gets a workflow variable
func (ctx *ExecutionContext) GetVariable(key string) (string, bool) {
	value, exists := ctx.WorkflowVariables[key]
	return value, exists
}

// SetStepStatus records the status of a completed step
func (ctx *ExecutionContext) SetStepStatus(stepName, status string) {
	ctx.PreviousStepStatus[stepName] = status
}

// SetStepOutputs records multiple outputs from a completed step
func (ctx *ExecutionContext) SetStepOutputs(stepName string, outputs map[string]string) {
	if ctx.PreviousStepOutputs[stepName] == nil {
		ctx.PreviousStepOutputs[stepName] = make(map[string]string)
	}
	for k, v := range outputs {
		ctx.PreviousStepOutputs[stepName][k] = v
	}
}

// SetStepOutput records a single output from a completed step
func (ctx *ExecutionContext) SetStepOutput(stepName, key, value string) {
	if ctx.PreviousStepOutputs[stepName] == nil {
		ctx.PreviousStepOutputs[stepName] = make(map[string]string)
	}
	ctx.PreviousStepOutputs[stepName][key] = value
}

// GetStepOutput retrieves an output value from a previous step
func (ctx *ExecutionContext) GetStepOutput(stepName, key string) (string, bool) {
	if outputs, exists := ctx.PreviousStepOutputs[stepName]; exists {
		value, found := outputs[key]
		return value, found
	}
	return "", false
}

// GetAllStepOutputs retrieves all outputs from a previous step
func (ctx *ExecutionContext) GetAllStepOutputs(stepName string) (map[string]string, bool) {
	outputs, exists := ctx.PreviousStepOutputs[stepName]
	return outputs, exists
}

// ShouldExecuteStep determines if a step should be executed based on its conditions
func (ctx *ExecutionContext) ShouldExecuteStep(step types.Step) (bool, string) {
	// Merge all variable sources (priority: step env > workflow vars > context env)
	mergedEnv := make(map[string]string)

	// Start with context environment
	for k, v := range ctx.Environment {
		mergedEnv[k] = v
	}

	// Add workflow variables
	for k, v := range ctx.WorkflowVariables {
		mergedEnv[k] = v
	}

	// Add step environment (highest priority)
	for k, v := range step.Env {
		mergedEnv[k] = v
	}

	// Evaluate "when" condition (simple keywords)
	if step.When != "" {
		shouldRun, reason := ctx.evaluateWhen(step.When)
		if !shouldRun {
			return false, reason
		}
	}

	// Evaluate "unless" condition (must be false to run)
	if step.Unless != "" {
		result, err := ctx.evaluateCondition(step.Unless, mergedEnv)
		if err != nil {
			return false, fmt.Sprintf("unless condition error: %v", err)
		}
		if result {
			return false, fmt.Sprintf("unless condition '%s' is true", step.Unless)
		}
	}

	// Evaluate "if" condition (must be true to run)
	if step.If != "" {
		result, err := ctx.evaluateCondition(step.If, mergedEnv)
		if err != nil {
			return false, fmt.Sprintf("if condition error: %v", err)
		}
		if !result {
			return false, fmt.Sprintf("if condition '%s' is false", step.If)
		}
	}

	return true, ""
}

// evaluateWhen evaluates simple "when" keywords
func (ctx *ExecutionContext) evaluateWhen(when string) (bool, string) {
	when = strings.ToLower(strings.TrimSpace(when))

	switch when {
	case "always":
		return true, ""

	case "on_success", "success":
		// Run only if all previous steps succeeded
		for stepName, status := range ctx.PreviousStepStatus {
			if status == "failed" {
				return false, fmt.Sprintf("when=on_success but step '%s' failed", stepName)
			}
		}
		return true, ""

	case "on_failure", "failure":
		// Run only if any previous step failed
		for _, status := range ctx.PreviousStepStatus {
			if status == "failed" {
				return true, ""
			}
		}
		return false, "when=on_failure but no steps have failed"

	case "manual":
		// Requires manual approval (for now, we skip manual steps)
		return false, "when=manual requires manual approval"

	default:
		return false, fmt.Sprintf("unknown when condition: %s", when)
	}
}

// evaluateCondition evaluates a boolean condition expression
func (ctx *ExecutionContext) evaluateCondition(condition string, env map[string]string) (bool, error) {
	condition = strings.TrimSpace(condition)

	// Replace environment variables
	condition = ctx.replaceVariables(condition, env)

	// Simple expression evaluation
	// Supports: ==, !=, <, >, <=, >=, contains, startsWith, endsWith, matches

	// Check for comparison operators
	operators := []string{"==", "!=", "<=", ">=", "<", ">"}
	for _, op := range operators {
		if strings.Contains(condition, op) {
			parts := strings.SplitN(condition, op, 2)
			if len(parts) == 2 {
				left := strings.TrimSpace(parts[0])
				right := strings.TrimSpace(parts[1])
				return ctx.compareValues(left, right, op)
			}
		}
	}

	// Check for string operations
	if strings.Contains(condition, "contains") {
		return ctx.evaluateContains(condition)
	}
	if strings.Contains(condition, "startsWith") {
		return ctx.evaluateStartsWith(condition)
	}
	if strings.Contains(condition, "endsWith") {
		return ctx.evaluateEndsWith(condition)
	}
	if strings.Contains(condition, "matches") {
		return ctx.evaluateMatches(condition)
	}

	// Check for boolean literals
	conditionLower := strings.ToLower(condition)
	if conditionLower == "true" {
		return true, nil
	}
	if conditionLower == "false" {
		return false, nil
	}

	// Check for step status references (e.g., "step1.success")
	if strings.Contains(condition, ".") {
		parts := strings.SplitN(condition, ".", 2)
		if len(parts) == 2 {
			stepName := strings.TrimSpace(parts[0])
			statusCheck := strings.ToLower(strings.TrimSpace(parts[1]))

			status, exists := ctx.PreviousStepStatus[stepName]
			if !exists {
				return false, fmt.Errorf("step '%s' not found in context", stepName)
			}

			switch statusCheck {
			case "success", "succeeded":
				return status == "success", nil
			case "failed", "failure":
				return status == "failed", nil
			case "skipped":
				return status == "skipped", nil
			default:
				return false, fmt.Errorf("unknown status check: %s", statusCheck)
			}
		}
	}

	// If condition is just a variable name, check if it exists and is non-empty
	if !strings.ContainsAny(condition, " \t\n") {
		value, exists := env[condition]
		return exists && value != "" && value != "false" && value != "0", nil
	}

	return false, fmt.Errorf("unable to evaluate condition: %s", condition)
}

// replaceVariables replaces ${VAR} and $VAR with their values
// Supports: $VAR, ${VAR}, ${step.output}, ${workflow.VAR}
func (ctx *ExecutionContext) replaceVariables(str string, env map[string]string) string {
	// Replace ${VAR} style (including step.output and workflow.VAR)
	re := regexp.MustCompile(`\$\{([^}]+)\}`)
	str = re.ReplaceAllStringFunc(str, func(match string) string {
		varName := match[2 : len(match)-1] // Remove ${ and }

		// Check for step output reference: ${step.output}
		if strings.Contains(varName, ".") {
			parts := strings.SplitN(varName, ".", 2)
			if len(parts) == 2 {
				stepName := parts[0]
				outputKey := parts[1]

				// Check if it's a workflow variable reference
				if stepName == "workflow" {
					if val, exists := ctx.WorkflowVariables[outputKey]; exists {
						return val
					}
				} else {
					// Check step outputs
					if val, found := ctx.GetStepOutput(stepName, outputKey); found {
						return val
					}
				}
			}
		}

		// Check env variables (step, workflow, context, system)
		if val, exists := env[varName]; exists {
			return val
		}
		if val, exists := ctx.WorkflowVariables[varName]; exists {
			return val
		}
		// Check system environment
		if val := os.Getenv(varName); val != "" {
			return val
		}
		return match // Return original if not found
	})

	// Replace $VAR style (word boundaries only, not dot notation)
	re = regexp.MustCompile(`\$([A-Za-z_][A-Za-z0-9_]*)(?:\.([A-Za-z_][A-Za-z0-9_]*))?`)
	str = re.ReplaceAllStringFunc(str, func(match string) string {
		// Check if this is a step.output reference
		if strings.Contains(match, ".") {
			parts := strings.Split(match[1:], ".") // Remove $
			if len(parts) == 2 {
				stepName := parts[0]
				outputKey := parts[1]

				// Check if it's a workflow variable reference
				if stepName == "workflow" {
					if val, exists := ctx.WorkflowVariables[outputKey]; exists {
						return val
					}
				} else {
					// Check step outputs
					if val, found := ctx.GetStepOutput(stepName, outputKey); found {
						return val
					}
				}
			}
			return match
		}

		varName := match[1:] // Remove $
		if val, exists := env[varName]; exists {
			return val
		}
		if val, exists := ctx.WorkflowVariables[varName]; exists {
			return val
		}
		// Check system environment
		if val := os.Getenv(varName); val != "" {
			return val
		}
		return match // Return original if not found
	})

	return str
}

// compareValues compares two values using the specified operator
func (ctx *ExecutionContext) compareValues(left, right, op string) (bool, error) {
	left = strings.Trim(left, `"'`)
	right = strings.Trim(right, `"'`)

	// Try numeric comparison first
	leftNum, leftErr := strconv.ParseFloat(left, 64)
	rightNum, rightErr := strconv.ParseFloat(right, 64)

	if leftErr == nil && rightErr == nil {
		switch op {
		case "==":
			return leftNum == rightNum, nil
		case "!=":
			return leftNum != rightNum, nil
		case "<":
			return leftNum < rightNum, nil
		case ">":
			return leftNum > rightNum, nil
		case "<=":
			return leftNum <= rightNum, nil
		case ">=":
			return leftNum >= rightNum, nil
		}
	}

	// String comparison
	switch op {
	case "==":
		return left == right, nil
	case "!=":
		return left != right, nil
	case "<":
		return left < right, nil
	case ">":
		return left > right, nil
	case "<=":
		return left <= right, nil
	case ">=":
		return left >= right, nil
	default:
		return false, fmt.Errorf("unknown operator: %s", op)
	}
}

// evaluateContains checks if a string contains a substring
func (ctx *ExecutionContext) evaluateContains(condition string) (bool, error) {
	// Format: "string contains substring" or string.contains(substring)
	if strings.Contains(condition, ".contains(") {
		re := regexp.MustCompile(`(.+)\.contains\((.+)\)`)
		matches := re.FindStringSubmatch(condition)
		if len(matches) == 3 {
			str := strings.Trim(matches[1], `"' `)
			substr := strings.Trim(matches[2], `"' `)
			return strings.Contains(str, substr), nil
		}
	}

	parts := strings.Split(condition, "contains")
	if len(parts) == 2 {
		str := strings.Trim(parts[0], `"' `)
		substr := strings.Trim(parts[1], `"' `)
		return strings.Contains(str, substr), nil
	}

	return false, fmt.Errorf("invalid contains expression: %s", condition)
}

// evaluateStartsWith checks if a string starts with a prefix
func (ctx *ExecutionContext) evaluateStartsWith(condition string) (bool, error) {
	if strings.Contains(condition, ".startsWith(") {
		re := regexp.MustCompile(`(.+)\.startsWith\((.+)\)`)
		matches := re.FindStringSubmatch(condition)
		if len(matches) == 3 {
			str := strings.Trim(matches[1], `"' `)
			prefix := strings.Trim(matches[2], `"' `)
			return strings.HasPrefix(str, prefix), nil
		}
	}

	parts := strings.Split(condition, "startsWith")
	if len(parts) == 2 {
		str := strings.Trim(parts[0], `"' `)
		prefix := strings.Trim(parts[1], `"' `)
		return strings.HasPrefix(str, prefix), nil
	}

	return false, fmt.Errorf("invalid startsWith expression: %s", condition)
}

// evaluateEndsWith checks if a string ends with a suffix
func (ctx *ExecutionContext) evaluateEndsWith(condition string) (bool, error) {
	if strings.Contains(condition, ".endsWith(") {
		re := regexp.MustCompile(`(.+)\.endsWith\((.+)\)`)
		matches := re.FindStringSubmatch(condition)
		if len(matches) == 3 {
			str := strings.Trim(matches[1], `"' `)
			suffix := strings.Trim(matches[2], `"' `)
			return strings.HasSuffix(str, suffix), nil
		}
	}

	parts := strings.Split(condition, "endsWith")
	if len(parts) == 2 {
		str := strings.Trim(parts[0], `"' `)
		suffix := strings.Trim(parts[1], `"' `)
		return strings.HasSuffix(str, suffix), nil
	}

	return false, fmt.Errorf("invalid endsWith expression: %s", condition)
}

// evaluateMatches checks if a string matches a regex pattern
func (ctx *ExecutionContext) evaluateMatches(condition string) (bool, error) {
	if strings.Contains(condition, ".matches(") {
		re := regexp.MustCompile(`(.+)\.matches\((.+)\)`)
		matches := re.FindStringSubmatch(condition)
		if len(matches) == 3 {
			str := strings.Trim(matches[1], `"' `)
			pattern := strings.Trim(matches[2], `"' `)
			matched, err := regexp.MatchString(pattern, str)
			return matched, err
		}
	}

	parts := strings.Split(condition, "matches")
	if len(parts) == 2 {
		str := strings.Trim(parts[0], `"' `)
		pattern := strings.Trim(parts[1], `"' `)
		matched, err := regexp.MatchString(pattern, str)
		return matched, err
	}

	return false, fmt.Errorf("invalid matches expression: %s", condition)
}

// InterpolateResourceParams recursively interpolates variables in resource parameters
// Supports string values containing ${step.output}, ${workflow.VAR}, $VAR references
func (ctx *ExecutionContext) InterpolateResourceParams(params map[string]interface{}, env map[string]string) map[string]interface{} {
	if params == nil {
		return nil
	}

	result := make(map[string]interface{})
	for key, value := range params {
		result[key] = ctx.interpolateValue(value, env)
	}
	return result
}

// interpolateValue recursively interpolates variables in a value of any type
func (ctx *ExecutionContext) interpolateValue(value interface{}, env map[string]string) interface{} {
	switch v := value.(type) {
	case string:
		// Interpolate variables in string
		return ctx.replaceVariables(v, env)

	case map[string]interface{}:
		// Recursively interpolate nested map
		result := make(map[string]interface{})
		for k, val := range v {
			result[k] = ctx.interpolateValue(val, env)
		}
		return result

	case []interface{}:
		// Recursively interpolate array elements
		result := make([]interface{}, len(v))
		for i, val := range v {
			result[i] = ctx.interpolateValue(val, env)
		}
		return result

	case map[interface{}]interface{}:
		// Handle YAML-style map with interface{} keys
		result := make(map[string]interface{})
		for k, val := range v {
			keyStr := fmt.Sprintf("%v", k)
			result[keyStr] = ctx.interpolateValue(val, env)
		}
		return result

	default:
		// Return non-string values as-is (numbers, booleans, etc.)
		return value
	}
}
