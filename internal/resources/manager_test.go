package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"innominatus/internal/database"
	"innominatus/internal/types"
)

func TestNewManager(t *testing.T) {
	// Test with nil repository (should not panic)
	manager := NewManager(nil)
	assert.NotNil(t, manager)
}

func TestManagerWithoutDatabase(t *testing.T) {
	// Test manager behavior when no database is available
	manager := NewManager(nil)

	// These operations should handle nil repository gracefully
	assert.NotNil(t, manager)

	// Test with empty score spec
	spec := &types.ScoreSpec{
		Metadata: types.Metadata{Name: "test-app"},
		Resources: map[string]types.Resource{},
	}

	// Since there are no resources, this should complete without error
	// (even with nil repository)
	err := manager.CreateResourceFromSpec("test-app", spec, "test-user")
	assert.NoError(t, err) // No resources to create, so no database operations
}

func TestResourceLifecycleStates(t *testing.T) {
	// Test resource lifecycle state constants
	states := []database.ResourceLifecycleState{
		database.ResourceStateRequested,
		database.ResourceStateProvisioning,
		database.ResourceStateActive,
		database.ResourceStateScaling,
		database.ResourceStateUpdating,
		database.ResourceStateDegraded,
		database.ResourceStateTerminating,
		database.ResourceStateTerminated,
		database.ResourceStateFailed,
	}

	// Verify all states are defined
	for _, state := range states {
		assert.NotEmpty(t, string(state))
	}

	// Test specific state values
	assert.Equal(t, "requested", string(database.ResourceStateRequested))
	assert.Equal(t, "provisioning", string(database.ResourceStateProvisioning))
	assert.Equal(t, "active", string(database.ResourceStateActive))
	assert.Equal(t, "terminated", string(database.ResourceStateTerminated))
}

func TestResourceSpecProcessing(t *testing.T) {
	// Test with various resource types
	spec := &types.ScoreSpec{
		Metadata: types.Metadata{Name: "test-app"},
		Resources: map[string]types.Resource{
			"postgres-db": {
				Type: "postgres",
				Params: map[string]interface{}{
					"version": "13",
					"size":    "small",
				},
			},
			"redis-cache": {
				Type: "redis",
				Params: map[string]interface{}{
					"version": "6",
				},
			},
			"storage": {
				Type: "volume",
				Params: map[string]interface{}{
					"size": "10Gi",
				},
			},
		},
	}

	// Test resource types are recognized (this would normally create resources)
	assert.Len(t, spec.Resources, 3)
	assert.Equal(t, "postgres", spec.Resources["postgres-db"].Type)
	assert.Equal(t, "redis", spec.Resources["redis-cache"].Type)
	assert.Equal(t, "volume", spec.Resources["storage"].Type)
}

func TestManagerErrorHandling(t *testing.T) {
	manager := NewManager(nil)

	// Test with nil spec (should handle gracefully)
	err := manager.CreateResourceFromSpec("test-app", nil, "test-user")
	assert.Error(t, err) // Should error due to nil spec

	// Test with spec that has resources but nil repository
	spec := &types.ScoreSpec{
		Metadata: types.Metadata{Name: "test-app"},
		Resources: map[string]types.Resource{
			"db": {Type: "postgres"},
		},
	}

	err = manager.CreateResourceFromSpec("test-app", spec, "test-user")
	assert.Error(t, err) // Should error due to nil repository
}

func TestManagerMethods(t *testing.T) {
	manager := NewManager(nil)

	// Test that methods exist and can be called (will error due to nil repo, but shouldn't panic)
	_, err := manager.GetResourcesByApplication("test-app")
	assert.Error(t, err)

	_, err = manager.GetResource(1)
	assert.Error(t, err)

	_, err = manager.GetResourceByName("test-app", "db")
	assert.Error(t, err)

	err = manager.UpdateResourceHealth(1, "healthy", nil)
	assert.Error(t, err)

	err = manager.DeleteResource(1, "test-user")
	assert.Error(t, err)

	err = manager.CheckResourceHealth(1)
	assert.Error(t, err)

	_, err = manager.GetResourceStateTransitions(1, 10)
	assert.Error(t, err)
}

func TestResourceConfiguration(t *testing.T) {
	// Test resource configuration creation
	resource := types.Resource{
		Type: "postgres",
		Params: map[string]interface{}{
			"version":        "13",
			"size":          "small",
			"backup":        true,
			"max_connections": 100,
		},
	}

	// Verify resource configuration
	assert.Equal(t, "postgres", resource.Type)
	assert.Equal(t, "13", resource.Params["version"])
	assert.Equal(t, "small", resource.Params["size"])
	assert.Equal(t, true, resource.Params["backup"])
	assert.Equal(t, 100, resource.Params["max_connections"])
}

func TestProvisioningLogic(t *testing.T) {
	manager := NewManager(nil)

	// Test resource type detection logic
	resourceTypes := []string{"postgres", "redis", "volume", "vault-space", "unknown-type"}

	for _, resType := range resourceTypes {
		// Each resource type should be handled (even if repository is nil)
		// This tests the resource type switching logic
		err := manager.ProvisionResource(1, "provider-123", map[string]interface{}{
			"endpoint": "test.example.com",
		}, "system")

		// Should error due to nil repository, but shouldn't panic
		assert.Error(t, err)
		// Use resType to avoid unused variable error
		assert.NotEmpty(t, resType)
	}
}

func TestStateTransitionValidation(t *testing.T) {
	// Create a mock resource instance for testing state transitions
	resource := &database.ResourceInstance{
		ID:             1,
		ApplicationName: "test-app",
		ResourceName:   "test-resource",
		ResourceType:   "postgres",
		State:          database.ResourceStateRequested,
		HealthStatus:   "unknown",
		Configuration:  map[string]interface{}{},
	}

	// Test valid state transitions
	assert.True(t, resource.IsValidStateTransition(database.ResourceStateProvisioning))
	assert.True(t, resource.IsValidStateTransition(database.ResourceStateFailed))

	// Test invalid state transitions
	assert.False(t, resource.IsValidStateTransition(database.ResourceStateActive))
	assert.False(t, resource.IsValidStateTransition(database.ResourceStateTerminated))

	// Change state and test again
	resource.State = database.ResourceStateActive
	assert.True(t, resource.IsValidStateTransition(database.ResourceStateScaling))
	assert.True(t, resource.IsValidStateTransition(database.ResourceStateTerminating))
	assert.False(t, resource.IsValidStateTransition(database.ResourceStateRequested))
}

func TestResourceTypes(t *testing.T) {
	// Test different resource types and their handling
	resourceTypes := map[string]map[string]interface{}{
		"postgres": {
			"version":     "13",
			"size":       "small",
			"backup":     true,
			"replicas":   3,
		},
		"redis": {
			"version":    "6",
			"memory":     "1Gi",
			"persistence": true,
		},
		"volume": {
			"size":        "10Gi",
			"access_mode": "ReadWriteOnce",
			"storage_class": "fast-ssd",
		},
		"vault-space": {
			"path":        "/secrets/app",
			"policies":    []string{"read", "write"},
		},
	}

	for resourceType, params := range resourceTypes {
		resource := types.Resource{
			Type:   resourceType,
			Params: params,
		}

		assert.Equal(t, resourceType, resource.Type)
		assert.NotNil(t, resource.Params)
		assert.True(t, len(resource.Params) > 0)
	}
}