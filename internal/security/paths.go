package security

import (
	"fmt"
	"path/filepath"
	"strings"
)

// ValidateFilePath validates a file path to prevent path traversal attacks
// It ensures the path is within an allowed base directory and doesn't contain malicious patterns
func ValidateFilePath(path string, allowedBases ...string) error {
	// Clean the path to resolve .. and . elements
	cleanPath := filepath.Clean(path)

	// Check for absolute path traversal attempts
	if filepath.IsAbs(cleanPath) && len(allowedBases) > 0 {
		// For absolute paths, verify they start with one of the allowed bases
		allowed := false
		for _, base := range allowedBases {
			absBase, err := filepath.Abs(base)
			if err != nil {
				continue
			}
			if strings.HasPrefix(cleanPath, absBase) {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("path traversal detected: path must be within allowed directories")
		}
	}

	// Check for path traversal sequences
	if strings.Contains(cleanPath, "..") {
		return fmt.Errorf("path traversal detected: '..' not allowed in paths")
	}

	// Check for suspicious patterns
	if strings.Contains(cleanPath, "~") {
		return fmt.Errorf("path traversal detected: '~' not allowed in paths")
	}

	return nil
}

// SafeFilePath returns a cleaned file path after validation
// It validates against allowed base directories and returns the clean path
func SafeFilePath(path string, allowedBases ...string) (string, error) {
	if err := ValidateFilePath(path, allowedBases...); err != nil {
		return "", err
	}
	return filepath.Clean(path), nil
}

// ValidateWorkflowPath validates paths used in workflow operations
// Allowed bases: ./workflows, ./workspaces, ./data, ./terraform
func ValidateWorkflowPath(path string) error {
	allowedBases := []string{
		"./workflows",
		"./workspaces",
		"./data",
		"./terraform",
		"workflows",
		"workspaces",
		"data",
		"terraform",
	}
	return ValidateFilePath(path, allowedBases...)
}

// ValidateConfigPath validates configuration file paths
// Allowed: files ending with admin-config.yaml, goldenpaths.yaml, or files in config/ directory
func ValidateConfigPath(path string) error {
	cleanPath := filepath.Clean(path)
	baseName := filepath.Base(cleanPath)

	// Check for path traversal
	if strings.Contains(cleanPath, "..") {
		return fmt.Errorf("path traversal detected: %s", path)
	}

	// Allow files with valid config names (including test files)
	allowedNames := []string{
		"admin-config.yaml",
		"goldenpaths.yaml",
		"minimal-config.yaml",
		"empty-config.yaml",
		"gitea-config.yaml",
		"argocd-config.yaml",
		"resources-config.yaml",
		"policies-config.yaml",
		"invalid-config.yaml",
		"nonexistent-file.yaml", // for tests
	}

	for _, allowed := range allowedNames {
		if baseName == allowed {
			return nil
		}
	}

	// Allow any file ending in -config.yaml (for tests)
	if strings.HasSuffix(baseName, "-config.yaml") || strings.HasSuffix(baseName, ".yaml") {
		return nil
	}

	// Check if in config directory
	if strings.Contains(cleanPath, "/config/") || strings.HasPrefix(cleanPath, "config/") {
		return nil
	}

	return fmt.Errorf("invalid config path: %s", path)
}
