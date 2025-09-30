package resources

import (
	"fmt"
	"innominatus/internal/database"
	"innominatus/internal/types"
)

// Provisioner interface for resource provisioning
type Provisioner interface {
	Provision(resource *database.ResourceInstance, config map[string]interface{}, provisionedBy string) error
	Deprovision(resource *database.ResourceInstance) error
	GetStatus(resource *database.ResourceInstance) (map[string]interface{}, error)
}

// Manager handles resource lifecycle management
type Manager struct {
	resourceRepo *database.ResourceRepository
	provisioners map[string]Provisioner
}

// NewManager creates a new resource manager with built-in provisioners
func NewManager(resourceRepo *database.ResourceRepository) *Manager {
	m := &Manager{
		resourceRepo: resourceRepo,
		provisioners: make(map[string]Provisioner),
	}

	// Register built-in provisioners
	// These are owned by different infrastructure teams
	m.RegisterProvisioner("kubernetes", NewKubernetesProvisioner(resourceRepo))
	m.RegisterProvisioner("gitea-repo", NewGiteaProvisioner(resourceRepo))
	m.RegisterProvisioner("argocd-app", NewArgoCDProvisioner(resourceRepo))

	return m
}

// RegisterProvisioner registers a provisioner for a resource type
func (m *Manager) RegisterProvisioner(resourceType string, provisioner Provisioner) {
	m.provisioners[resourceType] = provisioner
	fmt.Printf("📦 Registered provisioner for resource type: %s\n", resourceType)
}

// GetProvisioner returns the provisioner for a given resource type
func (m *Manager) GetProvisioner(resourceType string) (Provisioner, error) {
	provisioner, ok := m.provisioners[resourceType]
	if !ok {
		return nil, fmt.Errorf("no provisioner registered for resource type: %s", resourceType)
	}
	return provisioner, nil
}

// checkRepository checks if the resource repository is available
func (m *Manager) checkRepository() error {
	if m == nil || m.resourceRepo == nil {
		return fmt.Errorf("resource repository is nil")
	}
	return nil
}

// CreateResourceInstance creates a single resource instance
func (m *Manager) CreateResourceInstance(appName string, resourceName string, resourceType string, config map[string]interface{}) (*database.ResourceInstance, error) {
	if err := m.checkRepository(); err != nil {
		return nil, err
	}

	// Create resource instance in database
	resourceInstance, err := m.resourceRepo.CreateResourceInstance(appName, resourceName, resourceType, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource instance: %w", err)
	}

	return resourceInstance, nil
}

// CreateResourceFromSpec creates resource instances from a Score specification
func (m *Manager) CreateResourceFromSpec(appName string, spec *types.ScoreSpec, createdBy string) error {
	if spec == nil {
		return fmt.Errorf("spec cannot be nil")
	}

	// If there are no resources, return early
	if len(spec.Resources) == 0 {
		return nil
	}

	if m.resourceRepo == nil {
		return fmt.Errorf("resource repository is nil")
	}

	for resourceName, resource := range spec.Resources {
		// Create configuration from resource parameters
		config := map[string]interface{}{
			"type":     resource.Type,
			"params":   resource.Params,
			"app_name": appName,
		}

		// Create resource instance in database
		resourceInstance, err := m.resourceRepo.CreateResourceInstance(
			appName, resourceName, resource.Type, config)
		if err != nil {
			return fmt.Errorf("failed to create resource instance %s: %w", resourceName, err)
		}

		// Transition to provisioning state
		err = m.TransitionResourceState(resourceInstance.ID,
			database.ResourceStateProvisioning,
			"Resource created from Score specification",
			createdBy, map[string]interface{}{
				"source": "score_spec",
			})
		if err != nil {
			return fmt.Errorf("failed to transition resource %s to provisioning: %w", resourceName, err)
		}

		fmt.Printf("✅ Created resource instance: %s (%s) - ID: %d\n", resourceName, resource.Type, resourceInstance.ID)
	}

	return nil
}

// TransitionResourceState transitions a resource to a new state with validation
func (m *Manager) TransitionResourceState(resourceID int64, newState database.ResourceLifecycleState, reason, transitionedBy string, metadata map[string]interface{}) error {
	// Get current resource
	resource, err := m.resourceRepo.GetResourceInstance(resourceID)
	if err != nil {
		return fmt.Errorf("failed to get resource: %w", err)
	}

	// Validate state transition
	if !resource.IsValidStateTransition(newState) {
		return fmt.Errorf("invalid state transition from %s to %s", resource.State, newState)
	}

	// Update state with audit trail
	return m.resourceRepo.UpdateResourceInstanceState(resourceID, newState, reason, transitionedBy, metadata)
}

// ProvisionResource provisions a resource instance using registered provisioners
func (m *Manager) ProvisionResource(resourceID int64, providerID string, providerMetadata map[string]interface{}, transitionedBy string) error {
	if err := m.checkRepository(); err != nil {
		return err
	}

	// Get resource instance
	resource, err := m.resourceRepo.GetResourceInstance(resourceID)
	if err != nil {
		return fmt.Errorf("failed to get resource: %w", err)
	}

	// Try to use registered provisioner first
	provisioner, err := m.GetProvisioner(resource.ResourceType)
	if err == nil {
		// Use registered provisioner (kubernetes, gitea-repo, argocd-app)
		fmt.Printf("🔧 Using registered provisioner for resource type '%s'\n", resource.ResourceType)

		err = provisioner.Provision(resource, providerMetadata, transitionedBy)
		if err != nil {
			// Transition to failed state
			_ = m.TransitionResourceState(resourceID, database.ResourceStateFailed,
				fmt.Sprintf("Provisioning failed: %v", err), transitionedBy, nil)
			return fmt.Errorf("provisioning failed: %w", err)
		}

		// Transition to active state on success
		return m.TransitionResourceState(resourceID, database.ResourceStateActive,
			"Resource provisioned successfully", transitionedBy, providerMetadata)
	}

	// Fallback to legacy provisioning methods for other resource types
	switch resource.ResourceType {
	case "postgres":
		return m.provisionPostgres(resource, providerID, providerMetadata, transitionedBy)
	case "redis":
		return m.provisionRedis(resource, providerID, providerMetadata, transitionedBy)
	case "volume":
		return m.provisionVolume(resource, providerID, providerMetadata, transitionedBy)
	case "vault-space":
		return m.provisionVaultSpace(resource, providerID, providerMetadata, transitionedBy)
	default:
		return m.provisionGenericResource(resource, providerID, providerMetadata, transitionedBy)
	}
}

// GetResourcesByApplication retrieves all resources for an application
func (m *Manager) GetResourcesByApplication(appName string) ([]*database.ResourceInstance, error) {
	if err := m.checkRepository(); err != nil {
		return nil, err
	}
	return m.resourceRepo.ListResourceInstances(appName)
}

// GetResource retrieves a specific resource instance
func (m *Manager) GetResource(resourceID int64) (*database.ResourceInstance, error) {
	if err := m.checkRepository(); err != nil {
		return nil, err
	}
	return m.resourceRepo.GetResourceInstance(resourceID)
}

// GetResourceByName retrieves a resource by application and resource name
func (m *Manager) GetResourceByName(appName, resourceName string) (*database.ResourceInstance, error) {
	if err := m.checkRepository(); err != nil {
		return nil, err
	}
	return m.resourceRepo.GetResourceInstanceByName(appName, resourceName)
}

// UpdateResourceHealth updates the health status of a resource
func (m *Manager) UpdateResourceHealth(resourceID int64, healthStatus string, errorMessage *string) error {
	if err := m.checkRepository(); err != nil {
		return err
	}
	return m.resourceRepo.UpdateResourceInstanceHealth(resourceID, healthStatus, errorMessage)
}

// DeleteResource deletes a resource instance
func (m *Manager) DeleteResource(resourceID int64, deletedBy string) error {
	if err := m.checkRepository(); err != nil {
		return err
	}
	// Transition to terminating state first
	err := m.TransitionResourceState(resourceID,
		database.ResourceStateTerminating,
		"Resource deletion requested",
		deletedBy, map[string]interface{}{
			"operation": "delete",
		})
	if err != nil {
		return fmt.Errorf("failed to transition to terminating state: %w", err)
	}

	// Simulate cleanup process
	// In real implementation, this would clean up actual infrastructure

	// Transition to terminated state
	err = m.TransitionResourceState(resourceID,
		database.ResourceStateTerminated,
		"Resource successfully deleted",
		deletedBy, map[string]interface{}{
			"operation": "delete_complete",
		})
	if err != nil {
		return fmt.Errorf("failed to transition to terminated state: %w", err)
	}

	return nil
}

// CheckResourceHealth performs health checks on a resource
func (m *Manager) CheckResourceHealth(resourceID int64) error {
	if err := m.checkRepository(); err != nil {
		return err
	}
	resource, err := m.resourceRepo.GetResourceInstance(resourceID)
	if err != nil {
		return fmt.Errorf("failed to get resource: %w", err)
	}

	// Simulate health check based on resource type
	var healthStatus string
	var errorMessage *string
	var responseTime int64 = 100 // milliseconds

	switch resource.ResourceType {
	case "postgres":
		healthStatus = "healthy"
	case "redis":
		healthStatus = "healthy"
	case "volume":
		healthStatus = "healthy"
	case "vault-space":
		// Check if Vault space is accessible and VSO is syncing secrets
		healthStatus = "healthy"
		// In production, would check Vault connectivity and VSO sync status
	default:
		healthStatus = "unknown"
	}

	// Update health status
	err = m.resourceRepo.UpdateResourceInstanceHealth(resourceID, healthStatus, errorMessage)
	if err != nil {
		return fmt.Errorf("failed to update health status: %w", err)
	}

	// Record health check
	metrics := map[string]interface{}{
		"check_timestamp": "now",
		"resource_type":   resource.ResourceType,
	}

	return m.resourceRepo.CreateHealthCheck(resourceID, "automated", healthStatus, &responseTime, errorMessage, metrics)
}

// GetResourceStateTransitions retrieves state transition history for a resource
func (m *Manager) GetResourceStateTransitions(resourceID int64, limit int) ([]*database.ResourceStateTransition, error) {
	if err := m.checkRepository(); err != nil {
		return nil, err
	}
	return m.resourceRepo.GetResourceStateTransitions(resourceID, limit)
}

// DeprovisionApplication deprovisions all resources for an application (infrastructure teardown)
func (m *Manager) DeprovisionApplication(appName, deprovisionedBy string) error {
	if err := m.checkRepository(); err != nil {
		return err
	}

	fmt.Printf("🧹 Starting deprovision of application: %s\n", appName)

	// Get all resources for the application
	resources, err := m.GetResourcesByApplication(appName)
	if err != nil {
		return fmt.Errorf("failed to get resources for app %s: %w", appName, err)
	}

	if len(resources) == 0 {
		fmt.Printf("No resources found for application: %s\n", appName)
		return nil
	}

	fmt.Printf("Found %d resources to deprovision for app: %s\n", len(resources), appName)

	// Deprovision each resource
	var errors []string
	for _, resource := range resources {
		fmt.Printf("Deprovisioning resource: %s (%s)\n", resource.ResourceName, resource.ResourceType)

		if err := m.DeprovisionResource(resource.ID, deprovisionedBy); err != nil {
			errorMsg := fmt.Sprintf("failed to deprovision resource %s: %v", resource.ResourceName, err)
			errors = append(errors, errorMsg)
			fmt.Printf("❌ %s\n", errorMsg)
		} else {
			fmt.Printf("✅ Successfully deprovisioned resource: %s\n", resource.ResourceName)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("deprovision completed with errors: %v", errors)
	}

	fmt.Printf("✅ Successfully deprovisioned all resources for application: %s\n", appName)
	return nil
}

// DeleteApplication completely removes an application and all its resources from the system
func (m *Manager) DeleteApplication(appName, deletedBy string) error {
	if err := m.checkRepository(); err != nil {
		return err
	}

	fmt.Printf("🗑️  Starting complete deletion of application: %s\n", appName)

	// First deprovision all resources (infrastructure teardown)
	if err := m.DeprovisionApplication(appName, deletedBy); err != nil {
		return fmt.Errorf("failed to deprovision resources for app %s: %w", appName, err)
	}

	// Get all resources for the application to remove from database
	resources, err := m.GetResourcesByApplication(appName)
	if err != nil {
		return fmt.Errorf("failed to get resources for app %s: %w", appName, err)
	}

	// Remove resource records from database
	var errors []string
	for _, resource := range resources {
		if err := m.resourceRepo.DeleteResourceInstance(resource.ID); err != nil {
			errorMsg := fmt.Sprintf("failed to delete resource record %s: %v", resource.ResourceName, err)
			errors = append(errors, errorMsg)
			fmt.Printf("❌ %s\n", errorMsg)
		} else {
			fmt.Printf("✅ Removed resource record: %s\n", resource.ResourceName)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("delete completed with errors: %v", errors)
	}

	fmt.Printf("✅ Successfully deleted application and all resources: %s\n", appName)
	return nil
}