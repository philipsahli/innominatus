package resources

import (
	"context"
	"fmt"
	"innominatus/internal/database"
)

// UpdateExternalResourceState updates the external state and reference URL of a delegated resource
// This is a convenience wrapper around the repository method with additional context support
func UpdateExternalResourceState(
	ctx context.Context,
	repo *database.ResourceRepository,
	id int64,
	externalState string,
	referenceURL string,
) error {
	if repo == nil {
		return fmt.Errorf("resource repository is nil")
	}

	// Validate external state
	validStates := map[string]bool{
		database.ExternalStateWaitingExternal:  true,
		database.ExternalStateBuildingExternal: true,
		database.ExternalStateHealthy:          true,
		database.ExternalStateError:            true,
		database.ExternalStateUnknown:          true,
	}

	if !validStates[externalState] {
		return fmt.Errorf("invalid external state: %s", externalState)
	}

	// Update the resource
	err := repo.UpdateExternalResourceState(id, externalState, referenceURL)
	if err != nil {
		return fmt.Errorf("failed to update external resource state: %w", err)
	}

	fmt.Printf("✅ Updated resource %d external state to '%s' (ref: %s)\n", id, externalState, referenceURL)
	return nil
}

// GetDelegatedResourcesByApp retrieves all delegated resources for a specific application
func GetDelegatedResourcesByApp(
	ctx context.Context,
	repo *database.ResourceRepository,
	appName string,
) ([]*database.ResourceInstance, error) {
	if repo == nil {
		return nil, fmt.Errorf("resource repository is nil")
	}

	resources, err := repo.GetDelegatedResources(appName)
	if err != nil {
		return nil, fmt.Errorf("failed to get delegated resources for app %s: %w", appName, err)
	}

	return resources, nil
}

// FilterResourcesByTypeAndApp filters resources by type (native, delegated, external)
func FilterResourcesByTypeAndApp(
	ctx context.Context,
	repo *database.ResourceRepository,
	appName string,
	resourceType string,
) ([]*database.ResourceInstance, error) {
	if repo == nil {
		return nil, fmt.Errorf("resource repository is nil")
	}

	// Validate resource type
	validTypes := map[string]bool{
		database.ResourceTypeNative:    true,
		database.ResourceTypeDelegated: true,
		database.ResourceTypeExternal:  true,
	}

	if !validTypes[resourceType] {
		return nil, fmt.Errorf("invalid resource type: %s", resourceType)
	}

	resources, err := repo.FilterResourcesByType(appName, resourceType)
	if err != nil {
		return nil, fmt.Errorf("failed to filter resources: %w", err)
	}

	return resources, nil
}

// SetResourceTypeDelegated is a helper to create a delegated resource instance
func SetResourceTypeDelegated(resource *database.ResourceInstance, provider, referenceURL string) {
	resource.Type = database.ResourceTypeDelegated
	providerPtr := provider
	resource.Provider = &providerPtr
	if referenceURL != "" {
		refPtr := referenceURL
		resource.ReferenceURL = &refPtr
	}
	externalState := database.ExternalStateWaitingExternal
	resource.ExternalState = &externalState
}
