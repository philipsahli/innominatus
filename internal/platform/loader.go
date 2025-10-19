package platform

import (
	"fmt"
	"innominatus/pkg/sdk"
	"os"
	"path/filepath"

	"github.com/Masterminds/semver/v3"
	"gopkg.in/yaml.v3"
)

// Loader loads platform manifests from filesystem
type Loader struct {
	coreVersion string
}

// NewLoader creates a new platform loader
func NewLoader(coreVersion string) *Loader {
	return &Loader{
		coreVersion: coreVersion,
	}
}

// LoadFromFile loads a platform manifest from a YAML file
func (l *Loader) LoadFromFile(path string) (*sdk.Platform, error) {
	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read platform file %s: %w", path, err)
	}

	// Parse YAML
	var platform sdk.Platform
	if err := yaml.Unmarshal(data, &platform); err != nil {
		return nil, fmt.Errorf("failed to parse platform YAML: %w", err)
	}

	// Validate platform
	if err := platform.Validate(); err != nil {
		return nil, fmt.Errorf("invalid platform manifest: %w", err)
	}

	// Check version compatibility
	if err := l.checkCompatibility(&platform); err != nil {
		return nil, fmt.Errorf("platform compatibility check failed: %w", err)
	}

	return &platform, nil
}

// LoadFromDirectory loads all platform manifests from a directory
func (l *Loader) LoadFromDirectory(dirPath string) ([]*sdk.Platform, error) {
	var platforms []*sdk.Platform

	// Check if directory exists
	info, err := os.Stat(dirPath)
	if err != nil {
		if os.IsNotExist(err) {
			return platforms, nil // Empty list, not an error
		}
		return nil, fmt.Errorf("failed to stat directory %s: %w", dirPath, err)
	}

	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", dirPath)
	}

	// Find all platform.yaml files
	err = filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Only process platform.yaml files
		if !info.IsDir() && (info.Name() == "platform.yaml" || info.Name() == "platform.yml") {
			platform, err := l.LoadFromFile(path)
			if err != nil {
				// Log warning but continue with other platforms
				fmt.Printf("Warning: failed to load platform from %s: %v\n", path, err)
				return nil
			}
			platforms = append(platforms, platform)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory %s: %w", dirPath, err)
	}

	return platforms, nil
}

// checkCompatibility verifies the platform is compatible with the core version
func (l *Loader) checkCompatibility(platform *sdk.Platform) error {
	// Parse core version
	coreVer, err := semver.NewVersion(l.coreVersion)
	if err != nil {
		return fmt.Errorf("invalid core version %s: %w", l.coreVersion, err)
	}

	// Parse min/max compatibility versions
	minVer, err := semver.NewVersion(platform.Compatibility.MinCoreVersion)
	if err != nil {
		return fmt.Errorf("invalid minCoreVersion %s: %w", platform.Compatibility.MinCoreVersion, err)
	}

	maxVer, err := semver.NewVersion(platform.Compatibility.MaxCoreVersion)
	if err != nil {
		return fmt.Errorf("invalid maxCoreVersion %s: %w", platform.Compatibility.MaxCoreVersion, err)
	}

	// Check compatibility
	if coreVer.LessThan(minVer) {
		return fmt.Errorf(
			"platform %s requires core version >= %s, but running %s",
			platform.Metadata.Name,
			platform.Compatibility.MinCoreVersion,
			l.coreVersion,
		)
	}

	if coreVer.GreaterThan(maxVer) {
		return fmt.Errorf(
			"platform %s requires core version <= %s, but running %s",
			platform.Metadata.Name,
			platform.Compatibility.MaxCoreVersion,
			l.coreVersion,
		)
	}

	return nil
}

// LoadBuiltinPlatform loads the built-in platform from the default location
func (l *Loader) LoadBuiltinPlatform() (*sdk.Platform, error) {
	// Try current directory first
	if _, err := os.Stat("platform.yaml"); err == nil {
		return l.LoadFromFile("platform.yaml")
	}

	// Try ./platforms/builtin/platform.yaml
	if _, err := os.Stat("platforms/builtin/platform.yaml"); err == nil {
		return l.LoadFromFile("platforms/builtin/platform.yaml")
	}

	return nil, fmt.Errorf("builtin platform.yaml not found in current directory or platforms/builtin/")
}
