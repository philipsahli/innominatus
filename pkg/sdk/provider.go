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

	// Capabilities declares what resource types this provider can handle
	Capabilities ProviderCapabilities `yaml:"capabilities,omitempty" json:"capabilities,omitempty"`

	// Workflows lists all workflows provided by this provider (unified provisioners + golden paths)
	Workflows []WorkflowMetadata `yaml:"workflows,omitempty" json:"workflows,omitempty"`

	// Provisioners lists the resource provisioners provided by this provider
	// DEPRECATED: Use Workflows with category="provisioner" instead. Will be removed in v2.0.
	Provisioners []ProvisionerMetadata `yaml:"provisioners,omitempty" json:"provisioners,omitempty"`

	// GoldenPaths lists the workflow templates provided by this provider
	// DEPRECATED: Use Workflows with category="goldenpath" instead. Will be removed in v2.0.
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

// ProviderCapabilities declares what resource types this provider can handle
// Used for automatic resource-to-provider matching during orchestration
type ProviderCapabilities struct {
	// ResourceTypes lists the Score resource types this provider can provision (simple format)
	// Example: ["postgres", "postgresql", "mysql"]
	// These map to the "type" field in Score spec resources
	// For backward compatibility - if specified, all resources default to first provisioner workflow for CREATE operations
	ResourceTypes []string `yaml:"resourceTypes,omitempty" json:"resourceTypes,omitempty"`

	// ResourceTypeCapabilities provides operation-specific workflow mapping (advanced format)
	// Example: Declare different workflows for CREATE, UPDATE, DELETE operations
	// If both ResourceTypes and ResourceTypeCapabilities are specified, ResourceTypeCapabilities takes precedence
	ResourceTypeCapabilities []ResourceTypeCapability `yaml:"resourceTypeCapabilities,omitempty" json:"resourceTypeCapabilities,omitempty"`
}

// ResourceTypeCapability defines CRUD operation workflows for a specific resource type
type ResourceTypeCapability struct {
	// Type is the resource type identifier (e.g., "postgres", "namespace")
	Type string `yaml:"type" json:"type"`

	// Operations maps CRUD operations to workflows
	// Keys: "create", "read", "update", "delete"
	Operations map[string]OperationWorkflow `yaml:"operations,omitempty" json:"operations,omitempty"`

	// AliasFor indicates this is an alias for another resource type
	// Example: "postgresql" is an alias for "postgres"
	AliasFor string `yaml:"aliasFor,omitempty" json:"aliasFor,omitempty"`
}

// OperationWorkflow defines which workflow(s) handle a specific operation
type OperationWorkflow struct {
	// Workflow specifies a single workflow for this operation
	Workflow string `yaml:"workflow,omitempty" json:"workflow,omitempty"`

	// Workflows specifies multiple workflows with tag-based disambiguation
	// Used when multiple workflows can handle the same operation (e.g., different update strategies)
	Workflows []WorkflowOption `yaml:"workflows,omitempty" json:"workflows,omitempty"`

	// Default specifies the default workflow when multiple workflows are available
	Default string `yaml:"default,omitempty" json:"default,omitempty"`
}

// WorkflowOption represents a workflow with associated tags for disambiguation
type WorkflowOption struct {
	// Name is the workflow identifier
	Name string `yaml:"name" json:"name"`

	// Tags are used for workflow selection (e.g., ["scaling"], ["config"], ["ha"])
	Tags []string `yaml:"tags,omitempty" json:"tags,omitempty"`
}

// WorkflowMetadata describes a workflow provided by the provider
// Workflows can be either provisioners (single-resource) or goldenpaths (multi-resource orchestration)
type WorkflowMetadata struct {
	// Name is the unique identifier for this workflow
	Name string `yaml:"name" json:"name"`

	// File is the path to the workflow YAML file
	File string `yaml:"file" json:"file"`

	// Version is the semantic version of this workflow
	Version string `yaml:"version,omitempty" json:"version,omitempty"`

	// Description provides a human-readable description
	Description string `yaml:"description,omitempty" json:"description,omitempty"`

	// Category indicates the workflow type: "provisioner" (single-resource) or "goldenpath" (multi-resource)
	// provisioner: Creates a single resource (database, namespace, bucket, etc.)
	// goldenpath: Orchestrates multiple workflows from different providers
	Category string `yaml:"category,omitempty" json:"category,omitempty"`

	// Operation indicates the CRUD operation this workflow performs: "create", "read", "update", "delete"
	// Optional - for backward compatibility, provisioner workflows without operation default to "create"
	Operation string `yaml:"operation,omitempty" json:"operation,omitempty"`

	// Tags are searchable keywords
	Tags []string `yaml:"tags,omitempty" json:"tags,omitempty"`
}

// GoldenPathMetadata is deprecated. Use WorkflowMetadata with category="goldenpath" instead.
// DEPRECATED: Will be removed in v2.0
type GoldenPathMetadata = WorkflowMetadata

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

	// Require either workflows or provisioners (for backward compat)
	if len(p.Workflows) == 0 && len(p.Provisioners) == 0 {
		return ErrInvalidProvider("at least one workflow or provisioner is required")
	}

	// Validate workflows
	for i, wf := range p.Workflows {
		if wf.Name == "" {
			return ErrInvalidProvider("workflows[%d].name is required", i)
		}
		if wf.File == "" {
			return ErrInvalidProvider("workflows[%d].file is required", i)
		}
		// Category is optional, defaults to "provisioner" if not specified
		// But if specified, must be valid
		if wf.Category != "" && wf.Category != "provisioner" && wf.Category != "goldenpath" {
			return ErrInvalidProvider("workflows[%d].category must be 'provisioner' or 'goldenpath', got '%s'", i, wf.Category)
		}
		// Operation is optional, but if specified must be valid
		if wf.Operation != "" && wf.Operation != "create" && wf.Operation != "read" && wf.Operation != "update" && wf.Operation != "delete" {
			return ErrInvalidProvider("workflows[%d].operation must be 'create', 'read', 'update', or 'delete', got '%s'", i, wf.Operation)
		}
	}

	// Validate provisioners (deprecated but still supported)
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

	// Validate resource type capabilities for circular references
	if err := p.validateAliasReferences(); err != nil {
		return err
	}

	return nil
}

// validateAliasReferences checks for circular alias references in resourceTypeCapabilities
func (p *Provider) validateAliasReferences() error {
	// Build alias map
	aliasMap := make(map[string]string)
	for _, rtc := range p.Capabilities.ResourceTypeCapabilities {
		if rtc.AliasFor != "" {
			aliasMap[rtc.Type] = rtc.AliasFor
		}
	}

	// Check each alias for circular references
	for aliasType := range aliasMap {
		visited := make(map[string]bool)
		current := aliasType

		for {
			if visited[current] {
				// Found a cycle
				return ErrInvalidProvider("circular alias reference detected: %s", aliasType)
			}
			visited[current] = true

			next, isAlias := aliasMap[current]
			if !isAlias {
				// Reached a non-alias type, no cycle
				break
			}
			current = next
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

// CanProvisionResourceType checks if this provider declares capability for a resource type
// Supports both simple resourceTypes format and advanced resourceTypeCapabilities format
func (p *Provider) CanProvisionResourceType(resourceType string) bool {
	// Check simple format (backward compatible)
	for _, rt := range p.Capabilities.ResourceTypes {
		if rt == resourceType {
			return true
		}
	}

	// Check advanced format (resourceTypeCapabilities)
	if len(p.Capabilities.ResourceTypeCapabilities) > 0 {
		for i := range p.Capabilities.ResourceTypeCapabilities {
			rtc := &p.Capabilities.ResourceTypeCapabilities[i]
			if rtc.Type == resourceType {
				return true
			}
			// Check if this is an alias pointing to another type
			// (e.g., postgresql is an alias for postgres)
			if rtc.AliasFor == resourceType {
				return true
			}
		}
	}

	return false
}

// GetProvisionerWorkflow finds the provisioner workflow for automatic resource provisioning
// Returns the first workflow with category="provisioner"
func (p *Provider) GetProvisionerWorkflow() *WorkflowMetadata {
	for i := range p.Workflows {
		if p.Workflows[i].Category == "provisioner" || p.Workflows[i].Category == "" {
			return &p.Workflows[i]
		}
	}
	return nil
}

// GetWorkflowForOperation finds the workflow for a specific resource type and operation
// Supports both simple resourceTypes format (backward compat) and advanced resourceTypeCapabilities format
// Parameters:
//   - resourceType: The resource type (e.g., "postgres")
//   - operation: The CRUD operation ("create", "read", "update", "delete")
//   - tags: Optional tags for workflow disambiguation when multiple workflows exist for an operation
//
// Returns the workflow name, or empty string if not found
func (p *Provider) GetWorkflowForOperation(resourceType, operation string, tags []string) string {
	// First try advanced resourceTypeCapabilities format
	if len(p.Capabilities.ResourceTypeCapabilities) > 0 {
		return p.getWorkflowFromCapabilities(resourceType, operation, tags)
	}

	// Fallback to simple resourceTypes format (backward compatibility)
	// Simple format only supports CREATE operation with first provisioner workflow
	if operation == "create" || operation == "" {
		for _, rt := range p.Capabilities.ResourceTypes {
			if rt == resourceType {
				// Return first provisioner workflow
				for i := range p.Workflows {
					wf := &p.Workflows[i]
					if wf.Category == "provisioner" || wf.Category == "" {
						// Check if workflow operation matches (or is unspecified, defaulting to create)
						if wf.Operation == "create" || wf.Operation == "" {
							return wf.Name
						}
					}
				}
			}
		}
	}

	return ""
}

// getWorkflowFromCapabilities resolves workflow using advanced resourceTypeCapabilities format
func (p *Provider) getWorkflowFromCapabilities(resourceType, operation string, tags []string) string {
	// Find the resource type capability
	var capability *ResourceTypeCapability
	for i := range p.Capabilities.ResourceTypeCapabilities {
		rtc := &p.Capabilities.ResourceTypeCapabilities[i]
		if rtc.Type == resourceType {
			capability = rtc
			break
		}
		// Check if this is an alias
		if rtc.AliasFor == resourceType {
			// Follow alias to find the primary type
			capability = p.findPrimaryCapability(rtc.AliasFor)
			break
		}
	}

	if capability == nil {
		return ""
	}

	// Handle aliases - resolve to primary type
	if capability.AliasFor != "" {
		primaryCapability := p.findPrimaryCapability(capability.AliasFor)
		if primaryCapability != nil {
			capability = primaryCapability
		}
	}

	// Get operation workflow
	opWorkflow, exists := capability.Operations[operation]
	if !exists {
		return ""
	}

	// Case 1: Single workflow specified
	if opWorkflow.Workflow != "" {
		return opWorkflow.Workflow
	}

	// Case 2: Multiple workflows with tag-based disambiguation
	if len(opWorkflow.Workflows) > 0 {
		// Try to find workflow matching tags
		if len(tags) > 0 {
			bestMatch := p.findBestWorkflowMatch(opWorkflow.Workflows, tags)
			if bestMatch != "" {
				return bestMatch
			}
		}

		// Use default if specified
		if opWorkflow.Default != "" {
			return opWorkflow.Default
		}

		// Return first workflow as fallback
		if len(opWorkflow.Workflows) > 0 {
			return opWorkflow.Workflows[0].Name
		}
	}

	return ""
}

// findPrimaryCapability finds the primary (non-alias) capability for a resource type
func (p *Provider) findPrimaryCapability(resourceType string) *ResourceTypeCapability {
	for i := range p.Capabilities.ResourceTypeCapabilities {
		rtc := &p.Capabilities.ResourceTypeCapabilities[i]
		if rtc.Type == resourceType && rtc.AliasFor == "" {
			return rtc
		}
	}
	return nil
}

// findBestWorkflowMatch finds the workflow that best matches the provided tags
// Returns the workflow name with the most tag matches
func (p *Provider) findBestWorkflowMatch(workflows []WorkflowOption, tags []string) string {
	bestMatch := ""
	bestScore := 0

	for _, wf := range workflows {
		score := 0
		for _, tag := range tags {
			if p.containsTag(wf.Tags, tag) {
				score++
			}
		}
		if score > bestScore {
			bestScore = score
			bestMatch = wf.Name
		}
	}

	return bestMatch
}

// containsTag checks if a tag exists in a tag list
func (p *Provider) containsTag(tags []string, tag string) bool {
	for _, t := range tags {
		if t == tag {
			return true
		}
	}
	return false
}

// SupportsOperation checks if the provider supports a specific operation for a resource type
func (p *Provider) SupportsOperation(resourceType, operation string) bool {
	// Check advanced format
	if len(p.Capabilities.ResourceTypeCapabilities) > 0 {
		for i := range p.Capabilities.ResourceTypeCapabilities {
			rtc := &p.Capabilities.ResourceTypeCapabilities[i]
			if rtc.Type == resourceType || rtc.AliasFor == resourceType {
				// If it's an alias, resolve to primary
				if rtc.AliasFor != "" {
					primaryCapability := p.findPrimaryCapability(rtc.AliasFor)
					if primaryCapability != nil {
						_, exists := primaryCapability.Operations[operation]
						return exists
					}
				}
				_, exists := rtc.Operations[operation]
				return exists
			}
		}
		return false
	}

	// Check simple format (only supports create)
	if operation == "create" || operation == "" {
		return p.CanProvisionResourceType(resourceType)
	}

	return false
}

// Legacy type aliases for backward compatibility
type Platform = Provider
type PlatformMetadata = ProviderMetadata
type PlatformCompatibility = ProviderCompatibility
