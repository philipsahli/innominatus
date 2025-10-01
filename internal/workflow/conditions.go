package workflow

import (
	"fmt"
	"innominatus/internal/types"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// ExecutionContext holds the context for evaluating conditions
type ExecutionContext struct {
	PreviousStepStatus map[string]string // Map of step name -> status ("success", "failed", "skipped")
	PreviousStepOutputs map[string]string // Map of step name -> output
	Environment        map[string]string // Environment variables
	WorkflowStatus     string            // Overall workflow status
}

// NewExecutionContext creates a new execution context
func NewExecutionContext() *ExecutionContext {
	return &ExecutionContext{
		PreviousStepStatus:  make(map[string]string),
		PreviousStepOutputs: make(map[string]string),
		Environment:         make(map[string]string),
		WorkflowStatus:      "running",
	}
}

// SetStepStatus records the status of a completed step
func (ctx *ExecutionContext) SetStepStatus(stepName, status string) {
	ctx.PreviousStepStatus[stepName] = status
}

// SetStepOutput records the output of a completed step
func (ctx *ExecutionContext) SetStepOutput(stepName, output string) {
	ctx.PreviousStepOutputs[stepName] = output
}

// ShouldExecuteStep determines if a step should be executed based on its conditions
func (ctx *ExecutionContext) ShouldExecuteStep(step types.Step) (bool, string) {
	// Merge step environment variables with context environment
	mergedEnv := make(map[string]string)
	for k, v := range ctx.Environment {
		mergedEnv[k] = v
	}
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
func (ctx *ExecutionContext) replaceVariables(str string, env map[string]string) string {
	// Replace ${VAR} style
	re := regexp.MustCompile(`\$\{([^}]+)\}`)
	str = re.ReplaceAllStringFunc(str, func(match string) string {
		varName := match[2 : len(match)-1] // Remove ${ and }
		if val, exists := env[varName]; exists {
			return val
		}
		// Check system environment
		if val := os.Getenv(varName); val != "" {
			return val
		}
		return match // Return original if not found
	})

	// Replace $VAR style (word boundaries)
	re = regexp.MustCompile(`\$([A-Za-z_][A-Za-z0-9_]*)`)
	str = re.ReplaceAllStringFunc(str, func(match string) string {
		varName := match[1:] // Remove $
		if val, exists := env[varName]; exists {
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
