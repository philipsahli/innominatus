package validation

import (
	"fmt"
	"os"
)

// StartupValidator orchestrates all configuration validations at startup
type StartupValidator struct {
	suite *ValidationSuite
}

// NewStartupValidator creates a comprehensive startup validator
func NewStartupValidator() (*StartupValidator, error) {
	suite := NewValidationSuite("Application Startup")

	// Add admin configuration validator if config exists
	if _, err := os.Stat("admin-config.yaml"); err == nil {
		adminValidator, err := NewAdminConfigValidator("admin-config.yaml")
		if err != nil {
			return nil, fmt.Errorf("failed to create admin config validator: %w", err)
		}
		suite.AddValidator(adminValidator)
	}

	// Add golden paths validator if config exists
	if _, err := os.Stat("goldenpaths.yaml"); err == nil {
		goldenPathsValidator, err := NewGoldenPathsValidator("")
		if err != nil {
			return nil, fmt.Errorf("failed to create golden paths validator: %w", err)
		}
		suite.AddValidator(goldenPathsValidator)
	}

	// Add database validator (always present as it uses environment variables)
	dbValidator := NewDatabaseValidator()
	suite.AddValidator(dbValidator)

	// Add users validator if users file exists
	if _, err := os.Stat("users.yaml"); err == nil {
		usersValidator, err := NewUsersValidator("")
		if err != nil {
			return nil, fmt.Errorf("failed to create users validator: %w", err)
		}
		suite.AddValidator(usersValidator)
	}

	return &StartupValidator{suite: suite}, nil
}

// ValidateAll runs all startup validations
func (sv *StartupValidator) ValidateAll() *ValidationSummary {
	return sv.suite.ValidateAll()
}

// ValidateAllWithExit runs all validations and exits if there are errors
func (sv *StartupValidator) ValidateAllWithExit() *ValidationSummary {
	summary := sv.ValidateAll()

	// Always print the summary
	summary.PrintSummary()

	// Exit with error code if validation failed
	if !summary.Valid {
		fmt.Printf("❌ Configuration validation failed. Please fix the errors above before starting the application.\n")
		os.Exit(1)
	}

	if summary.WarningCount > 0 {
		fmt.Printf("⚠️  Configuration validation passed with %d warnings. Consider addressing these for optimal operation.\n", summary.WarningCount)
	}

	return summary
}

// ValidateConfiguration is a convenience function for validating all configurations
func ValidateConfiguration() *ValidationSummary {
	validator, err := NewStartupValidator()
	if err != nil {
		fmt.Printf("❌ Failed to create startup validator: %v\n", err)
		os.Exit(1)
	}

	return validator.ValidateAll()
}

// ValidateConfigurationWithExit is a convenience function that exits on validation failure
func ValidateConfigurationWithExit() {
	validator, err := NewStartupValidator()
	if err != nil {
		fmt.Printf("❌ Failed to create startup validator: %v\n", err)
		os.Exit(1)
	}

	validator.ValidateAllWithExit()
}

// ValidationMode represents different validation modes
type ValidationMode int

const (
	ValidationModeStartup ValidationMode = iota // Full validation on startup
	ValidationModeFast                          // Fast validation for CLI operations
	ValidationModeRequired                      // Only required/critical validations
)

// ValidateWithMode runs validation with a specific mode
func ValidateWithMode(mode ValidationMode) *ValidationSummary {
	switch mode {
	case ValidationModeStartup:
		return ValidateConfiguration()
	case ValidationModeFast:
		return ValidateConfigurationFast()
	case ValidationModeRequired:
		return ValidateConfigurationRequired()
	default:
		return ValidateConfiguration()
	}
}

// ValidateConfigurationFast runs fast validations (skips connectivity tests)
func ValidateConfigurationFast() *ValidationSummary {
	suite := NewValidationSuite("Fast Configuration Check")

	// Only add file-based validations (skip database connectivity)
	if _, err := os.Stat("admin-config.yaml"); err == nil {
		if adminValidator, err := NewAdminConfigValidator("admin-config.yaml"); err == nil {
			suite.AddValidator(adminValidator)
		}
	}

	if _, err := os.Stat("goldenpaths.yaml"); err == nil {
		if goldenPathsValidator, err := NewGoldenPathsValidator(""); err == nil {
			suite.AddValidator(goldenPathsValidator)
		}
	}

	if _, err := os.Stat("users.yaml"); err == nil {
		if usersValidator, err := NewUsersValidator(""); err == nil {
			suite.AddValidator(usersValidator)
		}
	}

	return suite.ValidateAll()
}

// ValidateConfigurationRequired runs only critical validations
func ValidateConfigurationRequired() *ValidationSummary {
	suite := NewValidationSuite("Required Configuration Check")

	// Only add critical validations
	if _, err := os.Stat("users.yaml"); err == nil {
		if usersValidator, err := NewUsersValidator(""); err == nil {
			suite.AddValidator(usersValidator)
		}
	} else {
		// If no users file, create a minimal validator to report this critical issue
		suite.AddValidator(&RequiredFileValidator{
			fileName:  "users.yaml",
			component: "User Authentication",
		})
	}

	return suite.ValidateAll()
}

// RequiredFileValidator is a simple validator for checking required files
type RequiredFileValidator struct {
	fileName  string
	component string
}

func (v *RequiredFileValidator) Validate() *ValidationResult {
	result := &ValidationResult{
		Valid:     true,
		Errors:    make([]string, 0),
		Warnings:  make([]string, 0),
		Component: v.component,
	}

	if _, err := os.Stat(v.fileName); os.IsNotExist(err) {
		result.Errors = append(result.Errors, fmt.Sprintf("Required file '%s' not found", v.fileName))
		result.Valid = false
	}

	return result
}

func (v *RequiredFileValidator) GetComponent() string {
	return v.component
}