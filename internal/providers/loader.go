package providers

import (
	"fmt"
	"innominatus/internal/types"
	"innominatus/internal/workflow"
	"innominatus/pkg/sdk"
	"os"
	"path/filepath"
	"strings"

	"github.com/Masterminds/semver/v3"
	"gopkg.in/yaml.v3"
)

// Loader loads provider manifests from filesystem
type Loader struct {
	coreVersion string
}

// NewLoader creates a new provider loader
func NewLoader(coreVersion string) *Loader {
	return &Loader{
		coreVersion: coreVersion,
	}
}

// LoadFromFile loads a provider manifest from a YAML file
func (l *Loader) LoadFromFile(path string) (*sdk.Provider, error) {
	// Read file
	// #nosec G304 -- path is user-provided config file path, validated by caller
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read provider file %s: %w", path, err)
	}

	// Parse YAML
	var provider sdk.Provider
	if err := yaml.Unmarshal(data, &provider); err != nil {
		return nil, fmt.Errorf("failed to parse provider YAML: %w", err)
	}

	// Migrate old format to new (backward compatibility)
	l.migrateProvider(&provider)

	// Validate provider
	if err := provider.Validate(); err != nil {
		return nil, fmt.Errorf("invalid provider manifest: %w", err)
	}

	// Check version compatibility
	if err := l.checkCompatibility(&provider); err != nil {
		return nil, fmt.Errorf("provider compatibility check failed: %w", err)
	}

	// Validate all workflow files (use directory of provider.yaml as base)
	providerDir := filepath.Dir(path)
	if err := l.validateProviderWorkflows(providerDir, &provider); err != nil {
		return nil, fmt.Errorf("provider workflow validation failed: %w", err)
	}

	return &provider, nil
}

// LoadFromDirectory loads all provider manifests from a directory
func (l *Loader) LoadFromDirectory(dirPath string) ([]*sdk.Provider, error) {
	var providers []*sdk.Provider

	// Check if directory exists
	info, err := os.Stat(dirPath)
	if err != nil {
		if os.IsNotExist(err) {
			return providers, nil // Empty list, not an error
		}
		return nil, fmt.Errorf("failed to stat directory %s: %w", dirPath, err)
	}

	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", dirPath)
	}

	// Find all provider.yaml files (also support legacy platform.yaml)
	err = filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Process provider.yaml or legacy platform.yaml files
		if !info.IsDir() && (info.Name() == "provider.yaml" || info.Name() == "provider.yml" || info.Name() == "platform.yaml" || info.Name() == "platform.yml") {
			provider, err := l.LoadFromFile(path)
			if err != nil {
				// Log warning but continue with other providers
				fmt.Printf("Warning: failed to load provider from %s: %v\n", path, err)
				return nil
			}
			providers = append(providers, provider)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory %s: %w", dirPath, err)
	}

	// Early conflict detection: check for duplicate capability claims within loaded providers
	if err := l.checkProviderConflicts(providers); err != nil {
		return nil, fmt.Errorf("provider conflicts detected: %w", err)
	}

	return providers, nil
}

// checkProviderConflicts performs early conflict detection on loaded providers
func (l *Loader) checkProviderConflicts(providers []*sdk.Provider) error {
	// Build map of resource type -> set of providers claiming it
	resourceTypeMap := make(map[string]map[string]bool)

	for _, provider := range providers {
		providerName := provider.Metadata.Name

		// Check simple format
		for _, resourceType := range provider.Capabilities.ResourceTypes {
			if resourceTypeMap[resourceType] == nil {
				resourceTypeMap[resourceType] = make(map[string]bool)
			}
			resourceTypeMap[resourceType][providerName] = true
		}

		// Check advanced format
		for _, rtc := range provider.Capabilities.ResourceTypeCapabilities {
			// Only check primary types, not aliases
			if rtc.AliasFor == "" {
				if resourceTypeMap[rtc.Type] == nil {
					resourceTypeMap[rtc.Type] = make(map[string]bool)
				}
				resourceTypeMap[rtc.Type][providerName] = true
			}
		}
	}

	// Detect conflicts
	var conflicts []string
	for resourceType, providerSet := range resourceTypeMap {
		if len(providerSet) > 1 {
			providerList := make([]string, 0, len(providerSet))
			for provider := range providerSet {
				providerList = append(providerList, provider)
			}
			conflicts = append(conflicts, fmt.Sprintf("resource type '%s' claimed by: %v", resourceType, providerList))
		}
	}

	if len(conflicts) > 0 {
		return fmt.Errorf("capability conflicts:\n  - %v", conflicts)
	}

	return nil
}

// checkCompatibility verifies the provider is compatible with the core version
func (l *Loader) checkCompatibility(provider *sdk.Provider) error {
	// Skip compatibility check for development builds
	if l.coreVersion == "dev" || l.coreVersion == "unknown" {
		return nil
	}

	// Parse core version
	coreVer, err := semver.NewVersion(l.coreVersion)
	if err != nil {
		return fmt.Errorf("invalid core version %s: %w", l.coreVersion, err)
	}

	// Parse min/max compatibility versions
	minVer, err := semver.NewVersion(provider.Compatibility.MinCoreVersion)
	if err != nil {
		return fmt.Errorf("invalid minCoreVersion %s: %w", provider.Compatibility.MinCoreVersion, err)
	}

	maxVer, err := semver.NewVersion(provider.Compatibility.MaxCoreVersion)
	if err != nil {
		return fmt.Errorf("invalid maxCoreVersion %s: %w", provider.Compatibility.MaxCoreVersion, err)
	}

	// Check compatibility
	if coreVer.LessThan(minVer) {
		return fmt.Errorf(
			"provider %s requires core version >= %s, but running %s",
			provider.Metadata.Name,
			provider.Compatibility.MinCoreVersion,
			l.coreVersion,
		)
	}

	if coreVer.GreaterThan(maxVer) {
		return fmt.Errorf(
			"provider %s requires core version <= %s, but running %s",
			provider.Metadata.Name,
			provider.Compatibility.MaxCoreVersion,
			l.coreVersion,
		)
	}

	return nil
}

// LoadBuiltinProvider loads the built-in provider from the default location
func (l *Loader) LoadBuiltinProvider() (*sdk.Provider, error) {
	// Try current directory first (provider.yaml or legacy platform.yaml)
	if _, err := os.Stat("provider.yaml"); err == nil {
		return l.LoadFromFile("provider.yaml")
	}
	if _, err := os.Stat("platform.yaml"); err == nil {
		return l.LoadFromFile("platform.yaml")
	}

	// Try ./providers/builtin/
	if _, err := os.Stat("providers/builtin/provider.yaml"); err == nil {
		return l.LoadFromFile("providers/builtin/provider.yaml")
	}
	if _, err := os.Stat("providers/builtin/platform.yaml"); err == nil {
		return l.LoadFromFile("providers/builtin/platform.yaml")
	}

	// Try legacy ./platforms/builtin/
	if _, err := os.Stat("platforms/builtin/provider.yaml"); err == nil {
		return l.LoadFromFile("platforms/builtin/provider.yaml")
	}
	if _, err := os.Stat("platforms/builtin/platform.yaml"); err == nil {
		return l.LoadFromFile("platforms/builtin/platform.yaml")
	}

	return nil, fmt.Errorf("builtin provider.yaml not found in current directory, providers/builtin/, or platforms/builtin/")
}

// migrateProvider migrates old provider format to new unified workflows format
// This provides backward compatibility for providers using provisioners[] and goldenpaths[] fields
func (l *Loader) migrateProvider(provider *sdk.Provider) {
	// If workflows are already populated, no migration needed
	if len(provider.Workflows) > 0 {
		return
	}

	// Migrate goldenpaths to workflows with category="goldenpath"
	if len(provider.GoldenPaths) > 0 {
		for _, gp := range provider.GoldenPaths {
			workflow := gp // GoldenPathMetadata is now an alias for WorkflowMetadata
			if workflow.Category == "" {
				workflow.Category = "goldenpath"
			}
			provider.Workflows = append(provider.Workflows, workflow)
		}
	}

	// Note: We don't automatically migrate provisioners to workflows because
	// provisioners don't have a workflow file reference. Product teams should
	// manually add workflow files and update their provider.yaml to use workflows.
	// The old provisioners[] field will continue to work for backward compatibility.
}

// validateProviderWorkflows validates all workflow files referenced by a provider
func (l *Loader) validateProviderWorkflows(providerDir string, provider *sdk.Provider) error {
	validator := workflow.NewWorkflowValidator()
	var allErrors []string

	for _, workflowMeta := range provider.Workflows {
		// Construct full path to workflow file
		workflowPath := filepath.Join(providerDir, workflowMeta.File)

		// Read and parse workflow YAML
		data, err := os.ReadFile(workflowPath)
		if err != nil {
			return fmt.Errorf("workflow '%s': failed to read file %s: %w",
				workflowMeta.Name, workflowPath, err)
		}

		var wf types.Workflow
		if err := yaml.Unmarshal(data, &wf); err != nil {
			return fmt.Errorf("workflow '%s': failed to parse YAML: %w",
				workflowMeta.Name, err)
		}

		// Validate workflow
		if errors := validator.ValidateWorkflow(&wf); len(errors) > 0 {
			formatted := workflow.FormatValidationErrors(workflowMeta.Name, errors)
			allErrors = append(allErrors, formatted)
		}
	}

	if len(allErrors) > 0 {
		return fmt.Errorf("provider has invalid workflows:\n%s",
			strings.Join(allErrors, "\n"))
	}

	return nil
}
