package orchestration

import (
	"fmt"

	"innominatus/internal/providers"
	"innominatus/pkg/sdk"
)

// Resolver matches resource types to providers and their workflows
type Resolver struct {
	registry *providers.Registry
}

// NewResolver creates a new resolver instance
func NewResolver(registry *providers.Registry) *Resolver {
	return &Resolver{
		registry: registry,
	}
}

// ResolveProviderForResource finds the provider and workflow for a given resource type
// Returns the provider, provisioner workflow, and any error
// Defaults to CREATE operation for backward compatibility
func (r *Resolver) ResolveProviderForResource(resourceType string) (*sdk.Provider, *sdk.WorkflowMetadata, error) {
	return r.ResolveWorkflowForOperation(resourceType, "create", nil)
}

// ResolveWorkflowForOperation finds the provider and workflow for a specific resource type and operation
// Supports CRUD operations: create, read, update, delete
// Parameters:
//   - resourceType: The resource type (e.g., "postgres", "namespace")
//   - operation: The CRUD operation ("create", "read", "update", "delete")
//   - tags: Optional tags for workflow disambiguation when multiple workflows exist
//
// Returns the provider, workflow metadata, and any error
func (r *Resolver) ResolveWorkflowForOperation(resourceType, operation string, tags []string) (*sdk.Provider, *sdk.WorkflowMetadata, error) {
	allProviders := r.registry.ListProviders()

	var matchedProviders []*sdk.Provider

	// Find all providers that declare capability for this resource type
	for _, provider := range allProviders {
		if provider.CanProvisionResourceType(resourceType) {
			matchedProviders = append(matchedProviders, provider)
		}
	}

	// Error if no provider found
	if len(matchedProviders) == 0 {
		return nil, nil, fmt.Errorf("no provider found for resource type '%s'", resourceType)
	}

	// Error if multiple providers claim the same resource type
	if len(matchedProviders) > 1 {
		providerNames := make([]string, len(matchedProviders))
		for i, p := range matchedProviders {
			providerNames[i] = p.Metadata.Name
		}
		return nil, nil, fmt.Errorf("multiple providers claim resource type '%s': %v (disambiguation needed)", resourceType, providerNames)
	}

	// Found exactly one provider
	provider := matchedProviders[0]

	// Check if provider supports the requested operation
	if !provider.SupportsOperation(resourceType, operation) {
		return nil, nil, fmt.Errorf("provider '%s' does not support operation '%s' for resource type '%s'",
			provider.Metadata.Name, operation, resourceType)
	}

	// Get the workflow for this operation
	workflowName := provider.GetWorkflowForOperation(resourceType, operation, tags)
	if workflowName == "" {
		return nil, nil, fmt.Errorf("provider '%s' declares capability for '%s' but has no workflow for operation '%s'",
			provider.Metadata.Name, resourceType, operation)
	}

	// Find the workflow metadata by name
	workflow := r.FindWorkflowByName(provider, workflowName)
	if workflow == nil {
		return nil, nil, fmt.Errorf("provider '%s' references workflow '%s' but it does not exist",
			provider.Metadata.Name, workflowName)
	}

	return provider, workflow, nil
}

// FindWorkflowByName searches for a workflow by name in the provider's workflow list
func (r *Resolver) FindWorkflowByName(provider *sdk.Provider, workflowName string) *sdk.WorkflowMetadata {
	for i := range provider.Workflows {
		if provider.Workflows[i].Name == workflowName {
			return &provider.Workflows[i]
		}
	}
	return nil
}

// ValidateProviders checks for conflicts in provider capabilities at registration time
func (r *Resolver) ValidateProviders() error {
	allProviders := r.registry.ListProviders()

	// Build map of resource type -> set of unique providers
	resourceTypeMap := make(map[string]map[string]bool)

	for _, provider := range allProviders {
		providerName := provider.Metadata.Name

		// Check simple format (backward compatible)
		for _, resourceType := range provider.Capabilities.ResourceTypes {
			if resourceTypeMap[resourceType] == nil {
				resourceTypeMap[resourceType] = make(map[string]bool)
			}
			resourceTypeMap[resourceType][providerName] = true
		}

		// Check advanced resourceTypeCapabilities format
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

	// Check for conflicts (convert sets to lists for error reporting)
	var conflicts []string
	for resourceType, providerSet := range resourceTypeMap {
		if len(providerSet) > 1 {
			// Convert set to sorted list
			providerList := make([]string, 0, len(providerSet))
			for provider := range providerSet {
				providerList = append(providerList, provider)
			}
			conflicts = append(conflicts, fmt.Sprintf("resource type '%s' claimed by: %v", resourceType, providerList))
		}
	}

	if len(conflicts) > 0 {
		return fmt.Errorf("provider capability conflicts detected:\n  - %v", conflicts)
	}

	return nil
}
