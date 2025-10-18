package sdk

// Platform represents a platform implementation with its metadata and capabilities
// Platforms are defined via platform.yaml manifests
type Platform struct {
	// APIVersion is the schema version (e.g., "innominatus.io/v1")
	APIVersion string `yaml:"apiVersion" json:"apiVersion"`

	// Kind must be "Platform"
	Kind string `yaml:"kind" json:"kind"`

	// Metadata contains platform identification and versioning
	Metadata PlatformMetadata `yaml:"metadata" json:"metadata"`

	// Compatibility defines core version requirements
	Compatibility PlatformCompatibility `yaml:"compatibility" json:"compatibility"`

	// Provisioners lists the resource provisioners provided by this platform
	Provisioners []ProvisionerMetadata `yaml:"provisioners" json:"provisioners"`

	// GoldenPaths lists the workflow templates provided by this platform
	GoldenPaths []GoldenPathMetadata `yaml:"goldenpaths,omitempty" json:"goldenpaths,omitempty"`

	// Configuration contains platform-specific configuration
	Configuration map[string]interface{} `yaml:"configuration,omitempty" json:"configuration,omitempty"`
}

// PlatformMetadata contains identification and versioning information
type PlatformMetadata struct {
	// Name is the unique identifier for this platform
	// Example: "aws", "azure", "acme-internal"
	Name string `yaml:"name" json:"name"`

	// Version is the semantic version of this platform
	// Example: "1.2.3", "2.0.0-beta.1"
	Version string `yaml:"version" json:"version"`

	// Description provides a human-readable description
	Description string `yaml:"description,omitempty" json:"description,omitempty"`

	// Author identifies the platform maintainer
	Author string `yaml:"author,omitempty" json:"author,omitempty"`

	// Homepage is the URL to the platform documentation
	Homepage string `yaml:"homepage,omitempty" json:"homepage,omitempty"`

	// Repository is the source code repository URL
	Repository string `yaml:"repository,omitempty" json:"repository,omitempty"`

	// License identifies the software license
	License string `yaml:"license,omitempty" json:"license,omitempty"`

	// Tags are searchable keywords for discovery
	Tags []string `yaml:"tags,omitempty" json:"tags,omitempty"`
}

// PlatformCompatibility defines version constraints for core compatibility
type PlatformCompatibility struct {
	// MinCoreVersion is the minimum compatible core version
	// Example: "1.0.0"
	MinCoreVersion string `yaml:"minCoreVersion" json:"minCoreVersion"`

	// MaxCoreVersion is the maximum compatible core version
	// Example: "2.0.0"
	MaxCoreVersion string `yaml:"maxCoreVersion" json:"maxCoreVersion"`
}

// GoldenPathMetadata describes a workflow template provided by the platform
type GoldenPathMetadata struct {
	// Name is the unique identifier for this golden path
	Name string `yaml:"name" json:"name"`

	// File is the path to the workflow YAML file
	File string `yaml:"file" json:"file"`

	// Version is the semantic version of this golden path
	Version string `yaml:"version" json:"version"`

	// Description provides a human-readable description
	Description string `yaml:"description,omitempty" json:"description,omitempty"`

	// Category groups golden paths (deployment, cleanup, environment, etc.)
	Category string `yaml:"category,omitempty" json:"category,omitempty"`

	// Tags are searchable keywords
	Tags []string `yaml:"tags,omitempty" json:"tags,omitempty"`
}

// Validate checks if the platform manifest is valid
func (p *Platform) Validate() error {
	if p.APIVersion == "" {
		return ErrInvalidPlatform("apiVersion is required")
	}
	if p.Kind != "Platform" {
		return ErrInvalidPlatform("kind must be 'Platform'")
	}
	if p.Metadata.Name == "" {
		return ErrInvalidPlatform("metadata.name is required")
	}
	if p.Metadata.Version == "" {
		return ErrInvalidPlatform("metadata.version is required")
	}
	if p.Compatibility.MinCoreVersion == "" {
		return ErrInvalidPlatform("compatibility.minCoreVersion is required")
	}
	if len(p.Provisioners) == 0 {
		return ErrInvalidPlatform("at least one provisioner is required")
	}

	// Validate provisioners
	for i, prov := range p.Provisioners {
		if prov.Name == "" {
			return ErrInvalidPlatform("provisioners[%d].name is required", i)
		}
		if prov.Type == "" {
			return ErrInvalidPlatform("provisioners[%d].type is required", i)
		}
		if prov.Version == "" {
			return ErrInvalidPlatform("provisioners[%d].version is required", i)
		}
	}

	return nil
}

// GetProvisionerByType finds a provisioner by its type
func (p *Platform) GetProvisionerByType(resourceType string) *ProvisionerMetadata {
	for i := range p.Provisioners {
		if p.Provisioners[i].Type == resourceType {
			return &p.Provisioners[i]
		}
	}
	return nil
}

// GetProvisionerByName finds a provisioner by its name
func (p *Platform) GetProvisionerByName(name string) *ProvisionerMetadata {
	for i := range p.Provisioners {
		if p.Provisioners[i].Name == name {
			return &p.Provisioners[i]
		}
	}
	return nil
}
