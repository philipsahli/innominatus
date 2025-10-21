package sdk

// Provider represents a provider implementation with its metadata and capabilities
// Providers are defined via provider.yaml manifests (or legacy platform.yaml)
type Provider struct {
	// APIVersion is the schema version (e.g., "innominatus.io/v1")
	APIVersion string `yaml:"apiVersion" json:"apiVersion"`

	// Kind must be "Provider" (or legacy "Platform")
	Kind string `yaml:"kind" json:"kind"`

	// Metadata contains provider identification and versioning
	Metadata ProviderMetadata `yaml:"metadata" json:"metadata"`

	// Compatibility defines core version requirements
	Compatibility ProviderCompatibility `yaml:"compatibility" json:"compatibility"`

	// Provisioners lists the resource provisioners provided by this provider
	Provisioners []ProvisionerMetadata `yaml:"provisioners" json:"provisioners"`

	// GoldenPaths lists the workflow templates provided by this provider
	GoldenPaths []GoldenPathMetadata `yaml:"goldenpaths,omitempty" json:"goldenpaths,omitempty"`

	// Configuration contains provider-specific configuration
	Configuration map[string]interface{} `yaml:"configuration,omitempty" json:"configuration,omitempty"`
}

// ProviderMetadata contains identification and versioning information
type ProviderMetadata struct {
	// Name is the unique identifier for this provider
	// Example: "aws", "azure", "ecommerce", "analytics"
	Name string `yaml:"name" json:"name"`

	// Version is the semantic version of this provider
	// Example: "1.2.3", "2.0.0-beta.1"
	Version string `yaml:"version" json:"version"`

	// Category indicates provider type: "infrastructure" or "service"
	// infrastructure: AWS, Azure, GCP (platform teams)
	// service: ecommerce, analytics, ML (product teams)
	Category string `yaml:"category,omitempty" json:"category,omitempty"`

	// Description provides a human-readable description
	Description string `yaml:"description,omitempty" json:"description,omitempty"`

	// Author identifies the provider maintainer
	Author string `yaml:"author,omitempty" json:"author,omitempty"`

	// Homepage is the URL to the provider documentation
	Homepage string `yaml:"homepage,omitempty" json:"homepage,omitempty"`

	// Repository is the source code repository URL
	Repository string `yaml:"repository,omitempty" json:"repository,omitempty"`

	// License identifies the software license
	License string `yaml:"license,omitempty" json:"license,omitempty"`

	// Tags are searchable keywords for discovery
	Tags []string `yaml:"tags,omitempty" json:"tags,omitempty"`
}

// ProviderCompatibility defines version constraints for core compatibility
type ProviderCompatibility struct {
	// MinCoreVersion is the minimum compatible core version
	// Example: "1.0.0"
	MinCoreVersion string `yaml:"minCoreVersion" json:"minCoreVersion"`

	// MaxCoreVersion is the maximum compatible core version
	// Example: "2.0.0"
	MaxCoreVersion string `yaml:"maxCoreVersion" json:"maxCoreVersion"`
}

// GoldenPathMetadata describes a workflow template provided by the provider
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

// Validate checks if the provider manifest is valid
func (p *Provider) Validate() error {
	if p.APIVersion == "" {
		return ErrInvalidProvider("apiVersion is required")
	}
	// Accept both "Provider" and legacy "Platform" for backward compatibility
	if p.Kind != "Provider" && p.Kind != "Platform" {
		return ErrInvalidProvider("kind must be 'Provider' (or legacy 'Platform')")
	}
	if p.Metadata.Name == "" {
		return ErrInvalidProvider("metadata.name is required")
	}
	if p.Metadata.Version == "" {
		return ErrInvalidProvider("metadata.version is required")
	}
	if p.Compatibility.MinCoreVersion == "" {
		return ErrInvalidProvider("compatibility.minCoreVersion is required")
	}
	if len(p.Provisioners) == 0 {
		return ErrInvalidProvider("at least one provisioner is required")
	}

	// Validate provisioners
	for i, prov := range p.Provisioners {
		if prov.Name == "" {
			return ErrInvalidProvider("provisioners[%d].name is required", i)
		}
		if prov.Type == "" {
			return ErrInvalidProvider("provisioners[%d].type is required", i)
		}
		if prov.Version == "" {
			return ErrInvalidProvider("provisioners[%d].version is required", i)
		}
	}

	return nil
}

// GetProvisionerByType finds a provisioner by its type
func (p *Provider) GetProvisionerByType(resourceType string) *ProvisionerMetadata {
	for i := range p.Provisioners {
		if p.Provisioners[i].Type == resourceType {
			return &p.Provisioners[i]
		}
	}
	return nil
}

// GetProvisionerByName finds a provisioner by its name
func (p *Provider) GetProvisionerByName(name string) *ProvisionerMetadata {
	for i := range p.Provisioners {
		if p.Provisioners[i].Name == name {
			return &p.Provisioners[i]
		}
	}
	return nil
}

// Legacy type aliases for backward compatibility
type Platform = Provider
type PlatformMetadata = ProviderMetadata
type PlatformCompatibility = ProviderCompatibility
