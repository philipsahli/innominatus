package security

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

// AllowedCommands defines the whitelist of commands that can be executed
var AllowedCommands = map[string]bool{
	"terraform":        true,
	"kubectl":          true,
	"ansible-playbook": true,
	"git":              true,
	"docker":           true,
	"helm":             true,
	"curl":             true,
	"lsof":             true,
	"pkill":            true,
}

// ValidateCommand checks if a command is in the allowed list
func ValidateCommand(command string) error {
	if !AllowedCommands[command] {
		return fmt.Errorf("command not allowed: %s", command)
	}
	return nil
}

// ValidateCommandArgs validates command arguments to prevent injection
func ValidateCommandArgs(args []string) error {
	// Check for dangerous patterns
	dangerousPatterns := []string{
		";",      // command chaining
		"&&",     // command chaining
		"||",     // command chaining
		"|",      // piping (allow -| for kubectl)
		"`",      // command substitution
		"$(",     // command substitution
		"<(",     // process substitution
		">(",     // process substitution
		"../",    // path traversal
		"~/",     // home directory expansion
	}

	for _, arg := range args {
		// Skip empty args
		if arg == "" {
			continue
		}

		// Allow kubectl pipe notation specifically
		if strings.HasPrefix(arg, "-|") {
			continue
		}

		for _, pattern := range dangerousPatterns {
			if strings.Contains(arg, pattern) {
				return fmt.Errorf("dangerous pattern in argument: %s (contains %s)", arg, pattern)
			}
		}

		// Check for null bytes
		if strings.Contains(arg, "\x00") {
			return fmt.Errorf("null byte in argument: %s", arg)
		}
	}

	return nil
}

// SafeCommand creates an exec.Cmd after validating the command and arguments
func SafeCommand(command string, args ...string) (*exec.Cmd, error) {
	// Validate command
	if err := ValidateCommand(command); err != nil {
		return nil, err
	}

	// Validate arguments
	if err := ValidateCommandArgs(args); err != nil {
		return nil, err
	}

	// Create and return the command
	return exec.Command(command, args...), nil // #nosec G204 - command and args validated above
}

// ValidateResourceName validates Kubernetes resource names
func ValidateResourceName(name string) error {
	// Kubernetes resource names must be lowercase alphanumeric with hyphens
	validName := regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)
	if !validName.MatchString(name) {
		return fmt.Errorf("invalid resource name: %s (must be lowercase alphanumeric with hyphens)", name)
	}

	if len(name) > 253 {
		return fmt.Errorf("resource name too long: %s (max 253 characters)", name)
	}

	return nil
}

// ValidateNamespace validates Kubernetes namespace names
func ValidateNamespace(namespace string) error {
	return ValidateResourceName(namespace)
}
